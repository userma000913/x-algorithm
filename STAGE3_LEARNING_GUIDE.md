# 第三阶段学习指南：深入各组件

> **适合人群**：已完成第二阶段学习  
> **预计时间**：3-5天  
> **目标**：深入理解各个组件的具体实现和工作原理

---

## 📚 学习目标

完成第三阶段后，你应该能够：

1. ✅ 理解 Sources（候选源）如何获取候选
2. ✅ 理解 Filters（过滤器）的过滤逻辑
3. ✅ 理解 Hydrators（增强器）如何补充数据
4. ✅ 理解 Scorers（打分器）如何计算分数
5. ✅ 能够阅读和理解各组件的代码实现

---

## 🎯 第一部分：Sources（候选源）

### 1.1 概述

**作用**：从不同数据源获取候选内容

**执行方式**：并行执行

**组件**：
- `ThunderSource`：站内内容（关注账号的帖子）
- `PhoenixSource`：站外内容（ML检索）

### 1.2 ThunderSource（站内内容源）

**文件位置**：`home-mixer/sources/thunder_source.rs`

**工作原理**：

```rust
pub struct ThunderSource {
    pub thunder_client: Arc<ThunderClient>,
}

impl Source<ScoredPostsQuery, PostCandidate> for ThunderSource {
    async fn get_candidates(&self, query: &ScoredPostsQuery) -> Result<Vec<PostCandidate>, String> {
        // 1. 获取 Thunder 客户端连接
        let channel = self.thunder_client.get_random_channel(ThunderCluster::Amp)?;
        let mut client = InNetworkPostsServiceClient::new(channel);
        
        // 2. 构建请求
        let request = GetInNetworkPostsRequest {
            user_id: query.user_id as u64,
            following_user_ids: query.user_features.followed_user_ids.iter().map(|&id| id as u64).collect(),
            max_results: p::THUNDER_MAX_RESULTS,  // 例如：500
            exclude_tweet_ids: vec![],
            algorithm: "default".to_string(),
            debug: false,
            is_video_request: false,
        };
        
        // 3. 调用 Thunder 服务
        let response = client.get_in_network_posts(request).await?;
        
        // 4. 转换为 PostCandidate
        let candidates: Vec<PostCandidate> = response
            .into_inner()
            .posts
            .into_iter()
            .map(|post| {
                PostCandidate {
                    tweet_id: post.post_id,
                    author_id: post.author_id as u64,
                    in_reply_to_tweet_id: post.in_reply_to_post_id,
                    ancestors: ...,  // 构建祖先链
                    served_type: Some(pb::ServedType::ForYouInNetwork),
                    ..Default::default()
                }
            })
            .collect();
        
        Ok(candidates)
    }
}
```

**关键点**：
- 从 Thunder 服务获取用户关注账号的帖子
- Thunder 是内存存储，查询速度很快（亚毫秒级）
- 返回的候选只有基本信息（ID、作者ID等），没有完整内容

**任务清单**：
- [ ] 阅读 `home-mixer/sources/thunder_source.rs`
- [ ] 理解如何调用 Thunder 服务
- [ ] 理解如何构建请求参数
- [ ] 理解如何转换响应为 PostCandidate

### 1.3 PhoenixSource（站外内容源）

**文件位置**：`home-mixer/sources/phoenix_source.rs`

**工作原理**：

```rust
pub struct PhoenixSource {
    pub phoenix_retrieval_client: Arc<dyn PhoenixRetrievalClient + Send + Sync>,
}

impl Source<ScoredPostsQuery, PostCandidate> for PhoenixSource {
    async fn get_candidates(&self, query: &ScoredPostsQuery) -> Result<Vec<PostCandidate>, String> {
        // 1. 准备检索请求
        // 使用用户的交互历史作为查询
        
        // 2. 调用 Phoenix Retrieval 服务
        // 使用 Two-Tower 模型进行相似度搜索
        
        // 3. 返回 Top-K 候选（例如：500条）
        
        // 4. 转换为 PostCandidate
    }
}
```

**关键点**：
- 调用 Phoenix Retrieval（Two-Tower 模型）
- 使用用户特征和历史进行相似度搜索
- 从全局语料库中发现相关内容

**任务清单**：
- [ ] 阅读 `home-mixer/sources/phoenix_source.rs`
- [ ] 理解如何调用 Phoenix Retrieval
- [ ] 理解检索请求的构建
- [ ] 理解检索结果的转换

