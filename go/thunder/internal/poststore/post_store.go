package poststore

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"x-algorithm-go/proto/thunder"
)

// PostStore is a thread-safe store for posts grouped by user ID
type PostStore struct {
	// Full post data indexed by post_id
	posts sync.Map // map[int64]*thunder.LightPost

	// Maps user_id to a deque of TinyPost references for original posts (non-reply, non-retweet)
	originalPostsByUser sync.Map // map[int64]*PostDeque

	// Maps user_id to a deque of TinyPost references for replies and retweets
	secondaryPostsByUser sync.Map // map[int64]*PostDeque

	// Maps user_id to a deque of TinyPost references for video posts
	videoPostsByUser sync.Map // map[int64]*PostDeque

	// Deleted posts set
	deletedPosts sync.Map // map[int64]bool

	// Retention period for posts in seconds
	retentionSeconds uint64

	// Request timeout for get_posts_by_users iteration (0 = no timeout)
	requestTimeout time.Duration

	// Mutex for operations that need synchronization
	mu sync.RWMutex
}

// PostDeque represents a deque of TinyPost references
type PostDeque struct {
	mu    sync.RWMutex
	posts []*TinyPost
}

// NewPostDeque creates a new PostDeque
func NewPostDeque() *PostDeque {
	return &PostDeque{
		posts: make([]*TinyPost, 0),
	}
}

// PushBack adds a post to the back of the deque
func (d *PostDeque) PushBack(post *TinyPost) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.posts = append(d.posts, post)
}

// Front returns the front post without removing it
func (d *PostDeque) Front() *TinyPost {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.posts) == 0 {
		return nil
	}
	return d.posts[0]
}

// PopFront removes and returns the front post
func (d *PostDeque) PopFront() *TinyPost {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.posts) == 0 {
		return nil
	}
	post := d.posts[0]
	d.posts = d.posts[1:]
	return post
}

// Len returns the length of the deque
func (d *PostDeque) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.posts)
}

// IsEmpty returns true if the deque is empty
func (d *PostDeque) IsEmpty() bool {
	return d.Len() == 0
}

// Iter returns an iterator over posts (from newest to oldest)
func (d *PostDeque) Iter() []*TinyPost {
	d.mu.RLock()
	defer d.mu.RUnlock()
	// Return a copy, reversed (newest first)
	result := make([]*TinyPost, len(d.posts))
	for i := 0; i < len(d.posts); i++ {
		result[i] = d.posts[len(d.posts)-1-i]
	}
	return result
}

// Sort sorts posts by created_at timestamp (oldest first)
func (d *PostDeque) Sort() {
	d.mu.Lock()
	defer d.mu.Unlock()
	sort.Slice(d.posts, func(i, j int) bool {
		return d.posts[i].CreatedAt < d.posts[j].CreatedAt
	})
}

// NewPostStore creates a new empty PostStore with the specified retention period and request timeout
func NewPostStore(retentionSeconds uint64, requestTimeoutMs uint64) *PostStore {
	return &PostStore{
		retentionSeconds: retentionSeconds,
		requestTimeout:   time.Duration(requestTimeoutMs) * time.Millisecond,
	}
}

// MarkAsDeleted marks posts as deleted
func (ps *PostStore) MarkAsDeleted(posts []*thunder.TweetDeleteEvent) {
	for _, post := range posts {
		ps.posts.Delete(post.PostID)
		ps.deletedPosts.Store(post.PostID, true)

		// Add to delete event tracking
		key := int64(-1) // DELETE_EVENT_KEY
		deque := ps.getOrCreateDeque(&ps.originalPostsByUser, key)
		deque.PushBack(NewTinyPost(post.PostID, post.DeletedAt))
	}
}

