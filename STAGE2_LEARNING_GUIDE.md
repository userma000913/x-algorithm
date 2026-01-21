# 第二阶段学习指南：理解数据流

> **适合人群**：已完成第一阶段学习  
> **预计时间**：2-3天  
> **目标**：追踪一个请求的完整处理流程，理解各阶段的数据转换

---

## 📚 学习目标

完成第二阶段后，你应该能够：

1. ✅ 理解一个请求从入口到返回的完整流程
2. ✅ 理解每个阶段的数据结构和转换
3. ✅ 理解并行执行 vs 顺序执行的区别
4. ✅ 能够追踪代码执行路径
5. ✅ 理解错误处理和日志记录机制

---

## 🎯 第一步：理解请求入口

### 1.1 gRPC 服务入口

**文件位置**：`home-mixer/server.rs`

**关键代码**：

```go
// 示例：Go gRPC 服务入口（等价于 Rust 的 server.rs 入口逻辑）
//
// 说明：
// - 这里假设你已经用 protobuf 生成了 pb 包
// - pipeline.Execute(ctx, query) 返回 pipelineResult
// - 这是“结构示例”，方便你看懂数据流（不是可直接运行的完整工程）

type HomeMixerServer struct {
	pb.UnimplementedScoredPostsServiceServer
	pipeline *PhoenixCandidatePipeline
}

func (s *HomeMixerServer) GetScoredPosts(
	ctx context.Context,
	req *pb.ScoredPostsQuery,
) (*pb.ScoredPostsResponse, error) {
	// 1) 参数校验
	if req.GetViewerId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "viewer_id must be specified")
	}

	// 2) 构建内部 Query（等价于 ScoredPostsQuery::new(...)）
	query := NewScoredPostsQuery(
		req.GetViewerId(),
		req.GetClientAppId(),
		req.GetCountryCode(),
		req.GetLanguageCode(),
		req.GetSeenIds(),
		req.GetServedIds(),
		req.GetInNetworkOnly(),
		req.GetIsBottomRequest(),
		req.GetBloomFilterEntries(),
	)

	// 3) 执行候选管道（等价于 self.phx_candidate_pipeline.execute(query).await）
	pipelineResult, err := s.pipeline.Execute(ctx, query)
	if err != nil {
		// 这里你可以根据错误类型决定返回 codes.Internal / codes.DeadlineExceeded 等
		return nil, status.Errorf(codes.Internal, "pipeline execute failed: %v", err)
	}

	// 4) 转换为响应格式
	scoredPosts := make([]*pb.ScoredPost, 0, len(pipelineResult.SelectedCandidates))
	for _, c := range pipelineResult.SelectedCandidates {
		scoredPosts = append(scoredPosts, &pb.ScoredPost{
			TweetId:               uint64(c.TweetID),
			AuthorId:              uint64(c.AuthorID),
			RetweetedTweetId:      uint64(ptrOrZero(c.RetweetedTweetID)),
			RetweetedUserId:       uint64(ptrOrZero(c.RetweetedUserID)),
			InReplyToTweetId:      uint64(ptrOrZero(c.InReplyToTweetID)),
			Score:                 float32(floatOrZero(c.Score)),
			InNetwork:             boolOrFalse(c.InNetwork),
			ServedType:            int32(intOrZero(c.ServedType)),
			LastScoredTimestampMs: uint64(ptrOrZero(c.LastScoredAtMs)),
			PredictionRequestId:   uint64(ptrOrZero(c.PredictionRequestID)),
			Ancestors:             toU64Slice(c.Ancestors),
			ScreenNames:           c.ScreenNames,
			VisibilityReason:      c.VisibilityReason,
		})
	}

	return &pb.ScoredPostsResponse{ScoredPosts: scoredPosts}, nil
}
```

### 1.2 请求数据结构

**输入**：`ScoredPostsQuery`（gRPC 协议）
- `viewer_id`：用户 ID
- `client_app_id`：客户端应用 ID
- `country_code`：国家代码
- `language_code`：语言代码
- `seen_ids`：已看过的帖子 ID 列表
- `served_ids`：本次会话已服务的帖子 ID 列表
- `in_network_only`：是否只要站内内容
- `is_bottom_request`：是否是底部请求（分页）
- `bloom_filter_entries`：布隆过滤器条目（去重）

**输出**：`ScoredPostsResponse`
- `scored_posts`：排序后的帖子列表

### 1.2.1（补充）Go 示例里用到的“占位辅助函数”

