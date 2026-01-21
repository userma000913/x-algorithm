package hydrators

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// VideoDurationCandidateHydrator 增强候选的视频时长信息
type VideoDurationCandidateHydrator struct {
	tesClient TweetEntityServiceClient
}

// NewVideoDurationCandidateHydrator 创建新的 VideoDurationCandidateHydrator 实例
func NewVideoDurationCandidateHydrator(client TweetEntityServiceClient) *VideoDurationCandidateHydrator {
	return &VideoDurationCandidateHydrator{
		tesClient: client,
	}
}

// MediaEntities 表示媒体实体列表
type MediaEntities []MediaEntity

// MediaEntity 表示单个媒体实体
type MediaEntity struct {
	MediaInfo *MediaInfo
}

// MediaInfo 表示媒体信息（可以是视频、图片等）
type MediaInfo struct {
	VideoInfo *VideoInfo
}

// VideoInfo 表示视频信息
type VideoInfo struct {
	DurationMillis int32
}

// Hydrate 实现 Hydrator 接口
func (h *VideoDurationCandidateHydrator) Hydrate(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	// 提取所有 tweet_id
	tweetIDs := make([]int64, len(candidates))
	for i, c := range candidates {
		tweetIDs[i] = c.TweetID
	}

	// 批量获取媒体实体
	mediaEntitiesMap, err := h.tesClient.GetTweetMediaEntities(ctx, tweetIDs)
	if err != nil {
		return nil, err
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	hydrated := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		hydrated[i] = candidate.Clone()

		// 获取媒体实体
		mediaEntities := mediaEntitiesMap[candidate.TweetID]
		if mediaEntities != nil {
			// 查找视频信息
			for _, entity := range *mediaEntities {
				if entity.MediaInfo != nil && entity.MediaInfo.VideoInfo != nil {
					duration := entity.MediaInfo.VideoInfo.DurationMillis
					hydrated[i].VideoDurationMs = &duration
					break // 只取第一个视频
				}
			}
		}
	}

	return hydrated, nil
}

// Update 更新单个候选的增强字段
func (h *VideoDurationCandidateHydrator) Update(candidate *pipeline.Candidate, hydrated *pipeline.Candidate) {
	if hydrated.VideoDurationMs != nil {
		candidate.VideoDurationMs = hydrated.VideoDurationMs
	}
}

// UpdateAll 批量更新候选的增强字段
func (h *VideoDurationCandidateHydrator) UpdateAll(candidates []*pipeline.Candidate, hydrated []*pipeline.Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		h.Update(candidates[i], hydrated[i])
	}
}

// Name 返回 Hydrator 名称
func (h *VideoDurationCandidateHydrator) Name() string {
	return "VideoDurationCandidateHydrator"
}

// Enable 决定是否启用（VideoDurationCandidateHydrator 总是启用）
func (h *VideoDurationCandidateHydrator) Enable(query *pipeline.Query) bool {
	return true
}
