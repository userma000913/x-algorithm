package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// VFClientImpl implements VisibilityFilteringClient interface
type VFClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewVFClient creates a new Visibility Filtering client
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

// GetVisibilityResults implements VisibilityFilteringClient interface
func (c *VFClientImpl) GetVisibilityResults(
	ctx context.Context,
	tweetIDs []int64,
	isInNetwork bool,
	userID int64,
) (map[int64]*string, error) {
	// TODO: Implement actual VF gRPC call
	// For now, return all tweet IDs as visible (nil means visible)
	_ = ctx
	_ = isInNetwork
	_ = userID
	results := make(map[int64]*string)
	for _, tweetID := range tweetIDs {
		results[tweetID] = nil // nil means visible
	}
	return results, nil
}

// Close closes the gRPC connection
func (c *VFClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
