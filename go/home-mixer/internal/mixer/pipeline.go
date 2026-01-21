package mixer

import (
	"context"
	"time"

	"github.com/x-algorithm/go/home-mixer/internal/clients"
	"github.com/x-algorithm/go/home-mixer/internal/filters"
	"github.com/x-algorithm/go/home-mixer/internal/hydrators"
	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/query_hydrators"
	"github.com/x-algorithm/go/home-mixer/internal/scorers"
	"github.com/x-algorithm/go/home-mixer/internal/selectors"
	"github.com/x-algorithm/go/home-mixer/internal/side_effects"
	"github.com/x-algorithm/go/home-mixer/internal/sources"
)

// PhoenixCandidatePipeline 配置完整的推荐管道
// 组装所有组件：Query Hydrators, Sources, Hydrators, Filters, Scorers, Selector 等
type PhoenixCandidatePipeline struct {
	Pipeline *pipeline.CandidatePipeline
}

// PipelineConfig 配置管道的所有组件
type PipelineConfig struct {
	// 客户端（需要外部注入）
	ThunderClient           sources.ThunderClient
	PhoenixRetrievalClient  sources.PhoenixRetrievalClient
	PhoenixRankingClient    scorers.PhoenixRankingClient // 新增
	TESClient               hydrators.TweetEntityServiceClient
	GizmoduckClient         hydrators.GizmoduckClient
	VFClient                hydrators.VisibilityFilteringClient
	UASFetcher              query_hydrators.UserActionSequenceFetcher
	StratoClient            query_hydrators.StratoClient
	StratoClientForCache    side_effects.StratoClient // 用于 Side Effect 的 Strato 客户端
	
	// 配置参数
	ThunderMaxResults       int
	PhoenixMaxResults       int
	TopK                    int
	MaxAge                  time.Duration
}

