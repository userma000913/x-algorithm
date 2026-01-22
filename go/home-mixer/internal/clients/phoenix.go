package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/candidate-pipeline/pipeline"
	"x-algorithm-go/home-mixer/internal/sources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PhoenixRetrievalClientImpl 实现 PhoenixRetrievalClient 接口
type PhoenixRetrievalClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewPhoenixRetrievalClient 创建一个新的 Phoenix 检索客户端
func NewPhoenixRetrievalClient(address string) (sources.PhoenixRetrievalClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Phoenix 检索服务失败: %w", err)
	}

	return &PhoenixRetrievalClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// Retrieve 实现 PhoenixRetrievalClient 接口
func (c *PhoenixRetrievalClientImpl) Retrieve(
	ctx context.Context,
	userID uint64,
	sequence *pipeline.UserActionSequence,
	maxResults int,
) (*sources.RetrievalResponse, error) {
	// 用于本地学习/测试的模拟实现
	// 根据用户动作序列返回测试站外帖子
	
	_ = ctx
	
	// 生成模拟站外候选
	candidates := make([]sources.ScoredCandidate, 0)
	currentTime := int64(1704067200) // 2024-01-01 00:00:00 UTC
	
	// 创建来自随机作者的测试帖子（站外）
	for i := 0; i < maxResults && i < 50; i++ {
		// 生成作者 ID（与 userID 不同以确保站外）
		authorID := uint64(1000000 + i)
		tweetID := int64(authorID)*1000000 + currentTime + int64(i)
		
		candidates = append(candidates, sources.ScoredCandidate{
			Candidate: &sources.TweetInfo{
				TweetID:         tweetID,
				AuthorID:        authorID,
				InReplyToTweetID: 0,
			},
		})
	}
	
	return &sources.RetrievalResponse{
		TopKCandidates: []sources.ScoredCandidatesGroup{
			{
				Candidates: candidates,
			},
		},
	}, nil
}

// Close 关闭 gRPC 连接
func (c *PhoenixRetrievalClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PhoenixRankingClientImpl 实现 PhoenixRankingClient 接口
type PhoenixRankingClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewPhoenixRankingClient 创建一个新的 Phoenix 排序客户端
func NewPhoenixRankingClient(address string) (*PhoenixRankingClientImpl, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Phoenix 排序服务失败: %w", err)
	}

	return &PhoenixRankingClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// Rank 实现 PhoenixRankingClient 接口
func (c *PhoenixRankingClientImpl) Rank(
	ctx context.Context,
	req interface{}, // TODO: 定义正确的请求类型
) (interface{}, error) {
	// 模拟实现 - 这通常会调用 Phoenix 排序服务
	// 对于本地学习，我们返回模拟预测
	_ = ctx
	_ = req
	return nil, fmt.Errorf("Phoenix 排序需要正确的请求类型 - 使用带有模拟客户端的 scorers.NewPhoenixScorer")
}


// Close 关闭 gRPC 连接
func (c *PhoenixRankingClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
