# Go 实现 - X Algorithm 推荐系统

这是 X Algorithm 推荐系统的 Go 语言实现。

## 项目结构

```
go/
├── cmd/
│   └── server/              # 服务入口 ✅
│       └── main.go
├── internal/
│   ├── mixer/               # Home Mixer（业务层）✅
│   │   └── server.go        # gRPC 服务实现
│   ├── sources/             # 候选源实现 ✅
│   │   ├── thunder.go       # Thunder Source
│   │   ├── phoenix.go       # Phoenix Source
│   │   └── mock.go          # Mock 实现（测试用）
│   ├── filters/             # 过滤器实现 ✅
│   │   ├── age.go           # Age Filter
│   │   ├── duplicate.go     # Duplicate Filter
│   │   ├── self_tweet.go    # Self Tweet Filter
│   │   ├── previously_seen.go # Previously Seen Filter
│   │   ├── previously_served.go # Previously Served Filter
│   │   ├── muted_keyword.go # Muted Keyword Filter
│   │   ├── author_socialgraph.go # Author Socialgraph Filter
│   │   ├── retweet_dedup.go # Retweet Deduplication Filter
│   │   ├── core_data_hydration.go # Core Data Hydration Filter
│   │   ├── ineligible_subscription.go # Ineligible Subscription Filter
│   │   ├── vf.go # Visibility Filtering Filter
│   │   └── dedup_conversation.go # Dedup Conversation Filter
│   ├── hydrators/           # 增强器实现 ✅
│   │   ├── core_data.go     # Core Data Hydrator
│   │   ├── in_network.go    # In Network Hydrator
│   │   ├── gizmoduck.go     # Gizmoduck Hydrator
│   │   └── video_duration.go # Video Duration Hydrator
│   ├── query_hydrators/     # Query 增强器实现 ✅
│   │   ├── user_action_seq.go # User Action Sequence Hydrator
│   │   ├── user_features.go # User Features Hydrator
│   │   └── mock.go          # Mock 实现
│   ├── side_effects/        # Side Effects 实现 ✅
│   │   └── cache_request_info.go # Cache Request Info
│   ├── scorers/             # 打分器实现 ✅
│   │   ├── phoenix.go       # Phoenix Scorer
│   │   ├── weighted.go      # Weighted Scorer
│   │   ├── author_diversity.go # Author Diversity Scorer
│   │   └── oon.go           # OON Scorer
│   ├── selectors/           # 选择器实现 ✅
│   │   └── top_k.go         # TopK Selector
│   ├── pipeline/            # 管道框架（核心）✅
│   │   ├── types.go         # 数据结构定义
│   │   ├── pipeline.go      # Pipeline 执行引擎
│   │   ├── source.go        # Source 接口
│   │   ├── filter.go        # Filter 接口
│   │   ├── hydrator.go      # Hydrator 接口
│   │   ├── scorer.go        # Scorer 接口
│   │   ├── selector.go      # Selector 接口
│   │   ├── query_hydrator.go # QueryHydrator 接口
│   │   ├── side_effect.go   # SideEffect 接口
│   │   └── utils.go         # 辅助函数
│   └── utils/               # 工具函数 ✅
│       └── request.go       # 请求ID生成
├── pkg/
│   └── proto/               # gRPC 协议 ✅
│       └── scored_posts.proto
└── go.mod                   # Go 模块定义 ✅
```

## 已完成的工作

### Phase 1: 基础数据结构 ✅

已完成所有基础数据结构和接口定义。

### Phase 2: Pipeline 执行引擎 ✅

已完成 Pipeline 核心实现，包括：
- `CandidatePipeline` 结构体
- `Execute()` 主流程方法
- 所有阶段的执行方法（并行/顺序）
- 错误处理和日志记录

### Phase 3: gRPC 服务层 ✅

已完成 gRPC 服务层实现：
- Proto 文件定义 (`pkg/proto/scored_posts.proto`)
- gRPC 服务实现 (`internal/mixer/server.go`)
- 服务入口 (`cmd/server/main.go`)

