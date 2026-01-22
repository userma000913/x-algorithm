package scorers

import (
	"context"
	"math"

	"x-algorithm-go/candidate-pipeline/pipeline"
	"x-algorithm-go/home-mixer/internal/utils"
)

// WeightedScorer 加权组合多个预测分数
type WeightedScorer struct {
	// 权重配置（可以从配置文件读取）
	Weights *ActionWeights
}

// ActionWeights 定义各种动作的权重
type ActionWeights struct {
	FavoriteWeight         float64
	ReplyWeight            float64
	RetweetWeight          float64
	PhotoExpandWeight      float64
	ClickWeight            float64
	ProfileClickWeight     float64
	VqvWeight              float64
	ShareWeight            float64
	ShareViaDmWeight       float64
	ShareViaCopyLinkWeight float64
	DwellWeight            float64
	QuoteWeight            float64
	QuotedClickWeight      float64
	ContDwellTimeWeight    float64
	FollowAuthorWeight     float64
	NotInterestedWeight    float64
	BlockAuthorWeight      float64
	MuteAuthorWeight       float64
	ReportWeight           float64
	
	// 配置参数
	MinVideoDurationMs     int32
	NegativeScoresOffset    float64
	WeightsSum             float64
	NegativeWeightsSum     float64
}

// DefaultActionWeights 返回默认权重配置
func DefaultActionWeights() *ActionWeights {
	// 这些是示例权重，实际应该从配置文件读取
	return &ActionWeights{
		FavoriteWeight:         1.0,
		ReplyWeight:            1.0,
		RetweetWeight:          1.0,
		PhotoExpandWeight:      0.5,
		ClickWeight:            0.5,
		ProfileClickWeight:     0.3,
		VqvWeight:              1.0,
		ShareWeight:            1.0,
		ShareViaDmWeight:       0.8,
		ShareViaCopyLinkWeight: 0.5,
		DwellWeight:            0.5,
		QuoteWeight:            1.0,
		QuotedClickWeight:      0.3,
		ContDwellTimeWeight:    0.1,
		FollowAuthorWeight:     0.5,
		NotInterestedWeight:    -1.0,
		BlockAuthorWeight:      -2.0,
		MuteAuthorWeight:       -1.5,
		ReportWeight:           -3.0,
		MinVideoDurationMs:     3000, // 3秒
		NegativeScoresOffset:    0.0,
		WeightsSum:             10.0, // 正权重之和（示例）
		NegativeWeightsSum:     -7.5, // 负权重之和（示例）
	}
}

// NewWeightedScorer 创建新的 WeightedScorer 实例
func NewWeightedScorer(weights *ActionWeights) *WeightedScorer {
	if weights == nil {
		weights = DefaultActionWeights()
	}
	return &WeightedScorer{
		Weights: weights,
	}
}

// Score 实现 Scorer 接口
func (s *WeightedScorer) Score(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	scored := make([]*pipeline.Candidate, len(candidates))
	
	for i, candidate := range candidates {
		// 克隆候选
		scored[i] = candidate.Clone()
		
		// 计算加权分数
		weightedScore := s.computeWeightedScore(candidate)
		
		// 归一化分数（使用与Rust版本一致的逻辑）
		normalizedScore := utils.NormalizeScore(candidate, weightedScore)
		
		// 只更新 WeightedScore 字段（与Rust版本一致）
		// Score 字段由后续的 AuthorDiversityScorer 设置
		scored[i].WeightedScore = &normalizedScore
	}
	
	return scored, nil
}

// computeWeightedScore 计算加权分数
func (s *WeightedScorer) computeWeightedScore(candidate *pipeline.Candidate) float64 {
	if candidate.PhoenixScores == nil {
		return 0.0
	}
	
	ps := candidate.PhoenixScores
	w := s.Weights
	
	// 计算 VQV 权重（需要视频时长）
	vqvWeight := s.vqvWeightEligibility(candidate)
	
	// 组合所有分数
	combinedScore := s.apply(ps.FavoriteScore, w.FavoriteWeight) +
		s.apply(ps.ReplyScore, w.ReplyWeight) +
		s.apply(ps.RetweetScore, w.RetweetWeight) +
		s.apply(ps.PhotoExpandScore, w.PhotoExpandWeight) +
		s.apply(ps.ClickScore, w.ClickWeight) +
		s.apply(ps.ProfileClickScore, w.ProfileClickWeight) +
		s.apply(ps.VqvScore, vqvWeight) +
		s.apply(ps.ShareScore, w.ShareWeight) +
		s.apply(ps.ShareViaDmScore, w.ShareViaDmWeight) +
		s.apply(ps.ShareViaCopyLinkScore, w.ShareViaCopyLinkWeight) +
		s.apply(ps.DwellScore, w.DwellWeight) +
		s.apply(ps.QuoteScore, w.QuoteWeight) +
		s.apply(ps.QuotedClickScore, w.QuotedClickWeight) +
		s.apply(ps.DwellTime, w.ContDwellTimeWeight) +
		s.apply(ps.FollowAuthorScore, w.FollowAuthorWeight) +
		s.apply(ps.NotInterestedScore, w.NotInterestedWeight) +
		s.apply(ps.BlockAuthorScore, w.BlockAuthorWeight) +
		s.apply(ps.MuteAuthorScore, w.MuteAuthorWeight) +
		s.apply(ps.ReportScore, w.ReportWeight)
	
	// 应用偏移
	return s.offsetScore(combinedScore)
}

// apply 应用权重
func (s *WeightedScorer) apply(score *float64, weight float64) float64 {
	if score == nil {
		return 0.0
	}
	return *score * weight
}

// vqvWeightEligibility 计算 VQV 权重（需要视频时长）
func (s *WeightedScorer) vqvWeightEligibility(candidate *pipeline.Candidate) float64 {
	if candidate.VideoDurationMs == nil {
		return 0.0
	}
	if *candidate.VideoDurationMs > s.Weights.MinVideoDurationMs {
		return s.Weights.VqvWeight
	}
	return 0.0
}

// offsetScore 应用分数偏移
func (s *WeightedScorer) offsetScore(combinedScore float64) float64 {
	w := s.Weights
	
	if w.WeightsSum == 0.0 {
		return math.Max(combinedScore, 0.0)
	}
	
	if combinedScore < 0.0 {
		return (combinedScore+w.NegativeWeightsSum)/w.WeightsSum*w.NegativeScoresOffset
	}
	
	return combinedScore + w.NegativeScoresOffset
}

// Update 更新单个候选的打分字段
// 只更新 WeightedScore 字段（与Rust版本一致）
func (s *WeightedScorer) Update(candidate *pipeline.Candidate, scored *pipeline.Candidate) {
	if scored.WeightedScore != nil {
		candidate.WeightedScore = scored.WeightedScore
	}
	// 注意：不更新 Score 字段，Score 字段由后续的 AuthorDiversityScorer 设置
}

// UpdateAll 批量更新候选的打分字段
func (s *WeightedScorer) UpdateAll(candidates []*pipeline.Candidate, scored []*pipeline.Candidate) {
	if len(candidates) != len(scored) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		s.Update(candidates[i], scored[i])
	}
}

// Name 返回 Scorer 名称
func (s *WeightedScorer) Name() string {
	return "WeightedScorer"
}

// Enable 决定是否启用（WeightedScorer 总是启用）
func (s *WeightedScorer) Enable(query *pipeline.Query) bool {
	return true
}