// InsertPosts inserts posts into the post store
func (ps *PostStore) InsertPosts(posts []*thunder.LightPost) {
	// Filter to keep only posts created in the last retention_seconds and not from the future
	currentTime := time.Now().Unix()
	filteredPosts := make([]*thunder.LightPost, 0, len(posts))

	for _, post := range posts {
		if post.CreatedAt < currentTime &&
			currentTime-post.CreatedAt <= int64(ps.retentionSeconds) {
			filteredPosts = append(filteredPosts, post)
		}
	}

	// Sort remaining posts by created_at timestamp
	sort.Slice(filteredPosts, func(i, j int) bool {
		return filteredPosts[i].CreatedAt < filteredPosts[j].CreatedAt
	})

	ps.insertPostsInternal(filteredPosts)
}

func (ps *PostStore) insertPostsInternal(posts []*thunder.LightPost) {
	for _, post := range posts {
		postID := post.PostID
		authorID := post.AuthorID
		createdAt := post.CreatedAt
		isOriginal := !post.IsReply && !post.IsRetweet

		// Check if post is deleted
		if _, deleted := ps.deletedPosts.Load(postID); deleted {
			continue
		}

		// Store the full post data
		if _, exists := ps.posts.LoadOrStore(postID, post); exists {
			// If already stored, don't add it again
			continue
		}

		// Create a TinyPost reference for the timeline
		tinyPost := NewTinyPost(postID, createdAt)

		// Add to appropriate user's posts timeline
		if isOriginal {
			deque := ps.getOrCreateDeque(&ps.originalPostsByUser, authorID)
			deque.PushBack(tinyPost)
		} else {
			deque := ps.getOrCreateDeque(&ps.secondaryPostsByUser, authorID)
			deque.PushBack(tinyPost)
		}

		// Check if post has video
		videoEligible := post.HasVideo

		// If this is a retweet and the retweeted post has video, mark has_video as true
		if !videoEligible && post.IsRetweet && post.SourcePostID != nil {
			if sourcePostVal, ok := ps.posts.Load(*post.SourcePostID); ok {
				sourcePost := sourcePostVal.(*thunder.LightPost)
				if !sourcePost.IsReply && sourcePost.HasVideo {
					videoEligible = true
				}
			}
		}

		if post.IsReply {
			videoEligible = false
		}

		// Also add to video posts timeline if post has video
		if videoEligible {
			deque := ps.getOrCreateDeque(&ps.videoPostsByUser, authorID)
			deque.PushBack(tinyPost)
		}
	}
}

// getOrCreateDeque gets or creates a PostDeque for the given user ID
func (ps *PostStore) getOrCreateDeque(m *sync.Map, userID int64) *PostDeque {
	if val, ok := m.Load(userID); ok {
		return val.(*PostDeque)
	}
	deque := NewPostDeque()
	if actual, loaded := m.LoadOrStore(userID, deque); loaded {
		return actual.(*PostDeque)
	}
	return deque
}

// GetVideosByUsers retrieves video posts from multiple users
func (ps *PostStore) GetVideosByUsers(
	userIDs []int64,
	excludeTweetIDs map[int64]bool,
	startTime time.Time,
	requestUserID int64,
) []*thunder.LightPost {
	return ps.getPostsFromMap(
		&ps.videoPostsByUser,
		userIDs,
		MAX_VIDEO_POSTS_PER_AUTHOR,
		excludeTweetIDs,
		make(map[int64]bool),
		startTime,
		requestUserID,
	)
}

// GetAllPostsByUsers retrieves all posts from multiple users
func (ps *PostStore) GetAllPostsByUsers(
	userIDs []int64,
	excludeTweetIDs map[int64]bool,
	startTime time.Time,
	requestUserID int64,
) []*thunder.LightPost {
	followingUsersSet := make(map[int64]bool)
	for _, userID := range userIDs {
		followingUsersSet[userID] = true
	}

	allPosts := ps.getPostsFromMap(
		&ps.originalPostsByUser,
		userIDs,
		MAX_ORIGINAL_POSTS_PER_AUTHOR,
		excludeTweetIDs,
		make(map[int64]bool),
		startTime,
		requestUserID,
	)

	secondaryPosts := ps.getPostsFromMap(
		&ps.secondaryPostsByUser,
		userIDs,
		MAX_REPLY_POSTS_PER_AUTHOR,
		excludeTweetIDs,
		followingUsersSet,
		startTime,
		requestUserID,
	)

	allPosts = append(allPosts, secondaryPosts...)
	return allPosts
}

