package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/home-mixer/internal/hydrators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TESClientImpl implements TweetEntityServiceClient interface
type TESClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewTESClient creates a new TES client
func NewTESClient(address string) (hydrators.TweetEntityServiceClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to TES service: %w", err)
	}

	return &TESClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// GetTweetCoreDatas implements TweetEntityServiceClient interface
func (c *TESClientImpl) GetTweetCoreDatas(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*hydrators.CoreData, error) {
	// Mock implementation for local learning/testing
	// Returns test core data for tweets
	
	_ = ctx
	
	result := make(map[int64]*hydrators.CoreData)
	
	for _, tweetID := range tweetIDs {
		// Extract author ID from tweet ID (simple mock logic)
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

// GetTweetMediaEntities implements TweetEntityServiceClient interface
func (c *TESClientImpl) GetTweetMediaEntities(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*hydrators.MediaEntities, error) {
	// Mock implementation - return empty media entities for most tweets
	_ = ctx
	_ = tweetIDs
	return make(map[int64]*hydrators.MediaEntities), nil
}

// GetSubscriptions implements TweetEntityServiceClient interface
func (c *TESClientImpl) GetSubscriptions(
	ctx context.Context,
	userID uint64,
	tweetIDs []int64,
) (map[int64]bool, error) {
	// TODO: Implement actual TES gRPC call
	_ = ctx
	_ = userID
	_ = tweetIDs
	return make(map[int64]bool), nil
}

// GetSubscriptionAuthorIDs implements TweetEntityServiceClient interface
func (c *TESClientImpl) GetSubscriptionAuthorIDs(
	ctx context.Context,
	tweetIDs []int64,
) (map[int64]*uint64, error) {
	// Mock implementation - return no subscription authors
	_ = ctx
	_ = tweetIDs
	return make(map[int64]*uint64), nil
}

// Close closes the gRPC connection
func (c *TESClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