// NewPhoenixCandidatePipeline 创建新的 PhoenixCandidatePipeline 实例
func NewPhoenixCandidatePipeline(config *PipelineConfig) *PhoenixCandidatePipeline {
	if config == nil {
		config = &PipelineConfig{
			ThunderMaxResults: 500,
			PhoenixMaxResults: 500,
			TopK:              50,
			MaxAge:            7 * 24 * time.Hour, // 7天
		}
	}

	// 1) Query Hydrators（并行执行）
	// Use mock clients if real clients are not provided
	uasFetcher := config.UASFetcher
	if uasFetcher == nil {
		uasFetcher = clients.NewMockUASFetcher()
	}
	
	stratoClient := config.StratoClient
	if stratoClient == nil {
		stratoClient = clients.NewMockStratoClient()
	}
	
	queryHydrators := []pipeline.QueryHydrator{
		query_hydrators.NewUserActionSeqQueryHydrator(uasFetcher),
		query_hydrators.NewUserFeaturesQueryHydrator(stratoClient),
	}

	// 2) Sources（并行执行）
	// Use mock clients if real clients are not provided
	phoenixRetrievalClient := config.PhoenixRetrievalClient
	if phoenixRetrievalClient == nil {
		phoenixRetrievalClient = clients.NewMockPhoenixRetrievalClient()
	}
	
	thunderClient := config.ThunderClient
	if thunderClient == nil {
		thunderClient = clients.NewMockThunderClient()
	}
	
	sourceList := []pipeline.Source{
		sources.NewPhoenixSource(phoenixRetrievalClient, config.PhoenixMaxResults),
		sources.NewThunderSource(thunderClient, config.ThunderMaxResults),
	}

	// 3) Hydrators（并行执行）
	// Use mock clients if real clients are not provided
	tesClient := config.TESClient
	if tesClient == nil {
		tesClient = clients.NewMockTESClient()
	}
	
	gizmoduckClient := config.GizmoduckClient
	if gizmoduckClient == nil {
		gizmoduckClient = clients.NewMockGizmoduckClient()
	}
	
	vfClient := config.VFClient
	if vfClient == nil {
		vfClient = clients.NewMockVFClient()
	}
	
	hydratorList := []pipeline.Hydrator{
		hydrators.NewInNetworkCandidateHydrator(), // 站内标记（需要先执行，因为依赖 UserFeatures）
		hydrators.NewCoreDataCandidateHydrator(tesClient),
		hydrators.NewVideoDurationCandidateHydrator(tesClient),
		hydrators.NewSubscriptionHydrator(tesClient),
		hydrators.NewGizmoduckCandidateHydrator(gizmoduckClient),
	}

	// 4) Pre-Scoring Filters（顺序执行）
	// 注意：顺序必须与Rust版本一致，因为Filter的执行顺序会影响结果
	filterList := []pipeline.Filter{
		filters.NewDropDuplicatesFilter(),        // 1. 去重
		filters.NewCoreDataHydrationFilter(),     // 2. 移除数据获取失败的候选
		filters.NewAgeFilter(config.MaxAge),      // 3. 年龄过滤
		filters.NewSelfTweetFilter(),             // 4. 移除自己的帖子
		filters.NewRetweetDeduplicationFilter(), // 5. 转发去重
		filters.NewIneligibleSubscriptionFilter(), // 6. 订阅过滤
		filters.NewPreviouslySeenPostsFilter(),   // 7. 移除已看过的帖子
		filters.NewPreviouslyServedPostsFilter(), // 8. 移除已服务的帖子（分页时）
		filters.NewMutedKeywordFilter(),         // 9. 移除包含静音关键词的帖子
		filters.NewAuthorSocialgraphFilter(),    // 10. 移除屏蔽/静音作者的帖子
	}

	// 5) Scorers（顺序执行）
	// Create mock PhoenixRankingClient for local learning if not provided
	var phoenixRankingClient scorers.PhoenixRankingClient
	if config.PhoenixRankingClient == nil {
		// Use mock client for local testing
		phoenixRankingClient = scorers.NewMockPhoenixRankingClient()
	} else {
		phoenixRankingClient = config.PhoenixRankingClient
	}
	
	scorerList := []pipeline.Scorer{
		scorers.NewPhoenixScorer(phoenixRankingClient),
		scorers.NewWeightedScorer(nil),          // 使用默认权重
		scorers.DefaultAuthorDiversityScorer(),  // 作者多样性调整
		scorers.DefaultOONScorer(),              // 站外内容调整
	}

	// 6) Selector
	selector := selectors.NewTopKScoreSelector(config.TopK)

	// 7) Post-Selection Hydrators（并行执行）
	postSelectionHydrators := []pipeline.Hydrator{
		hydrators.NewVFCandidateHydrator(vfClient),
	}

	// 8) Post-Selection Filters（顺序执行）
	postSelectionFilters := []pipeline.Filter{
		filters.NewVFFilter(),                    // 可见性过滤
		filters.NewDedupConversationFilter(),     // 对话去重
	}

	// 9) Side Effects（异步执行）
	// Use mock client if real client is not provided
	stratoClientForCache := config.StratoClientForCache
	if stratoClientForCache == nil {
		stratoClientForCache = clients.NewMockStratoClientForCache()
	}
	
	sideEffects := []pipeline.SideEffect{
		side_effects.NewCacheRequestInfoSideEffect(stratoClientForCache),
	}

	// 创建 Pipeline
	candidatePipeline := &pipeline.CandidatePipeline{
		QueryHydrators:         queryHydrators,
		Sources:                sourceList,
		Hydrators:              hydratorList,
		Filters:                filterList,
		Scorers:                scorerList,
		Selector:               selector,
		PostSelectionHydrators: postSelectionHydrators,
		PostSelectionFilters:   postSelectionFilters,
		SideEffects:            sideEffects,
		ResultSize:             config.TopK,
	}

	return &PhoenixCandidatePipeline{
		Pipeline: candidatePipeline,
	}
}

// Prod creates a production-ready pipeline configuration with real clients
// This method should be called with actual service addresses
func Prod(
	thunderAddr string,
	phoenixRetrievalAddr string,
	phoenixRankingAddr string,
	tesAddr string,
	gizmoduckAddr string,
	stratoAddr string,
	uasAddr string,
	vfAddr string,
) (*PhoenixCandidatePipeline, error) {
	// Note: For local learning/testing, this creates mock clients
	// In production, you would use real gRPC clients
	
	// Create mock clients (or real clients if addresses are provided)
	config := &PipelineConfig{
		ThunderMaxResults: 500,
		PhoenixMaxResults: 500,
		TopK:              50,
		MaxAge:            7 * 24 * time.Hour,
		// Clients are nil, which means mock implementations will be used
	}
	return NewPhoenixCandidatePipeline(config), nil
}

// NewMockPipeline creates a pipeline with all mock clients for local learning
func NewMockPipeline() *PhoenixCandidatePipeline {
	config := &PipelineConfig{
		ThunderMaxResults: 500,
		PhoenixMaxResults: 500,
		TopK:              50,
		MaxAge:            7 * 24 * time.Hour,
		// All clients are nil - mock implementations will be used via sources/hydrators/etc.
	}
	return NewPhoenixCandidatePipeline(config)
}

// Execute 执行管道（委托给内部的 CandidatePipeline）
func (p *PhoenixCandidatePipeline) Execute(ctx context.Context, query *pipeline.Query) (*pipeline.PipelineResult, error) {
	return p.Pipeline.Execute(ctx, query)
}
