# X Algorithm 项目概述

## 项目简介

这是 X（Twitter）的 "For You" 推荐系统核心算法实现。该系统负责为用户生成个性化的内容流，结合站内（In-Network）和站外（Out-of-Network）内容，使用基于 Grok 的 Transformer 模型进行排序。

### 核心价值

1. **完全基于 ML 的排序**：消除了所有手工特征工程，使用 Transformer 模型自动学习内容相关性
2. **两阶段推荐架构**：检索（Retrieval）+ 排序（Ranking），兼顾效率和准确性
3. **高性能实现**：Rust 服务层实现亚毫秒级响应，内存存储实现快速查询
4. **可组合架构**：基于 Trait 的设计，各组件可独立开发、测试和替换

---

## 技术架构

### 技术栈

| 层级 | 技术 | 用途 |
|------|------|------|
| 服务层 | Rust + Tokio | 高并发、低延迟的服务实现 |
| ML 层 | Python + JAX + Haiku | 灵活的 ML 模型实现 |
| 通信 | gRPC | 服务间通信 |
| 数据流 | Kafka | 实时事件流处理 |
| 存储 | 内存存储（Thunder） | 亚毫秒级查询 |

### 系统组件

#### 1. Home Mixer（编排层）
- **位置**：`home-mixer/`
- **职责**：协调整个推荐流程
- **接口**：gRPC 服务（`ScoredPostsService`）
- **核心**：实现 `CandidatePipeline` trait，组装各个阶段

#### 2. Thunder（站内内容源）
- **位置**：`thunder/`
- **职责**：
  - 从 Kafka 消费帖子创建/删除事件
  - 维护内存中的帖子存储（按用户组织）
  - 提供亚毫秒级的站内内容查询
- **特点**：自动清理过期数据，支持原始帖子、回复/转发、视频帖子

#### 3. Phoenix（ML 组件）
- **位置**：`phoenix/`
- **职责**：
  - **检索（Retrieval）**：Two-Tower 模型进行相似度搜索
  - **排序（Ranking）**：Transformer 模型预测交互概率
- **特点**：基于 Grok-1 架构，支持 Candidate Isolation

#### 4. Candidate Pipeline（管道框架）
- **位置**：`candidate-pipeline/`
- **职责**：提供可重用的推荐管道框架
- **核心 Trait**：
  - `Source`：获取候选
  - `Hydrator`：增强数据
  - `Filter`：过滤候选
  - `Scorer`：计算分数
  - `Selector`：选择最终结果

---

## 核心流程

### 完整处理流程

```
用户请求
  ↓
1. Query Hydration（查询增强）
   ├─ 获取用户交互历史
   └─ 获取用户特征（关注列表等）
  ↓
2. Candidate Sourcing（候选获取）
   ├─ Thunder Source：站内内容（关注账号的帖子）
   └─ Phoenix Source：站外内容（ML 检索）
  ↓
3. Candidate Hydration（候选增强）
   ├─ 补充帖子核心数据
   ├─ 补充作者信息
   └─ 补充视频时长等
  ↓
4. Pre-Scoring Filtering（打分前过滤）
   ├─ 去重、年龄过滤
   ├─ 移除自己的帖子
   ├─ 移除已看过的帖子
   └─ 移除屏蔽/静音的内容
  ↓
5. Scoring（打分）
   ├─ Phoenix Scorer：ML 预测交互概率
   ├─ Weighted Scorer：加权组合
   ├─ Author Diversity Scorer：多样性调整
   └─ OON Scorer：站外内容调整
  ↓
6. Selection（选择）
   └─ Top-K 选择（按分数排序）
  ↓
7. Post-Selection Hydration（选择后增强）
   └─ 补充可见性信息
  ↓
8. Post-Selection Filtering（选择后过滤）
   ├─ 可见性过滤（删除、垃圾、暴力等）
   └─ 对话去重
  ↓
9. Side Effects（副作用）
   └─ 异步缓存请求信息
  ↓
返回排序后的 Feed
```

### 关键设计决策

#### 1. 无手工特征工程
- 系统完全依赖 Grok Transformer 模型学习相关性
- 模型从用户交互历史中自动学习特征
- 显著减少了数据管道和服务基础设施的复杂性

#### 2. Candidate Isolation（候选隔离）
- 在 Transformer 推理时，候选之间不能相互关注
- 确保候选的分数不依赖于批次中的其他候选
- 使得分数一致且可缓存

