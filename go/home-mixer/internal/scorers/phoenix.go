package scorers

import (
	"context"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
)

// PhoenixScorer 使用 Phoenix 模型为候选打分
type PhoenixScorer struct {
	phoenixRankingClient PhoenixRankingClient
}

// PhoenixRankingClient 定义 Phoenix Ranking 客户端接口
type PhoenixRankingClient interface {
	// Rank 对候选进行排序预测
	Rank(ctx context.Context, req *RankingRequest) (*RankingResponse, error)
}

// RankingRequest 表示排序请求
type RankingRequest struct {
	UserID            uint64
	UserActionSequence *pipeline.UserActionSequence
	Candidates        []*pipeline.Candidate
	TweetInfos        []*TweetInfo // 用于预测的TweetInfo（转发时使用原帖ID）
}

// TweetInfo 表示用于预测的帖子信息
type TweetInfo struct {
	TweetID  int64
	AuthorID uint64
}

// RankingResponse 表示排序响应
type RankingResponse struct {
	Predictions    []PhoenixPrediction           // 按索引的预测（向后兼容）
	PredictionsMap map[uint64]*PhoenixPrediction // 按tweet_id的预测（推荐使用）
}

// PhoenixPrediction 表示 Phoenix 模型的预测结果
type PhoenixPrediction struct {
	// 正面动作分数
	FavoriteScore      float64
	ReplyScore         float64
	RetweetScore       float64
	PhotoExpandScore   float64
	ClickScore         float64
	ProfileClickScore  float64
	VqvScore           float64
	ShareScore         float64
	ShareViaDmScore    float64
	ShareViaCopyLinkScore float64
	DwellScore         float64
	QuoteScore         float64
	QuotedClickScore   float64
	FollowAuthorScore  float64
	
	// 负面动作分数
	NotInterestedScore float64
	BlockAuthorScore   float64
	MuteAuthorScore    float64
	ReportScore        float64
	
	// 连续动作
	DwellTime          float64
	
	// 元数据
	PredictionRequestID uint64
	LastScoredAtMs      uint64
}

// NewPhoenixScorer 创建新的 PhoenixScorer 实例
func NewPhoenixScorer(client PhoenixRankingClient) *PhoenixScorer {
	return &PhoenixScorer{
		phoenixRankingClient: client,
	}
}

// Score 实现 Scorer 接口
func (s *PhoenixScorer) Score(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	// 检查是否有 user_action_sequence
	// 如果没有用户历史，返回未改变的候选（与Rust版本一致）
	if query.UserActionSequence == nil {
		scored := make([]*pipeline.Candidate, len(candidates))
		for i, c := range candidates {
			scored[i] = c.Clone()
		}
		return scored, nil
	}

	// 构建请求 - 对于转发，使用原帖ID和作者ID
	tweetInfos := make([]*TweetInfo, len(candidates))
	for i, candidate := range candidates {
		// 对于转发，使用原帖ID和作者ID（与Rust版本一致）
		tweetID := uint64(candidate.TweetID)
		authorID := candidate.AuthorID
		if candidate.RetweetedTweetID != nil {
			tweetID = *candidate.RetweetedTweetID
		}
		if candidate.RetweetedUserID != nil {
			authorID = *candidate.RetweetedUserID
		}
		tweetInfos[i] = &TweetInfo{
			TweetID:  int64(tweetID),
			AuthorID: authorID,
		}
	}
	
	req := &RankingRequest{
		UserID:             uint64(query.UserID),
		UserActionSequence: query.UserActionSequence,
		Candidates:         candidates,
		TweetInfos:         tweetInfos, // 添加TweetInfos用于预测查找
	}

	// 调用 Phoenix Ranking 服务
	response, err := s.phoenixRankingClient.Rank(ctx, req)
	if err != nil {
		return nil, err
	}

	// 构建增强后的候选列表（保持顺序和数量一致）
	scored := make([]*pipeline.Candidate, len(candidates))
	for i, candidate := range candidates {
		// 克隆候选
		scored[i] = candidate.Clone()
		
		// 对于转发，使用原帖ID查找预测（与Rust版本一致）
		lookupTweetID := uint64(candidate.TweetID)
		if candidate.RetweetedTweetID != nil {
			lookupTweetID = *candidate.RetweetedTweetID
		}
		
		// 从响应中查找对应的预测（按tweet_id查找，而不是按索引）
		var pred *PhoenixPrediction
		if response.PredictionsMap != nil {
			// 如果响应包含map，使用map查找
			pred = response.PredictionsMap[lookupTweetID]
		} else if i < len(response.Predictions) {
			// 否则使用索引（向后兼容）
			pred = &response.Predictions[i]
		}
		
		if pred != nil {
			
			// 填充 PhoenixScores
			scored[i].PhoenixScores = &pipeline.PhoenixScores{
				FavoriteScore:      &pred.FavoriteScore,
				ReplyScore:         &pred.ReplyScore,
				RetweetScore:       &pred.RetweetScore,
				PhotoExpandScore:   &pred.PhotoExpandScore,
				ClickScore:         &pred.ClickScore,
				ProfileClickScore:  &pred.ProfileClickScore,
				VqvScore:           &pred.VqvScore,
				ShareScore:         &pred.ShareScore,
				ShareViaDmScore:    &pred.ShareViaDmScore,
				ShareViaCopyLinkScore: &pred.ShareViaCopyLinkScore,
				DwellScore:         &pred.DwellScore,
				QuoteScore:         &pred.QuoteScore,
				QuotedClickScore:   &pred.QuotedClickScore,
				FollowAuthorScore:  &pred.FollowAuthorScore,
				NotInterestedScore: &pred.NotInterestedScore,
				BlockAuthorScore:   &pred.BlockAuthorScore,
				MuteAuthorScore:    &pred.MuteAuthorScore,
				ReportScore:        &pred.ReportScore,
				DwellTime:          &pred.DwellTime,
			}
			
			// 填充元数据
			if pred.PredictionRequestID > 0 {
				scored[i].PredictionRequestID = &pred.PredictionRequestID
			}
			if pred.LastScoredAtMs > 0 {
				scored[i].LastScoredAtMs = &pred.LastScoredAtMs
			}
		}
	}

	return scored, nil
}

