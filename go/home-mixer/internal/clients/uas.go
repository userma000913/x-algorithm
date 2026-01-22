package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/home-mixer/internal/query_hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UASFetcherImpl 实现 UserActionSequenceFetcher 接口
type UASFetcherImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewUASFetcher 创建一个新的 UAS 获取器客户端
func NewUASFetcher(address string) (query_hydrators.UserActionSequenceFetcher, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 UAS 服务失败: %w", err)
	}

	return &UASFetcherImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetByUserID 实现 UserActionSequenceFetcher 接口
func (c *UASFetcherImpl) GetByUserID(
	ctx context.Context,
	userID int64,
) (*query_hydrators.UserActionSequenceData, error) {
	// 用于本地学习/测试的模拟实现
	// 返回测试用户动作序列（交互历史）
	
	_ = ctx
	
	// 生成一些模拟交互动作
	actions := make([]query_hydrators.UserActionData, 20)
	currentTime := int64(1704067200) // 2024-01-01 00:00:00 UTC
	
	for i := 0; i < 20; i++ {
		actionType := "favorite"
		if i%3 == 0 {
			actionType = "reply"
		} else if i%3 == 1 {
			actionType = "retweet"
		}
		
		actions[i] = query_hydrators.UserActionData{
			ActionType: actionType,
			TweetID:    currentTime + int64(i*100),
			Timestamp:  currentTime - int64(i*3600), // 动作分布在过去 20 小时内
		}
	}
	
	sequence := &query_hydrators.UserActionSequenceData{
		UserID:  userID,
		Actions: actions,
	}
	return sequence, nil
}

// Close 关闭 gRPC 连接
func (c *UASFetcherImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
