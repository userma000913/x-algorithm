# Go 实现 TODO 列表

> **目标**：根据 STAGE2_LEARNING_GUIDE.md 中的 Go 代码示例，逐步实现完整的推荐系统  
> **使用方式**：按照顺序逐个实现，每完成一个 TODO 项，标记为完成

---

## 📋 项目结构

```
go/
├── cmd/
│   └── server/
│       └── main.go                    # 服务入口
├── internal/
│   ├── pipeline/                      # 管道框架（核心）
│   │   ├── pipeline.go               # Pipeline 执行引擎
│   │   ├── types.go                  # 数据结构定义
│   │   ├── source.go                 # Source 接口
│   │   ├── filter.go                 # Filter 接口
│   │   ├── hydrator.go               # Hydrator 接口
│   │   ├── scorer.go                 # Scorer 接口
│   │   ├── selector.go               # Selector 接口
│   │   └── query_hydrator.go         # QueryHydrator 接口
│   ├── mixer/                        # Home Mixer（业务层）
│   │   ├── server.go                 # gRPC 服务实现
│   │   └── pipeline.go               # 管道配置
│   ├── sources/                      # 候选源实现
│   │   ├── thunder.go                # Thunder 源
│   │   └── phoenix.go                # Phoenix 检索源
│   ├── filters/                      # 过滤器实现
│   │   ├── age.go                    # 年龄过滤
│   │   ├── duplicate.go              # 去重过滤
│   │   ├── self_tweet.go             # 自己的帖子过滤
│   │   └── ...                       # 其他过滤器
│   ├── hydrators/                    # 增强器实现
│   │   ├── core_data.go              # 核心数据增强
│   │   ├── author.go                 # 作者信息增强
│   │   └── ...                       # 其他增强器
│   ├── scorers/                      # 打分器实现
│   │   ├── phoenix.go                # Phoenix 打分器
│   │   ├── weighted.go               # 加权打分器
│   │   └── ...                       # 其他打分器
│   └── utils/                        # 工具函数
│       ├── snowflake.go              # 雪花ID工具
│       └── helpers.go                # 辅助函数
├── pkg/
│   └── proto/                        # gRPC 协议（自动生成）
│       └── scored_posts.proto        # 协议定义
└── go.mod                            # Go 模块定义
```

---

## 🎯 Phase 1: 基础数据结构（优先级：最高）

### 1.1 定义核心数据结构

- [ ] **TODO-1.1**: 创建 `internal/pipeline/types.go`
  - [ ] 定义 `Query` 结构体（包含所有字段）
  - [ ] 定义 `Candidate` 结构体（包含所有字段）
  - [ ] 定义 `PipelineResult` 结构体
  - [ ] 定义 `FilterResult` 结构体
  - [ ] 定义 `PhoenixScores` 结构体
  - [ ] 定义 `UserActionSequence` 结构体
  - [ ] 定义 `UserFeatures` 结构体
  - [ ] 添加必要的辅助方法（Clone, 指针处理等）

**参考**：STAGE2_LEARNING_GUIDE.md 中的数据结构示例

---

### 1.2 定义接口

- [ ] **TODO-1.2**: 创建 `internal/pipeline/source.go`
  - [ ] 定义 `Source` 接口
    ```go
    type Source interface {
        GetCandidates(ctx context.Context, query *Query) ([]*Candidate, error)
        Name() string
        Enable(query *Query) bool
    }
    ```

- [ ] **TODO-1.3**: 创建 `internal/pipeline/filter.go`
  - [ ] 定义 `Filter` 接口
    ```go
    type Filter interface {
        Filter(ctx context.Context, query *Query, candidates []*Candidate) (*FilterResult, error)
        Name() string
        Enable(query *Query) bool
    }
    ```

- [ ] **TODO-1.4**: 创建 `internal/pipeline/hydrator.go`
  - [ ] 定义 `Hydrator` 接口
    ```go
    type Hydrator interface {
        Hydrate(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
        Name() string
        Enable(query *Query) bool
        Update(candidate *Candidate, hydrated *Candidate)
        UpdateAll(candidates []*Candidate, hydrated []*Candidate)
    }
    ```

- [ ] **TODO-1.5**: 创建 `internal/pipeline/scorer.go`
  - [ ] 定义 `Scorer` 接口
    ```go
    type Scorer interface {
        Score(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
        Name() string
        Enable(query *Query) bool
        Update(candidate *Candidate, scored *Candidate)
        UpdateAll(candidates []*Candidate, scored []*Candidate)
    }
    ```

- [ ] **TODO-1.6**: 创建 `internal/pipeline/selector.go`
  - [ ] 定义 `Selector` 接口
    ```go
    type Selector interface {
        Select(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate
        Name() string
        Enable(query *Query) bool
    }
    ```