上面的 Go 代码为了表达“字段可能为空（类似 Rust 的 `Option<T>`）”，用了几种占位函数。你实现自己的项目时，可以用更规范的方式（比如 protobuf 的 `optional` 字段，或自己定义 `NullXXX` 类型）。这里给一个**最简可读**的写法：

```go
func ptrOrZero[T ~int64 | ~uint64](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}

func floatOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func boolOrFalse(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func intOrZero(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func toU64Slice(xs []int64) []uint64 {
	out := make([]uint64, 0, len(xs))
	for _, x := range xs {
		out = append(out, uint64(x))
	}
	return out
}
```

### 1.3 任务清单

- [ ] 阅读 `home-mixer/server.rs`
- [ ] 理解 gRPC 服务如何接收请求
- [ ] 理解如何构建 `ScoredPostsQuery`
- [ ] 理解如何调用管道执行
- [ ] 理解如何转换结果并返回

---

## 🔄 第二步：理解管道执行引擎

### 2.1 管道执行流程

**文件位置**：`candidate-pipeline/candidate_pipeline.rs`

**核心方法**：`execute`

```go
// Go 版 Pipeline.Execute（等价于 Rust 的 candidate_pipeline.rs::execute）
func (p *CandidatePipeline) Execute(ctx context.Context, query *Query) (*PipelineResult, error) {
	// 1) Query Hydration（并行）
	hydratedQuery := p.hydrateQuery(ctx, query)

	// 2) Candidate Sourcing（并行）
	candidates := p.fetchCandidates(ctx, hydratedQuery)

	// 3) Candidate Hydration（并行）
	hydratedCandidates := p.hydrateCandidates(ctx, hydratedQuery, candidates)

	// 4) Pre-Scoring Filtering（顺序）
	keptCandidates, filteredCandidates := p.filterCandidates(ctx, hydratedQuery, hydratedCandidates)

	// 5) Scoring（顺序）
	scoredCandidates := p.scoreCandidates(ctx, hydratedQuery, keptCandidates)

	// 6) Selection（排序/截断）
	selectedCandidates := p.selector.Select(ctx, hydratedQuery, scoredCandidates)

	// 7) Post-Selection Hydration（并行）
	postHydrated := p.hydratePostSelection(ctx, hydratedQuery, selectedCandidates)

	// 8) Post-Selection Filtering（顺序）
	finalCandidates, postFiltered := p.filterPostSelection(ctx, hydratedQuery, postHydrated)
	filteredCandidates = append(filteredCandidates, postFiltered...)

	// 9) 截断到结果大小
	if p.resultSize > 0 && len(finalCandidates) > p.resultSize {
		finalCandidates = finalCandidates[:p.resultSize]
	}

	// 10) Side Effects（异步，不阻塞主链路）
	go p.runSideEffects(context.WithoutCancel(ctx), hydratedQuery, finalCandidates)

	return &PipelineResult{
		RetrievedCandidates: hydratedCandidates,
		FilteredCandidates:  filteredCandidates,
		SelectedCandidates:  finalCandidates,
		Query:               hydratedQuery,
	}, nil
}
```

### 2.2 数据流转换

```
ScoredPostsQuery（输入）
    ↓
hydrate_query()
    ↓
ScoredPostsQuery（增强后，包含用户历史、特征）
    ↓
fetch_candidates()
    ↓
Vec<PostCandidate>（初始候选，只有ID和基本信息）
    ↓
hydrate()
    ↓
Vec<PostCandidate>（增强后，包含完整数据）
    ↓
filter()
    ↓
Vec<PostCandidate>（过滤后，移除不符合条件的）
    ↓
score()
    ↓
Vec<PostCandidate>（打分后，包含分数）
    ↓
select()
    ↓
Vec<PostCandidate>（选择后，Top-K）
    ↓
hydrate_post_selection()
    ↓
Vec<PostCandidate>（选择后增强）
    ↓
filter_post_selection()
    ↓
Vec<PostCandidate>（最终候选）
    ↓
转换为 ScoredPostsResponse（输出）
```

### 2.3 任务清单

- [ ] 阅读 `candidate-pipeline/candidate_pipeline.rs` 的 `execute` 方法
- [ ] 理解每个阶段的输入输出
- [ ] 画出数据流转换图
- [ ] 理解 `PipelineResult` 的结构

---

## ⚡ 第三步：理解并行 vs 顺序执行

### 3.1 并行执行的阶段

