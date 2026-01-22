package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/candidate-pipeline/pipeline"
	"x-algorithm-go/home-mixer/internal/query_hydrators"
	"x-algorithm-go/home-mixer/internal/side_effects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StratoClientImpl 为查询增强器实现 StratoClient 接口
type StratoClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewStratoClient 为查询增强器创建一个新的 Strato 客户端
func NewStratoClient(address string) (query_hydrators.StratoClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Strato 服务失败: %w", err)
	}

	return &StratoClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetUserFeatures 实现 StratoClient 接口
func (c *StratoClientImpl) GetUserFeatures(
	ctx context.Context,
	userID int64,
) (*pipeline.UserFeatures, error) {
	// 用于本地学习/测试的模拟实现
	// 返回测试用户特征（关注列表）
	
	_ = ctx
	
	// 生成一些模拟关注用户
	followedUserIDs := make([]int64, 10)
	for i := 0; i < 10; i++ {
		// 生成不同的用户 ID
		followedUserIDs[i] = userID + 100 + int64(i*10)
	}
	
	features := &pipeline.UserFeatures{
		FollowedUserIDs: followedUserIDs,
	}
	return features, nil
}

// Close 关闭 gRPC 连接
func (c *StratoClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// StratoClientForCacheImpl 为副作用实现 StratoClient 接口
type StratoClientForCacheImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewStratoClientForCache 为副作用创建一个新的 Strato 客户端
func NewStratoClientForCache(address string) (side_effects.StratoClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Strato 服务失败: %w", err)
	}

	return &StratoClientForCacheImpl{
		conn:    conn,
		address: address,
	}, nil
}

// StoreRequestInfo 为副作用实现 StratoClient 接口
func (c *StratoClientForCacheImpl) StoreRequestInfo(
	ctx context.Context,
	userID int64,
	postIDs []int64,
) error {
	// TODO: 实现实际的 Strato 缓存调用
	_ = ctx
	_ = userID
	_ = postIDs
	return nil
}

// Close 关闭 gRPC 连接
func (c *StratoClientForCacheImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
