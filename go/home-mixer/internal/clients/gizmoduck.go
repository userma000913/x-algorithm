package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GizmoduckClientImpl implements GizmoduckClient interface
type GizmoduckClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewGizmoduckClient creates a new Gizmoduck client
func NewGizmoduckClient(address string) (hydrators.GizmoduckClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Gizmoduck service: %w", err)
	}

	return &GizmoduckClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetUsers implements GizmoduckClient interface
func (c *GizmoduckClientImpl) GetUsers(
	ctx context.Context,
	userIDs []int64,
) (map[int64]*hydrators.GizmoduckUserResult, error) {
	// Mock implementation for local learning/testing
	// Returns test user profile data
	
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

// Close closes the gRPC connection
func (c *GizmoduckClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
