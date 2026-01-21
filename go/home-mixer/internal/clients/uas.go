package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/home-mixer/internal/query_hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UASFetcherImpl implements UserActionSequenceFetcher interface
type UASFetcherImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewUASFetcher creates a new UAS Fetcher client
func NewUASFetcher(address string) (query_hydrators.UserActionSequenceFetcher, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to UAS service: %w", err)
	}

	return &UASFetcherImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetByUserID implements UserActionSequenceFetcher interface
func (c *UASFetcherImpl) GetByUserID(
	ctx context.Context,
	userID int64,
) (*query_hydrators.UserActionSequenceData, error) {
	// Mock implementation for local learning/testing
	// Returns test user action sequence (engagement history)
	
	_ = ctx
	
	// Generate some mock engagement actions
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
			Timestamp:  currentTime - int64(i*3600), // Actions spread over last 20 hours
		}
	}
	
	sequence := &query_hydrators.UserActionSequenceData{
		UserID:  userID,
		Actions: actions,
	}
	return sequence, nil
}

// Close closes the gRPC connection
func (c *UASFetcherImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
