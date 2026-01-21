package pipeline

// Query 表示一个推荐请求的查询对象
// 包含用户信息、请求参数以及增强后的用户特征和历史
type Query struct {
	// 基础字段
	UserID         int64
	ClientAppID    int32
	CountryCode    string
	LanguageCode   string
	SeenIDs        []int64
	ServedIDs      []int64
	InNetworkOnly  bool
	IsBottomRequest bool
	BloomFilterEntries []BloomFilterEntry
	RequestID      string

	// 增强后的字段（通过 Query Hydrators 填充）
	UserActionSequence *UserActionSequence
	UserFeatures      UserFeatures
}

// Clone 创建 Query 的深拷贝
func (q *Query) Clone() *Query {
	if q == nil {
		return nil
	}
	
	clone := &Query{
		UserID:          q.UserID,
		ClientAppID:    q.ClientAppID,
		CountryCode:    q.CountryCode,
		LanguageCode:    q.LanguageCode,
		InNetworkOnly:  q.InNetworkOnly,
		IsBottomRequest: q.IsBottomRequest,
		RequestID:       q.RequestID,
	}
	
	// 深拷贝切片
	if q.SeenIDs != nil {
		clone.SeenIDs = make([]int64, len(q.SeenIDs))
		copy(clone.SeenIDs, q.SeenIDs)
	}
	if q.ServedIDs != nil {
		clone.ServedIDs = make([]int64, len(q.ServedIDs))
		copy(clone.ServedIDs, q.ServedIDs)
	}
	if q.BloomFilterEntries != nil {
		clone.BloomFilterEntries = make([]BloomFilterEntry, len(q.BloomFilterEntries))
		copy(clone.BloomFilterEntries, q.BloomFilterEntries)
	}
	
	// 深拷贝指针字段
	if q.UserActionSequence != nil {
		clone.UserActionSequence = q.UserActionSequence.Clone()
	}
	clone.UserFeatures = q.UserFeatures.Clone()
	
	return clone
}

// BloomFilterEntry 表示布隆过滤器条目（用于去重）
type BloomFilterEntry struct {
	// Data 包含序列化的布隆过滤器位数组数据
	// 格式可能包括：位数组字节 + 可选的元数据（哈希函数数量等）
	Data []byte
}

// UserActionSequence 表示用户的交互历史序列
// 包含用户最近的点赞、转发、回复等动作
type UserActionSequence struct {
	UserID    uint64
	Metadata  *UserActionSequenceMeta
	// 用户动作列表（简化表示，实际可能需要更复杂的结构）
	Actions   []UserAction
}

// Clone 创建 UserActionSequence 的深拷贝
func (uas *UserActionSequence) Clone() *UserActionSequence {
	if uas == nil {
		return nil
	}
	clone := &UserActionSequence{
		UserID: uas.UserID,
	}
	if uas.Metadata != nil {
		clone.Metadata = uas.Metadata.Clone()
	}
	if uas.Actions != nil {
		clone.Actions = make([]UserAction, len(uas.Actions))
		copy(clone.Actions, uas.Actions)
	}
	return clone
}

// UserActionSequenceMeta 表示用户动作序列的元数据
type UserActionSequenceMeta struct {
	Length                    uint64
	FirstSequenceTime         uint64
	LastSequenceTime          uint64
	LastModifiedEpochMs      uint64
	PreviousKafkaPublishEpochMs uint64
}

// Clone 创建 UserActionSequenceMeta 的深拷贝
func (m *UserActionSequenceMeta) Clone() *UserActionSequenceMeta {
	if m == nil {
		return nil
	}
	return &UserActionSequenceMeta{
		Length:                    m.Length,
		FirstSequenceTime:         m.FirstSequenceTime,
		LastSequenceTime:          m.LastSequenceTime,
		LastModifiedEpochMs:      m.LastModifiedEpochMs,
		PreviousKafkaPublishEpochMs: m.PreviousKafkaPublishEpochMs,
	}
}

// UserAction 表示单个用户动作
type UserAction struct {
	// 根据实际需求定义字段
	// 这里先定义基本结构
	ActionType string
	TweetID    int64
	Timestamp  int64
}

// UserFeatures 表示用户特征
// 包含关注列表、屏蔽列表、静音列表等
type UserFeatures struct {
	MutedKeywords    []string
	BlockedUserIDs   []int64
	MutedUserIDs     []int64
	FollowedUserIDs  []int64
	SubscribedUserIDs []int64
}

