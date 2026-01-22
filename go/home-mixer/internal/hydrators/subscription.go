package hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// SubscriptionHydrator 增强候选的订阅状态信息
type SubscriptionHydrator struct {
	tesClient TweetEntityServiceClient
}

// NewSubscriptionHydrator 创建新的 SubscriptionHydrator 实例
func NewSubscriptionHydrator(client TweetEntityServiceClient) *SubscriptionHydrator {
	return &SubscriptionHydrator{
		tesClient: client,
	}
}

// Hydrate 实现 Hydrator 接口
func (h *SubscriptionHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	// 提取所有 tweet_id
	tweetIDs := make([]int64, len(candidates))
	for i, c := range candidates {
		tweetIDs[i] = c.TweetID
	}

	// 批量获取订阅作者ID
	subscriptionAuthorIDs, err := h.tesClient.GetSubscriptionAuthorIDs(ctx, tweetIDs)
	if err != nil {
		return nil, err
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		hydrated[i] = candidate.Clone()

		// 获取订阅作者ID
		if authorID, ok := subscriptionAuthorIDs[candidate.TweetID]; ok && authorID != nil {
			hydrated[i].SubscriptionAuthorID = authorID
		}
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
func (h *SubscriptionHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	if hydrated.SubscriptionAuthorID != nil {
		candidate.SubscriptionAuthorID = hydrated.SubscriptionAuthorID
	}
}

// UpdateAll 批量更新候选的增强字段
func (h *SubscriptionHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *SubscriptionHydrator) Name() string {
	return "SubscriptionHydrator"
}

// Enable 决定是否启用（SubscriptionHydrator 总是启用）
func (h *SubscriptionHydrator) Enable(query *pipeline.Query) bool {
	return true
}