### 1.4 实践练习

**练习1**：理解 Source 的 Trait 定义
- 阅读 `candidate-pipeline/source.rs`
- 理解 `Source` trait 的定义
- 理解 `get_candidates` 方法的签名

**练习2**：添加日志
- 在 `ThunderSource` 中添加日志，记录获取的候选数量
- 在 `PhoenixSource` 中添加日志，记录检索耗时

---

## 🔍 第二部分：Filters（过滤器）

### 2.1 概述

**作用**：移除不符合条件的候选

**执行方式**：顺序执行（每个 filter 基于前一个的结果）

**过滤器列表**（按执行顺序）：
1. `DropDuplicatesFilter`：去重
2. `CoreDataHydrationFilter`：移除数据获取失败的
3. `AgeFilter`：移除过期的
4. `SelfTweetFilter`：移除自己的帖子
5. `RetweetDeduplicationFilter`：转发去重
6. `IneligibleSubscriptionFilter`：移除无法访问的付费内容
7. `PreviouslySeenPostsFilter`：移除已看过的
8. `PreviouslyServedPostsFilter`：移除已服务的
9. `MutedKeywordFilter`：移除包含静音关键词的
10. `AuthorSocialgraphFilter`：移除屏蔽/静音作者的

### 2.2 AgeFilter（年龄过滤器）

**文件位置**：`home-mixer/filters/age_filter.rs`

**工作原理**：

```rust
pub struct AgeFilter {
    pub max_age: Duration,  // 例如：7天
}

impl Filter<ScoredPostsQuery, PostCandidate> for AgeFilter {
    async fn filter(
        &self,
        _query: &ScoredPostsQuery,
        candidates: Vec<PostCandidate>,
    ) -> Result<FilterResult<PostCandidate>, String> {
        // 使用 partition 将候选分为两部分
        let (kept, removed): (Vec<_>, Vec<_>) = candidates
            .into_iter()
            .partition(|c| self.is_within_age(c.tweet_id));
        
        Ok(FilterResult { kept, removed })
    }
}

impl AgeFilter {
    fn is_within_age(&self, tweet_id: i64) -> bool {
        // 从 tweet_id（雪花ID）中提取创建时间
        snowflake::duration_since_creation_opt(tweet_id)
            .map(|age| age <= self.max_age)  // 检查是否在最大年龄内
            .unwrap_or(false)  // 如果无法提取时间，返回 false（移除）
    }
}
```

**关键点**：
- 使用雪花ID（Snowflake ID）提取帖子创建时间
- 移除超过 `max_age` 的帖子（例如：7天）
- 使用 `partition` 高效地分离保留和移除的候选

**任务清单**：
- [ ] 阅读 `home-mixer/filters/age_filter.rs`
- [ ] 理解如何从 tweet_id 提取时间
- [ ] 理解 `partition` 的作用
- [ ] 理解 `FilterResult` 的结构

### 2.3 SelfTweetFilter（自己的帖子过滤器）

**文件位置**：`home-mixer/filters/self_tweet_filter.rs`

**工作原理**：

```rust
pub struct SelfTweetFilter;

impl Filter<ScoredPostsQuery, PostCandidate> for SelfTweetFilter {
    async fn filter(
        &self,
        query: &ScoredPostsQuery,
        candidates: Vec<PostCandidate>,
    ) -> Result<FilterResult<PostCandidate>, String> {
        let viewer_id = query.user_id as u64;
        
        // 移除作者ID等于用户ID的帖子
        let (kept, removed): (Vec<_>, Vec<_>) = candidates
            .into_iter()
            .partition(|c| c.author_id != viewer_id);
        
        Ok(FilterResult { kept, removed })
    }
}
```

**关键点**：
- 简单的条件判断：`author_id != viewer_id`
- 移除用户自己发的帖子

**任务清单**：
- [ ] 阅读 `home-mixer/filters/self_tweet_filter.rs`
- [ ] 理解过滤逻辑
- [ ] 理解为什么需要这个过滤器

### 2.4 其他重要过滤器

#### DropDuplicatesFilter（去重过滤器）

**作用**：移除重复的帖子ID

**实现**：使用 HashSet 记录已见过的ID

#### PreviouslySeenPostsFilter（已看过过滤器）

**作用**：移除用户已经看过的帖子

**实现**：检查 `query.seen_ids` 列表