**并行执行**：可以同时运行，互不依赖

#### Query Hydrators（查询增强器）

```go
// Go 版：并行执行 Query Hydrators，并把各自结果 merge 到 query（等价于 join_all + update）
func (p *CandidatePipeline) hydrateQuery(ctx context.Context, query *Query) *Query {
	hydrated := query.Clone()

	type item struct {
		h QueryHydrator
		r any
		e error
	}

	hydrators := make([]QueryHydrator, 0, len(p.queryHydrators))
	for _, h := range p.queryHydrators {
		if h.Enable(query) {
			hydrators = append(hydrators, h)
		}
	}

	ch := make(chan item, len(hydrators))
	var wg sync.WaitGroup
	for _, h := range hydrators {
		wg.Add(1)
		go func(h QueryHydrator) {
			defer wg.Done()
			res, err := h.Hydrate(ctx, query)
			ch <- item{h: h, r: res, e: err}
		}(h)
	}
	wg.Wait()
	close(ch)

	for it := range ch {
		if it.e != nil {
			// 记录错误，不中断流程（和 Rust 版一致）
			log.Printf("request_id=%s stage=QueryHydrator component=%s failed: %v",
				query.RequestID, it.h.Name(), it.e)
			continue
		}
		it.h.Update(hydrated, it.r)
	}

	return hydrated
}
```

**为什么可以并行？**
- 每个 hydrator 获取不同的数据
- 它们之间不相互依赖
- 最后合并结果即可

#### Sources（候选源）

```go
// Go 版：并行执行 Sources，最后合并所有候选（等价于 join_all + append）
func (p *CandidatePipeline) fetchCandidates(ctx context.Context, query *Query) []*Candidate {
	sources := make([]Source, 0, len(p.sources))
	for _, s := range p.sources {
		if s.Enable(query) {
			sources = append(sources, s)
		}
	}

	type item struct {
		s Source
		c []*Candidate
		e error
	}
	ch := make(chan item, len(sources))

	var wg sync.WaitGroup
	for _, s := range sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			cs, err := s.GetCandidates(ctx, query)
			ch <- item{s: s, c: cs, e: err}
		}(s)
	}
	wg.Wait()
	close(ch)

	var collected []*Candidate
	for it := range ch {
		if it.e != nil {
			log.Printf("request_id=%s stage=Source component=%s failed: %v",
				query.RequestID, it.s.Name(), it.e)
			continue
		}
		log.Printf("request_id=%s stage=Source component=%s fetched %d candidates",
			query.RequestID, it.s.Name(), len(it.c))
		collected = append(collected, it.c...)
	}
	return collected
}
```

**为什么可以并行？**
- Thunder Source 和 Phoenix Source 独立运行
- 它们从不同的数据源获取候选
- 最后合并即可

#### Hydrators（候选增强器）

```go
// Go 版：并行执行 Hydrators，并把每个 hydrator 的结果 merge 回 candidates
// 注意：为简化理解，这里选择“每个 hydrator 返回同长度的 hydratedCandidates，用来逐个 update”
func (p *CandidatePipeline) hydrateCandidates(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	hydrators := make([]Hydrator, 0, len(p.hydrators))
	for _, h := range p.hydrators {
		if h.Enable(query) {
			hydrators = append(hydrators, h)
		}
	}

	type item struct {
		h Hydrator
		r []*Candidate
		e error
	}
	ch := make(chan item, len(hydrators))
	var wg sync.WaitGroup
	for _, h := range hydrators {
		wg.Add(1)
		go func(h Hydrator) {
			defer wg.Done()
			hydrated, err := h.Hydrate(ctx, query, candidates)
			ch <- item{h: h, r: hydrated, e: err}
		}(h)
	}
	wg.Wait()
	close(ch)

	expectedLen := len(candidates)
	for it := range ch {
		if it.e != nil {
			log.Printf("request_id=%s stage=Hydrator component=%s failed: %v",
				query.RequestID, it.h.Name(), it.e)
			continue
		}
		if len(it.r) != expectedLen {
			log.Printf("request_id=%s stage=Hydrator component=%s skipped: length_mismatch expected=%d got=%d",
				query.RequestID, it.h.Name(), expectedLen, len(it.r))
			continue
		}
		// merge：逐个 candidate update（等价于 Rust 的 update_all）
		for i := 0; i < expectedLen; i++ {
			it.h.Update(candidates[i], it.r[i])
		}
	}
	return candidates
}
```

