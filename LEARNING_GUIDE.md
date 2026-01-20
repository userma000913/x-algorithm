# X Algorithm 学习指南

## 目录

- [项目概述](#项目概述)
- [系统架构](#系统架构)
- [核心流程详解](#核心流程详解)
- [学习路径](#学习路径)
- [关键技术点](#关键技术点)

---

## 项目概述

### 项目简介

这是 X（Twitter）的 "For You" 推荐系统核心算法实现。该系统负责为用户生成个性化的内容流，结合站内（In-Network）和站外（Out-of-Network）内容，使用基于 Grok 的 Transformer 模型进行排序。

### 核心特点

1. **两阶段推荐架构**
   - **检索（Retrieval）**：从百万级候选集中快速召回相关内容
   - **排序（Ranking）**：使用 Transformer 模型对召回结果进行精确排序

2. **完全基于 ML 的排序**
   - 消除了所有手工特征工程
   - 使用 Grok-1 Transformer 架构理解用户交互历史
   - 模型自动学习内容相关性

3. **可组合的管道架构**
   - 基于 Trait 的灵活设计
   - 各组件可独立开发、测试和替换
   - 支持并行执行和优雅的错误处理

4. **高性能实现**
   - Rust 实现服务层（低延迟、高并发）
   - Python/JAX 实现 ML 模型（灵活、高效）
   - 内存存储（Thunder）实现亚毫秒级查询

### 技术栈

- **服务层**：Rust + Tokio（异步运行时）
- **ML 模型**：Python + JAX + Haiku
- **通信协议**：gRPC
- **数据流**：Kafka（实时事件流）
- **存储**：内存存储（Thunder）

---

## 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                    FOR YOU FEED REQUEST                     │
└─────────────────────────────────────────────────────────────────────────────┘
                                               │
                                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                         HOME MIXER                          │
│                                    (Orchestration Layer)                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                   QUERY HYDRATION                   │   │
│   │  ┌──────────────────────────┐    ┌──────────────────────────────┐   │   │
│   │  │ User Action Sequence     │    │ User Features                │   │   │
│   │  │ (engagement history)     │    │ (following list, etc.)       │   │   │
│   │  └──────────────────────────┘    └──────────────────────────────┘   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                  CANDIDATE SOURCES                  │   │
│   │         ┌─────────────────────────────┐    ┌──────────────────┐      │   │
│   │         │        THUNDER              │    │ PHOENIX RETRIEVAL│      │   │
│   │         │    (In-Network Posts)       │    │ (Out-of-Network) │      │   │
│   │         │                             │    │                  │      │   │
│   │         │  Posts from accounts        │    │ ML-based search  │      │   │
│   │         │  you follow                 │    │ across corpus     │      │   │
│   │         └─────────────────────────────┘    └──────────────────┘      │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                      HYDRATION                      │   │
│   │  Fetch: core metadata, author info, media, video duration, etc.     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                      FILTERING                      │   │
│   │  Remove: duplicates, old posts, self-posts, blocked, muted, etc.   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                       SCORING                       │   │
│   │  ┌──────────────────────────┐                                       │   │
│   │  │  Phoenix Scorer          │    Grok Transformer predicts:        │   │
│   │  │  (ML Predictions)        │    P(like), P(reply), P(repost)...    │   │
│   │  └──────────────────────────┘                                       │   │
│   │               │                                                      │   │
│   │               ▼                                                      │   │
│   │  ┌──────────────────────────┐                                       │   │
│   │  │  Weighted Scorer         │    Weighted Score = Σ(weight × P)    │   │
│   │  │  (Combine predictions)   │                                       │   │
│   │  └──────────────────────────┘                                       │   │
│   │               │                                                      │   │
│   │               ▼                                                      │   │
│   │  ┌──────────────────────────┐                                       │   │
│   │  │  Author Diversity       │    Attenuate repeated authors         │   │
│   │  │  Scorer                 │    for feed diversity                  │   │
│   │  └──────────────────────────┘                                       │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                                      SELECTION                      │   │
│   │                    Sort by score, select top K                     │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                              │                              │
│                                              ▼                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                              FILTERING (Post-Selection)             │   │
│   │                 Visibility filtering (deleted/spam/violence)         │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                               │
                                               ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                     RANKED FEED RESPONSE                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. Home Mixer（编排层）
- **位置**：`home-mixer/`
- **职责**：协调整个推荐流程，组装 For You Feed
- **接口**：gRPC 服务（`ScoredPostsService`）

#### 2. Thunder（站内内容源）
- **位置**：`thunder/`
- **职责**：
  - 从 Kafka 消费帖子创建/删除事件
  - 维护内存中的帖子存储
  - 提供亚毫秒级的站内内容查询

#### 3. Phoenix（ML 组件）
- **位置**：`phoenix/`
- **职责**：
  - **检索**：Two-Tower 模型进行相似度搜索
  - **排序**：Transformer 模型预测交互概率

#### 4. Candidate Pipeline（管道框架）
- **位置**：`candidate-pipeline/`
- **职责**：提供可重用的推荐管道框架
- **核心 Trait**：`Source`、`Hydrator`、`Filter`、`Scorer`、`Selector`

---

## 核心流程详解

### 完整请求处理流程

#### 阶段 1：Query Hydration（查询增强）

**目的**：获取用户上下文信息

**执行方式**：并行执行所有 Query Hydrators

**组件**：
1. **UserActionSeqQueryHydrator**
   - 获取用户最近的交互历史（点赞、转发、回复等）
   - 用于理解用户兴趣和偏好

2. **UserFeaturesQueryHydrator**
   - 获取用户特征（关注列表、偏好设置等）
   - 用于后续的过滤和排序

**代码位置**：
- `home-mixer/query_hydrators/user_action_seq_query_hydrator.rs`
- `home-mixer/query_hydrators/user_features_query_hydrator.rs`

**关键代码**：
```rust
async fn hydrate_query(&self, query: Q) -> Q {
    // 并行执行所有 query hydrators
    let hydrate_futures = hydrators.iter().map(|h| h.hydrate(&query));
    let results = join_all(hydrate_futures).await;
    // 合并结果到 query
}
```

---

#### 阶段 2：Candidate Sourcing（候选获取）

**目的**：从不同数据源获取候选内容

**执行方式**：并行执行所有 Sources

**组件**：
1. **ThunderSource（站内内容）**
   - 从 Thunder 获取用户关注账号的最新帖子
   - 支持原始帖子、回复/转发、视频帖子

2. **PhoenixSource（站外内容）**
   - 调用 Phoenix Retrieval 服务
   - 使用 Two-Tower 模型进行相似度搜索
   - 从全局语料库中发现相关内容

**代码位置**：
- `home-mixer/sources/thunder_source.rs`
- `home-mixer/sources/phoenix_source.rs`
- `thunder/posts/post_store.rs`

**关键代码**：
```rust
async fn fetch_candidates(&self, query: &Q) -> Vec<C> {
    // 并行执行所有 sources
    let source_futures = sources.iter().map(|s| s.get_candidates(query));
    let results = join_all(source_futures).await;
    // 合并所有候选
}
```

---

#### 阶段 3：Candidate Hydration（候选增强）

**目的**：为候选内容补充必要的元数据

**执行方式**：并行执行所有 Hydrators

**组件**：
1. **InNetworkCandidateHydrator**
   - 标记内容是否来自用户关注的账号

2. **CoreDataCandidateHydrator**
   - 获取帖子核心数据（文本、媒体等）

3. **VideoDurationCandidateHydrator**
   - 获取视频时长信息

4. **SubscriptionHydrator**
   - 获取订阅状态信息

5. **GizmoduckCandidateHydrator**
   - 获取作者信息（用户名、认证状态等）

**代码位置**：
- `home-mixer/candidate_hydrators/`

**关键代码**：
```rust
async fn hydrate(&self, query: &Q, candidates: Vec<C>) -> Vec<C> {
    // 并行执行所有 hydrators
    let hydrate_futures = hydrators.iter().map(|h| h.hydrate(query, &candidates));
    let results = join_all(hydrate_futures).await;
    // 更新所有候选
}
```

---

#### 阶段 4：Pre-Scoring Filtering（打分前过滤）

**目的**：移除不符合条件的候选

**执行方式**：顺序执行所有 Filters（每个 filter 基于前一个的结果）

**过滤器列表**：
1. **DropDuplicatesFilter**：移除重复的帖子 ID
2. **CoreDataHydrationFilter**：移除核心数据获取失败的帖子
3. **AgeFilter**：移除超过年龄阈值的帖子
4. **SelfTweetFilter**：移除用户自己的帖子
5. **RetweetDeduplicationFilter**：去重相同内容的转发
6. **IneligibleSubscriptionFilter**：移除用户无法访问的付费内容
7. **PreviouslySeenPostsFilter**：移除用户已看过的帖子
8. **PreviouslyServedPostsFilter**：移除本次会话已服务的帖子
9. **MutedKeywordFilter**：移除包含用户静音关键词的帖子
10. **AuthorSocialgraphFilter**：移除被屏蔽/静音作者的帖子

**代码位置**：
- `home-mixer/filters/`

**关键代码**：
```rust
async fn filter(&self, query: &Q, candidates: Vec<C>) -> (Vec<C>, Vec<C>) {
    let mut kept = candidates;
    let mut removed = Vec::new();
    // 顺序执行每个 filter
    for filter in filters {
        let (kept_new, removed_new) = filter.filter(query, kept).await;
        kept = kept_new;
        removed.extend(removed_new);
    }
    (kept, removed)
}
```

---

#### 阶段 5：Scoring（打分）

**目的**：计算每个候选的相关性分数

**执行方式**：顺序执行所有 Scorers（每个 scorer 基于前一个的结果）

**打分器列表**：
1. **PhoenixScorer（ML 预测）**
   - 调用 Phoenix Ranking 模型
   - 输入：用户上下文 + 候选帖子
   - 输出：多个交互类型的概率
     - P(like), P(reply), P(repost), P(quote)
     - P(click), P(profile_click), P(video_view)
     - P(photo_expand), P(share), P(dwell)
     - P(follow_author)
     - P(not_interested), P(block_author), P(mute_author), P(report)

2. **WeightedScorer（加权组合）**
   - 将多个预测概率组合成单一分数
   - 公式：`Final Score = Σ(weight_i × P(action_i))`
   - 正面动作（点赞、转发）使用正权重
   - 负面动作（屏蔽、静音）使用负权重

3. **AuthorDiversityScorer（作者多样性）**
   - 衰减重复作者的分值
   - 确保 Feed 的多样性

4. **OONScorer（站外内容调整）**
   - 调整站外内容的分数

**代码位置**：
- `home-mixer/scorers/phoenix_scorer.rs`
- `home-mixer/scorers/weighted_scorer.rs`
- `home-mixer/scorers/author_diversity_scorer.rs`
- `phoenix/recsys_model.py`

**关键代码**：
```rust
async fn score(&self, query: &Q, mut candidates: Vec<C>) -> Vec<C> {
    // 顺序执行每个 scorer
    for scorer in scorers {
        let scored = scorer.score(query, &candidates).await?;
        scorer.update_all(&mut candidates, scored);
    }
    candidates
}
```

---

#### 阶段 6：Selection（选择）

**目的**：根据分数排序并选择 Top-K

**执行方式**：同步执行

**组件**：
- **TopKScoreSelector**
  - 按最终分数降序排序
  - 选择前 K 个候选（K = `params::RESULT_SIZE`）

**代码位置**：
- `home-mixer/selectors/top_k_score_selector.rs`

**关键代码**：
```rust
fn select(&self, query: &Q, candidates: Vec<C>) -> Vec<C> {
    let mut sorted = candidates;
    sorted.sort_by(|a, b| b.score.cmp(&a.score));
    sorted.truncate(self.result_size());
    sorted
}
```

---

#### 阶段 7：Post-Selection Hydration（选择后增强）

**目的**：为已选择的候选补充额外数据

**执行方式**：并行执行

**组件**：
- **VFCandidateHydrator**
  - 获取可见性过滤信息（删除、垃圾、暴力等）

**代码位置**：
- `home-mixer/candidate_hydrators/vf_candidate_hydrator.rs`

---

#### 阶段 8：Post-Selection Filtering（选择后过滤）

**目的**：最终验证和过滤

**执行方式**：顺序执行

**过滤器列表**：
1. **VFFilter**
   - 移除已删除、垃圾、暴力、血腥等内容

2. **DedupConversationFilter**
   - 去重同一对话线程的多个分支

**代码位置**：
- `home-mixer/filters/vf_filter.rs`
- `home-mixer/filters/dedup_conversation_filter.rs`

---

#### 阶段 9：Side Effects（副作用）

**目的**：异步执行非关键操作

**执行方式**：并行执行（异步，不阻塞响应）

**组件**：
- **CacheRequestInfoSideEffect**
  - 缓存请求信息供后续使用

**代码位置**：
- `home-mixer/side_effects/cache_request_info_side_effect.rs`

**关键代码**：
```rust
fn run_side_effects(&self, input: Arc<SideEffectInput<Q, C>>) {
    tokio::spawn(async move {
        // 异步执行，不阻塞主流程
        let futures = side_effects.iter().map(|se| se.run(input.clone()));
        join_all(futures).await;
    });
}
```

---

### 数据流示例

```
用户请求 (viewer_id=123)
    ↓
Query Hydration
    ├─→ UserActionSeqQueryHydrator: 获取最近100条交互历史
    └─→ UserFeaturesQueryHydrator: 获取关注列表、偏好设置
    ↓
Candidate Sourcing
    ├─→ ThunderSource: 返回500条站内帖子
    └─→ PhoenixSource: 返回500条站外帖子
    ↓ (合并: 1000条候选)
Candidate Hydration
    ├─→ CoreDataCandidateHydrator: 补充帖子内容
    ├─→ GizmoduckCandidateHydrator: 补充作者信息
    └─→ VideoDurationCandidateHydrator: 补充视频时长
    ↓ (1000条候选，已增强)
Pre-Scoring Filtering
    ├─→ DropDuplicatesFilter: 移除50条重复 → 950条
    ├─→ AgeFilter: 移除100条过期 → 850条
    ├─→ SelfTweetFilter: 移除10条自己的 → 840条
    └─→ ... (其他过滤器)
    ↓ (假设最终保留: 600条候选)
Scoring
    ├─→ PhoenixScorer: 调用ML模型，获取交互概率
    ├─→ WeightedScorer: 计算加权分数
    ├─→ AuthorDiversityScorer: 调整多样性
    └─→ OONScorer: 调整站外内容分数
    ↓ (600条候选，已打分)
Selection
    └─→ TopKScoreSelector: 选择Top 50
    ↓ (50条候选)
Post-Selection Hydration
    └─→ VFCandidateHydrator: 补充可见性信息
    ↓ (50条候选)
Post-Selection Filtering
    ├─→ VFFilter: 移除5条不可见 → 45条
    └─→ DedupConversationFilter: 移除2条重复对话 → 43条
    ↓ (43条最终候选)
Side Effects
    └─→ CacheRequestInfoSideEffect: 异步缓存
    ↓
返回给用户 (43条排序后的帖子)
```

---

## 学习路径

### 第一阶段：理解整体架构（1-2天）

#### 目标
- 理解系统的整体架构和设计理念
- 掌握推荐系统的基本概念

#### 任务清单
- [ ] 阅读 `README.md`，理解系统架构图
- [ ] 阅读 `phoenix/README.md`，理解 ML 组件
- [ ] 画出自己的架构图，标注各组件职责
- [ ] 理解两阶段推荐（检索 + 排序）的概念

#### 关键概念
- **两阶段推荐**：先检索（召回），再排序（精排）
- **Candidate Pipeline**：可组合的管道架构
- **Candidate Isolation**：排序时候选之间不相互关注

---

### 第二阶段：理解数据流（2-3天）

#### 目标
- 追踪一个请求的完整处理流程
- 理解各阶段的数据转换

#### 任务清单
- [ ] 阅读 `home-mixer/server.rs`，理解入口点
- [ ] 阅读 `home-mixer/candidate_pipeline/phoenix_candidate_pipeline.rs`，理解管道配置
- [ ] 阅读 `candidate-pipeline/candidate_pipeline.rs`，理解执行逻辑
- [ ] 画出数据流图，标注每个阶段的输入输出
- [ ] 理解并行执行（sources, hydrators）vs 顺序执行（filters, scorers）

#### 关键文件
```
home-mixer/server.rs                          # 入口
home-mixer/candidate_pipeline/phoenix_candidate_pipeline.rs  # 管道配置
candidate-pipeline/candidate_pipeline.rs      # 执行引擎
```

#### 实践建议
- 在代码中添加注释，标注每个阶段的作用
- 使用调试器追踪一个请求的完整流程
- 记录每个阶段候选数量的变化

---

### 第三阶段：深入各组件（3-5天）

#### 3.1 候选源（Sources）（1天）

**目标**：理解如何获取候选内容

**任务清单**：
- [ ] 阅读 `home-mixer/sources/thunder_source.rs`
- [ ] 阅读 `home-mixer/sources/phoenix_source.rs`
- [ ] 阅读 `thunder/posts/post_store.rs`，理解内存存储
- [ ] 理解 Thunder 如何从 Kafka 消费事件
- [ ] 理解 Phoenix Retrieval 的调用方式

**关键问题**：
- Thunder 如何实现亚毫秒级查询？
- Phoenix Source 如何调用检索服务？
- 两个 Source 的结果如何合并？

---

#### 3.2 过滤器（Filters）（1天）

**目标**：理解各种过滤逻辑

**任务清单**：
- [ ] 阅读所有过滤器实现
- [ ] 理解过滤器的执行顺序
- [ ] 理解为什么过滤器要顺序执行
- [ ] 分析每个过滤器的过滤条件

**重点过滤器**：
- `AgeFilter`：如何判断帖子年龄？
- `AuthorSocialgraphFilter`：如何检查作者关系？
- `PreviouslySeenPostsFilter`：如何判断用户是否看过？

**实践建议**：
- 尝试添加一个新的过滤器（如：过滤特定语言的帖子）
- 分析过滤器的性能影响

---

#### 3.3 增强器（Hydrators）（1天）

**目标**：理解如何补充候选数据

**任务清单**：
- [ ] 阅读所有 Hydrator 实现
- [ ] 理解 Hydrator 的并行执行机制
- [ ] 理解如何优雅处理 Hydrator 失败
- [ ] 分析每个 Hydrator 获取的数据

**关键问题**：
- 为什么 Hydrator 可以并行执行？
- 如何处理 Hydrator 返回数据长度不匹配？
- 如何避免重复调用外部服务？

---

#### 3.4 打分器（Scorers）（2天）

**目标**：理解打分逻辑和 ML 模型调用

**任务清单**：
- [ ] 阅读 `home-mixer/scorers/phoenix_scorer.rs`
- [ ] 阅读 `home-mixer/scorers/weighted_scorer.rs`
- [ ] 阅读 `home-mixer/scorers/author_diversity_scorer.rs`
- [ ] 理解打分器的执行顺序
- [ ] 理解如何组合多个预测概率

**关键问题**：
- Phoenix Scorer 如何调用 ML 模型？
- Weighted Scorer 的权重如何设置？
- Author Diversity Scorer 如何确保多样性？

**实践建议**：
- 尝试修改权重，观察结果变化
- 分析不同 Scorer 对最终分数的影响

---

### 第四阶段：理解 ML 模型（5-7天）

#### 4.1 检索模型（Two-Tower）（2-3天）

**目标**：理解检索阶段的 ML 模型

**任务清单**：
- [ ] 阅读 `phoenix/recsys_retrieval_model.py`
- [ ] 理解 Two-Tower 架构
- [ ] 理解 User Tower 如何编码用户特征
- [ ] 理解 Candidate Tower 如何编码帖子
- [ ] 理解相似度搜索（点积）

**关键概念**：
- **User Tower**：编码用户特征和交互历史
- **Candidate Tower**：编码所有帖子
- **相似度搜索**：使用点积找到最相关的帖子

**实践建议**：
- 运行 `uv run run_retrieval.py`
- 阅读 `phoenix/test_recsys_retrieval_model.py`
- 尝试修改模型参数，观察效果

---

#### 4.2 排序模型（Transformer）（3-4天）

**目标**：理解排序阶段的 Transformer 模型

**任务清单**：
- [ ] 阅读 `phoenix/recsys_model.py`
- [ ] 理解基于 Grok-1 的 Transformer 架构
- [ ] 理解 Candidate Isolation 的注意力掩码
- [ ] 理解多动作预测
- [ ] 理解 Hash-Based Embeddings

**关键概念**：
- **Candidate Isolation**：候选之间不能相互关注
- **注意力掩码**：控制 Transformer 的注意力模式
- **多动作预测**：同时预测多个交互类型的概率

**实践建议**：
- 运行 `uv run run_ranker.py`
- 阅读 `phoenix/test_recsys_model.py`
- 画出注意力掩码的可视化图
- 理解为什么需要 Candidate Isolation

---

### 第五阶段：实践与实验（持续）

#### 5.1 运行代码

**任务清单**：
- [ ] 设置开发环境（Rust + Python）
- [ ] 编译 Rust 代码
- [ ] 运行 Phoenix 模型测试
- [ ] 理解如何启动服务

**命令**：
```bash
# 运行排序模型
cd phoenix
uv run run_ranker.py

# 运行检索模型
uv run run_retrieval.py

# 运行测试
uv run pytest test_recsys_model.py test_recsys_retrieval_model.py
```

---

#### 5.2 阅读测试

**任务清单**：
- [ ] 阅读 `phoenix/test_recsys_model.py`
- [ ] 阅读 `phoenix/test_recsys_retrieval_model.py`
- [ ] 理解测试用例的设计
- [ ] 理解模型的预期行为

---

#### 5.3 修改实验

**实验建议**：
1. **修改过滤器顺序**
   - 调整过滤器的执行顺序
   - 观察对结果的影响

2. **修改打分权重**
   - 调整 Weighted Scorer 的权重
   - 观察不同权重对排序的影响

3. **添加新过滤器**
   - 实现一个新的过滤器（如：语言过滤）
   - 集成到管道中

4. **分析性能**
   - 测量各阶段的耗时
   - 识别性能瓶颈

---

## 关键技术点

### 1. Rust 异步编程

**关键概念**：
- `async/await` 语法
- `tokio` 异步运行时
- `futures` 和 `join_all` 并行执行

**学习资源**：
- Rust Async Book
- Tokio 文档

---

### 2. Trait 设计模式

**关键 Trait**：
- `Source`：获取候选
- `Hydrator`：增强数据
- `Filter`：过滤候选
- `Scorer`：计算分数
- `Selector`：选择最终结果

**设计优势**：
- 可组合性
- 可测试性
- 可扩展性

---

### 3. JAX/Haiku ML 框架

**关键概念**：
- JAX：NumPy 的加速版本，支持自动微分
- Haiku：JAX 的神经网络库
- 函数式编程风格

**学习资源**：
- JAX 文档
- Haiku 文档

---

### 4. Transformer 架构

**关键概念**：
- 自注意力机制（Self-Attention）
- 多头注意力（Multi-Head Attention）
- 位置编码（Positional Encoding）
- 注意力掩码（Attention Mask）

**本项目特点**：
- 基于 Grok-1 架构
- Candidate Isolation（候选隔离）
- 多动作预测

---

### 5. 推荐系统核心概念

**关键概念**：
- **两阶段推荐**：检索 + 排序
- **特征工程**：本项目消除了手工特征
- **多样性**：Author Diversity Scorer
- **冷启动**：如何处理新用户
- **实时性**：Thunder 的实时数据流

---

## 推荐学习顺序总结

### Week 1: 整体架构 + 数据流
- Day 1-2: 阅读文档，理解架构
- Day 3-5: 追踪代码，理解数据流

### Week 2: Sources + Filters + Hydrators
- Day 1: Sources（候选获取）
- Day 2: Filters（过滤逻辑）
- Day 3: Hydrators（数据增强）

### Week 3: Scorers + ML 模型基础
- Day 1-2: Scorers（打分逻辑）
- Day 3-5: ML 模型基础（Two-Tower）

### Week 4: Phoenix 模型深入 + 实验
- Day 1-3: Transformer 模型深入
- Day 4-5: 实验和优化

---

## 学习建议

### 1. 先理解整体，再深入细节
- 不要一开始就陷入代码细节
- 先理解数据流和架构设计

### 2. 画流程图
- 画出数据在各阶段的流转
- 标注每个阶段的输入输出

### 3. 运行代码
- 实际运行代码，观察执行流程
- 使用调试器追踪变量变化

### 4. 阅读测试
- 测试代码是最好的文档
- 理解预期行为和边界情况

### 5. 做笔记
- 记录关键设计决策
- 记录不理解的地方，后续深入研究

### 6. 实践实验
- 尝试修改代码
- 观察修改对结果的影响
- 理解各组件的作用

---

## 常见问题

### Q: 为什么过滤器要顺序执行？
A: 因为每个过滤器可能依赖前一个过滤器的结果。例如，`CoreDataHydrationFilter` 需要先执行 `CoreDataCandidateHydrator`。

### Q: 为什么 Hydrator 可以并行执行？
A: 因为 Hydrator 之间是独立的，它们只是补充不同的数据，不相互依赖。

### Q: Candidate Isolation 的作用是什么？
A: 确保候选的分数不依赖于批次中的其他候选，使得分数一致且可缓存。

### Q: 为什么使用 Hash-Based Embeddings？
A: 减少嵌入表的大小，同时保持表达能力。

---

## 下一步

完成基础学习后，可以：
1. 深入研究某个特定组件
2. 优化性能瓶颈
3. 添加新功能
4. 理解生产环境的部署

祝学习顺利！