#### AuthorSocialgraphFilter（作者关系过滤器）

**作用**：移除来自屏蔽/静音作者的帖子

**实现**：检查作者是否在屏蔽/静音列表中

### 2.5 实践练习

**练习1**：理解 Filter 的 Trait 定义
- 阅读 `candidate-pipeline/filter.rs`
- 理解 `Filter` trait 的定义
- 理解 `filter` 方法的签名和返回值

**练习2**：实现一个简单的过滤器
- 实现一个 `LanguageFilter`：只保留特定语言的帖子
- 集成到管道中

**练习3**：分析过滤器的执行顺序
- 理解为什么 `CoreDataHydrationFilter` 必须在 `CoreDataCandidateHydrator` 之后
- 理解为什么某些过滤器必须在其他过滤器之后

---

## 💧 第三部分：Hydrators（增强器）

### 3.1 概述

**作用**：为候选补充额外的数据

**执行方式**：并行执行（每个 hydrator 补充不同的数据）

**Hydrator 列表**：
1. `InNetworkCandidateHydrator`：标记是否站内内容
2. `CoreDataCandidateHydrator`：获取帖子核心数据
3. `VideoDurationCandidateHydrator`：获取视频时长
4. `SubscriptionHydrator`：获取订阅状态
5. `GizmoduckCandidateHydrator`：获取作者信息

### 3.2 CoreDataCandidateHydrator（核心数据增强器）

**文件位置**：`home-mixer/candidate_hydrators/core_data_candidate_hydrator.rs`

**工作原理**：

```rust
pub struct CoreDataCandidateHydrator {
    pub tes_client: Arc<dyn TESClient + Send + Sync>,  // Tweet Entity Service 客户端
}

impl Hydrator<ScoredPostsQuery, PostCandidate> for CoreDataCandidateHydrator {
    async fn hydrate(
        &self,
        _query: &ScoredPostsQuery,
        candidates: &[PostCandidate],
    ) -> Result<Vec<PostCandidate>, String> {
        // 1. 提取所有 tweet_id
        let tweet_ids = candidates.iter().map(|c| c.tweet_id).collect::<Vec<_>>();
        
        // 2. 批量获取核心数据
        let post_features = self.tes_client.get_tweet_core_datas(tweet_ids.clone()).await?;
        
        // 3. 为每个候选补充数据
        let mut hydrated_candidates = Vec::with_capacity(candidates.len());
        for tweet_id in tweet_ids {
            let core_data = post_features.get(&tweet_id);
            let hydrated = PostCandidate {
                author_id: core_data.map(|x| x.author_id).unwrap_or_default(),
                retweeted_user_id: core_data.and_then(|x| x.source_user_id),
                retweeted_tweet_id: core_data.and_then(|x| x.source_tweet_id),
                in_reply_to_tweet_id: core_data.and_then(|x| x.in_reply_to_tweet_id),
                tweet_text: core_data.map(|x| x.text.clone()).unwrap_or_default(),
                ..Default::default()
            };
            hydrated_candidates.push(hydrated);
        }
        
        Ok(hydrated_candidates)
    }
    
    fn update(&self, candidate: &mut PostCandidate, hydrated: PostCandidate) {
        // 更新候选的字段
        candidate.retweeted_user_id = hydrated.retweeted_user_id;
        candidate.retweeted_tweet_id = hydrated.retweeted_tweet_id;
        candidate.in_reply_to_tweet_id = hydrated.in_reply_to_tweet_id;
        candidate.tweet_text = hydrated.tweet_text;
    }
}
```

**关键点**：
- 批量获取数据（提高效率）
- 使用 `TESClient`（Tweet Entity Service）获取帖子核心数据
- 补充：文本内容、转发信息、回复信息等
- 如果数据获取失败，使用默认值

**任务清单**：
- [ ] 阅读 `home-mixer/candidate_hydrators/core_data_candidate_hydrator.rs`
- [ ] 理解批量获取的机制
- [ ] 理解 `update` 方法的作用
- [ ] 理解如何处理数据获取失败

### 3.3 GizmoduckCandidateHydrator（作者信息增强器）

**作用**：获取作者信息（用户名、认证状态等）

**实现**：调用 Gizmoduck 服务获取用户信息

### 3.4 实践练习