**为什么可以并行？**
- 每个 hydrator 补充不同的数据
- 它们之间不相互依赖
- 最后更新候选即可

### 3.2 顺序执行的阶段

**顺序执行**：必须按顺序运行，后面的依赖前面的结果

#### Filters（过滤器）

```go
// Go 版：顺序执行 Filters（每个 filter 以上一个 filter 的 kept 作为输入）
func (p *CandidatePipeline) filterCandidates(ctx context.Context, query *Query, candidates []*Candidate) (kept []*Candidate, removed []*Candidate) {
	kept = candidates
	for _, f := range p.filters {
		if !f.Enable(query) {
			continue
		}
		backup := append([]*Candidate(nil), kept...) // 备份，以防失败（等价于 Rust clone）
		res, err := f.Filter(ctx, query, kept)
		if err != nil {
			log.Printf("request_id=%s stage=Filter component=%s failed: %v",
				query.RequestID, f.Name(), err)
			kept = backup // 恢复备份
			continue
		}
		kept = res.Kept
		removed = append(removed, res.Removed...)
	}
	log.Printf("request_id=%s stage=Filter kept %d, removed %d", query.RequestID, len(kept), len(removed))
	return kept, removed
}
```

**为什么必须顺序执行？**
- 每个 filter 基于前一个 filter 的结果
- 例如：`CoreDataHydrationFilter` 需要先执行 `CoreDataCandidateHydrator`
- 如果并行执行，可能使用过时的数据

#### Scorers（打分器）

```go
// Go 版：顺序执行 Scorers（每个 scorer 基于上一个 scorer 已经更新过的 candidates）
func (p *CandidatePipeline) scoreCandidates(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	expectedLen := len(candidates)
	for _, s := range p.scorers {
		if !s.Enable(query) {
			continue
		}
		scored, err := s.Score(ctx, query, candidates)
		if err != nil {
			log.Printf("request_id=%s stage=Scorer component=%s failed: %v",
				query.RequestID, s.Name(), err)
			continue
		}
		if len(scored) != expectedLen {
			log.Printf("request_id=%s stage=Scorer component=%s skipped: length_mismatch expected=%d got=%d",
				query.RequestID, s.Name(), expectedLen, len(scored))
			continue
		}
		for i := 0; i < expectedLen; i++ {
			s.Update(candidates[i], scored[i])
		}
	}
	return candidates
}
```

**为什么必须顺序执行？**
- 每个 scorer 基于前一个 scorer 的结果
- 例如：`WeightedScorer` 需要先执行 `PhoenixScorer` 获取预测
- `AuthorDiversityScorer` 需要先执行 `WeightedScorer` 获取基础分数

### 3.3 任务清单

- [ ] 理解 `join_all` 的作用（并行执行）
- [ ] 理解为什么某些阶段可以并行，某些必须顺序
- [ ] 阅读并行执行的代码（hydrate_query, fetch_candidates, hydrate）
- [ ] 阅读顺序执行的代码（filter, score）
- [ ] 理解错误处理机制（备份、恢复）

---

## 📊 第四步：理解数据结构转换

### 4.1 Query（查询对象）

**初始状态**：
```rust
ScoredPostsQuery {
    user_id: 123,
    client_app_id: ...,
    country_code: ...,
    language_code: ...,
    seen_ids: [...],
    served_ids: [...],
    // 还没有用户历史、特征
}
```

**增强后**：
```rust
ScoredPostsQuery {
    user_id: 123,
    // ... 其他字段
    user_action_sequence: Some(UserActionSequence {
        // 用户最近的交互历史
        // 点赞、转发、回复等
    }),
    user_features: UserFeatures {
        followed_user_ids: [...],  // 关注列表
        // ... 其他特征
    },
}
```

### 4.2 Candidate（候选对象）

**初始状态**（从 Source 获取）：
```rust
PostCandidate {
    tweet_id: 12345,
    author_id: 67890,
    // 只有基本信息，没有内容、作者信息等
}
```

**增强后**（经过 Hydrators）：
```rust
PostCandidate {
    tweet_id: 12345,
    author_id: 67890,
    // 核心数据
    core_data: Some(CoreData { ... }),
    // 作者信息
    author_screen_name: Some("username"),
    author_verified: Some(true),
    // 视频时长
    video_duration_ms: Some(60000),
    // 订阅状态
    subscription_status: Some(...),
    // 是否站内内容
    in_network: Some(true),
}
```

**过滤后**：
- 数量减少（移除不符合条件的）
- 数据结构不变

