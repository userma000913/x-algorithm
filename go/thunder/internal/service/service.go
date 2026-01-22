package service

import (
	"context"
	"log"
	"sort"
	"time"

	"x-algorithm-go/thunder/internal/config"
	"x-algorithm-go/thunder/internal/poststore"
	"x-algorithm-go/thunder/internal/strato"
	"x-algorithm-go/proto/thunder"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnimplementedInNetworkPostsServiceServer must be embedded to have forward compatible implementations
type UnimplementedInNetworkPostsServiceServer struct{}

// ThunderServiceImpl implements the InNetworkPostsService
type ThunderServiceImpl struct {
	UnimplementedInNetworkPostsServiceServer

	// PostStore for retrieving posts by user ID
	postStore *poststore.PostStore

	// StratoClient for fetching following lists when not provided
	stratoClient *strato.StratoClient

	// Semaphore to limit concurrent requests and prevent overload
	requestSemaphore *semaphore.Weighted
}

// NewThunderService creates a new ThunderServiceImpl
func NewThunderService(
	postStore *poststore.PostStore,
	stratoClient *strato.StratoClient,
	maxConcurrentRequests int64,
) *ThunderServiceImpl {
	log.Printf("Initializing ThunderService with max_concurrent_requests=%d", maxConcurrentRequests)
	return &ThunderServiceImpl{
		postStore:        postStore,
		stratoClient:     stratoClient,
		requestSemaphore: semaphore.NewWeighted(maxConcurrentRequests),
	}
}

// GetInNetworkPosts gets posts from users in the network
func (s *ThunderServiceImpl) GetInNetworkPosts(
	ctx context.Context,
	req *thunder.GetInNetworkPostsRequest,
) (*thunder.GetInNetworkPostsResponse, error) {
	// Try to acquire semaphore permit without blocking
	// If we're at capacity, reject immediately with RESOURCE_EXHAUSTED
	if !s.requestSemaphore.TryAcquire(1) {
		return nil, status.Error(codes.ResourceExhausted, "Server at capacity, please retry")
	}
	defer s.requestSemaphore.Release(1)

	startTime := time.Now()

	if req.Debug {
		log.Printf("Received GetInNetworkPosts request: user_id=%d, following_count=%d, exclude_tweet_ids=%d",
			req.UserID, len(req.FollowingUserIDs), len(req.ExcludeTweetIDs))
	}

	// If following_user_id list is empty, fetch it from Strato
	followingUserIDs := req.FollowingUserIDs
	if len(followingUserIDs) == 0 && req.Debug {
		log.Printf("Following list is empty, fetching from Strato for user %d", req.UserID)

		followingList, err := s.stratoClient.FetchFollowingList(ctx, int64(req.UserID), config.MAX_INPUT_LIST_SIZE)
		if err != nil {
			log.Printf("Failed to fetch following list from Strato for user %d: %v", req.UserID, err)
			return nil, status.Errorf(codes.Internal, "Failed to fetch following list: %v", err)
		}

		log.Printf("Fetched %d following users from Strato for user %d", len(followingList), req.UserID)
		followingUserIDs = make([]uint64, len(followingList))
		for i, id := range followingList {
			followingUserIDs[i] = uint64(id)
		}
	}

	// Limit following_user_ids and exclude_tweet_ids to first K entries
	if len(followingUserIDs) > config.MAX_INPUT_LIST_SIZE {
		log.Printf("Limiting following_user_ids from %d to %d entries for user %d",
			len(followingUserIDs), config.MAX_INPUT_LIST_SIZE, req.UserID)
		followingUserIDs = followingUserIDs[:config.MAX_INPUT_LIST_SIZE]
	}

	excludeTweetIDs := req.ExcludeTweetIDs
	if len(excludeTweetIDs) > config.MAX_INPUT_LIST_SIZE {
		log.Printf("Limiting exclude_tweet_ids from %d to %d entries for user %d",
			len(excludeTweetIDs), config.MAX_INPUT_LIST_SIZE, req.UserID)
		excludeTweetIDs = excludeTweetIDs[:config.MAX_INPUT_LIST_SIZE]
	}

	// Default max_results if not specified
	maxResults := int(req.MaxResults)
	if maxResults == 0 {
		if req.IsVideoRequest {
			maxResults = config.MAX_VIDEO_POSTS_TO_RETURN
		} else {
			maxResults = config.MAX_POSTS_TO_RETURN
		}
	}

	// Convert to int64 slices
	followingUserIDsInt64 := make([]int64, len(followingUserIDs))
	for i, id := range followingUserIDs {
		followingUserIDsInt64[i] = int64(id)
	}

	excludeTweetIDsSet := make(map[int64]bool)
	for _, id := range excludeTweetIDs {
		excludeTweetIDsSet[int64(id)] = true
	}

	// Fetch posts
	var allPosts []*thunder.LightPost
	if req.IsVideoRequest {
		allPosts = s.postStore.GetVideosByUsers(
			followingUserIDsInt64,
			excludeTweetIDsSet,
			startTime,
			int64(req.UserID),
		)
	} else {
		allPosts = s.postStore.GetAllPostsByUsers(
			followingUserIDsInt64,
			excludeTweetIDsSet,
			startTime,
			int64(req.UserID),
		)
	}

	// Score posts by recency (newer posts first)
	scoredPosts := scoreRecent(allPosts, maxResults)

	if req.Debug {
		log.Printf("Returning %d posts for user %d", len(scoredPosts), req.UserID)
	}

	// Record metrics
	duration := time.Since(startTime)
	recordMetrics(req, len(scoredPosts), len(allPosts), duration)

	return &thunder.GetInNetworkPostsResponse{
		Posts: scoredPosts,
	}, nil
}

// recordMetrics records metrics for GetInNetworkPosts request
func recordMetrics(req *thunder.GetInNetworkPostsRequest, returnedCount int, foundCount int, duration time.Duration) {
	// TODO: 当指标包可用时集成
	_ = req
	_ = returnedCount
	_ = foundCount
	_ = duration
}

// AnalyzeAndReportPostStatistics analyzes and reports post statistics
func (s *ThunderServiceImpl) AnalyzeAndReportPostStatistics(posts []*thunder.LightPost) {
	if len(posts) == 0 {
		return
	}

	// Calculate statistics
	uniqueAuthors := make(map[int64]bool)
	var oldestTimestamp int64 = 0
	var newestTimestamp int64 = 0
	replyCount := 0

	for _, post := range posts {
		uniqueAuthors[int64(post.AuthorID)] = true
		
		if oldestTimestamp == 0 || post.CreatedAt < oldestTimestamp {
			oldestTimestamp = post.CreatedAt
		}
		if newestTimestamp == 0 || post.CreatedAt > newestTimestamp {
			newestTimestamp = post.CreatedAt
		}
		
		if post.InReplyToPostID != nil && *post.InReplyToPostID != 0 {
			replyCount++
		}
	}

	// Calculate metrics
	freshnessSeconds := int64(0)
	if newestTimestamp > 0 {
		freshnessSeconds = time.Now().Unix() - newestTimestamp
	}
	
	timeRangeSeconds := int64(0)
	if oldestTimestamp > 0 && newestTimestamp > 0 {
		timeRangeSeconds = newestTimestamp - oldestTimestamp
	}
	
	replyRatio := float64(replyCount) / float64(len(posts))
	postsPerAuthor := float64(len(posts)) / float64(len(uniqueAuthors))

	// Log statistics
	log.Printf("Post Statistics: %d posts, %d unique authors, %.2f posts/author, %.2f reply ratio, %d freshness seconds, %d time range seconds",
		len(posts), len(uniqueAuthors), postsPerAuthor, replyRatio, freshnessSeconds, timeRangeSeconds)

	// TODO: 当指标包集成时记录指标
	_ = freshnessSeconds
	_ = timeRangeSeconds
	_ = replyRatio
	_ = len(uniqueAuthors)
	_ = postsPerAuthor
}

// scoreRecent scores posts by recency (created_at timestamp, newer posts first)
func scoreRecent(posts []*thunder.LightPost, maxResults int) []*thunder.LightPost {
	// Sort by created_at descending (newest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].CreatedAt > posts[j].CreatedAt
	})

	// Limit to max results
	if len(posts) > maxResults {
		return posts[:maxResults]
	}
	return posts
}
