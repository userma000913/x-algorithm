package hydrators

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// VFCandidateHydrator 增强候选的可见性信息（Visibility Filtering）
// 检查帖子是否可见（未删除、非垃圾内容等）
type VFCandidateHydrator struct {
	vfClient VisibilityFilteringClient
}

// VisibilityFilteringClient 定义可见性过滤客户端接口
type VisibilityFilteringClient interface {
	// GetVisibilityResults 批量获取可见性检查结果
	GetVisibilityResults(ctx context.Context, tweetIDs []int64, isInNetwork bool, userID int64) (map[int64]*string, error)
}

// NewVFCandidateHydrator 创建新的 VFCandidateHydrator 实例
func NewVFCandidateHydrator(client VisibilityFilteringClient) *VFCandidateHydrator {
	return &VFCandidateHydrator{
		vfClient: client,
	}
}

// Hydrate 实现 Hydrator 接口
func (h *VFCandidateHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	// 分离站内和站外内容
	var inNetworkIDs []int64
	var oonIDs []int64

	for _, candidate := range candidates {
		isInNetwork := false
		if candidate.InNetwork != nil {
			isInNetwork = *candidate.InNetwork
		}

		if isInNetwork {
			inNetworkIDs = append(inNetworkIDs, candidate.TweetID)
		} else {
			oonIDs = append(oonIDs, candidate.TweetID)
		}
	}

	// 并行获取可见性结果
	type result struct {
		results map[int64]*string
		err     error
	}

	ch := make(chan result, 2)
	
	// 获取站内内容可见性
	go func() {
		results, err := h.vfClient.GetVisibilityResults(ctx, inNetworkIDs, true, query.UserID)
		ch <- result{results: results, err: err}
	}()

	// 获取站外内容可见性
	go func() {
		results, err := h.vfClient.GetVisibilityResults(ctx, oonIDs, false, query.UserID)
		ch <- result{results: results, err: err}
	}()

	// 合并结果
	visibilityResults := make(map[int64]*string)
	for i := 0; i < 2; i++ {
		res := <-ch
		if res.err != nil {
			return nil, res.err
		}
		for k, v := range res.results {
			visibilityResults[k] = v
		}
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		hydrated[i] = candidate.Clone()

		// 获取可见性原因
		if reason, ok := visibilityResults[candidate.TweetID]; ok {
			hydrated[i].VisibilityReason = reason
		}
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
func (h *VFCandidateHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	if hydrated.VisibilityReason != nil {
		candidate.VisibilityReason = hydrated.VisibilityReason
	}
}

// UpdateAll 批量更新候选的增强字段
func (h *VFCandidateHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *VFCandidateHydrator) Name() string {
	return "VFCandidateHydrator"
}

// Enable 决定是否启用（VFCandidateHydrator 总是启用）
func (h *VFCandidateHydrator) Enable(query *pipeline.Query) bool {
	return true
}