// Clone 创建 UserFeatures 的深拷贝
func (uf *UserFeatures) Clone() UserFeatures {
	if uf == nil {
		return UserFeatures{}
	}
	clone := UserFeatures{}
	if uf.MutedKeywords != nil {
		clone.MutedKeywords = make([]string, len(uf.MutedKeywords))
		copy(clone.MutedKeywords, uf.MutedKeywords)
	}
	if uf.BlockedUserIDs != nil {
		clone.BlockedUserIDs = make([]int64, len(uf.BlockedUserIDs))
		copy(clone.BlockedUserIDs, uf.BlockedUserIDs)
	}
	if uf.MutedUserIDs != nil {
		clone.MutedUserIDs = make([]int64, len(uf.MutedUserIDs))
		copy(clone.MutedUserIDs, uf.MutedUserIDs)
	}
	if uf.FollowedUserIDs != nil {
		clone.FollowedUserIDs = make([]int64, len(uf.FollowedUserIDs))
		copy(clone.FollowedUserIDs, uf.FollowedUserIDs)
	}
	if uf.SubscribedUserIDs != nil {
		clone.SubscribedUserIDs = make([]int64, len(uf.SubscribedUserIDs))
		copy(clone.SubscribedUserIDs, uf.SubscribedUserIDs)
	}
	return clone
}

// Candidate 表示一个候选帖子
// 包含帖子ID、作者信息、内容、分数等
type Candidate struct {
	// 基础字段
	TweetID    int64
	AuthorID   uint64
	TweetText  string
	
	// 关系字段
	InReplyToTweetID *uint64
	RetweetedTweetID *uint64
	RetweetedUserID  *uint64
	
	// Phoenix 预测分数
	PhoenixScores *PhoenixScores
	
	// 分数相关字段
	PredictionRequestID *uint64
	LastScoredAtMs      *uint64
	WeightedScore        *float64
	Score                *float64
	
	// 元数据字段
	ServedType           *int32
	InNetwork             *bool
	Ancestors             []uint64
	VideoDurationMs       *int32
	AuthorFollowersCount  *int32
	AuthorScreenName      *string
	RetweetedScreenName   *string
	VisibilityReason      *string
	SubscriptionAuthorID  *uint64
}

// Clone 创建 Candidate 的深拷贝
func (c *Candidate) Clone() *Candidate {
	if c == nil {
		return nil
	}
	clone := &Candidate{
		TweetID:  c.TweetID,
		AuthorID: c.AuthorID,
		TweetText: c.TweetText,
	}
	
	// 深拷贝指针字段
	if c.InReplyToTweetID != nil {
		val := *c.InReplyToTweetID
		clone.InReplyToTweetID = &val
	}
	if c.RetweetedTweetID != nil {
		val := *c.RetweetedTweetID
		clone.RetweetedTweetID = &val
	}
	if c.RetweetedUserID != nil {
		val := *c.RetweetedUserID
		clone.RetweetedUserID = &val
	}
	if c.PhoenixScores != nil {
		clone.PhoenixScores = c.PhoenixScores.Clone()
	}
	if c.PredictionRequestID != nil {
		val := *c.PredictionRequestID
		clone.PredictionRequestID = &val
	}
	if c.LastScoredAtMs != nil {
		val := *c.LastScoredAtMs
		clone.LastScoredAtMs = &val
	}
	if c.WeightedScore != nil {
		val := *c.WeightedScore
		clone.WeightedScore = &val
	}
	if c.Score != nil {
		val := *c.Score
		clone.Score = &val
	}
	if c.ServedType != nil {
		val := *c.ServedType
		clone.ServedType = &val
	}
	if c.InNetwork != nil {
		val := *c.InNetwork
		clone.InNetwork = &val
	}
	if c.VideoDurationMs != nil {
		val := *c.VideoDurationMs
		clone.VideoDurationMs = &val
	}
	if c.AuthorFollowersCount != nil {
		val := *c.AuthorFollowersCount
		clone.AuthorFollowersCount = &val
	}
	if c.AuthorScreenName != nil {
		val := *c.AuthorScreenName
		clone.AuthorScreenName = &val
	}
	if c.RetweetedScreenName != nil {
		val := *c.RetweetedScreenName
		clone.RetweetedScreenName = &val
	}
	if c.VisibilityReason != nil {
		val := *c.VisibilityReason
		clone.VisibilityReason = &val
	}
	if c.SubscriptionAuthorID != nil {
		val := *c.SubscriptionAuthorID
		clone.SubscriptionAuthorID = &val
	}
	
	// 深拷贝切片
	if c.Ancestors != nil {
		clone.Ancestors = make([]uint64, len(c.Ancestors))
		copy(clone.Ancestors, c.Ancestors)
	}
	
	return clone
}

