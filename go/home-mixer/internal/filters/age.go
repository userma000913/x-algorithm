package filters

import (
	"context"
	"time"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/utils"
)

// AgeFilter 过滤掉超过指定年龄的帖子
type AgeFilter struct {
	MaxAge time.Duration
}

// NewAgeFilter 创建新的 AgeFilter 实例
func NewAgeFilter(maxAge time.Duration) *AgeFilter {
	return &AgeFilter{
		MaxAge: maxAge,
	}
}

// Filter 实现 Filter 接口
func (f *AgeFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		if utils.IsWithinAge(candidate.TweetID, f.MaxAge) {
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
func (f *AgeFilter) Name() string {
	return "AgeFilter"
}

// Enable 决定是否启用（AgeFilter 总是启用）
func (f *AgeFilter) Enable(query *pipeline.Query) bool {
	return true
}