// Import config constants
const (
	MAX_VIDEO_POSTS_PER_AUTHOR   = 5
	MAX_ORIGINAL_POSTS_PER_AUTHOR = 10
	MAX_REPLY_POSTS_PER_AUTHOR    = 5
	MAX_TINY_POSTS_PER_USER_SCAN  = 1000
)

// getPostsFromMap retrieves posts from a specific map
func (ps *PostStore) getPostsFromMap(
	postsMap *sync.Map,
	userIDs []int64,
	maxPerUser int,
	excludeTweetIDs map[int64]bool,
	followingUsers map[int64]bool,
	startTime time.Time,
	requestUserID int64,
) []*thunder.LightPost {
	lightPosts := make([]*thunder.LightPost, 0)
	totalEligible := 0

	for i, userID := range userIDs {
		// Check timeout
		if ps.requestTimeout > 0 && time.Since(startTime) >= ps.requestTimeout {
			log.Printf("Timed out fetching posts for user=%d; Processed: %d/%d",
				requestUserID, i, len(userIDs))
			break
		}

		if dequeVal, ok := postsMap.Load(userID); ok {
			deque := dequeVal.(*PostDeque)
			tinyPosts := deque.Iter()
			totalEligible += len(tinyPosts)

			// Filter and process posts
			count := 0
			for _, tinyPost := range tinyPosts {
				if count >= MAX_TINY_POSTS_PER_USER_SCAN {
					break
				}
				if excludeTweetIDs[tinyPost.PostID] {
					continue
				}

				// Get full post data
				if postVal, ok := ps.posts.Load(tinyPost.PostID); ok {
					post := postVal.(*thunder.LightPost)

					// Check if deleted
					if _, deleted := ps.deletedPosts.Load(post.PostID); deleted {
						continue
					}

					// Filter retweets from request user
					if post.IsRetweet && post.SourceUserID != nil && *post.SourceUserID == requestUserID {
						continue
					}

					// Filter replies based on following users
					if len(followingUsers) > 0 {
						if post.InReplyToPostID != nil {
							if repliedToVal, ok := ps.posts.Load(*post.InReplyToPostID); ok {
								repliedToPost := repliedToVal.(*thunder.LightPost)
								if !repliedToPost.IsRetweet && !repliedToPost.IsReply {
									// Original post, include
								} else {
									// Check if reply to original or followed user
									if post.ConversationID != nil {
										replyToOriginal := repliedToPost.InReplyToPostID != nil &&
											*repliedToPost.InReplyToPostID == *post.ConversationID
										replyToFollowed := post.InReplyToUserID != nil &&
											followingUsers[*post.InReplyToUserID]
										if !(replyToOriginal && replyToFollowed) {
											continue
										}
									} else {
										continue
									}
								}
							} else {
								continue
							}
						}
					}

					lightPosts = append(lightPosts, post)
					count++
					if len(lightPosts) >= maxPerUser*len(userIDs) {
						break
					}
				}
			}
		}
	}

	return lightPosts
}

// FinalizeInit finalizes initialization by sorting and trimming
func (ps *PostStore) FinalizeInit(ctx context.Context) error {
	if err := ps.SortAllUserPosts(ctx); err != nil {
		return err
	}
	ps.TrimOldPosts(ctx)

	// Remove deleted posts
	ps.deletedPosts.Range(func(key, value interface{}) bool {
		ps.posts.Delete(key)
		return true
	})

	return nil
}

// SortAllUserPosts sorts all user post lists by creation time (oldest first)
func (ps *PostStore) SortAllUserPosts(ctx context.Context) error {
	ps.originalPostsByUser.Range(func(key, value interface{}) bool {
		deque := value.(*PostDeque)
		deque.Sort()
		return true
	})

	ps.secondaryPostsByUser.Range(func(key, value interface{}) bool {
		deque := value.(*PostDeque)
		deque.Sort()
		return true
	})

	ps.videoPostsByUser.Range(func(key, value interface{}) bool {
		deque := value.(*PostDeque)
		deque.Sort()
		return true
	})

	return nil
}

