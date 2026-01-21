package selectors

import (
	"context"
	"math"
	"sort"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// TopKScoreSelector 按分数排序并选择 Top-K 候选
type TopKScoreSelector struct {
	K int // 要选择的候选数量，0 表示不限制
}

// NewTopKScoreSelector 创建新的 TopKScoreSelector 实例
func NewTopKScoreSelector(k int) *TopKScoreSelector {
	return &TopKScoreSelector{
		K: k,
	}
}

// Select 实现 Selector 接口
func (s *TopKScoreSelector) Select(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) []*pipeline.Candidate {
	// 排序
	sorted := s.Sort(candidates)
	
	// 截断到 Top-K
	if s.K > 0 && len(sorted) > s.K {
		sorted = sorted[:s.K]
	}
	
	return sorted
}

// Name 返回 Selector 名称
func (s *TopKScoreSelector) Name() string {
	return "TopKScoreSelector"
}

// Enable 决定是否启用（TopKScoreSelector 总是启用）
func (s *TopKScoreSelector) Enable(query *pipeline.Query) bool {
	return true
}

// Score 从候选对象中提取分数用于排序
func (s *TopKScoreSelector) Score(candidate *pipeline.Candidate) float64 {
	if candidate.Score != nil {
		return *candidate.Score
	}
	// 如果没有分数，返回负无穷（与Rust版本一致）
	return math.Inf(-1)
}

// Sort 按分数降序排序候选列表
func (s *TopKScoreSelector) Sort(candidates []*pipeline.Candidate) []*pipeline.Candidate {
	// 创建副本以避免修改原切片
	sorted := make([]*pipeline.Candidate, len(candidates))
	copy(sorted, candidates)
	
	// 按分数降序排序
	sort.Slice(sorted, func(i, j int) bool {
		return s.Score(sorted[i]) > s.Score(sorted[j])
	})
	
	return sorted
}

// Size 返回要选择的候选数量
func (s *TopKScoreSelector) Size() *int {
	if s.K > 0 {
		return &s.K
	}
	return nil
}
