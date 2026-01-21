package sources

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// MockThunderClient 是 ThunderClient 的 Mock 实现（用于测试）
type MockThunderClient struct {
	Posts []LightPost
}

// GetInNetworkPosts 实现 ThunderClient 接口
func (m *MockThunderClient) GetInNetworkPosts(ctx context.Context, req *GetInNetworkPostsRequest) (*GetInNetworkPostsResponse, error) {
	// 简单的 mock 实现：返回预设的帖子
	// 实际应该根据 req 进行过滤
	posts := m.Posts
	if len(posts) > req.MaxResults {
		posts = posts[:req.MaxResults]
	}
	return &GetInNetworkPostsResponse{
		Posts: posts,
	}, nil
}

// MockPhoenixRetrievalClient 是 PhoenixRetrievalClient 的 Mock 实现（用于测试）
type MockPhoenixRetrievalClient struct {
	Candidates []TweetInfo
}

// Retrieve 实现 PhoenixRetrievalClient 接口
func (m *MockPhoenixRetrievalClient) Retrieve(ctx context.Context, userID uint64, sequence *pipeline.UserActionSequence, maxResults int) (*RetrievalResponse, error) {
	if sequence == nil {
		return nil, fmt.Errorf("sequence is required")
	}

	// 简单的 mock 实现：返回预设的候选
	candidates := make([]ScoredCandidate, 0, len(m.Candidates))
	for _, tweetInfo := range m.Candidates {
		if len(candidates) >= maxResults {
			break
		}
		candidates = append(candidates, ScoredCandidate{
			Candidate: &tweetInfo,
		})
	}

	return &RetrievalResponse{
		TopKCandidates: []ScoredCandidatesGroup{
			{
				Candidates: candidates,
			},
		},
	}, nil
}