**练习1**：理解 Hydrator 的 Trait 定义
- 阅读 `candidate-pipeline/hydrator.rs`
- 理解 `Hydrator` trait 的定义
- 理解 `hydrate` 和 `update` 方法

**练习2**：理解并行执行
- 添加日志，记录每个 hydrator 的执行时间
- 观察它们是否真的并行执行

**练习3**：理解数据长度检查
- 理解为什么需要检查 `hydrated.len() == expected_len`
- 理解如果长度不匹配会发生什么

---

## 📊 第四部分：Scorers（打分器）

### 4.1 概述

**作用**：计算候选的相关性分数

**执行方式**：顺序执行（每个 scorer 基于前一个的结果）

**Scorer 列表**（按执行顺序）：
1. `PhoenixScorer`：ML 预测（调用 Phoenix 模型）
2. `WeightedScorer`：加权组合
3. `AuthorDiversityScorer`：多样性调整
4. `OONScorer`：站外内容调整

### 4.2 PhoenixScorer（Phoenix 打分器）

**文件位置**：`home-mixer/scorers/phoenix_scorer.rs`

**工作原理**：

```rust
pub struct PhoenixScorer {
    pub phoenix_client: Arc<dyn PhoenixPredictionClient + Send + Sync>,
}

impl Scorer<ScoredPostsQuery, PostCandidate> for PhoenixScorer {
    async fn score(
        &self,
        query: &ScoredPostsQuery,
        candidates: &[PostCandidate],
    ) -> Result<Vec<PostCandidate>, String> {
        // 1. 准备预测请求
        let user_id = query.user_id as u64;
        let sequence = query.user_action_sequence.clone();
        let tweet_infos: Vec<TweetInfo> = candidates.iter().map(|c| {
            TweetInfo {
                tweet_id: c.retweeted_tweet_id.unwrap_or(c.tweet_id as u64),
                author_id: c.retweeted_user_id.unwrap_or(c.author_id),
                ..Default::default()
            }
        }).collect();
        
        // 2. 调用 Phoenix 模型
        let response = self.phoenix_client.predict(user_id, sequence, tweet_infos).await?;
        
        // 3. 提取预测结果
        let predictions_map = self.build_predictions_map(&response);
        
        // 4. 为每个候选分配预测分数
        let scored_candidates = candidates.iter().map(|c| {
            let phoenix_scores = predictions_map
                .get(&lookup_tweet_id)
                .map(|preds| self.extract_phoenix_scores(preds))
                .unwrap_or_default();
            
            PostCandidate {
                phoenix_scores,
                prediction_request_id: Some(prediction_request_id),
                last_scored_at_ms,
                ..Default::default()
            }
        }).collect();
        
        Ok(scored_candidates)
    }
}
```

**关键点**：
- 调用 Phoenix Ranking 模型（Transformer）
- 输入：用户历史 + 候选帖子
- 输出：多个动作的概率（点赞、转发、回复等）
- 如果预测失败，返回默认值（不中断流程）

**任务清单**：
- [ ] 阅读 `home-mixer/scorers/phoenix_scorer.rs`
- [ ] 理解如何构建预测请求
- [ ] 理解如何解析预测结果
- [ ] 理解 `PhoenixScores` 的结构

### 4.3 WeightedScorer（加权打分器）

**文件位置**：`home-mixer/scorers/weighted_scorer.rs`

**工作原理**：

```rust
pub struct WeightedScorer;

impl Scorer<ScoredPostsQuery, PostCandidate> for WeightedScorer {
    async fn score(
        &self,
        _query: &ScoredPostsQuery,
        candidates: &[PostCandidate],
    ) -> Result<Vec<PostCandidate>, String> {
        let scored = candidates.iter().map(|c| {
            // 计算加权分数
            let weighted_score = Self::compute_weighted_score(c);
            let normalized_weighted_score = normalize_score(c, weighted_score);
            
            PostCandidate {
                weighted_score: Some(normalized_weighted_score),
                ..Default::default()
            }
        }).collect();
        
        Ok(scored)
    }
}

impl WeightedScorer {
    fn compute_weighted_score(candidate: &PostCandidate) -> f64 {
        let s: &PhoenixScores = &candidate.phoenix_scores;
        
        // 加权组合多个预测
        let combined_score = 
            Self::apply(s.favorite_score, p::FAVORITE_WEIGHT) +
            Self::apply(s.reply_score, p::REPLY_WEIGHT) +
            Self::apply(s.retweet_score, p::RETWEET_WEIGHT) +
            Self::apply(s.click_score, p::CLICK_WEIGHT) +
            // ... 其他动作
            Self::apply(s.not_interested_score, p::NOT_INTERESTED_WEIGHT) +  // 负权重
            Self::apply(s.block_author_score, p::BLOCK_AUTHOR_WEIGHT);  // 负权重
        
        Self::offset_score(combined_score)
    }
}
```

