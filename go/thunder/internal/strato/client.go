package strato

import (
	"context"
	"fmt"
	"log"
)

// StratoClient is a client for fetching user data from Strato service
type StratoClient struct {
	// In a real implementation, this would contain gRPC client or HTTP client
	// For now, we'll use a mock implementation
}

// NewStratoClient creates a new StratoClient
func NewStratoClient() *StratoClient {
	log.Println("Initialized StratoClient")
	return &StratoClient{}
}

// FetchFollowingList fetches the following list for a user
// Returns a list of user IDs that the user follows
func (c *StratoClient) FetchFollowingList(ctx context.Context, userID int64, maxSize int) ([]int64, error) {
	// TODO: Implement actual Strato client call
	// This is a placeholder implementation
	// In a real implementation, this would make a gRPC or HTTP call to Strato service

	log.Printf("FetchFollowingList called for user %d (maxSize=%d) - using mock implementation", userID, maxSize)

	// Mock implementation - return empty list
	// In production, this would call the actual Strato service
	return []int64{}, nil
}

// MockFetchFollowingList is a mock implementation for testing
func (c *StratoClient) MockFetchFollowingList(ctx context.Context, userID int64, maxSize int) ([]int64, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", userID)
	}

	// Return a mock following list
	// In real implementation, this would be fetched from Strato service
	mockFollowingList := make([]int64, 0, maxSize)
	for i := int64(1); i <= int64(maxSize) && i <= 100; i++ {
		mockFollowingList = append(mockFollowingList, userID*1000+i)
	}

	return mockFollowingList, nil
}
