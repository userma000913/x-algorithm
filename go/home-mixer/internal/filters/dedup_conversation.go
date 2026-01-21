package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// DedupConversationFilter 对话去重，每个对话分支只保留分数最高的候选
type DedupConversationFilter struct{}

// NewDedupConversationFilter 创建新的 DedupConversationFilter 实例
func NewDedupConversationFilter() *DedupConversationFilter {
	return &DedupConversationFilter{}
}

// Filter 实现 Filter 接口
func (f *DedupConversationFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate
	
	// 记录每个对话的最佳候选（conversation_id -> (index_in_kept, score)）
	bestPerConversation := make(map[uint64]struct {
		index int
		score float64
	})

	for _, candidate := range candidates {
		conversationID := getConversationID(candidate)
		score := 0.0
		if candidate.Score != nil {
			score = *candidate.Score
		}

		// 检查是否已有该对话的候选
		if best, exists := bestPerConversation[conversationID]; exists {
			if score > best.score {
				// 当前候选分数更高，替换之前的
				removed = append(removed, kept[best.index])
				kept[best.index] = candidate
				bestPerConversation[conversationID] = struct {
					index int
					score float64
				}{index: best.index, score: score}
			} else {
				// 当前候选分数较低，移除
				removed = append(removed, candidate)
			}
		} else {
			// 第一次遇到这个对话，保留
			idx := len(kept)
			bestPerConversation[conversationID] = struct {
				index int
				score float64
			}{index: idx, score: score}
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// getConversationID 获取对话ID
// 使用 ancestors 中的最小值，如果没有则使用 tweet_id
func getConversationID(candidate *pipeline.Candidate) uint64 {
	if len(candidate.Ancestors) > 0 {
		minID := candidate.Ancestors[0]
		for _, id := range candidate.Ancestors[1:] {
			if id < minID {
				minID = id
			}
		}
		return minID
	}
	return uint64(candidate.TweetID)
}

// Name 返回 Filter 名称
func (f *DedupConversationFilter) Name() string {
	return "DedupConversationFilter"
}

// Enable 决定是否启用（DedupConversationFilter 总是启用）
func (f *DedupConversationFilter) Enable(query *pipeline.Query) bool {
	return true
}