**打分后**：
```rust
PostCandidate {
    // ... 之前的字段
    // 新增：Phoenix 预测分数
    phoenix_scores: Some(PhoenixScores {
        favorite_score: Some(0.8),
        reply_score: Some(0.3),
        retweet_score: Some(0.5),
        // ... 其他动作的分数
    }),
    // 新增：加权分数
    score: Some(0.75),
    // 新增：多样性调整后的分数
    // score: Some(0.70),  // 如果作者重复，分数会降低
}
```

**选择后**：
- 数量减少到 Top-K（例如 50）
- 按分数排序

### 4.3 任务清单

- [ ] 阅读 `home-mixer/candidate_pipeline/query.rs`（Query 结构）
- [ ] 阅读 `home-mixer/candidate_pipeline/candidate.rs`（Candidate 结构）
- [ ] 理解每个阶段如何修改数据结构
- [ ] 画出数据结构转换图

---

## 🔍 第五步：追踪一个完整请求

### 5.1 示例请求追踪

假设用户 ID 123 请求推荐：

```
1. 【入口】server.rs::get_scored_posts()
   输入：viewer_id=123, seen_ids=[100, 200]
   
2. 【Query Hydration】candidate_pipeline.rs::hydrate_query()
   并行执行：
   - UserActionSeqQueryHydrator：获取用户最近的交互历史
   - UserFeaturesQueryHydrator：获取关注列表
   输出：ScoredPostsQuery（包含用户历史、关注列表）
   
3. 【Candidate Sourcing】candidate_pipeline.rs::fetch_candidates()
   并行执行：
   - ThunderSource：从 Thunder 获取站内帖子（500条）
   - PhoenixSource：从 Phoenix Retrieval 获取站外帖子（500条）
   输出：Vec<PostCandidate>（1000条候选，只有ID）
   
4. 【Candidate Hydration】candidate_pipeline.rs::hydrate()
   并行执行：
   - CoreDataCandidateHydrator：获取帖子内容
   - GizmoduckCandidateHydrator：获取作者信息
   - VideoDurationCandidateHydrator：获取视频时长
   - SubscriptionHydrator：获取订阅状态
   - InNetworkCandidateHydrator：标记是否站内
   输出：Vec<PostCandidate>（1000条，数据完整）
   
5. 【Pre-Scoring Filtering】candidate_pipeline.rs::filter()
   顺序执行：
   - DropDuplicatesFilter：移除重复 → 950条
   - CoreDataHydrationFilter：移除数据获取失败的 → 900条
   - AgeFilter：移除过期的 → 800条
   - SelfTweetFilter：移除自己的 → 790条
   - ... 其他过滤器
   输出：Vec<PostCandidate>（假设最终 600条）
   
6. 【Scoring】candidate_pipeline.rs::score()
   顺序执行：
   - PhoenixScorer：调用 ML 模型，获取预测概率
   - WeightedScorer：计算加权分数
   - AuthorDiversityScorer：调整多样性
   - OONScorer：调整站外内容分数
   输出：Vec<PostCandidate>（600条，包含分数）
   
7. 【Selection】candidate_pipeline.rs::select()
   TopKScoreSelector：按分数排序，选择 Top 50
   输出：Vec<PostCandidate>（50条）
   
8. 【Post-Selection Hydration】candidate_pipeline.rs::hydrate_post_selection()
   VFCandidateHydrator：获取可见性信息
   输出：Vec<PostCandidate>（50条，包含可见性信息）
   
9. 【Post-Selection Filtering】candidate_pipeline.rs::filter_post_selection()
   顺序执行：
   - VFFilter：移除不可见的 → 45条
   - DedupConversationFilter：移除重复对话 → 43条
   输出：Vec<PostCandidate>（43条）
   
10. 【Side Effects】candidate_pipeline.rs::run_side_effects()
    异步执行：
    - CacheRequestInfoSideEffect：缓存请求信息
    不阻塞响应
    
11. 【返回】server.rs::get_scored_posts()
    转换为 ScoredPostsResponse
    返回：43条排序后的帖子
```

### 5.2 任务清单

- [ ] 使用调试器追踪一个请求
- [ ] 记录每个阶段的输入输出
- [ ] 记录每个阶段的候选数量变化
- [ ] 理解为什么某些阶段数量减少
- [ ] 理解为什么某些阶段可以并行

---

## 🛠️ 第六步：理解错误处理

### 6.1 错误处理策略

