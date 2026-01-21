package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// SelfTweetFilter 移除用户自己发的帖子
type SelfTweetFilter struct{}

// NewSelfTweetFilter 创建新的 SelfTweetFilter 实例
func NewSelfTweetFilter() *SelfTweetFilter {
	return &SelfTweetFilter{}
}

// Filter 实现 Filter 接口
func (f *SelfTweetFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	viewerID := uint64(query.UserID)
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		if candidate.AuthorID == viewerID {
			// 作者是查看者自己，移除
			removed = append(removed, candidate)
		} else {
			// 保留
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *SelfTweetFilter) Name() string {
	return "SelfTweetFilter"
}

// Enable 决定是否启用（SelfTweetFilter 总是启用）
func (f *SelfTweetFilter) Enable(query *pipeline.Query) bool {
	return true
}
