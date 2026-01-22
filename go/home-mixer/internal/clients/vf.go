package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// VFClientImpl 实现 VisibilityFilteringClient 接口
type VFClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewVFClient 创建一个新的可见性过滤客户端
func NewVFClient(address string) (hydrators.VisibilityFilteringClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VF service: %w", err)
	}

	return &VFClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetVisibilityResults 实现 VisibilityFilteringClient 接口
func (c *VFClientImpl) GetVisibilityResults(
	ctx context.Context,
	tweetIDs []int64,
	isInNetwork bool,
	userID int64,
) (map[int64]*string, error) {
	// TODO: 实现实际的 VF gRPC 调用
	// 目前返回所有推文 ID 为可见（nil 表示可见）
	_ = ctx
	_ = isInNetwork
	_ = userID
	results := make(map[int64]*string)
	for _, tweetID := range tweetIDs {
		results[tweetID] = nil // nil 表示可见
	}
	return results, nil
}

// Close 关闭 gRPC 连接
func (c *VFClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