**注意**：需要先运行 `protoc` 生成 proto 代码，详见 `PROTO_SETUP.md`。

#### ✅ TODO-1.1: 核心数据结构 (`internal/pipeline/types.go`)

定义了以下核心数据结构：

- **Query**: 推荐请求的查询对象
  - 包含用户信息、请求参数
  - 包含增强后的用户特征和历史（UserActionSequence, UserFeatures）
  - 实现了 `Clone()` 方法用于深拷贝

- **Candidate**: 候选帖子对象
  - 包含帖子ID、作者信息、内容
  - 包含 Phoenix 预测分数（PhoenixScores）
  - 包含各种元数据字段
  - 实现了 `Clone()` 和 `GetScreenNames()` 方法

- **PhoenixScores**: Phoenix 模型预测的各种交互概率分数
  - 正面动作分数（点赞、转发、回复等）
  - 负面动作分数（屏蔽、静音、举报等）
  - 连续动作（停留时间等）

- **UserFeatures**: 用户特征
  - 关注列表、屏蔽列表、静音列表等

- **UserActionSequence**: 用户交互历史序列
  - 包含用户最近的点赞、转发、回复等动作

- **PipelineResult**: 管道执行结果
  - 检索到的候选、被过滤的候选、最终选择的候选

- **FilterResult**: 过滤器执行结果
  - 保留的候选、移除的候选

#### ✅ TODO-1.2: Source 接口 (`internal/pipeline/source.go`)

定义了 `Source` 接口：
- `GetCandidates(ctx, query)`: 获取候选列表
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

#### ✅ TODO-1.3: Filter 接口 (`internal/pipeline/filter.go`)

定义了 `Filter` 接口：
- `Filter(ctx, query, candidates)`: 过滤候选列表
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

#### ✅ TODO-1.4: Hydrator 接口 (`internal/pipeline/hydrator.go`)

定义了 `Hydrator` 接口：
- `Hydrate(ctx, query, candidates)`: 增强候选列表
- `Update(candidate, hydrated)`: 更新单个候选
- `UpdateAll(candidates, hydrated)`: 批量更新候选
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

#### ✅ TODO-1.5: Scorer 接口 (`internal/pipeline/scorer.go`)

定义了 `Scorer` 接口：
- `Score(ctx, query, candidates)`: 为候选打分
- `Update(candidate, scored)`: 更新单个候选
- `UpdateAll(candidates, scored)`: 批量更新候选
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

#### ✅ TODO-1.6: Selector 接口 (`internal/pipeline/selector.go`)

定义了 `Selector` 接口：
- `Select(ctx, query, candidates)`: 选择候选列表
- `Score(candidate)`: 提取分数
- `Sort(candidates)`: 排序候选
- `Size()`: 返回选择数量
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

#### ✅ TODO-1.7: QueryHydrator 接口 (`internal/pipeline/query_hydrator.go`)

定义了 `QueryHydrator` 接口：
- `Hydrate(ctx, query)`: 增强查询对象
- `Update(query, hydrated)`: 更新查询对象
- `Name()`: 返回名称
- `Enable(query)`: 决定是否执行

## 代码特点

1. **类型安全**: 使用 Go 的强类型系统，确保类型安全
2. **深拷贝支持**: 所有主要数据结构都实现了 `Clone()` 方法
3. **接口设计**: 清晰的接口定义，便于实现和测试
4. **上下文支持**: 所有异步操作都支持 `context.Context`，便于取消和超时控制
5. **错误处理**: 所有可能失败的操作都返回 `error`

### Phase 2: Pipeline 执行引擎 ✅

已完成 Pipeline 核心实现：
- `CandidatePipeline` 结构体（包含所有组件列表）
- `Execute()` 主流程方法
- `hydrateQuery()` - 并行执行 Query Hydrators
- `fetchCandidates()` - 并行执行 Sources
- `hydrateCandidates()` - 并行执行 Hydrators
- `filterCandidates()` - 顺序执行 Filters
- `scoreCandidates()` - 顺序执行 Scorers
- `selectCandidates()` - 执行 Selector
- `hydratePostSelection()` - 并行执行 Post-Selection Hydrators
- `filterPostSelection()` - 顺序执行 Post-Selection Filters
- `runSideEffects()` - 异步执行 Side Effects
- 完整的错误处理和日志记录