- [ ] **TODO-1.7**: 创建 `internal/pipeline/query_hydrator.go`
  - [ ] 定义 `QueryHydrator` 接口
    ```go
    type QueryHydrator interface {
        Hydrate(ctx context.Context, query *Query) (*Query, error)
        Name() string
        Enable(query *Query) bool
        Update(query *Query, hydrated *Query)
    }
    ```

---

## 🔧 Phase 2: Pipeline 执行引擎（优先级：最高）

### 2.1 Pipeline 核心实现

- [ ] **TODO-2.1**: 创建 `internal/pipeline/pipeline.go`
  - [ ] 定义 `CandidatePipeline` 结构体（包含所有组件列表）
  - [ ] 实现 `Execute` 方法（主流程）
  - [ ] 实现 `hydrateQuery` 方法（并行执行 Query Hydrators）
  - [ ] 实现 `fetchCandidates` 方法（并行执行 Sources）
  - [ ] 实现 `hydrateCandidates` 方法（并行执行 Hydrators）
  - [ ] 实现 `filterCandidates` 方法（顺序执行 Filters）
  - [ ] 实现 `scoreCandidates` 方法（顺序执行 Scorers）
  - [ ] 实现 `selectCandidates` 方法（执行 Selector）
  - [ ] 实现 `hydratePostSelection` 方法（并行执行 Post-Selection Hydrators）
  - [ ] 实现 `filterPostSelection` 方法（顺序执行 Post-Selection Filters）
  - [ ] 实现 `runSideEffects` 方法（异步执行 Side Effects）
  - [ ] 添加错误处理和日志记录

**参考**：STAGE2_LEARNING_GUIDE.md 中的 Pipeline.Execute 示例

---

## 🌐 Phase 3: gRPC 服务层（优先级：高）

### 3.1 协议定义

- [ ] **TODO-3.1**: 创建 `pkg/proto/scored_posts.proto`
  - [ ] 定义 `ScoredPostsQuery` 消息
  - [ ] 定义 `ScoredPostsResponse` 消息
  - [ ] 定义 `ScoredPost` 消息
  - [ ] 定义 `ScoredPostsService` 服务

- [ ] **TODO-3.2**: 生成 Go 代码
  - [ ] 安装 protoc 和 Go 插件
  - [ ] 运行 `protoc` 生成 Go 代码
  - [ ] 验证生成的代码

### 3.2 gRPC 服务实现

- [ ] **TODO-3.3**: 创建 `internal/mixer/server.go`
  - [ ] 定义 `HomeMixerServer` 结构体
  - [ ] 实现 `GetScoredPosts` 方法（gRPC 处理函数）
  - [ ] 实现参数验证
  - [ ] 实现 Query 构建
  - [ ] 实现 Pipeline 调用
  - [ ] 实现响应转换
  - [ ] 添加错误处理和日志

**参考**：STAGE2_LEARNING_GUIDE.md 中的 gRPC 服务入口示例

- [ ] **TODO-3.4**: 创建 `cmd/server/main.go`
  - [ ] 初始化 Pipeline
  - [ ] 创建 gRPC 服务器
  - [ ] 注册服务
  - [ ] 启动服务器
  - [ ] 添加优雅关闭

---

## 📥 Phase 4: Sources 实现（优先级：高）

### 4.1 Thunder Source

- [ ] **TODO-4.1**: 创建 `internal/sources/thunder.go`
  - [ ] 定义 `ThunderSource` 结构体
  - [ ] 实现 `GetCandidates` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法
  - [ ] 添加错误处理和日志

**注意**：Thunder 可以用内存存储（map）或 Redis 实现

### 4.2 Phoenix Source

- [ ] **TODO-4.2**: 创建 `internal/sources/phoenix.go`
  - [ ] 定义 `PhoenixSource` 结构体（包含 gRPC 客户端）
  - [ ] 实现 `GetCandidates` 方法（调用 Python 检索服务）
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法
  - [ ] 添加 gRPC 连接管理
  - [ ] 添加错误处理和重试逻辑

**参考**：MIGRATION_GUIDE_GO_PYTHON.md 中的 PhoenixSource 示例

---

## 🔍 Phase 5: Filters 实现（优先级：中）

### 5.1 基础过滤器

- [ ] **TODO-5.1**: 创建 `internal/filters/age.go`
  - [ ] 定义 `AgeFilter` 结构体
  - [ ] 实现 `Filter` 方法（检查帖子年龄）
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法
  - [ ] 使用雪花ID提取时间

