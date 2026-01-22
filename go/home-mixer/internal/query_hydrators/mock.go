package query_hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// MockUserActionSequenceFetcher 是 UserActionSequenceFetcher 的 Mock 实现（用于测试）
type MockUserActionSequenceFetcher struct {
	Sequences map[int64]*UserActionSequenceData
}

// GetByUserID 实现 UserActionSequenceFetcher 接口
func (m *MockUserActionSequenceFetcher) GetByUserID(ctx context.Context, userID int64) (*UserActionSequenceData, error) {
	if m.Sequences == nil {
		m.Sequences = make(map[int64]*UserActionSequenceData)
	}
	
	// 返回预设的数据，如果不存在则返回空序列
	data, ok := m.Sequences[userID]
	if !ok {
		return &UserActionSequenceData{
			UserID:  userID,
			Actions: []UserActionData{},
		}, nil
	}
	return data, nil
}

// MockStratoClient 是 StratoClient 的 Mock 实现（用于测试）
type MockStratoClient struct {
	Features map[int64]*pipeline.UserFeatures
}

// GetUserFeatures 实现 StratoClient 接口
func (m *MockStratoClient) GetUserFeatures(ctx context.Context, userID int64) (*pipeline.UserFeatures, error) {
	if m.Features == nil {
		m.Features = make(map[int64]*pipeline.UserFeatures)
	}
	
	// 返回预设的特征，如果不存在则返回空特征
	features, ok := m.Features[userID]
	if !ok {
		return &pipeline.UserFeatures{}, nil
	}
	return features, nil
}
