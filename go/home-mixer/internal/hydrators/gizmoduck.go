package hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// GizmoduckCandidateHydrator 增强候选的作者信息（用户名、粉丝数等）
type GizmoduckCandidateHydrator struct {
	gizmoduckClient GizmoduckClient
}

// GizmoduckClient 定义 Gizmoduck 客户端接口
type GizmoduckClient interface {
	// GetUsers 批量获取用户信息
	GetUsers(ctx context.Context, userIDs []int64) (map[int64]*GizmoduckUserResult, error)
}

// GizmoduckUserResult 表示 Gizmoduck 用户查询结果
type GizmoduckUserResult struct {
	User *GizmoduckUser
}

// GizmoduckUser 表示用户信息
type GizmoduckUser struct {
	UserID   uint64
	Profile  *GizmoduckUserProfile
	Counts   *GizmoduckUserCounts
}

// GizmoduckUserProfile 表示用户资料
type GizmoduckUserProfile struct {
	ScreenName string
}

// GizmoduckUserCounts 表示用户统计信息
type GizmoduckUserCounts struct {
	FollowersCount uint32
}

// NewGizmoduckCandidateHydrator 创建新的 GizmoduckCandidateHydrator 实例
func NewGizmoduckCandidateHydrator(client GizmoduckClient) *GizmoduckCandidateHydrator {
	return &GizmoduckCandidateHydrator{
		gizmoduckClient: client,
	}
}

// Hydrate 实现 Hydrator 接口
func (h *GizmoduckCandidateHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	// 收集所有需要查询的用户ID（作者和转发作者）
	userIDsSet := make(map[int64]bool)
	for _, candidate := range candidates {
		userIDsSet[int64(candidate.AuthorID)] = true
		if candidate.RetweetedUserID != nil {
			userIDsSet[int64(*candidate.RetweetedUserID)] = true
		}
	}

	// 转换为切片
	userIDs := make([]int64, 0, len(userIDsSet))
	for id := range userIDsSet {
		userIDs = append(userIDs, id)
	}

	// 批量获取用户信息
	users, err := h.gizmoduckClient.GetUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		hydrated[i] = candidate.Clone()

		// 获取作者信息
		authorID := int64(candidate.AuthorID)
		if userResult, ok := users[authorID]; ok && userResult != nil && userResult.User != nil {
			user := userResult.User
			if user.Profile != nil {
				screenName := user.Profile.ScreenName
				hydrated[i].AuthorScreenName = &screenName
			}
			if user.Counts != nil {
				followersCount := int32(user.Counts.FollowersCount)
				hydrated[i].AuthorFollowersCount = &followersCount
			}
		}

		// 获取转发作者信息
		if candidate.RetweetedUserID != nil {
			retweetedUserID := int64(*candidate.RetweetedUserID)
			if userResult, ok := users[retweetedUserID]; ok && userResult != nil && userResult.User != nil {
				user := userResult.User
				if user.Profile != nil {
					screenName := user.Profile.ScreenName
					hydrated[i].RetweetedScreenName = &screenName
				}
			}
		}
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
func (h *GizmoduckCandidateHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	if hydrated.AuthorScreenName != nil {
		candidate.AuthorScreenName = hydrated.AuthorScreenName
	}
	if hydrated.AuthorFollowersCount != nil {
		candidate.AuthorFollowersCount = hydrated.AuthorFollowersCount
	}
	if hydrated.RetweetedScreenName != nil {
		candidate.RetweetedScreenName = hydrated.RetweetedScreenName
	}
}

// UpdateAll 批量更新候选的增强字段
func (h *GizmoduckCandidateHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *GizmoduckCandidateHydrator) Name() string {
	return "GizmoduckCandidateHydrator"
}

// Enable 决定是否启用（GizmoduckCandidateHydrator 总是启用）
func (h *GizmoduckCandidateHydrator) Enable(query *pipeline.Query) bool {
	return true
}
