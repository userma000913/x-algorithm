package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GizmoduckClientImpl 实现 GizmoduckClient 接口
type GizmoduckClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewGizmoduckClient 创建一个新的 Gizmoduck 客户端
func NewGizmoduckClient(address string) (hydrators.GizmoduckClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 Gizmoduck 服务失败: %w", err)
	}

	return &GizmoduckClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetUsers 实现 GizmoduckClient 接口
func (c *GizmoduckClientImpl) GetUsers(
	ctx context.Context,
	userIDs []int64,
) (map[int64]*hydrators.GizmoduckUserResult, error) {
	// 用于本地学习/测试的模拟实现
	// 返回测试用户资料数据
	
	_ = ctx
	
	result := make(map[int64]*hydrators.GizmoduckUserResult)
	
	for _, userID := range userIDs {
		result[userID] = &hydrators.GizmoduckUserResult{
			User: &hydrators.GizmoduckUser{
				UserID: uint64(userID),
				Profile: &hydrators.GizmoduckUserProfile{
					ScreenName: fmt.Sprintf("user_%d", userID),
				},
				Counts: &hydrators.GizmoduckUserCounts{
					FollowersCount: 1000 + uint32(userID%10000),
				},
			},
		}
	}
	
	return result, nil
}

// Close 关闭 gRPC 连接
func (c *GizmoduckClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
