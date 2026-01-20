# Go + Python 迁移指南

> **目标**：参考本项目，使用 Go 语言 + Python 实现自己的推荐算法系统  
> **适用场景**：不懂 Rust，但熟悉 Go 和 Python 的开发者

---

## 📋 目录

- [架构对比](#架构对比)
- [技术栈映射](#技术栈映射)
- [实施步骤](#实施步骤)
- [代码示例](#代码示例)
- [注意事项](#注意事项)

---

## 🏗️ 架构对比

### 原项目架构（Rust + Python）

```
┌─────────────────────────────────────────────────────────┐
│                    Rust 服务层                           │
│  - Home Mixer（编排层）                                  │
│  - Candidate Pipeline（管道框架）                       │
│  - Thunder（内存存储）                                  │
│  - Sources/Filters/Hydrators/Scorers                    │
└────────────────────┬────────────────────────────────────┘
                     │ gRPC/HTTP
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  Python ML 层                           │
│  - Phoenix Retrieval（Two-Tower 检索）                 │
│  - Phoenix Ranking（Transformer 排序）                 │
└─────────────────────────────────────────────────────────┘
```

### 目标架构（Go + Python）

```
┌─────────────────────────────────────────────────────────┐
│                    Go 服务层                            │
│  - Home Mixer（编排层）                                  │
│  - Candidate Pipeline（管道框架）                       │
│  - Thunder（内存存储）                                  │
│  - Sources/Filters/Hydrators/Scorers                    │
└────────────────────┬────────────────────────────────────┘
                     │ gRPC/HTTP
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  Python ML 层                           │
│  - Phoenix Retrieval（Two-Tower 检索）                  │
│  - Phoenix Ranking（Transformer 排序）                  │
│  （保持不变，直接复用）                                  │
└─────────────────────────────────────────────────────────┘
```

**关键点**：
- **Go 部分**：替代 Rust 服务层，实现业务逻辑
- **Python 部分**：直接复用 Phoenix ML 模型，无需修改

---

## 🔄 技术栈映射

### Rust → Go 映射表

| Rust 组件 | Go 对应 | 说明 |
|-----------|---------|------|
| `tokio` (异步运行时) | `goroutine` + `channel` | Go 原生并发 |
| `tonic` (gRPC) | `google.golang.org/grpc` | Go gRPC 库 |
| `serde` (序列化) | `encoding/json` / `protobuf` | Go 标准库 |
| `Arc<T>` (原子引用计数) | `sync` 包 | Go 并发安全 |
| `Vec<T>` | `[]T` | Go 切片 |
| `Result<T, E>` | `(T, error)` | Go 错误处理 |
| `Option<T>` | `*T` 或自定义类型 | Go 指针或接口 |
| Trait | Interface | Go 接口 |

### Python 部分（保持不变）

| 组件 | 说明 |
|------|------|
| `phoenix/recsys_model.py` | Transformer 排序模型（直接复用） |
| `phoenix/recsys_retrieval_model.py` | Two-Tower 检索模型（直接复用） |
| `phoenix/run_ranker.py` | 排序服务（需要包装为 gRPC/HTTP 服务） |
| `phoenix/run_retrieval.py` | 检索服务（需要包装为 gRPC/HTTP 服务） |

---

## 📝 实施步骤

### 第一阶段：设计架构（1-2天）

#### 1.1 项目结构设计

```
your-recommendation-system/
├── go/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # 服务入口
│   ├── internal/
│   │   ├── pipeline/                # 管道框架
│   │   │   ├── pipeline.go          # 管道执行引擎
│   │   │   ├── source.go           # Source 接口
│   │   │   ├── filter.go           # Filter 接口
│   │   │   ├── hydrator.go         # Hydrator 接口
│   │   │   ├── scorer.go           # Scorer 接口
│   │   │   └── selector.go         # Selector 接口
│   │   ├── mixer/                  # Home Mixer
│   │   │   ├── server.go           # gRPC 服务
│   │   │   └── pipeline.go        # 管道配置
│   │   ├── sources/                # 候选源
│   │   │   ├── thunder.go         # 站内内容源
│   │   │   └── phoenix.go         # 站外内容源（调用 Python）
│   │   ├── filters/                # 过滤器
│   │   │   ├── age.go             # 年龄过滤
│   │   │   ├── duplicate.go       # 去重
│   │   │   └── ...
│   │   ├── hydrators/              # 增强器
│   │   │   ├── core_data.go       # 核心数据
│   │   │   └── ...
│   │   ├── scorers/                # 打分器
│   │   │   ├── phoenix.go         # Phoenix 打分（调用 Python）
│   │   │   └── weighted.go        # 加权打分
│   │   └── thunder/                # Thunder 内存存储
│   │       ├── store.go           # 存储实现
│   │       └── kafka.go           # Kafka 消费
│   ├── pkg/
│   │   └── proto/                  # gRPC 协议定义
│   └── go.mod
├── python/
│   ├── phoenix/                    # 复用原项目的 Phoenix
│   │   ├── recsys_model.py
│   │   ├── recsys_retrieval_model.py
│   │   ├── run_ranker.py
│   │   └── run_retrieval.py
│   ├── services/                   # Python 服务包装
│   │   ├── retrieval_service.py   # 检索服务（gRPC）
│   │   └── ranking_service.py     # 排序服务（gRPC）
│   └── requirements.txt
└── README.md
```

#### 1.2 接口设计

**定义 Go 接口（对应 Rust Trait）**：

```go
// internal/pipeline/source.go
package pipeline

type Source interface {
    GetCandidates(ctx context.Context, query *Query) ([]*Candidate, error)
    Name() string
    Enable(query *Query) bool
}

// internal/pipeline/filter.go
package pipeline

type Filter interface {
    Filter(ctx context.Context, query *Query, candidates []*Candidate) (*FilterResult, error)
    Name() string
    Enable(query *Query) bool
}

// internal/pipeline/hydrator.go
package pipeline

type Hydrator interface {
    Hydrate(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
    Name() string
    Enable(query *Query) bool
}

// internal/pipeline/scorer.go
package pipeline

type Scorer interface {
    Score(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
    Name() string
    Enable(query *Query) bool
}
```

### 第二阶段：实现管道框架（3-5天）

#### 2.1 管道执行引擎

**文件**：`go/internal/pipeline/pipeline.go`

```go
package pipeline

import (
    "context"
    "sync"
)

type Pipeline struct {
    QueryHydrators []QueryHydrator
    Sources        []Source
    Hydrators      []Hydrator
    Filters        []Filter
    Scorers        []Scorer
    Selector       Selector
    PostSelectionHydrators []Hydrator
    PostSelectionFilters   []Filter
    SideEffects    []SideEffect
    ResultSize     int
}

type PipelineResult struct {
    RetrievedCandidates []*Candidate
    FilteredCandidates  []*Candidate
    SelectedCandidates  []*Candidate
    Query               *Query
}

func (p *Pipeline) Execute(ctx context.Context, query *Query) (*PipelineResult, error) {
    // 1. Query Hydration（并行）
    hydratedQuery, err := p.hydrateQuery(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 2. Candidate Sourcing（并行）
    candidates, err := p.fetchCandidates(ctx, hydratedQuery)
    if err != nil {
        return nil, err
    }
    
    // 3. Candidate Hydration（并行）
    hydratedCandidates, err := p.hydrate(ctx, hydratedQuery, candidates)
    if err != nil {
        return nil, err
    }
    
    // 4. Pre-Scoring Filtering（顺序）
    keptCandidates, filteredCandidates, err := p.filter(ctx, hydratedQuery, hydratedCandidates)
    if err != nil {
        return nil, err
    }
    
    // 5. Scoring（顺序）
    scoredCandidates, err := p.score(ctx, hydratedQuery, keptCandidates)
    if err != nil {
        return nil, err
    }
    
    // 6. Selection
    selectedCandidates := p.selectCandidates(hydratedQuery, scoredCandidates)
    
    // 7. Post-Selection Hydration（并行）
    postHydratedCandidates, err := p.hydratePostSelection(ctx, hydratedQuery, selectedCandidates)
    if err != nil {
        return nil, err
    }
    
    // 8. Post-Selection Filtering（顺序）
    finalCandidates, postFilteredCandidates, err := p.filterPostSelection(ctx, hydratedQuery, postHydratedCandidates)
    if err != nil {
        return nil, err
    }
    
    // 9. Truncate
    if len(finalCandidates) > p.ResultSize {
        finalCandidates = finalCandidates[:p.ResultSize]
    }
    
    // 10. Side Effects（异步，不阻塞）
    go p.runSideEffects(ctx, hydratedQuery, finalCandidates)
    
    return &PipelineResult{
        RetrievedCandidates: hydratedCandidates,
        FilteredCandidates:  append(filteredCandidates, postFilteredCandidates...),
        SelectedCandidates:  finalCandidates,
        Query:               hydratedQuery,
    }, nil
}

// 并行执行 Query Hydrators
func (p *Pipeline) hydrateQuery(ctx context.Context, query *Query) (*Query, error) {
    var wg sync.WaitGroup
    var mu sync.Mutex
    hydratedQuery := query.Clone()
    errChan := make(chan error, len(p.QueryHydrators))
    
    for _, hydrator := range p.QueryHydrators {
        if !hydrator.Enable(query) {
            continue
        }
        wg.Add(1)
        go func(h QueryHydrator) {
            defer wg.Done()
            result, err := h.Hydrate(ctx, query)
            if err != nil {
                errChan <- err
                return
            }
            mu.Lock()
            hydrator.Update(hydratedQuery, result)
            mu.Unlock()
        }(hydrator)
    }
    
    wg.Wait()
    close(errChan)
    
    // 检查错误（可以选择忽略部分错误）
    for err := range errChan {
        if err != nil {
            // 记录错误，但不中断流程
            log.Printf("Query hydrator failed: %v", err)
        }
    }
    
    return hydratedQuery, nil
}

// 并行执行 Sources
func (p *Pipeline) fetchCandidates(ctx context.Context, query *Query) ([]*Candidate, error) {
    var wg sync.WaitGroup
    var mu sync.Mutex
    var allCandidates []*Candidate
    errChan := make(chan error, len(p.Sources))
    
    for _, source := range p.Sources {
        if !source.Enable(query) {
            continue
        }
        wg.Add(1)
        go func(s Source) {
            defer wg.Done()
            candidates, err := s.GetCandidates(ctx, query)
            if err != nil {
                errChan <- err
                return
            }
            mu.Lock()
            allCandidates = append(allCandidates, candidates...)
            mu.Unlock()
        }(source)
    }
    
    wg.Wait()
    close(errChan)
    
    // 检查错误
    for err := range errChan {
        if err != nil {
            log.Printf("Source failed: %v", err)
        }
    }
    
    return allCandidates, nil
}

// 顺序执行 Filters
func (p *Pipeline) filter(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, []*Candidate, error) {
    kept := candidates
    var allRemoved []*Candidate
    
    for _, filter := range p.Filters {
        if !filter.Enable(query) {
            continue
        }
        result, err := filter.Filter(ctx, query, kept)
        if err != nil {
            // 记录错误，继续下一个 filter
            log.Printf("Filter %s failed: %v", filter.Name(), err)
            continue
        }
        kept = result.Kept
        allRemoved = append(allRemoved, result.Removed...)
    }
    
    return kept, allRemoved, nil
}
```

#### 2.2 数据结构定义

**文件**：`go/internal/pipeline/types.go`

```go
package pipeline

import "time"

// Query 查询对象
type Query struct {
    UserID          int64
    ClientAppID     string
    CountryCode     string
    LanguageCode    string
    SeenIDs         []int64
    ServedIDs       []int64
    InNetworkOnly   bool
    IsBottomRequest bool
    
    // 增强后的字段
    UserActionSequence *UserActionSequence
    UserFeatures       *UserFeatures
}

// Candidate 候选对象
type Candidate struct {
    TweetID            int64
    AuthorID           int64
    RetweetedTweetID   *int64
    RetweetedUserID    *int64
    InReplyToTweetID   *int64
    TweetText          string
    
    // 增强后的字段
    CoreData           *CoreData
    AuthorScreenName   *string
    AuthorVerified     *bool
    VideoDurationMs    *int64
    InNetwork          *bool
    
    // 打分后的字段
    PhoenixScores      *PhoenixScores
    WeightedScore       *float64
    Score               *float64
}

// PhoenixScores Phoenix 模型预测的分数
type PhoenixScores struct {
    FavoriteScore    *float64
    ReplyScore       *float64
    RetweetScore     *float64
    ClickScore       *float64
    // ... 其他动作分数
}

// FilterResult 过滤结果
type FilterResult struct {
    Kept    []*Candidate
    Removed []*Candidate
}
```

### 第三阶段：实现核心组件（5-7天）

#### 3.1 Phoenix Source（调用 Python 检索服务）

**文件**：`go/internal/sources/phoenix.go`

```go
package sources

import (
    "context"
    "google.golang.org/grpc"
    "your-project/pkg/proto/phoenix"
)

type PhoenixSource struct {
    client phoenix.RetrievalServiceClient
    conn   *grpc.ClientConn
}

func NewPhoenixSource(address string) (*PhoenixSource, error) {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    return &PhoenixSource{
        client: phoenix.NewRetrievalServiceClient(conn),
        conn:   conn,
    }, nil
}

func (s *PhoenixSource) GetCandidates(ctx context.Context, query *pipeline.Query) ([]*pipeline.Candidate, error) {
    // 构建请求
    req := &phoenix.RetrieveRequest{
        UserId: query.UserID,
        UserActionSequence: convertToProto(query.UserActionSequence),
        MaxResults: 500,
    }
    
    // 调用 Python 服务
    resp, err := s.client.Retrieve(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 转换为 Candidate
    candidates := make([]*pipeline.Candidate, 0, len(resp.Candidates))
    for _, protoCandidate := range resp.Candidates {
        candidates = append(candidates, &pipeline.Candidate{
            TweetID:  protoCandidate.TweetId,
            AuthorID: protoCandidate.AuthorId,
        })
    }
    
    return candidates, nil
}

func (s *PhoenixSource) Name() string {
    return "PhoenixSource"
}

func (s *PhoenixSource) Enable(query *pipeline.Query) bool {
    return !query.InNetworkOnly
}
```

#### 3.2 Phoenix Scorer（调用 Python 排序服务）

**文件**：`go/internal/scorers/phoenix.go`

```go
package scorers

import (
    "context"
    "google.golang.org/grpc"
    "your-project/pkg/proto/phoenix"
)

type PhoenixScorer struct {
    client phoenix.RankingServiceClient
    conn   *grpc.ClientConn
}

func NewPhoenixScorer(address string) (*PhoenixScorer, error) {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        return nil, err
    }
    return &PhoenixScorer{
        client: phoenix.NewRankingServiceClient(conn),
        conn:   conn,
    }, nil
}

func (s *PhoenixScorer) Score(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) ([]*pipeline.Candidate, error) {
    // 构建请求
    req := &phoenix.RankRequest{
        UserId: query.UserID,
        UserActionSequence: convertToProto(query.UserActionSequence),
        Candidates: convertCandidatesToProto(candidates),
    }
    
    // 调用 Python 服务
    resp, err := s.client.Rank(ctx, req)
    if err != nil {
        return nil, err
    }
    
    // 更新候选的分数
    scoredCandidates := make([]*pipeline.Candidate, len(candidates))
    for i, candidate := range candidates {
        scoredCandidates[i] = candidate.Clone()
        if i < len(resp.Predictions) {
            scoredCandidates[i].PhoenixScores = convertPredictionsToScores(resp.Predictions[i])
        }
    }
    
    return scoredCandidates, nil
}

func (s *PhoenixScorer) Name() string {
    return "PhoenixScorer"
}

func (s *PhoenixScorer) Enable(query *pipeline.Query) bool {
    return query.UserActionSequence != nil
}
```

#### 3.3 实现过滤器示例

**文件**：`go/internal/filters/age.go`

```go
package filters

import (
    "context"
    "time"
    "your-project/internal/pipeline"
    "your-project/internal/util/snowflake"
)

type AgeFilter struct {
    MaxAge time.Duration
}

func NewAgeFilter(maxAge time.Duration) *AgeFilter {
    return &AgeFilter{MaxAge: maxAge}
}

func (f *AgeFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
    var kept, removed []*pipeline.Candidate
    
    for _, candidate := range candidates {
        age := snowflake.DurationSinceCreation(candidate.TweetID)
        if age <= f.MaxAge {
            kept = append(kept, candidate)
        } else {
            removed = append(removed, candidate)
        }
    }
    
    return &pipeline.FilterResult{
        Kept:    kept,
        Removed: removed,
    }, nil
}

func (f *AgeFilter) Name() string {
    return "AgeFilter"
}

func (f *AgeFilter) Enable(query *pipeline.Query) bool {
    return true
}
```

### 第四阶段：Python 服务包装（2-3天）

#### 4.1 检索服务（gRPC）

**文件**：`python/services/retrieval_service.py`

```python
import grpc
from concurrent import futures
import phoenix.recsys_retrieval_model as retrieval_model
import your_project.proto.phoenix_pb2 as pb
import your_project.proto.phoenix_pb2_grpc as pb_grpc

class RetrievalService(pb_grpc.RetrievalServiceServicer):
    def __init__(self):
        # 加载检索模型
        self.model = retrieval_model.load_model()
    
    def Retrieve(self, request, context):
        # 提取用户信息
        user_id = request.user_id
        user_action_sequence = request.user_action_sequence
        
        # 调用检索模型
        user_embedding = self.model.encode_user(user_action_sequence)
        top_k_candidates = self.model.retrieve(user_embedding, k=request.max_results)
        
        # 转换为响应
        candidates = []
        for candidate in top_k_candidates:
            candidates.append(pb.Candidate(
                tweet_id=candidate.tweet_id,
                author_id=candidate.author_id,
            ))
        
        return pb.RetrieveResponse(candidates=candidates)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb_grpc.add_RetrievalServiceServicer_to_server(RetrievalService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

#### 4.2 排序服务（gRPC）

**文件**：`python/services/ranking_service.py`

```python
import grpc
from concurrent import futures
import phoenix.recsys_model as ranking_model
import your_project.proto.phoenix_pb2 as pb
import your_project.proto.phoenix_pb2_grpc as pb_grpc

class RankingService(pb_grpc.RankingServiceServicer):
    def __init__(self):
        # 加载排序模型
        self.model = ranking_model.load_model()
    
    def Rank(self, request, context):
        # 提取信息
        user_id = request.user_id
        user_action_sequence = request.user_action_sequence
        candidates = request.candidates
        
        # 调用排序模型
        predictions = self.model.predict(
            user_action_sequence=user_action_sequence,
            candidates=candidates
        )
        
        # 转换为响应
        prediction_list = []
        for pred in predictions:
            prediction_list.append(pb.ActionPredictions(
                favorite_score=pred.favorite_score,
                reply_score=pred.reply_score,
                retweet_score=pred.retweet_score,
                # ... 其他动作分数
            ))
        
        return pb.RankResponse(predictions=prediction_list)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    pb_grpc.add_RankingServiceServicer_to_server(RankingService(), server)
    server.add_insecure_port('[::]:50052')
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
```

### 第五阶段：gRPC 协议定义（1-2天）

#### 5.1 定义 Protocol Buffers

**文件**：`proto/phoenix.proto`

```protobuf
syntax = "proto3";

package phoenix;

// 检索服务
service RetrievalService {
    rpc Retrieve(RetrieveRequest) returns (RetrieveResponse);
}

message RetrieveRequest {
    int64 user_id = 1;
    UserActionSequence user_action_sequence = 2;
    int32 max_results = 3;
}

message RetrieveResponse {
    repeated Candidate candidates = 1;
}

// 排序服务
service RankingService {
    rpc Rank(RankRequest) returns (RankResponse);
}

message RankRequest {
    int64 user_id = 1;
    UserActionSequence user_action_sequence = 2;
    repeated Candidate candidates = 3;
}

message RankResponse {
    repeated ActionPredictions predictions = 1;
}

// 通用消息
message Candidate {
    int64 tweet_id = 1;
    int64 author_id = 2;
}

message UserActionSequence {
    repeated Action actions = 1;
}

message Action {
    int64 tweet_id = 1;
    int64 author_id = 2;
    string action_type = 3;  // "like", "retweet", "reply", etc.
}

message ActionPredictions {
    double favorite_score = 1;
    double reply_score = 2;
    double retweet_score = 3;
    double click_score = 4;
    // ... 其他动作分数
}
```

#### 5.2 生成代码

```bash
# 生成 Go 代码
protoc --go_out=. --go-grpc_out=. proto/phoenix.proto

# 生成 Python 代码
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/phoenix.proto
```

### 第六阶段：集成测试（2-3天）

#### 6.1 启动服务

```bash
# 启动 Python 检索服务
cd python/services
python retrieval_service.py

# 启动 Python 排序服务
python ranking_service.py

# 启动 Go 主服务
cd go/cmd/server
go run main.go
```

#### 6.2 测试流程

1. **单元测试**：测试各个组件
2. **集成测试**：测试完整流程
3. **性能测试**：测试并发和延迟

---

## 💡 关键注意事项

### 1. 并发处理

**Go 的优势**：
- 使用 `goroutine` 实现并行执行
- 使用 `sync.WaitGroup` 等待所有 goroutine 完成
- 使用 `channel` 进行通信

**示例**：
```go
var wg sync.WaitGroup
for _, source := range sources {
    wg.Add(1)
    go func(s Source) {
        defer wg.Done()
        // 执行 source
    }(source)
}
wg.Wait()
```

### 2. 错误处理

**Go 的错误处理**：
- 使用 `(result, error)` 返回错误
- 可以选择忽略部分错误（记录日志）
- 关键错误需要中断流程

### 3. 性能优化

**Go 部分**：
- 使用连接池管理 gRPC 连接
- 批量请求减少网络开销
- 使用缓存减少重复计算

**Python 部分**：
- 使用 gRPC 异步处理
- 模型预加载
- 批量推理

### 4. 数据一致性

- 确保 Go 和 Python 之间的数据格式一致
- 使用 Protocol Buffers 保证类型安全
- 版本化 API 接口

---

## 📊 实施时间表

| 阶段 | 任务 | 预计时间 |
|------|------|----------|
| 第一阶段 | 设计架构 | 1-2天 |
| 第二阶段 | 实现管道框架 | 3-5天 |
| 第三阶段 | 实现核心组件 | 5-7天 |
| 第四阶段 | Python 服务包装 | 2-3天 |
| 第五阶段 | gRPC 协议定义 | 1-2天 |
| 第六阶段 | 集成测试 | 2-3天 |
| **总计** | | **14-22天** |

---

## 🎯 快速开始清单

- [ ] 搭建 Go 开发环境
- [ ] 搭建 Python 开发环境
- [ ] 设计项目结构
- [ ] 定义接口和数据结构
- [ ] 实现管道框架
- [ ] 实现 Sources（Thunder, Phoenix）
- [ ] 实现 Filters（Age, Duplicate, etc.）
- [ ] 实现 Hydrators（CoreData, Author, etc.）
- [ ] 实现 Scorers（Phoenix, Weighted, etc.）
- [ ] 包装 Python ML 服务为 gRPC
- [ ] 定义 gRPC 协议
- [ ] 集成测试
- [ ] 性能优化

---

## 🔗 参考资源

### Go 相关

- [Go 官方文档](https://go.dev/doc/)
- [gRPC Go 教程](https://grpc.io/docs/languages/go/)
- [Go 并发模式](https://go.dev/blog/pipelines)

### Python 相关

- [gRPC Python 教程](https://grpc.io/docs/languages/python/)
- [JAX 文档](https://jax.readthedocs.io/)

### 本项目参考

- `candidate-pipeline/` - 管道框架设计
- `home-mixer/` - 业务逻辑实现
- `phoenix/` - ML 模型（直接复用）

---

## 🚀 下一步

1. **开始实施**：按照步骤逐步实现
2. **参考原项目**：理解设计思路，用 Go 重新实现
3. **复用 Python 部分**：直接使用 Phoenix 模型
4. **测试验证**：确保功能正确
5. **性能优化**：根据实际需求优化

**祝你实施顺利！🎉**

记住：理解原项目的设计思路比直接翻译代码更重要！
