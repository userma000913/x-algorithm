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

```rust
pub struct HomeMixerServer {
    phx_candidate_pipeline: Arc<PhoenixCandidatePipeline>,
}

#[tonic::async_trait]
impl pb::scored_posts_service_server::ScoredPostsService for HomeMixerServer {
    async fn get_scored_posts(
        &self,
        request: Request<pb::ScoredPostsQuery>,
    ) -> Result<Response<ScoredPostsResponse>, Status> {
        // 1. 解析请求
        let proto_query = request.into_inner();
        
        // 2. 验证参数
        if proto_query.viewer_id == 0 {
            return Err(Status::invalid_argument("viewer_id must be specified"));
        }
        
        // 3. 构建查询对象
        let query = ScoredPostsQuery::new(...);
        
        // 4. 执行管道
        let pipeline_result = self.phx_candidate_pipeline.execute(query).await;
        
        // 5. 转换为响应格式
        let scored_posts: Vec<ScoredPost> = ...;
        
        // 6. 返回结果
        Ok(Response::new(ScoredPostsResponse { scored_posts }))
    }
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

```rust
async fn execute(&self, query: Q) -> PipelineResult<Q, C> {
    // 1. Query Hydration（查询增强）
    let hydrated_query = self.hydrate_query(query).await;
    
    // 2. Candidate Sourcing（候选获取）
    let candidates = self.fetch_candidates(&hydrated_query).await;
    
    // 3. Candidate Hydration（候选增强）
    let hydrated_candidates = self.hydrate(&hydrated_query, candidates).await;
    
    // 4. Pre-Scoring Filtering（打分前过滤）
    let (kept_candidates, mut filtered_candidates) = 
        self.filter(&hydrated_query, hydrated_candidates.clone()).await;
    
    // 5. Scoring（打分）
    let scored_candidates = self.score(&hydrated_query, kept_candidates).await;
    
    // 6. Selection（选择）
    let selected_candidates = self.select(&hydrated_query, scored_candidates);
    
    // 7. Post-Selection Hydration（选择后增强）
    let post_selection_hydrated_candidates = 
        self.hydrate_post_selection(&hydrated_query, selected_candidates).await;
    
    // 8. Post-Selection Filtering（选择后过滤）
    let (mut final_candidates, post_selection_filtered_candidates) = 
        self.filter_post_selection(&hydrated_query, post_selection_hydrated_candidates).await;
    
    // 9. 截断到结果大小
    final_candidates.truncate(self.result_size());
    
    // 10. Side Effects（副作用）
    self.run_side_effects(input);
    
    // 11. 返回结果
    PipelineResult { ... }
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

```rust
async fn hydrate_query(&self, query: Q) -> Q {
    // 并行执行所有 query hydrators
    let hydrate_futures = hydrators.iter().map(|h| h.hydrate(&query));
    let results = join_all(hydrate_futures).await;  // 等待所有完成
    
    // 合并结果
    let mut hydrated_query = query;
    for (hydrator, result) in hydrators.iter().zip(results) {
        hydrator.update(&mut hydrated_query, hydrated);
    }
    hydrated_query
}
```

**为什么可以并行？**
- 每个 hydrator 获取不同的数据
- 它们之间不相互依赖
- 最后合并结果即可

#### Sources（候选源）

```rust
async fn fetch_candidates(&self, query: &Q) -> Vec<C> {
    // 并行执行所有 sources
    let source_futures = sources.iter().map(|s| s.get_candidates(query));
    let results = join_all(source_futures).await;  // 等待所有完成
    
    // 合并结果
    let mut collected = Vec::new();
    for (source, result) in sources.iter().zip(results) {
        collected.append(&mut candidates);
    }
    collected
}
```

**为什么可以并行？**
- Thunder Source 和 Phoenix Source 独立运行
- 它们从不同的数据源获取候选
- 最后合并即可

#### Hydrators（候选增强器）

```rust
async fn hydrate(&self, query: &Q, candidates: Vec<C>) -> Vec<C> {
    // 并行执行所有 hydrators
    let hydrate_futures = hydrators.iter().map(|h| h.hydrate(query, &candidates));
    let results = join_all(hydrate_futures).await;
    
    // 更新所有候选
    for (hydrator, result) in hydrators.iter().zip(results) {
        hydrator.update_all(&mut candidates, hydrated);
    }
    candidates
}
```

**为什么可以并行？**
- 每个 hydrator 补充不同的数据
- 它们之间不相互依赖
- 最后更新候选即可

### 3.2 顺序执行的阶段

**顺序执行**：必须按顺序运行，后面的依赖前面的结果

#### Filters（过滤器）

```rust
async fn filter(&self, query: &Q, candidates: Vec<C>) -> (Vec<C>, Vec<C>) {
    let mut kept = candidates;
    let mut removed = Vec::new();
    
    // 顺序执行每个 filter
    for filter in filters {
        let backup = kept.clone();  // 备份，以防失败
        match filter.filter(query, kept).await {
            Ok(result) => {
                kept = result.kept;  // 使用过滤后的结果
                removed.extend(result.removed);
            }
            Err(err) => {
                kept = backup;  // 失败时恢复
            }
        }
    }
    (kept, removed)
}
```

**为什么必须顺序执行？**
- 每个 filter 基于前一个 filter 的结果
- 例如：`CoreDataHydrationFilter` 需要先执行 `CoreDataCandidateHydrator`
- 如果并行执行，可能使用过时的数据

#### Scorers（打分器）

```rust
async fn score(&self, query: &Q, mut candidates: Vec<C>) -> Vec<C> {
    // 顺序执行每个 scorer
    for scorer in scorers {
        match scorer.score(query, &candidates).await {
            Ok(scored) => {
                scorer.update_all(&mut candidates, scored);  // 更新候选
            }
            Err(err) => {
                // 记录错误，继续下一个
            }
        }
    }
    candidates
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

```rust
info!(
    "request_id={} stage={:?} component={} fetched {} candidates",
    request_id,
    PipelineStage::Source,
    source.name(),
    candidates.len()
);
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
