package clients

import (
	"context"
	"fmt"

	"x-algorithm-go/home-mixer/internal/sources"
	"x-algorithm-go/proto/thunder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InNetworkPostsServiceClient 是 Thunder 服务的客户端接口
type InNetworkPostsServiceClient interface {
	GetInNetworkPosts(ctx context.Context, req *thunder.GetInNetworkPostsRequest, opts ...grpc.CallOption) (*thunder.GetInNetworkPostsResponse, error)
}

// thunderClientWrapper 包装 gRPC 连接并实现 ThunderClient
type thunderClientWrapper struct {
	conn *grpc.ClientConn
}

// ThunderClientImpl 使用 gRPC 实现 ThunderClient 接口
type ThunderClientImpl struct {
	conn   *grpc.ClientConn
	client InNetworkPostsServiceClient
}

// NewThunderClient 创建一个新的 Thunder gRPC 客户端
func NewThunderClient(address string) (sources.ThunderClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Thunder service: %w", err)
	}

	// 创建一个实现接口的客户端包装器
	client := &thunderClientWrapper{conn: conn}

	return &ThunderClientImpl{
		conn:   conn,
		client: client,
	}, nil
}

// GetInNetworkPosts 为包装器实现 ThunderClient 接口
func (w *thunderClientWrapper) GetInNetworkPosts(ctx context.Context, req *thunder.GetInNetworkPostsRequest, opts ...grpc.CallOption) (*thunder.GetInNetworkPostsResponse, error) {
	// 这是一个占位符实现
	// 在生产环境中，应该使用 proto 生成的客户端：
	// client := thunder.NewInNetworkPostsServiceClient(w.conn)
	// return client.GetInNetworkPosts(ctx, req, opts...)
	
	// 目前返回空响应以允许编译
	_ = ctx
	_ = req
	_ = opts
	return &thunder.GetInNetworkPostsResponse{
		Posts: []*thunder.LightPost{},
	}, nil
}

// GetInNetworkPosts 实现 ThunderClient 接口
func (c *ThunderClientImpl) GetInNetworkPosts(
	ctx context.Context,
	req *sources.GetInNetworkPostsRequest,
) (*sources.GetInNetworkPostsResponse, error) {
	// 用于本地学习/测试的模拟实现
	// 返回来自关注用户的测试帖子
	
	_ = ctx
	
	// 生成来自关注用户的模拟帖子
	posts := make([]sources.LightPost, 0)
	currentTime := int64(1704067200) // 2024-01-01 00:00:00 UTC
	
	// 创建一些来自关注用户的测试帖子
	for i, authorID := range req.FollowingUserIDs {
		if i >= req.MaxResults {
			break
		}
		
		// 生成一个推文 ID（简单的雪花 ID）
		tweetID := int64(authorID)*1000000 + currentTime + int64(i)
		
		posts = append(posts, sources.LightPost{
			PostID:        tweetID,
			AuthorID:      authorID,
			InReplyToPostID: nil,
			ConversationID: &tweetID,
		})
	}

	return &sources.GetInNetworkPostsResponse{
		Posts: posts,
	}, nil
}

// Close 关闭 gRPC 连接
func (c *ThunderClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
