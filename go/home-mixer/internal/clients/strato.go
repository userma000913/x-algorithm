package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/query_hydrators"
	"github.com/x-algorithm/go/home-mixer/internal/side_effects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StratoClientImpl implements StratoClient interface for query hydrators
type StratoClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewStratoClient creates a new Strato client for query hydrators
func NewStratoClient(address string) (query_hydrators.StratoClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Strato service: %w", err)
	}

	return &StratoClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetUserFeatures implements StratoClient interface
func (c *StratoClientImpl) GetUserFeatures(
	ctx context.Context,
	userID int64,
) (*pipeline.UserFeatures, error) {
	// Mock implementation for local learning/testing
	// Returns test user features (following list)
	
	_ = ctx
	
	// Generate some mock followed users
	followedUserIDs := make([]int64, 10)
	for i := 0; i < 10; i++ {
		// Generate different user IDs
		followedUserIDs[i] = userID + 100 + int64(i*10)
	}
	
	features := &pipeline.UserFeatures{
		FollowedUserIDs: followedUserIDs,
	}
	return features, nil
}

// Close closes the gRPC connection
func (c *StratoClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// StratoClientForCacheImpl implements StratoClient interface for side effects
type StratoClientForCacheImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewStratoClientForCache creates a new Strato client for side effects
func NewStratoClientForCache(address string) (side_effects.StratoClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Strato service: %w", err)
	}

	return &StratoClientForCacheImpl{
		conn:    conn,
		address: address,
	}, nil
}

// StoreRequestInfo implements StratoClient interface for side effects
func (c *StratoClientForCacheImpl) StoreRequestInfo(
	ctx context.Context,
	userID int64,
	postIDs []int64,
) error {
	// TODO: Implement actual Strato caching call
	_ = ctx
	_ = userID
	_ = postIDs
	return nil
}

// Close closes the gRPC connection
func (c *StratoClientForCacheImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
