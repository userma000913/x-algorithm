package side_effects

import (
	"context"
	"os"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// CacheRequestInfoSideEffect 缓存请求信息供后续使用
// 异步执行，不阻塞主流程
type CacheRequestInfoSideEffect struct {
	stratoClient StratoClient
}

// StratoClient 定义 Strato 客户端接口（用于缓存）
type StratoClient interface {
	// StoreRequestInfo 存储请求信息
	StoreRequestInfo(ctx context.Context, userID int64, postIDs []int64) error
}

// NewCacheRequestInfoSideEffect 创建新的 CacheRequestInfoSideEffect 实例
func NewCacheRequestInfoSideEffect(client StratoClient) *CacheRequestInfoSideEffect {
	return &CacheRequestInfoSideEffect{
		stratoClient: client,
	}
}

// Run 实现 SideEffect 接口
func (s *CacheRequestInfoSideEffect) Run(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) error {
	// 提取帖子ID列表
	postIDs := make([]int64, len(candidates))
	for i, candidate := range candidates {
		postIDs[i] = candidate.TweetID
	}

	// 调用 Strato 客户端存储请求信息
	return s.stratoClient.StoreRequestInfo(ctx, query.UserID, postIDs)
}

// Name 返回 SideEffect 名称
func (s *CacheRequestInfoSideEffect) Name() string {
	return "CacheRequestInfoSideEffect"
}

// Enable 决定是否启用
// 只在生产环境且非 in_network_only 时启用
func (s *CacheRequestInfoSideEffect) Enable(query *pipeline.Query) bool {
	appEnv := os.Getenv("APP_ENV")
	return appEnv == "prod" && !query.InNetworkOnly
}
