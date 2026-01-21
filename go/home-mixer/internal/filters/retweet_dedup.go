package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// RetweetDeduplicationFilter 去重转发，只保留第一次出现的帖子
// （无论是原帖还是转发）
type RetweetDeduplicationFilter struct{}

// NewRetweetDeduplicationFilter 创建新的 RetweetDeduplicationFilter 实例
func NewRetweetDeduplicationFilter() *RetweetDeduplicationFilter {
	return &RetweetDeduplicationFilter{}
}

// Filter 实现 Filter 接口
func (f *RetweetDeduplicationFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	seenTweetIDs := make(map[uint64]bool)
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		if candidate.RetweetedTweetID != nil {
			// 这是一个转发
			retweetedID := *candidate.RetweetedTweetID
			// 如果已经见过这个帖子（作为原帖或转发），则移除
			if seenTweetIDs[retweetedID] {
				removed = append(removed, candidate)
			} else {
				seenTweetIDs[retweetedID] = true
				kept = append(kept, candidate)
			}
		} else {
			// 这是原帖，标记为已见过，这样转发它的帖子会被过滤
			seenTweetIDs[uint64(candidate.TweetID)] = true
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *RetweetDeduplicationFilter) Name() string {
	return "RetweetDeduplicationFilter"
}

// Enable 决定是否启用（RetweetDeduplicationFilter 总是启用）
func (f *RetweetDeduplicationFilter) Enable(query *pipeline.Query) bool {
	return true
}
