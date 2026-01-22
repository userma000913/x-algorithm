package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TESClientImpl 实现 TweetEntityServiceClient 接口
type TESClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewTESClient 创建一个新的 TES 客户端
func NewTESClient(address string) (hydrators.TweetEntityServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接 TES 服务失败: %w", err)
	}

	return &TESClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetTweetCoreDatas 实现 TweetEntityServiceClient 接口
func (c *TESClientImpl) GetTweetCoreDatas(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*hydrators.CoreData, error) {
	// 用于本地学习/测试的模拟实现
	// 返回推文的测试核心数据
	
	_ = ctx
	
	result := make(map[int64]*hydrators.CoreData)
	
	for _, tweetID := range tweetIDs {
		// 从推文 ID 提取作者 ID（简单的模拟逻辑）
		authorID := uint64(tweetID / 1000000)
		if authorID == 0 {
			authorID = uint64(tweetID % 100000)
		}
		
		result[tweetID] = &hydrators.CoreData{
			AuthorID:        authorID,
			Text:            fmt.Sprintf("Mock tweet text for tweet %d", tweetID),
			SourceTweetID:   nil,
			SourceUserID:    nil,
			InReplyToTweetID: nil,
		}
	}
	
	return result, nil
}

// GetTweetMediaEntities 实现 TweetEntityServiceClient 接口
func (c *TESClientImpl) GetTweetMediaEntities(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*hydrators.MediaEntities, error) {
	// 模拟实现 - 为大多数推文返回空的媒体实体
	_ = ctx
	_ = tweetIDs
	return make(map[int64]*hydrators.MediaEntities), nil
}

// GetSubscriptions 实现 TweetEntityServiceClient 接口
func (c *TESClientImpl) GetSubscriptions(
	ctx context.Context,
	userID uint64,
	tweetIDs []int64,
) (map[int64]bool, error) {
	// TODO: 实现实际的 TES gRPC 调用
	_ = ctx
	_ = userID
	_ = tweetIDs
	return make(map[int64]bool), nil
}

// GetSubscriptionAuthorIDs 实现 TweetEntityServiceClient 接口
func (c *TESClientImpl) GetSubscriptionAuthorIDs(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*uint64, error) {
	// 模拟实现 - 返回无订阅作者
	_ = ctx
	_ = tweetIDs
	return make(map[int64]*uint64), nil
}

// Close 关闭 gRPC 连接
func (c *TESClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