### Phase 3: gRPC 服务层 ✅

已完成 gRPC 服务层实现：

#### ✅ TODO-3.1: Proto 文件定义 (`pkg/proto/scored_posts.proto`)
- `ScoredPostsQuery` 消息定义
- `ScoredPostsResponse` 消息定义
- `ScoredPost` 消息定义
- `ScoredPostsService` 服务定义

**注意**：需要运行 `protoc` 生成 Go 代码，详见 `PROTO_SETUP.md`

#### ✅ TODO-3.3: gRPC 服务实现 (`internal/mixer/server.go`)
- `HomeMixerServer` 结构体
- `GetScoredPosts()` gRPC 处理函数
- 参数验证
- Query 构建
- Pipeline 调用
- 响应转换
- 错误处理和日志

#### ✅ TODO-3.4: 服务入口 (`cmd/server/main.go`)
- Pipeline 初始化
- gRPC 服务器创建
- 服务注册
- 服务器启动
- 优雅关闭处理

### Phase 4: Sources 实现 ✅

已完成 Sources 实现：

#### ✅ TODO-4.1: Thunder Source (`internal/sources/thunder.go`)
- `ThunderSource` 结构体
- `GetCandidates()` 方法实现
- 从 Thunder 服务获取站内内容（关注账号的帖子）
- 支持关注列表过滤
- 构建回复链（ancestors）
- `ThunderClient` 接口定义（便于测试和替换实现）

#### ✅ TODO-4.2: Phoenix Source (`internal/sources/phoenix.go`)
- `PhoenixSource` 结构体
- `GetCandidates()` 方法实现
- 从 Phoenix Retrieval 服务获取站外内容（ML 检索）
- 需要 `user_action_sequence`
- 只在 `!in_network_only` 时启用
- `PhoenixRetrievalClient` 接口定义

#### ✅ Mock 实现 (`internal/sources/mock.go`)
- `MockThunderClient` - 用于测试
- `MockPhoenixRetrievalClient` - 用于测试

### Phase 5: Filters 实现 ✅

已完成过滤器实现：

#### ✅ TODO-5.1: Age Filter (`internal/filters/age.go`)
- `AgeFilter` 结构体
- 基于雪花ID提取时间，过滤过期帖子
- 使用 `utils.IsWithinAge()` 判断

#### ✅ TODO-5.2: Duplicate Filter (`internal/filters/duplicate.go`)
- `DropDuplicatesFilter` 结构体
- 基于 `tweet_id` 去重
- 使用 map 跟踪已见过的ID

#### ✅ TODO-5.3: Self Tweet Filter (`internal/filters/self_tweet.go`)
- `SelfTweetFilter` 结构体
- 移除用户自己发的帖子
- 比较 `author_id` 和 `viewer_id`

#### ✅ PreviouslySeenPostsFilter (`internal/filters/previously_seen.go`)
- 移除用户已经看过的帖子
- 使用 `seen_ids` 和 `bloom_filter_entries` 判断
- 检查相关帖子ID（原帖、转发、回复）

#### ✅ PreviouslyServedPostsFilter (`internal/filters/previously_served.go`)
- 移除本次会话中已经服务过的帖子
- 只在 `is_bottom_request` 时启用（分页请求）

#### ✅ MutedKeywordFilter (`internal/filters/muted_keyword.go`)
- 移除包含用户静音关键词的帖子
- 不区分大小写匹配

#### ✅ AuthorSocialgraphFilter (`internal/filters/author_socialgraph.go`)
- 移除来自屏蔽/静音作者的帖子
- 检查 `blocked_user_ids` 和 `muted_user_ids`

