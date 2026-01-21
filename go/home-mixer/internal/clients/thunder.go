package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/home-mixer/internal/sources"
	"github.com/x-algorithm/go/pkg/proto/thunder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InNetworkPostsServiceClient is the client interface for Thunder service
type InNetworkPostsServiceClient interface {
	GetInNetworkPosts(ctx context.Context, req *thunder.GetInNetworkPostsRequest, opts ...grpc.CallOption) (*thunder.GetInNetworkPostsResponse, error)
}

// thunderClientWrapper wraps gRPC connection and implements ThunderClient
type thunderClientWrapper struct {
	conn *grpc.ClientConn
}

// ThunderClientImpl implements ThunderClient interface using gRPC
type ThunderClientImpl struct {
	conn   *grpc.ClientConn
	client InNetworkPostsServiceClient
}

// NewThunderClient creates a new Thunder gRPC client
func NewThunderClient(address string) (sources.ThunderClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Thunder service: %w", err)
	}

	// Create a client wrapper that implements the interface
	client := &thunderClientWrapper{conn: conn}

	return &ThunderClientImpl{
		conn:   conn,
		client: client,
	}, nil
}

// GetInNetworkPosts implements ThunderClient interface for wrapper
func (w *thunderClientWrapper) GetInNetworkPosts(ctx context.Context, req *thunder.GetInNetworkPostsRequest, opts ...grpc.CallOption) (*thunder.GetInNetworkPostsResponse, error) {
	// This is a placeholder implementation
	// In production, this would use the proto-generated client:
	// client := thunder.NewInNetworkPostsServiceClient(w.conn)
	// return client.GetInNetworkPosts(ctx, req, opts...)
	
	// For now, return empty response to allow compilation
	_ = ctx
	_ = req
	_ = opts
	return &thunder.GetInNetworkPostsResponse{
		Posts: []*thunder.LightPost{},
	}, nil
}

// GetInNetworkPosts implements ThunderClient interface
func (c *ThunderClientImpl) GetInNetworkPosts(
	ctx context.Context,
	req *sources.GetInNetworkPostsRequest,
) (*sources.GetInNetworkPostsResponse, error) {
	// Mock implementation for local learning/testing
	// Returns test posts from followed users
	
	_ = ctx
	
	// Generate mock posts from followed users
	posts := make([]sources.LightPost, 0)
	currentTime := int64(1704067200) // 2024-01-01 00:00:00 UTC
	
	// Create some test posts from followed users
	for i, authorID := range req.FollowingUserIDs {
		if i >= req.MaxResults {
			break
		}
		
		// Generate a tweet ID (simple snowflake-like ID)
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

// Close closes the gRPC connection
func (c *ThunderClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
