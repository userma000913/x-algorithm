package filters

import (
	"context"
	"strings"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// CoreDataHydrationFilter 移除核心数据获取失败的候选
// 检查 author_id 和 tweet_text 是否有效
type CoreDataHydrationFilter struct{}

// NewCoreDataHydrationFilter 创建新的 CoreDataHydrationFilter 实例
func NewCoreDataHydrationFilter() *CoreDataHydrationFilter {
	return &CoreDataHydrationFilter{}
}

// Filter 实现 Filter 接口
func (f *CoreDataHydrationFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		// 检查 author_id 和 tweet_text 是否有效
		if candidate.AuthorID == 0 || strings.TrimSpace(candidate.TweetText) == "" {
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
func (f *CoreDataHydrationFilter) Name() string {
	return "CoreDataHydrationFilter"
}

// Enable 决定是否启用（CoreDataHydrationFilter 总是启用）
func (f *CoreDataHydrationFilter) Enable(query *pipeline.Query) bool {
	return true
}
