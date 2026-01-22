package scorers

import (
	"context"
	"math"
	"sort"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// AuthorDiversityScorer 调整分数以确保 Feed 中作者多样性
// 重复出现的作者分数会衰减
type AuthorDiversityScorer struct {
	DecayFactor float64 // 衰减因子（0-1之间）
	Floor       float64 // 最低倍数（衰减的下限）
}

// DefaultAuthorDiversityScorer 创建默认的 AuthorDiversityScorer
func DefaultAuthorDiversityScorer() *AuthorDiversityScorer {
	return &AuthorDiversityScorer{
		DecayFactor: 0.8, // 默认衰减因子
		Floor:       0.5,  // 默认最低倍数
	}
}

// NewAuthorDiversityScorer 创建新的 AuthorDiversityScorer 实例
func NewAuthorDiversityScorer(decayFactor, floor float64) *AuthorDiversityScorer {
	return &AuthorDiversityScorer{
		DecayFactor: decayFactor,
		Floor:       floor,
	}
}

// multiplier 计算给定位置的倍数
func (s *AuthorDiversityScorer) multiplier(position int) float64 {
	return (1.0-s.Floor)*math.Pow(s.DecayFactor, float64(position)) + s.Floor
}

// Score 实现 Scorer 接口
func (s *AuthorDiversityScorer) Score(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	scored := make([]*pipeline.Candidate, len(candidates))
	authorCounts := make(map[uint64]int)

	// 创建索引和候选的配对，并按加权分数排序
	type indexedCandidate struct {
		index     int
		candidate *pipeline.Candidate
	}
	indexed := make([]indexedCandidate, len(candidates))
	for i, c := range candidates {
		indexed[i] = indexedCandidate{index: i, candidate: c}
	}

	// 按加权分数降序排序
	sort.Slice(indexed, func(i, j int) bool {
		scoreI := 0.0
		if indexed[i].candidate.WeightedScore != nil {
			scoreI = *indexed[i].candidate.WeightedScore
		}
		scoreJ := 0.0
		if indexed[j].candidate.WeightedScore != nil {
			scoreJ = *indexed[j].candidate.WeightedScore
		}
		return scoreI > scoreJ
	})

	// 按顺序处理每个候选，应用多样性调整
	for _, item := range indexed {
		originalIdx := item.index
		candidate := item.candidate

		// 获取该作者的当前出现次数
		position := authorCounts[candidate.AuthorID]
		authorCounts[candidate.AuthorID]++

		// 计算调整倍数
		multiplier := s.multiplier(position)

		// 应用调整
		var adjustedScore *float64
		if candidate.WeightedScore != nil {
			adjusted := *candidate.WeightedScore * multiplier
			adjustedScore = &adjusted
		}

		// 创建更新后的候选
		scored[originalIdx] = candidate.Clone()
		scored[originalIdx].Score = adjustedScore
	}

	return scored, nil
}

// Update 更新单个候选的打分字段
func (s *AuthorDiversityScorer) Update(candidate *pipeline.Candidate, scored *pipeline.Candidate) {
	if scored.Score != nil {
		candidate.Score = scored.Score
	}
}

// UpdateAll 批量更新候选的打分字段
func (s *AuthorDiversityScorer) UpdateAll(candidates []*pipeline.Candidate, scored []*pipeline.Candidate) {
	if len(candidates) != len(scored) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		s.Update(candidates[i], scored[i])
	}
}

// Name 返回 Scorer 名称
func (s *AuthorDiversityScorer) Name() string {
	return "AuthorDiversityScorer"
}

// Enable 决定是否启用（AuthorDiversityScorer 总是启用）
func (s *AuthorDiversityScorer) Enable(query *pipeline.Query) bool {
	return true
}
