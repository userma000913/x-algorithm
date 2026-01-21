package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// DropDuplicatesFilter 移除重复的帖子（基于 tweet_id）
type DropDuplicatesFilter struct{}

// NewDropDuplicatesFilter 创建新的 DropDuplicatesFilter 实例
func NewDropDuplicatesFilter() *DropDuplicatesFilter {
	return &DropDuplicatesFilter{}
}

// Filter 实现 Filter 接口
func (f *DropDuplicatesFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	seenIDs := make(map[int64]bool)
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		if seenIDs[candidate.TweetID] {
			// 已经见过，移除
			removed = append(removed, candidate)
		} else {
			// 第一次见到，保留
			seenIDs[candidate.TweetID] = true
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *DropDuplicatesFilter) Name() string {
	return "DropDuplicatesFilter"
}

// Enable 决定是否启用（DropDuplicatesFilter 总是启用）
func (f *DropDuplicatesFilter) Enable(query *pipeline.Query) bool {
	return true
}