#### ✅ RetweetDeduplicationFilter (`internal/filters/retweet_dedup.go`)
- 转发去重，只保留第一次出现的帖子
- 无论是原帖还是转发，都只保留第一次出现

#### ✅ CoreDataHydrationFilter (`internal/filters/core_data_hydration.go`)
- 移除核心数据获取失败的候选
- 检查 `author_id` 和 `tweet_text` 是否有效

#### ✅ IneligibleSubscriptionFilter (`internal/filters/ineligible_subscription.go`)
- 移除用户未订阅的订阅内容
- 只保留用户已订阅作者的订阅内容

#### ✅ VFFilter (`internal/filters/vf.go`)
- 移除可见性过滤标记为不可见的帖子
- 检查 `visibility_reason`，移除已删除、垃圾内容等

#### ✅ DedupConversationFilter (`internal/filters/dedup_conversation.go`)
- 对话去重，每个对话分支只保留分数最高的候选
- 使用 `ancestors` 识别对话分支

#### ✅ TODO-10.1: 雪花ID工具 (`internal/utils/snowflake.go`)
- `DurationSinceCreation()` - 从雪花ID提取创建时间
- `CreationTime()` - 获取创建时间
- `IsWithinAge()` - 判断是否在指定年龄内

### Phase 6: Hydrators 实现 ✅

已完成增强器实现：

#### ✅ TODO-6.1: Core Data Hydrator (`internal/hydrators/core_data.go`)
- `CoreDataCandidateHydrator` 结构体
- `Hydrate()` 方法实现（批量获取帖子核心数据）
- `Update()` 和 `UpdateAll()` 方法
- `TweetEntityServiceClient` 接口定义

#### ✅ InNetworkCandidateHydrator (`internal/hydrators/in_network.go`)
- 标记候选是否为站内内容（来自关注账号）
- 基于 `followed_user_ids` 判断

#### ✅ GizmoduckCandidateHydrator (`internal/hydrators/gizmoduck.go`)
- 增强候选的作者信息（用户名、粉丝数等）
- 批量获取作者和转发作者信息
- `GizmoduckClient` 接口定义

#### ✅ VideoDurationCandidateHydrator (`internal/hydrators/video_duration.go`)
- 增强候选的视频时长信息
- 从媒体实体中提取视频时长
- 使用 `TESClient` 获取媒体信息

#### ✅ SubscriptionHydrator (`internal/hydrators/subscription.go`)
- 增强候选的订阅状态信息
- 标记哪些帖子是订阅内容
- 使用 `TESClient` 获取订阅作者ID

#### ✅ VFCandidateHydrator (`internal/hydrators/vf.go`)
- 增强候选的可见性信息（Visibility Filtering）
- 检查帖子是否可见（未删除、非垃圾内容等）
- 并行处理站内和站外内容
- `VisibilityFilteringClient` 接口定义

### Phase 7: Scorers 实现 ✅

已完成打分器实现：

#### ✅ TODO-7.1: Phoenix Scorer (`internal/scorers/phoenix.go`)
- `PhoenixScorer` 结构体
- `Score()` 方法实现（调用 Phoenix Ranking 服务）
- `Update()` 和 `UpdateAll()` 方法
- 填充 `PhoenixScores` 和元数据
- `PhoenixRankingClient` 接口定义

#### ✅ TODO-7.2: Weighted Scorer (`internal/scorers/weighted.go`)
- `WeightedScorer` 结构体
- `Score()` 方法实现（加权组合多个预测）
- `computeWeightedScore()` 辅助方法
- 权重配置（`ActionWeights`）
- 支持视频时长条件（VQV权重）
- 分数偏移和归一化

#### ✅ AuthorDiversityScorer (`internal/scorers/author_diversity.go`)
- 调整分数以确保 Feed 中作者多样性
- 重复出现的作者分数会衰减
- 可配置的衰减因子和最低倍数

#### ✅ OONScorer (`internal/scorers/oon.go`)
- 调整站外内容（Out-of-Network）的分数
- 优先显示站内内容，降低站外内容的分数
- 可配置的权重因子

