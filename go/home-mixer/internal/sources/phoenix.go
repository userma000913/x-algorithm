package sources

import (
	"context"
	"fmt"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// PhoenixSource 从 Phoenix Retrieval 服务获取站外内容（ML 检索）
type PhoenixSource struct {
	// PhoenixRetrievalClient 用于调用 Phoenix Retrieval 服务
	phoenixRetrievalClient PhoenixRetrievalClient
	maxResults             int
}

// PhoenixRetrievalClient 定义 Phoenix Retrieval 客户端接口
type PhoenixRetrievalClient interface {
	// Retrieve 执行检索，返回相关候选
	Retrieve(ctx context.Context, userID uint64, sequence *pipeline.UserActionSequence, maxResults int) (*RetrievalResponse, error)
}

// RetrievalResponse 表示检索响应
type RetrievalResponse struct {
	TopKCandidates []ScoredCandidatesGroup
}

// ScoredCandidatesGroup 表示一组打分的候选
type ScoredCandidatesGroup struct {
	Candidates []ScoredCandidate
}

// ScoredCandidate 表示一个打分的候选
type ScoredCandidate struct {
	Candidate *TweetInfo
}

// TweetInfo 表示帖子信息
type TweetInfo struct {
	TweetID         int64
	AuthorID        uint64
	InReplyToTweetID uint64
}

// NewPhoenixSource 创建新的 PhoenixSource 实例
func NewPhoenixSource(client PhoenixRetrievalClient, maxResults int) *PhoenixSource {
	if maxResults <= 0 {
		maxResults = 500 // 默认值
	}
	return &PhoenixSource{
		phoenixRetrievalClient: client,
		maxResults:             maxResults,
	}
}

// GetCandidates 实现 Source 接口
func (s *PhoenixSource) GetCandidates(ctx context.Context, query *pipeline.Query) ([]*pipeline.Candidate, error) {
	// 检查是否有 user_action_sequence
	if query.UserActionSequence == nil {
		return nil, fmt.Errorf("PhoenixSource: missing user_action_sequence")
	}

	userID := uint64(query.UserID)

	// 调用 Phoenix Retrieval 服务
	response, err := s.phoenixRetrievalClient.Retrieve(ctx, userID, query.UserActionSequence, s.maxResults)
	if err != nil {
		return nil, fmt.Errorf("PhoenixSource: %w", err)
	}

	// 转换为 Candidate
	candidates := make([]*pipeline.Candidate, 0)
	for _, group := range response.TopKCandidates {
		for _, scoredCandidate := range group.Candidates {
			if scoredCandidate.Candidate == nil {
				continue
			}
			
			tweetInfo := scoredCandidate.Candidate
			// 与Rust版本一致：总是设置in_reply_to_tweet_id（即使为0）
			// Rust版本使用Some(tweet_info.in_reply_to_tweet_id)，所以Go版本也应该设置指针
			var inReplyToTweetID *uint64
			if tweetInfo.InReplyToTweetID != 0 {
				val := uint64(tweetInfo.InReplyToTweetID)
				inReplyToTweetID = &val
			} else {
				// 如果为0，也设置为指向0的指针（与Rust的Some(0)语义一致）
				zero := uint64(0)
				inReplyToTweetID = &zero
			}
			
			servedType := int32(1) // ForYouPhoenixRetrieval，实际应该使用常量
			candidate := &pipeline.Candidate{
				TweetID:          tweetInfo.TweetID,
				AuthorID:         tweetInfo.AuthorID,
				InReplyToTweetID: inReplyToTweetID,
				ServedType:       &servedType,
			}
			candidates = append(candidates, candidate)
		}
	}

	return candidates, nil
}

// Name 返回 Source 名称
func (s *PhoenixSource) Name() string {
	return "PhoenixSource"
}

// Enable 决定是否启用（只在非 in_network_only 时启用）
func (s *PhoenixSource) Enable(query *pipeline.Query) bool {
	return !query.InNetworkOnly
}
