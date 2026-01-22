package hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// InNetworkCandidateHydrator 标记候选是否为站内内容（来自关注账号）
type InNetworkCandidateHydrator struct{}

// NewInNetworkCandidateHydrator 创建新的 InNetworkCandidateHydrator 实例
func NewInNetworkCandidateHydrator() *InNetworkCandidateHydrator {
	return &InNetworkCandidateHydrator{}
}

// Hydrate 实现 Hydrator 接口
func (h *InNetworkCandidateHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	// 构建关注列表集合（用于快速查找）
	followedSet := make(map[int64]bool)
	for _, id := range query.UserFeatures.FollowedUserIDs {
		followedSet[id] = true
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	viewerID := int64(query.UserID)
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		hydrated[i] = candidate.Clone()

		// 判断是否为站内内容（作者在关注列表中，或者是自己的帖子）
		authorID := int64(candidate.AuthorID)
		isSelf := authorID == viewerID
		isInNetwork := isSelf || followedSet[authorID]
		hydrated[i].InNetwork = &isInNetwork
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
func (h *InNetworkCandidateHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	if hydrated.InNetwork != nil {
		candidate.InNetwork = hydrated.InNetwork
	}
}

// UpdateAll 批量更新候选的增强字段
func (h *InNetworkCandidateHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *InNetworkCandidateHydrator) Name() string {
	return "InNetworkCandidateHydrator"
}

// Enable 决定是否启用（InNetworkCandidateHydrator 总是启用）
func (h *InNetworkCandidateHydrator) Enable(query *pipeline.Query) bool {
	return true
}
