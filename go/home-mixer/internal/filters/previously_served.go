package filters

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// PreviouslyServedPostsFilter 移除本次会话中已经服务过的帖子
// 只在 is_bottom_request 时启用（分页请求）
type PreviouslyServedPostsFilter struct{}

// NewPreviouslyServedPostsFilter 创建新的 PreviouslyServedPostsFilter 实例
func NewPreviouslyServedPostsFilter() *PreviouslyServedPostsFilter {
	return &PreviouslyServedPostsFilter{}
}

// Filter 实现 Filter 接口
func (f *PreviouslyServedPostsFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	// 构建已服务的ID集合（用于快速查找）
	servedIDsSet := make(map[int64]bool)
	for _, id := range query.ServedIDs {
		servedIDsSet[id] = true
	}

	for _, candidate := range candidates {
		// 获取相关的帖子ID
		relatedIDs := getRelatedPostIDs(candidate)

		// 检查是否有任何相关ID在已服务的列表中
		shouldRemove := false
		for _, id := range relatedIDs {
			if servedIDsSet[id] {
				shouldRemove = true
				break
			}
		}

		if shouldRemove {
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
func (f *PreviouslyServedPostsFilter) Name() string {
	return "PreviouslyServedPostsFilter"
}

// Enable 决定是否启用（只在 is_bottom_request 时启用）
func (f *PreviouslyServedPostsFilter) Enable(query *pipeline.Query) bool {
	return query.IsBottomRequest
}