- [ ] **TODO-5.2**: 创建 `internal/filters/duplicate.go`
  - [ ] 定义 `DropDuplicatesFilter` 结构体
  - [ ] 实现 `Filter` 方法（使用 map 去重）
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

- [ ] **TODO-5.3**: 创建 `internal/filters/self_tweet.go`
  - [ ] 定义 `SelfTweetFilter` 结构体
  - [ ] 实现 `Filter` 方法（移除自己的帖子）
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

### 5.2 其他过滤器（可选，后续实现）

- [ ] **TODO-5.4**: 创建 `internal/filters/core_data_hydration.go`
  - [ ] 移除核心数据获取失败的候选

- [ ] **TODO-5.5**: 创建 `internal/filters/previously_seen.go`
  - [ ] 移除已看过的帖子

- [ ] **TODO-5.6**: 创建 `internal/filters/muted_keyword.go`
  - [ ] 移除包含静音关键词的帖子

- [ ] **TODO-5.7**: 创建 `internal/filters/author_socialgraph.go`
  - [ ] 移除屏蔽/静音作者的帖子

---

## 💧 Phase 6: Hydrators 实现（优先级：中）

### 6.1 核心数据增强器

- [ ] **TODO-6.1**: 创建 `internal/hydrators/core_data.go`
  - [ ] 定义 `CoreDataCandidateHydrator` 结构体
  - [ ] 实现 `Hydrate` 方法（批量获取帖子核心数据）
  - [ ] 实现 `Update` 和 `UpdateAll` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法
  - [ ] 添加外部服务调用（可以是 mock 或真实服务）

### 6.2 其他增强器（可选，后续实现）

- [ ] **TODO-6.2**: 创建 `internal/hydrators/author.go`
  - [ ] 获取作者信息（用户名、认证状态等）

- [ ] **TODO-6.3**: 创建 `internal/hydrators/video_duration.go`
  - [ ] 获取视频时长

- [ ] **TODO-6.4**: 创建 `internal/hydrators/in_network.go`
  - [ ] 标记是否站内内容

---

## 📊 Phase 7: Scorers 实现（优先级：高）

### 7.1 Phoenix Scorer

- [ ] **TODO-7.1**: 创建 `internal/scorers/phoenix.go`
  - [ ] 定义 `PhoenixScorer` 结构体（包含 gRPC 客户端）
  - [ ] 实现 `Score` 方法（调用 Python 排序服务）
  - [ ] 实现 `Update` 和 `UpdateAll` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法
  - [ ] 添加 gRPC 连接管理
  - [ ] 解析预测结果并填充 PhoenixScores

**参考**：MIGRATION_GUIDE_GO_PYTHON.md 中的 PhoenixScorer 示例

### 7.2 Weighted Scorer

- [ ] **TODO-7.2**: 创建 `internal/scorers/weighted.go`
  - [ ] 定义 `WeightedScorer` 结构体
  - [ ] 实现 `Score` 方法（加权组合多个预测）
  - [ ] 实现 `computeWeightedScore` 辅助方法
  - [ ] 实现权重配置（可以从配置文件读取）
  - [ ] 实现 `Update` 和 `UpdateAll` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

**参考**：STAGE2_LEARNING_GUIDE.md 中的加权打分逻辑

### 7.3 其他 Scorer（可选，后续实现）

- [ ] **TODO-7.3**: 创建 `internal/scorers/author_diversity.go`
  - [ ] 调整重复作者的分数

- [ ] **TODO-7.4**: 创建 `internal/scorers/oon.go`
  - [ ] 调整站外内容分数

---

## 🎯 Phase 8: Selector 实现（优先级：中）

- [ ] **TODO-8.1**: 创建 `internal/pipeline/selector.go`（如果还没创建）
  - [ ] 定义 `TopKScoreSelector` 结构体
  - [ ] 实现 `Select` 方法（按分数排序，选择 Top-K）
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

---

## 🔄 Phase 9: Query Hydrators 实现（优先级：中）

- [ ] **TODO-9.1**: 创建 `internal/query_hydrators/user_action_seq.go`
  - [ ] 定义 `UserActionSeqQueryHydrator` 结构体
  - [ ] 实现 `Hydrate` 方法（获取用户交互历史）
  - [ ] 实现 `Update` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

- [ ] **TODO-9.2**: 创建 `internal/query_hydrators/user_features.go`
  - [ ] 定义 `UserFeaturesQueryHydrator` 结构体
  - [ ] 实现 `Hydrate` 方法（获取用户特征，如关注列表）
  - [ ] 实现 `Update` 方法
  - [ ] 实现 `Name` 方法
  - [ ] 实现 `Enable` 方法

---

## 🛠️ Phase 10: 工具函数（优先级：低）

