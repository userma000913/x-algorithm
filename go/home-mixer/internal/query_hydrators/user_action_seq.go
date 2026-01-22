package query_hydrators

import (
	"context"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// UserActionSeqQueryHydrator 增强查询，添加用户交互历史序列
type UserActionSeqQueryHydrator struct {
	uasFetcher UserActionSequenceFetcher
}

// UserActionSequenceFetcher 定义用户动作序列获取器接口
type UserActionSequenceFetcher interface {
	// GetByUserID 根据用户ID获取用户动作序列
	GetByUserID(ctx context.Context, userID int64) (*UserActionSequenceData, error)
}

// UserActionSequenceData 表示用户动作序列数据（简化表示）
// 实际实现中可能需要更复杂的结构
type UserActionSequenceData struct {
	UserID    int64
	Actions   []UserActionData
	Metadata  *UserActionSequenceMetadata
}

// UserActionData 表示单个用户动作数据
type UserActionData struct {
	ActionType string
	TweetID    int64
	Timestamp  int64
}

// UserActionSequenceMetadata 表示用户动作序列的元数据
type UserActionSequenceMetadata struct {
	Length                    uint64
	FirstSequenceTime         uint64
	LastSequenceTime          uint64
	LastModifiedEpochMs      uint64
	PreviousKafkaPublishEpochMs uint64
}

// NewUserActionSeqQueryHydrator 创建新的 UserActionSeqQueryHydrator 实例
func NewUserActionSeqQueryHydrator(fetcher UserActionSequenceFetcher) *UserActionSeqQueryHydrator {
	return &UserActionSeqQueryHydrator{
		uasFetcher: fetcher,
	}
}

// Hydrate 实现 QueryHydrator 接口
func (h *UserActionSeqQueryHydrator) Hydrate(ctx context.Context, query *pipeline.Query) (*pipeline.Query, error) {
	// 获取用户动作序列
	uasData, err := h.uasFetcher.GetByUserID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	// 转换为内部 UserActionSequence 格式
	userActionSequence := h.convertToUserActionSequence(uasData)

	// 返回增强后的查询
	return &pipeline.Query{
		UserActionSequence: userActionSequence,
	}, nil
}

// convertToUserActionSequence 转换数据格式
func (h *UserActionSeqQueryHydrator) convertToUserActionSequence(data *UserActionSequenceData) *pipeline.UserActionSequence {
	if data == nil {
		return nil
	}

	sequence := &pipeline.UserActionSequence{
		UserID: uint64(data.UserID),
	}

	// 转换元数据
	if data.Metadata != nil {
		sequence.Metadata = &pipeline.UserActionSequenceMeta{
			Length:                    data.Metadata.Length,
			FirstSequenceTime:         data.Metadata.FirstSequenceTime,
			LastSequenceTime:          data.Metadata.LastSequenceTime,
			LastModifiedEpochMs:      data.Metadata.LastModifiedEpochMs,
			PreviousKafkaPublishEpochMs: data.Metadata.PreviousKafkaPublishEpochMs,
		}
	}

	// 转换动作列表
	if data.Actions != nil {
		sequence.Actions = make([]pipeline.UserAction, len(data.Actions))
		for i, action := range data.Actions {
			sequence.Actions[i] = pipeline.UserAction{
				ActionType: action.ActionType,
				TweetID:    action.TweetID,
				Timestamp:  action.Timestamp,
			}
		}
	}

	return sequence
}

// Update 更新查询对象的增强字段
func (h *UserActionSeqQueryHydrator) Update(query *pipeline.Query, hydrated *pipeline.Query) {
	if hydrated.UserActionSequence != nil {
		query.UserActionSequence = hydrated.UserActionSequence
	}
}

// Name 返回 QueryHydrator 名称
func (h *UserActionSeqQueryHydrator) Name() string {
	return "UserActionSeqQueryHydrator"
}

// Enable 决定是否启用（UserActionSeqQueryHydrator 总是启用）
func (h *UserActionSeqQueryHydrator) Enable(query *pipeline.Query) bool {
	return true
}