**关键点**：
- 将多个预测概率组合成单一分数
- 公式：`Σ(weight_i × P(action_i))`
- 正面动作（点赞、转发）使用正权重
- 负面动作（屏蔽、静音）使用负权重
- 最后进行分数归一化和偏移

**任务清单**：
- [ ] 阅读 `home-mixer/scorers/weighted_scorer.rs`
- [ ] 理解权重配置（`params.rs`）
- [ ] 理解分数归一化
- [ ] 理解为什么需要负权重

### 4.4 AuthorDiversityScorer（作者多样性打分器）

**作用**：衰减重复作者的分数，确保 Feed 多样性

**实现**：跟踪已出现的作者，降低重复作者的分数

### 4.5 实践练习

**练习1**：理解 Scorer 的 Trait 定义
- 阅读 `candidate-pipeline/scorer.rs`
- 理解 `Scorer` trait 的定义
- 理解 `score` 和 `update` 方法

**练习2**：修改权重
- 修改 `params.rs` 中的权重配置
- 观察对最终排序的影响

**练习3**：理解分数传递
- 追踪分数如何在各个 scorer 之间传递
- 理解每个 scorer 如何修改分数

---

## ✅ 第五步：自我检查

### 检查清单

完成以下检查，确保你理解了：

#### Sources
- [ ] 我能解释 ThunderSource 如何工作吗？
- [ ] 我能解释 PhoenixSource 如何工作吗？
- [ ] 我能解释为什么 Sources 可以并行执行吗？

#### Filters
- [ ] 我能解释至少3个过滤器的逻辑吗？
- [ ] 我能解释为什么 Filters 必须顺序执行吗？
- [ ] 我能解释过滤器的执行顺序吗？

#### Hydrators
- [ ] 我能解释至少2个 Hydrator 的逻辑吗？
- [ ] 我能解释为什么 Hydrators 可以并行执行吗？
- [ ] 我能解释如何处理数据获取失败吗？

#### Scorers
- [ ] 我能解释 PhoenixScorer 如何工作吗？
- [ ] 我能解释 WeightedScorer 如何工作吗？
- [ ] 我能解释为什么 Scorers 必须顺序执行吗？

---

## 🎓 实践练习总结

### 综合练习：实现一个新组件

选择一个简单的功能，实现一个新组件：

1. **新过滤器**：实现一个 `LanguageFilter`，只保留特定语言的帖子
2. **新 Hydrator**：实现一个 `SentimentHydrator`，补充帖子的情感分析结果
3. **新 Scorer**：实现一个 `RecencyScorer`，根据帖子新鲜度调整分数

### 代码阅读练习

选择3-5个组件，深入阅读代码：

1. 理解每个函数的作用
2. 理解数据如何流转
3. 理解错误处理机制
4. 理解性能优化点

---

## 📝 学习笔记模板

```
# 第三阶段学习笔记

## 日期：____

## Sources
ThunderSource：
[你的理解]

PhoenixSource：
[你的理解]

## Filters
AgeFilter：
[你的理解]

SelfTweetFilter：
[你的理解]

其他过滤器：
[你的理解]

## Hydrators
CoreDataCandidateHydrator：
[你的理解]

其他 Hydrator：
[你的理解]

## Scorers
PhoenixScorer：
[你的理解]

WeightedScorer：
[你的理解]

其他 Scorer：
[你的理解]

## 不懂的地方
[记录不懂的地方]

## 收获
[记录学到的知识]
```

---

## 🚀 下一步

完成第三阶段后，你应该：

1. ✅ 理解各个组件的实现
2. ✅ 能够阅读和理解代码
3. ✅ 理解各组件的协作方式

**准备好进入第四阶段了吗？**

第四阶段将深入学习：
- ML 模型（Two-Tower 检索模型）
- ML 模型（Transformer 排序模型）
- Candidate Isolation 机制

---

**祝你学习顺利！🎉**

记住：深入理解各个组件是理解整个系统的关键，多读代码，多实践！