- [ ] **TODO-10.1**: 创建 `internal/utils/snowflake.go`
  - [ ] 实现 `DurationSinceCreation` 函数（从雪花ID提取时间）
  - [ ] 实现 `CreationTime` 函数

- [ ] **TODO-10.2**: 创建 `internal/utils/helpers.go`
  - [ ] 实现 `ptrOrZero` 辅助函数
  - [ ] 实现 `floatOrZero` 辅助函数
  - [ ] 实现 `boolOrFalse` 辅助函数
  - [ ] 实现 `intOrZero` 辅助函数
  - [ ] 实现 `toU64Slice` 辅助函数

---

## 🏗️ Phase 11: Pipeline 配置（优先级：高）

- [ ] **TODO-11.1**: 创建 `internal/mixer/pipeline.go`
  - [ ] 定义 `PhoenixCandidatePipeline` 结构体
  - [ ] 实现 `NewPhoenixCandidatePipeline` 构造函数
  - [ ] 配置所有 Query Hydrators
  - [ ] 配置所有 Sources
  - [ ] 配置所有 Hydrators
  - [ ] 配置所有 Filters（按正确顺序）
  - [ ] 配置所有 Scorers（按正确顺序）
  - [ ] 配置 Selector
  - [ ] 配置 Post-Selection Hydrators
  - [ ] 配置 Post-Selection Filters
  - [ ] 配置 Side Effects（如果有）

**参考**：原项目的 `phoenix_candidate_pipeline.rs`

---

## 🧪 Phase 12: 测试和验证（优先级：高）

### 12.1 单元测试

- [ ] **TODO-12.1**: 为 Pipeline 编写单元测试
  - [ ] 测试 Execute 方法
  - [ ] 测试并行执行
  - [ ] 测试顺序执行
  - [ ] 测试错误处理

- [ ] **TODO-12.2**: 为 Filters 编写单元测试
  - [ ] 测试 AgeFilter
  - [ ] 测试 DropDuplicatesFilter
  - [ ] 测试 SelfTweetFilter

- [ ] **TODO-12.3**: 为 Scorers 编写单元测试
  - [ ] 测试 WeightedScorer
  - [ ] Mock PhoenixScorer 测试

### 12.2 集成测试

- [ ] **TODO-12.4**: 编写端到端测试
  - [ ] Mock Python 服务
  - [ ] 测试完整流程
  - [ ] 验证结果正确性

---

## 🚀 Phase 13: 部署和优化（优先级：低）

- [ ] **TODO-13.1**: 添加配置管理
  - [ ] 使用配置文件（YAML/JSON）
  - [ ] 环境变量支持
  - [ ] 默认值设置

- [ ] **TODO-13.2**: 添加监控和日志
  - [ ] 集成日志库（如 logrus）
  - [ ] 添加性能指标（Prometheus）
  - [ ] 添加追踪（OpenTelemetry）

- [ ] **TODO-13.3**: 性能优化
  - [ ] 连接池管理
  - [ ] 缓存实现
  - [ ] 批量请求优化

- [ ] **TODO-13.4**: 文档完善
  - [ ] API 文档
  - [ ] 部署文档
  - [ ] 开发文档

---

## 📝 实施建议

### 最小可行实现（MVP）

如果想快速验证，可以先实现：

1. ✅ Phase 1: 基础数据结构
2. ✅ Phase 2: Pipeline 执行引擎（简化版）
3. ✅ Phase 3: gRPC 服务层
4. ✅ Phase 4: 一个 Source（PhoenixSource）
5. ✅ Phase 5: 一个 Filter（AgeFilter）
6. ✅ Phase 7: 一个 Scorer（PhoenixScorer）
7. ✅ Phase 8: Selector
8. ✅ Phase 11: Pipeline 配置

**预计时间**：1-2周

### 完整实现

按照 TODO 列表逐步实现所有功能。

**预计时间**：4-6周

---

## 🎯 使用方式

1. **选择 TODO 项**：告诉我你想实现哪个 TODO（例如："实现 TODO-2.1"）
2. **我提供代码**：我会根据 STAGE2_LEARNING_GUIDE.md 中的示例提供完整代码
3. **你实现和测试**：你实现代码并进行测试
4. **标记完成**：完成后标记 ✅，继续下一个

---

## 📚 参考文档

- **STAGE2_LEARNING_GUIDE.md** - 数据流和代码示例（Go 版本）
- **MIGRATION_GUIDE_GO_PYTHON.md** - 详细迁移指南
- **QUICK_START_GO_PYTHON.md** - 快速开始指南

---

**准备好了吗？告诉我你想从哪个 TODO 开始！🚀**