#### 并行执行的错误处理

```rust
// Query Hydrators
for (hydrator, result) in hydrators.iter().zip(results) {
    match result {
        Ok(hydrated) => {
            hydrator.update(&mut hydrated_query, hydrated);
        }
        Err(err) => {
            // 记录错误，但不中断流程
            error!("hydrator {} failed: {}", hydrator.name(), err);
            // 继续处理其他 hydrator 的结果
        }
    }
}
```

**策略**：一个失败不影响其他，继续处理成功的

#### 顺序执行的错误处理

```rust
// Filters
for filter in filters {
    let backup = candidates.clone();  // 备份
    match filter.filter(query, candidates).await {
        Ok(result) => {
            candidates = result.kept;
        }
        Err(err) => {
            error!("filter {} failed: {}", filter.name(), err);
            candidates = backup;  // 恢复备份
        }
    }
}
```

**策略**：失败时恢复备份，继续下一个 filter

### 6.2 日志记录

每个阶段都有详细的日志：

```go
// Go 版日志（建议统一结构，便于 grep / 日志平台检索）
log.Printf(
	"request_id=%s stage=%s component=%s fetched=%d",
	requestID,
	"Source",
	source.Name(),
	len(candidates),
)
```

**日志包含**：
- `request_id`：请求ID（用于追踪）
- `stage`：阶段（Source, Filter, Scorer等）
- `component`：组件名称
- 相关信息（候选数量、错误信息等）

### 6.3 任务清单

- [ ] 理解错误处理策略
- [ ] 理解为什么并行和顺序的错误处理不同
- [ ] 理解日志记录的作用
- [ ] 理解 `request_id` 的作用（追踪）

---

## ✅ 第七步：自我检查

### 检查清单

完成以下检查，确保你理解了：

#### 数据流理解
- [ ] 我能说出一个请求的完整流程吗？
- [ ] 我能解释每个阶段的输入输出吗？
- [ ] 我能解释数据结构的转换吗？

#### 执行模式理解
- [ ] 我能解释哪些阶段可以并行吗？
- [ ] 我能解释哪些阶段必须顺序执行吗？
- [ ] 我能解释为什么吗？

#### 代码追踪
- [ ] 我能追踪代码执行路径吗？
- [ ] 我能理解错误处理机制吗？
- [ ] 我能理解日志记录吗？

### 如果还有不懂的

1. **重新阅读代码**
   - 重点关注 `candidate_pipeline.rs` 的 `execute` 方法
   - 理解每个阶段的实现

2. **使用调试器**
   - 设置断点
   - 追踪变量变化
   - 观察数据转换

3. **画图帮助理解**
   - 画出数据流图
   - 标注每个阶段的输入输出
   - 标注并行/顺序执行

---

## 🎓 实践练习

### 练习1：追踪数据流

选择一个请求，手动追踪数据流：

1. 记录初始 Query 的内容
2. 记录每个阶段后 Query/Candidate 的变化
3. 记录候选数量的变化
4. 记录最终返回的结果

### 练习2：理解并行执行

修改代码，添加日志：

1. 在并行执行的阶段添加时间戳
2. 观察它们是否真的并行执行
3. 理解 `join_all` 的作用

### 练习3：理解顺序执行

修改代码，添加日志：

1. 在顺序执行的阶段添加时间戳
2. 观察它们的执行顺序
3. 理解为什么必须顺序执行

---

## 📝 学习笔记模板

```
# 第二阶段学习笔记

## 日期：____

## 请求入口
[你的理解]

## 管道执行流程
[画出流程图]

## 并行 vs 顺序执行
并行执行的阶段：
[列出并解释]

顺序执行的阶段：
[列出并解释]

## 数据结构转换
Query 的转换：
[记录变化]

Candidate 的转换：
[记录变化]

## 错误处理
[你的理解]

## 不懂的地方
[记录不懂的地方]

## 收获
[记录学到的知识]
```

---

## 🚀 下一步

完成第二阶段后，你应该：

1. ✅ 理解数据流的完整路径
2. ✅ 理解每个阶段的数据转换
3. ✅ 理解并行和顺序执行的区别

**准备好进入第三阶段了吗？**

第三阶段将深入学习：
- Sources（候选源）的具体实现
- Filters（过滤器）的具体实现
- Hydrators（增强器）的具体实现
- Scorers（打分器）的具体实现

---

**祝你学习顺利！🎉**

记住：理解数据流是理解整个系统的关键，多画图，多追踪代码！