#### 3. Hash-Based Embeddings
- 使用多个哈希函数进行嵌入查找
- 减少嵌入表大小，同时保持表达能力

#### 4. 多动作预测
- 不是预测单一的"相关性"分数
- 同时预测多个交互类型的概率（点赞、转发、回复等）
- 通过加权组合得到最终分数

#### 5. 可组合管道架构
- `candidate-pipeline` crate 提供灵活的框架
- 分离管道执行和监控与业务逻辑
- 支持并行执行和优雅的错误处理
- 易于添加新的 sources、hydrations、filters 和 scorers

---

## 数据流详解

### 输入数据

**用户请求（ScoredPostsQuery）**：
- `viewer_id`：用户 ID
- `client_app_id`：客户端应用 ID
- `country_code`：国家代码
- `language_code`：语言代码
- `seen_ids`：已看过的帖子 ID 列表
- `served_ids`：本次会话已服务的帖子 ID 列表
- `in_network_only`：是否只要站内内容
- `is_bottom_request`：是否是底部请求（用于分页）
- `bloom_filter_entries`：布隆过滤器条目（用于去重）

### 中间数据

**Query（查询对象）**：
- 用户交互历史（User Action Sequence）
- 用户特征（关注列表、偏好设置等）

**Candidate（候选对象）**：
- 帖子 ID、作者 ID
- 帖子内容、媒体信息
- 作者信息（用户名、认证状态等）
- 视频时长、订阅状态
- 是否站内内容
- 各种分数（Phoenix 分数、加权分数、最终分数）

### 输出数据

**ScoredPostsResponse**：
- `scored_posts`：排序后的帖子列表
  - `tweet_id`：帖子 ID
  - `author_id`：作者 ID
  - `score`：最终分数
  - `in_network`：是否站内内容
  - `served_type`：服务类型
  - 其他元数据

---

## 性能特点

### 延迟优化

1. **并行执行**：
   - Sources 并行执行
   - Hydrators 并行执行
   - Side Effects 异步执行（不阻塞响应）

2. **内存存储**：
   - Thunder 使用内存存储，实现亚毫秒级查询
   - 自动清理过期数据，控制内存使用

3. **缓存**：
   - 缓存用户特征和交互历史
   - 缓存 ML 模型预测结果（得益于 Candidate Isolation）

### 可扩展性

1. **水平扩展**：
   - 无状态服务设计
   - 可以水平扩展多个实例

2. **组件解耦**：
   - 各组件通过 Trait 接口解耦
   - 可以独立优化和替换

3. **异步处理**：
   - 使用 Tokio 异步运行时
   - 支持高并发请求处理

---

## 代码结构

### 目录组织

```
x-algorithm/
├── candidate-pipeline/          # 管道框架（可重用）
│   ├── candidate_pipeline.rs    # 执行引擎
│   ├── source.rs                 # Source trait
│   ├── hydrator.rs              # Hydrator trait
│   ├── filter.rs                # Filter trait
│   ├── scorer.rs                # Scorer trait
│   └── selector.rs               # Selector trait
│
├── home-mixer/                   # 编排层（业务逻辑）
│   ├── server.rs                # gRPC 服务入口
│   ├── candidate_pipeline/      # 管道实现
│   │   └── phoenix_candidate_pipeline.rs
│   ├── sources/                 # 候选源
│   │   ├── thunder_source.rs
│   │   └── phoenix_source.rs
│   ├── filters/                 # 过滤器
│   │   ├── age_filter.rs
│   │   ├── author_socialgraph_filter.rs
│   │   └── ...
│   ├── scorers/                 # 打分器
│   │   ├── phoenix_scorer.rs
│   │   ├── weighted_scorer.rs
│   │   └── ...
│   └── candidate_hydrators/     # 候选增强器
│       ├── core_data_candidate_hydrator.rs
│       └── ...
│
├── thunder/                      # 站内内容源
│   ├── posts/
│   │   └── post_store.rs        # 内存存储
│   ├── kafka/                   # Kafka 事件处理
│   └── thunder_service.rs       # 服务实现
│
└── phoenix/                      # ML 组件
    ├── recsys_model.py          # 排序模型（Transformer）
    ├── recsys_retrieval_model.py # 检索模型（Two-Tower）
    ├── run_ranker.py            # 排序模型运行脚本
    └── run_retrieval.py          # 检索模型运行脚本
```

