package hydrators

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// CoreDataCandidateHydrator 增强候选的核心数据（帖子内容、作者信息等）
type CoreDataCandidateHydrator struct {
	tesClient TweetEntityServiceClient
}

// TweetEntityServiceClient 定义 Tweet Entity Service 客户端接口
type TweetEntityServiceClient interface {
	// GetTweetCoreDatas 批量获取帖子核心数据
	GetTweetCoreDatas(ctx context.Context, tweetIDs []int64) (map[int64]*CoreData, error)
	// GetTweetMediaEntities 批量获取帖子媒体实体
	GetTweetMediaEntities(ctx context.Context, tweetIDs []int64) (map[int64]*MediaEntities, error)
	// GetSubscriptionAuthorIDs 批量获取订阅作者ID
	GetSubscriptionAuthorIDs(ctx context.Context, tweetIDs []int64) (map[int64]*uint64, error)
}

// CoreData 表示帖子的核心数据
type CoreData struct {
	AuthorID        uint64
	Text            string
	SourceTweetID   *uint64
	SourceUserID    *uint64
	InReplyToTweetID *uint64
}

// NewCoreDataCandidateHydrator 创建新的 CoreDataCandidateHydrator 实例
func NewCoreDataCandidateHydrator(client TweetEntityServiceClient) *CoreDataCandidateHydrator {
	return &CoreDataCandidateHydrator{
		tesClient: client,
	}
}

// Hydrate 实现 Hydrator 接口
func (h *CoreDataCandidateHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	// 提取所有 tweet_id
	tweetIDs := make([]int64, len(candidates))
	for i, c := range candidates {
		tweetIDs[i] = c.TweetID
	}

	// 批量获取核心数据
	coreDatas, err := h.tesClient.GetTweetCoreDatas(ctx, tweetIDs)
	if err != nil {
		return nil, err
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	// 与Rust版本一致：总是创建新的Candidate，只包含需要更新的字段
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 获取核心数据
		coreData := coreDatas[candidate.TweetID]
		
		// 创建新的Candidate（与Rust版本的Default::default()语义一致）
		// 只设置需要更新的字段，其他字段保持默认值
		hydrated[i] = &pipeline.Candidate{}
		
		if coreData != nil {
			// 更新字段（与Rust版本一致）
			hydrated[i].TweetText = coreData.Text
			hydrated[i].AuthorID = coreData.AuthorID // 总是设置（即使为0，与Rust的unwrap_or_default一致）
			hydrated[i].RetweetedTweetID = coreData.SourceTweetID
			hydrated[i].RetweetedUserID = coreData.SourceUserID
			hydrated[i].InReplyToTweetID = coreData.InReplyToTweetID
		} else {
			// 如果core_data不存在，使用默认值（与Rust版本一致）
			hydrated[i].TweetText = ""
			hydrated[i].AuthorID = 0 // unwrap_or_default()的结果
		}
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
// 注意：与Rust版本一致，不更新AuthorID（AuthorID在创建候选时已设置）
func (h *CoreDataCandidateHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	candidate.TweetText = hydrated.TweetText
	// 注意：Rust版本不更新author_id，只更新以下字段
	candidate.RetweetedTweetID = hydrated.RetweetedTweetID
	candidate.RetweetedUserID = hydrated.RetweetedUserID
	candidate.InReplyToTweetID = hydrated.InReplyToTweetID
}

// UpdateAll 批量更新候选的增强字段
func (h *CoreDataCandidateHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *CoreDataCandidateHydrator) Name() string {
	return "CoreDataCandidateHydrator"
}

// Enable 决定是否启用（CoreDataCandidateHydrator 总是启用）
func (h *CoreDataCandidateHydrator) Enable(query *pipeline.Query) bool {
	return true
}
