package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/utils"
)

// PreviouslySeenPostsFilter 移除用户已经看过的帖子
// 使用 seen_ids 和 bloom_filter_entries 来判断
type PreviouslySeenPostsFilter struct{}

// NewPreviouslySeenPostsFilter 创建新的 PreviouslySeenPostsFilter 实例
func NewPreviouslySeenPostsFilter() *PreviouslySeenPostsFilter {
	return &PreviouslySeenPostsFilter{}
}

// Filter 实现 Filter 接口
func (f *PreviouslySeenPostsFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	// 构建已看过的ID集合（用于快速查找）
	seenIDsSet := make(map[int64]bool)
	for _, id := range query.SeenIDs {
		seenIDsSet[id] = true
	}

	// 从 bloom_filter_entries 构建 Bloom Filter 列表
	bloomFilters := make([]*utils.BloomFilter, 0, len(query.BloomFilterEntries))
	for _, entry := range query.BloomFilterEntries {
		bf := utils.NewBloomFilterFromEntry(entry)
		if bf != nil {
			bloomFilters = append(bloomFilters, bf)
		}
	}

	// 检查每个候选
	for _, candidate := range candidates {
		// 获取相关的帖子ID（包括原帖、转发、回复等）
		relatedIDs := getRelatedPostIDs(candidate)

		// 检查是否有任何相关ID在已看过的列表中或Bloom Filter中
		shouldRemove := false
		for _, id := range relatedIDs {
			// 首先检查精确的seen_ids
			if seenIDsSet[id] {
				shouldRemove = true
				break
			}

			// 然后检查Bloom Filter
			for _, bf := range bloomFilters {
				if bf.MayContain(id) {
					shouldRemove = true
					break
				}
			}
			if shouldRemove {
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

// getRelatedPostIDs 获取候选相关的所有帖子ID
func getRelatedPostIDs(candidate *pipeline.Candidate) []int64 {
	ids := []int64{candidate.TweetID}
	
	if candidate.RetweetedTweetID != nil {
		ids = append(ids, int64(*candidate.RetweetedTweetID))
	}
	if candidate.InReplyToTweetID != nil {
		ids = append(ids, int64(*candidate.InReplyToTweetID))
	}
	
	return ids
}

// Name 返回 Filter 名称
func (f *PreviouslySeenPostsFilter) Name() string {
	return "PreviouslySeenPostsFilter"
}

// Enable 决定是否启用（PreviouslySeenPostsFilter 总是启用）
func (f *PreviouslySeenPostsFilter) Enable(query *pipeline.Query) bool {
	return true
}