### 关键文件说明

| 文件 | 作用 |
|------|------|
| `candidate-pipeline/candidate_pipeline.rs` | 管道执行引擎，协调各阶段 |
| `home-mixer/server.rs` | gRPC 服务入口，处理请求 |
| `home-mixer/candidate_pipeline/phoenix_candidate_pipeline.rs` | 管道配置，组装各组件 |
| `phoenix/recsys_model.py` | Transformer 排序模型 |
| `phoenix/recsys_retrieval_model.py` | Two-Tower 检索模型 |

---

## 学习路径

### 快速开始（1周）

1. **Day 1-2：理解架构**
   - 阅读 `README.md`
   - 阅读 `phoenix/README.md`
   - 画出架构图

2. **Day 3-4：追踪数据流**
   - 阅读 `home-mixer/server.rs`
   - 阅读 `candidate-pipeline/candidate_pipeline.rs`
   - 理解各阶段的输入输出

3. **Day 5：运行代码**
   - 设置开发环境
   - 运行 Phoenix 模型测试
   - 理解执行流程

### 深入学习（3-4周）

#### Week 1：核心组件
- Sources：理解候选获取
- Filters：理解过滤逻辑
- Hydrators：理解数据增强

#### Week 2：打分系统
- Scorers：理解打分逻辑
- ML 模型调用：理解如何调用 Phoenix

#### Week 3：ML 模型
- Two-Tower 检索模型
- Transformer 排序模型
- Candidate Isolation 机制

#### Week 4：实践实验
- 修改过滤器
- 调整打分权重
- 性能优化

详细学习路径请参考 `LEARNING_GUIDE.md`。

---

## 关键概念

### 1. 两阶段推荐

**检索（Retrieval）**：
- 从百万级候选集中快速召回相关内容
- 使用 Two-Tower 模型进行相似度搜索
- 返回 Top-K 候选（通常 K=1000）

**排序（Ranking）**：
- 对检索到的候选进行精确排序
- 使用 Transformer 模型预测交互概率
- 返回 Top-K 结果（通常 K=50）

### 2. Candidate Pipeline

可组合的管道框架，包含以下阶段：

1. **Query Hydration**：增强查询（用户上下文）
2. **Source**：获取候选
3. **Hydration**：增强候选数据
4. **Filter**：过滤不符合条件的候选
5. **Scorer**：计算候选分数
6. **Selector**：选择最终结果
7. **Post-Selection Processing**：选择后处理
8. **Side Effects**：异步副作用

### 3. Candidate Isolation

在 Transformer 推理时，使用特殊的注意力掩码：
- 候选可以关注用户和历史
- 候选之间不能相互关注（只能自关注）
- 确保候选分数独立且可缓存

### 4. 多动作预测

模型同时预测多个交互类型的概率：
- 正面动作：点赞、转发、回复、分享等
- 负面动作：不感兴趣、屏蔽、静音、举报等
- 通过加权组合得到最终分数

---

## 扩展阅读

### 相关论文

1. **Two-Stage Recommendation Systems**
   - 检索 + 排序的两阶段架构

2. **Transformer for Recommendation**
   - Transformer 在推荐系统中的应用

3. **Candidate Isolation**
   - 确保候选分数独立性的方法

### 相关技术

1. **JAX/Haiku**
   - JAX 文档：https://jax.readthedocs.io/
   - Haiku 文档：https://dm-haiku.readthedocs.io/

2. **Rust Async**
   - Tokio 文档：https://tokio.rs/
   - Rust Async Book：https://rust-lang.github.io/async-book/

3. **gRPC**
   - gRPC 文档：https://grpc.io/

---

## 贡献指南

### 代码风格

- **Rust**：遵循 Rust 官方风格指南
- **Python**：遵循 PEP 8

### 测试

- 所有新功能需要添加测试
- 运行测试：`cargo test`（Rust）或 `uv run pytest`（Python）

### 文档

- 新功能需要更新文档
- 代码注释要清晰明了

---

## 许可证

本项目采用 Apache License 2.0 许可证。详见 `LICENSE` 文件。

---

## 联系方式

如有问题或建议，请参考项目文档或提交 Issue。

---

**最后更新**：2024年
