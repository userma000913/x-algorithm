package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// AuthorSocialgraphFilter 移除来自屏蔽/静音作者的帖子
type AuthorSocialgraphFilter struct{}

// NewAuthorSocialgraphFilter 创建新的 AuthorSocialgraphFilter 实例
func NewAuthorSocialgraphFilter() *AuthorSocialgraphFilter {
	return &AuthorSocialgraphFilter{}
}

// Filter 实现 Filter 接口
func (f *AuthorSocialgraphFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	// 早期返回优化：如果没有屏蔽和静音列表，直接返回所有候选（与Rust版本一致）
	if len(query.UserFeatures.BlockedUserIDs) == 0 && len(query.UserFeatures.MutedUserIDs) == 0 {
		return &pipeline.FilterResult{
			Kept:    candidates,
			Removed: []*pipeline.Candidate{},
		}, nil
	}

	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	// 构建屏蔽和静音作者ID集合（用于快速查找）
	blockedSet := make(map[int64]bool)
	for _, id := range query.UserFeatures.BlockedUserIDs {
		blockedSet[id] = true
	}

	mutedSet := make(map[int64]bool)
	for _, id := range query.UserFeatures.MutedUserIDs {
		mutedSet[id] = true
	}

	for _, candidate := range candidates {
		authorID := int64(candidate.AuthorID)

		// 检查作者是否被屏蔽或静音
		if blockedSet[authorID] || mutedSet[authorID] {
			removed = append(removed, candidate)
		} else {
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *AuthorSocialgraphFilter) Name() string {
	return "AuthorSocialgraphFilter"
}

// Enable 决定是否启用（AuthorSocialgraphFilter 总是启用）
func (f *AuthorSocialgraphFilter) Enable(query *pipeline.Query) bool {
	return true
}