// GetScreenNames 获取候选相关的用户名映射
// 返回 author_id -> screen_name 的映射
func (c *Candidate) GetScreenNames() map[uint64]string {
	screenNames := make(map[uint64]string)
	if c.AuthorScreenName != nil {
		screenNames[c.AuthorID] = *c.AuthorScreenName
	}
	if c.RetweetedScreenName != nil && c.RetweetedUserID != nil {
		screenNames[*c.RetweetedUserID] = *c.RetweetedScreenName
	}
	return screenNames
}

// PhoenixScores 表示 Phoenix 模型预测的各种交互概率分数
type PhoenixScores struct {
	// 正面动作分数
	FavoriteScore      *float64
	ReplyScore         *float64
	RetweetScore       *float64
	PhotoExpandScore   *float64
	ClickScore         *float64
	ProfileClickScore  *float64
	VqvScore           *float64
	ShareScore         *float64
	ShareViaDmScore    *float64
	ShareViaCopyLinkScore *float64
	DwellScore         *float64
	QuoteScore         *float64
	QuotedClickScore   *float64
	FollowAuthorScore  *float64
	
	// 负面动作分数
	NotInterestedScore *float64
	BlockAuthorScore   *float64
	MuteAuthorScore    *float64
	ReportScore        *float64
	
	// 连续动作
	DwellTime          *float64
}

// Clone 创建 PhoenixScores 的深拷贝
func (ps *PhoenixScores) Clone() *PhoenixScores {
	if ps == nil {
		return nil
	}
	clone := &PhoenixScores{}
	
	// 深拷贝所有指针字段
	if ps.FavoriteScore != nil {
		val := *ps.FavoriteScore
		clone.FavoriteScore = &val
	}
	if ps.ReplyScore != nil {
		val := *ps.ReplyScore
		clone.ReplyScore = &val
	}
	if ps.RetweetScore != nil {
		val := *ps.RetweetScore
		clone.RetweetScore = &val
	}
	if ps.PhotoExpandScore != nil {
		val := *ps.PhotoExpandScore
		clone.PhotoExpandScore = &val
	}
	if ps.ClickScore != nil {
		val := *ps.ClickScore
		clone.ClickScore = &val
	}
	if ps.ProfileClickScore != nil {
		val := *ps.ProfileClickScore
		clone.ProfileClickScore = &val
	}
	if ps.VqvScore != nil {
		val := *ps.VqvScore
		clone.VqvScore = &val
	}
	if ps.ShareScore != nil {
		val := *ps.ShareScore
		clone.ShareScore = &val
	}
	if ps.ShareViaDmScore != nil {
		val := *ps.ShareViaDmScore
		clone.ShareViaDmScore = &val
	}
	if ps.ShareViaCopyLinkScore != nil {
		val := *ps.ShareViaCopyLinkScore
		clone.ShareViaCopyLinkScore = &val
	}
	if ps.DwellScore != nil {
		val := *ps.DwellScore
		clone.DwellScore = &val
	}
	if ps.QuoteScore != nil {
		val := *ps.QuoteScore
		clone.QuoteScore = &val
	}
	if ps.QuotedClickScore != nil {
		val := *ps.QuotedClickScore
		clone.QuotedClickScore = &val
	}
	if ps.FollowAuthorScore != nil {
		val := *ps.FollowAuthorScore
		clone.FollowAuthorScore = &val
	}
	if ps.NotInterestedScore != nil {
		val := *ps.NotInterestedScore
		clone.NotInterestedScore = &val
	}
	if ps.BlockAuthorScore != nil {
		val := *ps.BlockAuthorScore
		clone.BlockAuthorScore = &val
	}
	if ps.MuteAuthorScore != nil {
		val := *ps.MuteAuthorScore
		clone.MuteAuthorScore = &val
	}
	if ps.ReportScore != nil {
		val := *ps.ReportScore
		clone.ReportScore = &val
	}
	if ps.DwellTime != nil {
		val := *ps.DwellTime
		clone.DwellTime = &val
	}
	
	return clone
}

// PipelineResult 表示管道执行的结果
type PipelineResult struct {
	RetrievedCandidates []*Candidate // 检索到的候选（增强后）
	FilteredCandidates   []*Candidate // 被过滤掉的候选
	SelectedCandidates   []*Candidate // 最终选择的候选
	Query                *Query       // 增强后的查询对象
}

// FilterResult 表示过滤器执行的结果
type FilterResult struct {
	Kept    []*Candidate // 保留的候选
	Removed []*Candidate // 移除的候选
}