// Update 更新单个候选的打分字段
func (s *PhoenixScorer) Update(candidate *pipeline.Candidate, scored *pipeline.Candidate) {
	if scored.PhoenixScores != nil {
		candidate.PhoenixScores = scored.PhoenixScores.Clone()
	}
	if scored.PredictionRequestID != nil {
		candidate.PredictionRequestID = scored.PredictionRequestID
	}
	if scored.LastScoredAtMs != nil {
		candidate.LastScoredAtMs = scored.LastScoredAtMs
	}
}

// UpdateAll 批量更新候选的打分字段
func (s *PhoenixScorer) UpdateAll(candidates []*pipeline.Candidate, scored []*pipeline.Candidate) {
	if len(candidates) != len(scored) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		s.Update(candidates[i], scored[i])
	}
}

// Name 返回 Scorer 名称
func (s *PhoenixScorer) Name() string {
	return "PhoenixScorer"
}

// Enable 决定是否启用（PhoenixScorer 总是启用）
func (s *PhoenixScorer) Enable(query *pipeline.Query) bool {
	return true
}

// MockPhoenixRankingClient is a mock implementation for local learning/testing
type MockPhoenixRankingClient struct{}

// NewMockPhoenixRankingClient creates a new mock Phoenix Ranking client
func NewMockPhoenixRankingClient() PhoenixRankingClient {
	return &MockPhoenixRankingClient{}
}

// Rank implements PhoenixRankingClient interface (mock version)
func (c *MockPhoenixRankingClient) Rank(
	ctx context.Context,
	req *RankingRequest,
) (*RankingResponse, error) {
	// Mock implementation for local learning/testing
	// Returns mock predictions for all candidates
	// 使用PredictionsMap支持按tweet_id查找（处理retweet情况）
	
	_ = ctx
	
	if req == nil || len(req.Candidates) == 0 {
		return &RankingResponse{
			Predictions:    []PhoenixPrediction{},
			PredictionsMap: make(map[uint64]*PhoenixPrediction),
		}, nil
	}
	
	predictions := make([]PhoenixPrediction, len(req.Candidates))
	predictionsMap := make(map[uint64]*PhoenixPrediction)
	
	for i, candidate := range req.Candidates {
		// 确定用于查找预测的tweet_id（对于转发，使用原帖ID）
		lookupTweetID := uint64(candidate.TweetID)
		if candidate.RetweetedTweetID != nil {
			lookupTweetID = *candidate.RetweetedTweetID
		}
		// Generate mock prediction scores based on candidate attributes
		// This simulates what the Phoenix model would predict
		
		baseScore := 0.3 + float64(i%10)*0.05 // Vary base scores
		
		pred := PhoenixPrediction{
			// Positive actions (higher scores for better candidates)
			FavoriteScore:      baseScore + 0.1,
			ReplyScore:         baseScore * 0.3,
			RetweetScore:       baseScore * 0.5,
			PhotoExpandScore:   baseScore * 0.2,
			ClickScore:         baseScore + 0.05,
			ProfileClickScore:  baseScore * 0.15,
			VqvScore:           baseScore * 0.4,
			ShareScore:         baseScore * 0.3,
			ShareViaDmScore:    baseScore * 0.2,
			ShareViaCopyLinkScore: baseScore * 0.1,
			DwellScore:         baseScore * 0.6,
			QuoteScore:         baseScore * 0.25,
			QuotedClickScore:   baseScore * 0.1,
			FollowAuthorScore:  baseScore * 0.2,
			
			// Negative actions (lower scores)
			NotInterestedScore: baseScore * 0.05,
			BlockAuthorScore:   baseScore * 0.01,
			MuteAuthorScore:    baseScore * 0.02,
			ReportScore:        baseScore * 0.01,
			
			// Continuous actions
			DwellTime: baseScore * 2.0,
			
			// Metadata
			PredictionRequestID: lookupTweetID,
			LastScoredAtMs:      uint64(candidate.TweetID % 1000000000),
		}
		
		predictions[i] = pred
		// 使用lookupTweetID作为key（对于转发，多个转发会共享同一个原帖的预测）
		predictionsMap[lookupTweetID] = &pred
	}
	
	return &RankingResponse{
		Predictions:    predictions,
		PredictionsMap: predictionsMap,
	}, nil
}
