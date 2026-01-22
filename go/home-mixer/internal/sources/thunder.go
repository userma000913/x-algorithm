package sources

import (
	"context"
	"fmt"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// ThunderSource 从 Thunder 服务获取站内内容（关注账号的帖子）
type ThunderSource struct {
	// ThunderClient 用于调用 Thunder 服务
	// 这里先定义接口，实际客户端可以后续实现
	thunderClient ThunderClient
	maxResults    int
}

// ThunderClient 定义 Thunder 客户端接口
type ThunderClient interface {
	// GetInNetworkPosts 获取站内帖子
	GetInNetworkPosts(ctx context.Context, req *GetInNetworkPostsRequest) (*GetInNetworkPostsResponse, error)
}

// GetInNetworkPostsRequest 表示获取站内帖子的请求
type GetInNetworkPostsRequest struct {
	UserID           uint64
	FollowingUserIDs []uint64
	MaxResults       int
	ExcludeTweetIDs  []int64
	Algorithm        string
	Debug            bool
	IsVideoRequest   bool
}

// GetInNetworkPostsResponse 表示获取站内帖子的响应
type GetInNetworkPostsResponse struct {
	Posts []LightPost
}

// LightPost 表示轻量级帖子信息
type LightPost struct {
	PostID        int64
	AuthorID      uint64
	InReplyToPostID *int64
	ConversationID *int64
}

// NewThunderSource 创建新的 ThunderSource 实例
func NewThunderSource(client ThunderClient, maxResults int) *ThunderSource {
	if maxResults <= 0 {
		maxResults = 500 // 默认值
	}
	return &ThunderSource{
		thunderClient: client,
		maxResults:    maxResults,
	}
}

// GetCandidates 实现 Source 接口
func (s *ThunderSource) GetCandidates(ctx context.Context, query *pipeline.Query) ([]*pipeline.Candidate, error) {
	// 获取关注列表
	followingList := query.UserFeatures.FollowedUserIDs
	if len(followingList) == 0 {
		// 如果没有关注列表，返回空结果
		return []*pipeline.Candidate{}, nil
	}

	// 转换为 uint64
	followingUserIDs := make([]uint64, len(followingList))
	for i, id := range followingList {
		followingUserIDs[i] = uint64(id)
	}

	// 构建请求
	req := &GetInNetworkPostsRequest{
		UserID:           uint64(query.UserID),
		FollowingUserIDs: followingUserIDs,
		MaxResults:       s.maxResults,
		ExcludeTweetIDs:  []int64{}, // 可以从 query.SeenIDs 或 query.ServedIDs 填充
		Algorithm:        "default",
		Debug:            false,
		IsVideoRequest:   false,
	}

	// 调用 Thunder 服务
	response, err := s.thunderClient.GetInNetworkPosts(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ThunderSource: %w", err)
	}

	// 转换为 Candidate
	candidates := make([]*pipeline.Candidate, 0, len(response.Posts))
	for _, post := range response.Posts {
		var inReplyToTweetID *uint64
		if post.InReplyToPostID != nil {
			val := uint64(*post.InReplyToPostID)
			inReplyToTweetID = &val
		}

		// 构建 ancestors（回复链）
		var ancestors []uint64
		if inReplyToTweetID != nil {
			ancestors = append(ancestors, *inReplyToTweetID)
			if post.ConversationID != nil {
				convID := uint64(*post.ConversationID)
				if convID != *inReplyToTweetID {
					ancestors = append(ancestors, convID)
				}
			}
		}

		servedType := int32(0) // ForYouInNetwork，实际应该使用常量
		candidate := &pipeline.Candidate{
			TweetID:          post.PostID,
			AuthorID:         post.AuthorID,
			InReplyToTweetID: inReplyToTweetID,
			Ancestors:        ancestors,
			ServedType:       &servedType,
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// Name 返回 Source 名称
func (s *ThunderSource) Name() string {
	return "ThunderSource"
}

// Enable 决定是否启用（Thunder Source 总是启用）
func (s *ThunderSource) Enable(query *pipeline.Query) bool {
	return true
}
