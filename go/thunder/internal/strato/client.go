package strato

import (
	"context"
	"fmt"
	"log"
)

// StratoClient 是从 Strato 服务获取用户数据的客户端
type StratoClient struct {
	// 在实际实现中，这将包含 gRPC 客户端或 HTTP 客户端
	// 目前，我们将使用模拟实现
}

// NewStratoClient 创建新的 StratoClient
func NewStratoClient() *StratoClient {
	log.Println("Initialized StratoClient")
	return &StratoClient{}
}

// FetchFollowingList 获取用户的关注列表
// 返回用户关注的用户ID列表
func (c *StratoClient) FetchFollowingList(ctx context.Context, userID int64, maxSize int) ([]int64, error) {
	// TODO: 实现实际的 Strato 客户端调用
	// 这是一个占位实现
	// 在实际实现中，这将向 Strato 服务发起 gRPC 或 HTTP 调用

	log.Printf("FetchFollowingList called for user %d (maxSize=%d) - using mock implementation", userID, maxSize)

	// Mock implementation - return empty list
	// In production, this would call the actual Strato service
	return []int64{}, nil
}

// MockFetchFollowingList 是用于测试的模拟实现
func (c *StratoClient) MockFetchFollowingList(ctx context.Context, userID int64, maxSize int) ([]int64, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID: %d", userID)
	}

	// 返回模拟的关注列表
	// 在实际实现中，这将从 Strato 服务获取
	mockFollowingList := make([]int64, 0, maxSize)
	for i := int64(1); i <= int64(maxSize) && i <= 100; i++ {
		mockFollowingList = append(mockFollowingList, userID*1000+i)
	}

	return mockFollowingList, nil
}