// TrimOldPosts removes posts older than retention period
func (ps *PostStore) TrimOldPosts(ctx context.Context) int {
	currentTime := time.Now().Unix()
	totalTrimmed := 0

	trimMap := func(postsByUser *sync.Map) int {
		trimmed := 0
		var keysToRemove []int64

		postsByUser.Range(func(key, value interface{}) bool {
			userID := key.(int64)
			deque := value.(*PostDeque)

			for {
				front := deque.Front()
				if front == nil {
					break
				}
				if currentTime-int64(ps.retentionSeconds) > front.CreatedAt {
					deque.PopFront()
					ps.posts.Delete(front.PostID)
					if userID == -1 { // DELETE_EVENT_KEY
						ps.deletedPosts.Delete(front.PostID)
					}
					trimmed++
				} else {
					break
				}
			}

			if deque.IsEmpty() {
				keysToRemove = append(keysToRemove, userID)
			}
			return true
		})

		for _, key := range keysToRemove {
			postsByUser.Delete(key)
		}

		return trimmed
	}

	totalTrimmed += trimMap(&ps.originalPostsByUser)
	totalTrimmed += trimMap(&ps.secondaryPostsByUser)
	trimMap(&ps.videoPostsByUser)

	return totalTrimmed
}

// StartAutoTrim starts a background task that periodically trims old posts
func (ps *PostStore) StartAutoTrim(ctx context.Context, intervalMinutes uint64) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				trimmed := ps.TrimOldPosts(ctx)
				if trimmed > 0 {
					log.Printf("Auto-trim: removed %d old posts", trimmed)
				}
			}
		}
	}()
}

// StartStatsLogger starts a background task that periodically logs PostStore statistics
func (ps *PostStore) StartStatsLogger(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ps.logStats()
			}
		}
	}()
}

// logStats logs PostStore statistics and updates metrics
func (ps *PostStore) logStats() {
	userCount := 0
	totalPosts := 0
	deletedPosts := 0
	originalPostsCount := 0
	secondaryPostsCount := 0
	videoPostsCount := 0

	// Count users
	ps.originalPostsByUser.Range(func(key, value interface{}) bool {
		userCount++
		deque := value.(*PostDeque)
		originalPostsCount += deque.Len()
		return true
	})

	// Count posts
	ps.posts.Range(func(key, value interface{}) bool {
		totalPosts++
		return true
	})

	// Count deleted posts
	ps.deletedPosts.Range(func(key, value interface{}) bool {
		deletedPosts++
		return true
	})

	// Count secondary posts
	ps.secondaryPostsByUser.Range(func(key, value interface{}) bool {
		deque := value.(*PostDeque)
		secondaryPostsCount += deque.Len()
		return true
	})

	// Count video posts
	ps.videoPostsByUser.Range(func(key, value interface{}) bool {
		deque := value.(*PostDeque)
		videoPostsCount += deque.Len()
		return true
	})

	// Update metrics (if metrics package is available)
	// TODO: 当指标包可用时集成
	_ = userCount
	_ = totalPosts
	_ = deletedPosts
	_ = originalPostsCount
	_ = secondaryPostsCount
	_ = videoPostsCount

	log.Printf("PostStore Stats: %d users, %d total posts, %d deleted posts, %d original posts, %d secondary posts, %d video posts",
		userCount, totalPosts, deletedPosts, originalPostsCount, secondaryPostsCount, videoPostsCount)
}

// Clear clears all posts from the store
func (ps *PostStore) Clear() {
	ps.posts.Range(func(key, value interface{}) bool {
		ps.posts.Delete(key)
		return true
	})
	ps.originalPostsByUser.Range(func(key, value interface{}) bool {
		ps.originalPostsByUser.Delete(key)
		return true
	})
	ps.secondaryPostsByUser.Range(func(key, value interface{}) bool {
		ps.secondaryPostsByUser.Delete(key)
		return true
	})
	ps.videoPostsByUser.Range(func(key, value interface{}) bool {
		ps.videoPostsByUser.Delete(key)
		return true
	})
	log.Println("PostStore cleared")
}
