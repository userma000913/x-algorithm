package query_hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// UserFeaturesQueryHydrator 增强查询，添加用户特征（关注列表、屏蔽列表等）
type UserFeaturesQueryHydrator struct {
	stratoClient StratoClient
}

// StratoClient 定义 Strato 客户端接口
type StratoClient interface {
	// GetUserFeatures 获取用户特征
	GetUserFeatures(ctx context.Context, userID int64) (*pipeline.UserFeatures, error)
}

// NewUserFeaturesQueryHydrator 创建新的 UserFeaturesQueryHydrator 实例
func NewUserFeaturesQueryHydrator(client StratoClient) *UserFeaturesQueryHydrator {
	return &UserFeaturesQueryHydrator{
		stratoClient: client,
	}
}

// Hydrate 实现 QueryHydrator 接口
func (h *UserFeaturesQueryHydrator) Hydrate(ctx context.Context, query *pipeline.Query) (*pipeline.Query, error) {
	// 获取用户特征
	userFeatures, err := h.stratoClient.GetUserFeatures(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	// 返回增强后的查询
	return &pipeline.Query{
		UserFeatures: *userFeatures,
	}, nil
}

// Update 更新查询对象的增强字段
func (h *UserFeaturesQueryHydrator) Update(query *pipeline.Query, hydrated *pipeline.Query) {
	query.UserFeatures = hydrated.UserFeatures
}

// Name 返回 QueryHydrator 名称
func (h *UserFeaturesQueryHydrator) Name() string {
	return "UserFeaturesQueryHydrator"
}

// Enable 决定是否启用（UserFeaturesQueryHydrator 总是启用）
func (h *UserFeaturesQueryHydrator) Enable(query *pipeline.Query) bool {
	return true
}