### Phase 8: Selector 实现 ✅

#### ✅ TODO-8.1: TopK Selector (`internal/selectors/top_k.go`)
- `TopKScoreSelector` 结构体
- `Select()` 方法实现（按分数排序，选择 Top-K）
- `Score()` 方法提取分数
- `Sort()` 方法排序候选
- `Size()` 方法返回选择数量

### Phase 9: Query Hydrators 实现 ✅

已完成 Query Hydrators 实现：

#### ✅ TODO-9.1: UserActionSeqQueryHydrator (`internal/query_hydrators/user_action_seq.go`)
- `UserActionSeqQueryHydrator` 结构体
- `Hydrate()` 方法实现（获取用户交互历史）
- `Update()` 方法
- `UserActionSequenceFetcher` 接口定义

#### ✅ TODO-9.2: UserFeaturesQueryHydrator (`internal/query_hydrators/user_features.go`)
- `UserFeaturesQueryHydrator` 结构体
- `Hydrate()` 方法实现（获取用户特征，如关注列表）
- `Update()` 方法
- `StratoClient` 接口定义

#### ✅ Mock 实现 (`internal/query_hydrators/mock.go`)
- `MockUserActionSequenceFetcher` - 用于测试
- `MockStratoClient` - 用于测试

### Phase 11: Pipeline 配置 ✅

#### ✅ TODO-11.1: Pipeline 配置 (`internal/mixer/pipeline.go`)
- `PhoenixCandidatePipeline` 结构体
- `NewPhoenixCandidatePipeline()` 构造函数
- 配置所有组件：
  - Query Hydrators（UserActionSeq + UserFeatures）
  - Sources（Thunder + Phoenix）
  - Hydrators（InNetwork + Core Data + Gizmoduck + VideoDuration + Subscription）
  - Filters（Duplicate, CoreDataHydration, Age, Self Tweet, PreviouslySeen, PreviouslyServed, MutedKeyword, AuthorSocialgraph, RetweetDedup, IneligibleSubscription）
  - Scorers（Phoenix + Weighted + AuthorDiversity + OON）
  - Selector（TopK）
  - Post-Selection Hydrators（VF）
  - Post-Selection Filters（VF + DedupConversation）
  - Side Effects（CacheRequestInfo）

## 实现完成度

**第一部分核心功能已全部实现完成！** ✅

所有主要组件都已实现：
- ✅ 12个 Filters（包括 Pre-Scoring 和 Post-Selection）
- ✅ 6个 Hydrators（包括 Pre-Scoring 和 Post-Selection）
- ✅ 4个 Scorers
- ✅ 2个 Sources
- ✅ 2个 Query Hydrators
- ✅ 1个 Selector
- ✅ 完整的 Pipeline 配置

## 可选扩展

如果需要继续扩展，可以考虑：

- **Side Effects**（可选）
  - CacheRequestInfoSideEffect（缓存请求信息）
- **其他优化**（可选）
  - 性能优化
  - 监控和日志增强
  - 配置管理

## 参考文档

- `GO_IMPLEMENTATION_TODO.md` - 完整的实现计划
- `STAGE2_LEARNING_GUIDE.md` - 数据流和代码示例
- `MIGRATION_GUIDE_GO_PYTHON.md` - 详细迁移指南

## 构建和测试

### 前置要求

1. **Proto 代码**：
   - 已提供占位实现（`pkg/proto/scored_posts.pb.go`）
   - 如需实际运行，可以运行 protoc 生成代码（详见 `PROTO_SETUP.md`）
   - 当前占位实现已足够编译通过

2. **安装依赖**：
   ```bash
   cd go
   go mod tidy
   ```

### 构建

```bash
# 构建 pipeline 包
go build ./internal/pipeline/...

# 构建 mixer 包
go build ./internal/mixer/...

# 构建服务器
go build ./cmd/server/...
```

### 运行服务器

```bash
cd go
go run ./cmd/server/main.go -grpc_port=50051
```

### 测试

测试功能暂不实现。
