package filters

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/utils"
)

// MutedKeywordFilter 移除包含用户静音关键词的帖子
// 使用tokenizer进行精确的单词边界匹配
type MutedKeywordFilter struct {
	tokenizer *utils.TweetTokenizer
}

// NewMutedKeywordFilter 创建新的 MutedKeywordFilter 实例
func NewMutedKeywordFilter() *MutedKeywordFilter {
	return &MutedKeywordFilter{
		tokenizer: utils.NewTweetTokenizer(),
	}
}

// Filter 实现 Filter 接口
func (f *MutedKeywordFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	mutedKeywords := query.UserFeatures.MutedKeywords

	// 如果没有静音关键词，直接返回所有候选
	if len(mutedKeywords) == 0 {
		return &pipeline.FilterResult{
			Kept:    candidates,
			Removed: []*pipeline.Candidate{},
		}, nil
	}

	// 使用tokenizer对静音关键词进行分词，创建token序列
	tokenSequences := make([]*utils.TokenSequence, 0, len(mutedKeywords))
	for _, keyword := range mutedKeywords {
		tokens := f.tokenizer.Tokenize(keyword, true) // 使用小写
		if len(tokens) > 0 {
			tokenSequences = append(tokenSequences, utils.NewTokenSequence(tokens))
		}
	}

	// 创建UserMutes和匹配器
	userMutes := utils.NewUserMutes(tokenSequences)
	matcher := utils.NewMatchTweetGroup(userMutes)

	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	// 检查每个候选
	for _, candidate := range candidates {
		// 对推文文本进行分词
		tweetTokens := f.tokenizer.Tokenize(candidate.TweetText, true) // 使用小写
		tweetTokenSequence := utils.NewTokenSequence(tweetTokens)

		// 检查是否匹配任何静音关键词
		if matcher.Matches(tweetTokenSequence) {
			// 匹配静音关键词 - 应该被过滤掉
			removed = append(removed, candidate)
		} else {
			// 不匹配 - 保留
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// Name 返回 Filter 名称
func (f *MutedKeywordFilter) Name() string {
	return "MutedKeywordFilter"
}

// Enable 决定是否启用（MutedKeywordFilter 总是启用）
func (f *MutedKeywordFilter) Enable(query *pipeline.Query) bool {
	return true
}
