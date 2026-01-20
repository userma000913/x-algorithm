# Go + Python 快速开始指南

> **快速参考**：用 Go + Python 实现推荐系统的关键步骤

---

## 🎯 核心思路

```
原项目（Rust + Python）
    ↓
Go 替代 Rust 服务层
Python ML 部分直接复用
    ↓
你的项目（Go + Python）
```

---

## 📋 实施步骤（简化版）

### Step 1: 理解架构（1天）

**关键理解**：
- Go 部分：实现业务逻辑（管道、过滤、打分等）
- Python 部分：ML 模型（检索和排序），直接复用

**参考文档**：
- `README_CN.md` - 理解整体架构
- `STAGE1_LEARNING_GUIDE.md` - 理解推荐系统基础

### Step 2: 搭建项目结构（1天）

```
your-project/
├── go/                    # Go 服务层
│   ├── cmd/server/       # 服务入口
│   ├── internal/
│   │   ├── pipeline/     # 管道框架（核心）
│   │   ├── sources/      # 候选源
│   │   ├── filters/      # 过滤器
│   │   ├── hydrators/    # 增强器
│   │   └── scorers/      # 打分器
│   └── pkg/proto/        # gRPC 协议
└── python/               # Python ML 层
    ├── phoenix/          # 复用原项目
    └── services/         # gRPC 服务包装
```

### Step 3: 实现管道框架（3天）

**核心文件**：`go/internal/pipeline/pipeline.go`

**关键功能**：
- `Execute()` - 执行完整流程
- 并行执行：Sources, Hydrators
- 顺序执行：Filters, Scorers

**参考**：`candidate-pipeline/candidate_pipeline.rs`

### Step 4: 实现核心组件（5天）

**优先级排序**：

1. **Sources**（2天）
   - `ThunderSource` - 站内内容（可以用内存存储替代）
   - `PhoenixSource` - 调用 Python 检索服务

2. **Filters**（1天）
   - `AgeFilter` - 年龄过滤（简单）
   - `DropDuplicatesFilter` - 去重（简单）
   - `SelfTweetFilter` - 自己的帖子（简单）

3. **Hydrators**（1天）
   - `CoreDataHydrator` - 核心数据（调用外部服务）

4. **Scorers**（1天）
   - `PhoenixScorer` - 调用 Python 排序服务
   - `WeightedScorer` - 加权组合（纯 Go）

### Step 5: Python 服务包装（2天）

**两个服务**：

1. **检索服务**：`python/services/retrieval_service.py`
   - 包装 `phoenix/recsys_retrieval_model.py`
   - 提供 gRPC 接口

2. **排序服务**：`python/services/ranking_service.py`
   - 包装 `phoenix/recsys_model.py`
   - 提供 gRPC 接口

### Step 6: 定义 gRPC 协议（1天）

**文件**：`proto/phoenix.proto`

**两个服务**：
- `RetrievalService` - 检索
- `RankingService` - 排序

### Step 7: 集成测试（2天）

- 启动 Python 服务
- 启动 Go 服务
- 测试完整流程

---

## 🔑 关键技术点

### 1. Go 并发模式

```go
// 并行执行 Sources
var wg sync.WaitGroup
for _, source := range sources {
    wg.Add(1)
    go func(s Source) {
        defer wg.Done()
        candidates, _ := s.GetCandidates(ctx, query)
        // 合并结果
    }(source)
}
wg.Wait()
```

### 2. gRPC 调用 Python

```go
// Go 调用 Python 检索服务
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := phoenix.NewRetrievalServiceClient(conn)
resp, _ := client.Retrieve(ctx, req)
```

### 3. Python gRPC 服务

```python
# Python 提供 gRPC 服务
server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
pb_grpc.add_RetrievalServiceServicer_to_server(RetrievalService(), server)
server.add_insecure_port('[::]:50051')
server.start()
```

---

## 📝 最小可行实现（MVP）

### 最简版本（1周）

**只实现核心功能**：

1. **管道框架**（2天）
   - 基本的 Execute 流程
   - 并行/顺序执行

2. **一个 Source**（1天）
   - PhoenixSource（调用 Python）

3. **一个 Filter**（1天）
   - AgeFilter（简单实现）

4. **一个 Scorer**（1天）
   - PhoenixScorer（调用 Python）

5. **Python 服务**（1天）
   - 包装检索和排序服务

6. **测试**（1天）
   - 端到端测试

---

## 🛠️ 工具和库

### Go 依赖

```go
// go.mod
module your-project

require (
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.31.0
    // 其他依赖
)
```

### Python 依赖

```python
# requirements.txt
grpcio
grpcio-tools
jax
haiku
# 其他依赖
```

---

## ⚠️ 常见问题

### Q1: 如何复用 Python 模型？

**A**: 直接复制 `phoenix/` 目录，包装为 gRPC 服务即可。

### Q2: Thunder 如何实现？

**A**: 可以用 Go 的 `map` 或 `sync.Map` 实现内存存储，或者用 Redis。

### Q3: 如何保证性能？

**A**: 
- Go 部分：使用 goroutine 并发
- Python 部分：使用 gRPC 异步处理
- 连接池：复用 gRPC 连接

### Q4: 数据格式如何统一？

**A**: 使用 Protocol Buffers 定义接口，自动生成 Go 和 Python 代码。

---

## 📚 参考文档

- **详细指南**：`MIGRATION_GUIDE_GO_PYTHON.md`
- **原项目架构**：`README_CN.md`
- **学习路径**：`LEARNING_INDEX.md`

---

## 🚀 开始行动

1. ✅ 阅读 `MIGRATION_GUIDE_GO_PYTHON.md` 了解详细步骤
2. ✅ 搭建项目结构
3. ✅ 实现管道框架
4. ✅ 逐步实现各个组件
5. ✅ 集成测试

**祝你实施顺利！🎉**
