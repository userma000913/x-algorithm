package scorers

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// OONScorer 调整站外内容（Out-of-Network）的分数
// 优先显示站内内容，降低站外内容的分数
type OONScorer struct {
	OONWeightFactor float64 // 站外内容的权重因子（通常 < 1.0）
}

// DefaultOONScorer 创建默认的 OONScorer
func DefaultOONScorer() *OONScorer {
	return &OONScorer{
		OONWeightFactor: 0.9, // 默认站外内容权重为 0.9
	}
}

// NewOONScorer 创建新的 OONScorer 实例
func NewOONScorer(weightFactor float64) *OONScorer {
	return &OONScorer{
		OONWeightFactor: weightFactor,
	}
}

// Score 实现 Scorer 接口
func (s *OONScorer) Score(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	scored := make([]*pipeline.Candidate, len(candidates))

	for i, candidate := range candidates {
		// 克隆候选
		scored[i] = candidate.Clone()

		// 如果是站外内容，调整分数
		if candidate.Score != nil {
			if candidate.InNetwork != nil && !*candidate.InNetwork {
				// 站外内容，应用权重因子
				adjustedScore := *candidate.Score * s.OONWeightFactor
				scored[i].Score = &adjustedScore
			}
			// 站内内容保持原分数
		}
	}

	return scored, nil
}

// Update 更新单个候选的打分字段
func (s *OONScorer) Update(candidate *pipeline.Candidate, scored *pipeline.Candidate) {
	if scored.Score != nil {
		candidate.Score = scored.Score
	}
}

// UpdateAll 批量更新候选的打分字段
func (s *OONScorer) UpdateAll(candidates []*pipeline.Candidate, scored []*pipeline.Candidate) {
	if len(candidates) != len(scored) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		s.Update(candidates[i], scored[i])
	}
}

// Name 返回 Scorer 名称
func (s *OONScorer) Name() string {
	return "OONScorer"
}

// Enable 决定是否启用（OONScorer 总是启用）
func (s *OONScorer) Enable(query *pipeline.Query) bool {
	return true
}
