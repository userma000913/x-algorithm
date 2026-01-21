package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// IneligibleSubscriptionFilter 移除用户未订阅的订阅内容
// 只保留用户已订阅作者的订阅内容
type IneligibleSubscriptionFilter struct{}

// NewIneligibleSubscriptionFilter 创建新的 IneligibleSubscriptionFilter 实例
func NewIneligibleSubscriptionFilter() *IneligibleSubscriptionFilter {
	return &IneligibleSubscriptionFilter{}
}

// Filter 实现 Filter 接口
func (f *IneligibleSubscriptionFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	// 构建订阅用户ID集合（用于快速查找）
	subscribedSet := make(map[uint64]bool)
	for _, id := range query.UserFeatures.SubscribedUserIDs {
		subscribedSet[uint64(id)] = true
	}

	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		// 如果没有订阅作者ID，保留（不是订阅内容）
		if candidate.SubscriptionAuthorID == nil {
			kept = append(kept, candidate)
			continue
		}

		// 如果是订阅内容，检查用户是否订阅了该作者
		authorID := *candidate.SubscriptionAuthorID
		if subscribedSet[authorID] {
			kept = append(kept, candidate)
		} else {
			removed = append(removed, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *IneligibleSubscriptionFilter) Name() string {
	return "IneligibleSubscriptionFilter"
}

// Enable 决定是否启用（IneligibleSubscriptionFilter 总是启用）
func (f *IneligibleSubscriptionFilter) Enable(query *pipeline.Query) bool {
	return true
}
