# 服务架构说明

## 服务拆分

本项目包含两个完全独立的服务，与 Rust 项目结构保持一致：

### 1. Home Mixer 服务（推荐服务）

**位置**: `cmd/server/main.go`

**功能**:
- 提供 `ScoredPostsService` gRPC 接口
- 执行推荐系统管道（Pipeline）
- 返回排序后的帖子列表

**运行方式**:
```bash
# 开发模式
go run cmd/server/main.go --grpc_port=50051

# 编译后运行
go build -o bin/home-mixer cmd/server/main.go
./bin/home-mixer --grpc_port=50051
```

**默认端口**: 50051

**依赖**:
- Thunder 服务（通过 gRPC 调用）
- Phoenix 检索/排序服务
- 其他外部服务（TES, Gizmoduck, Strato, UAS, VF）

### 2. Thunder 服务（站内内容服务）

**位置**: `cmd/thunder/main.go`

**功能**:
- 提供 `InNetworkPostsService` gRPC 接口
- 监听 Kafka 事件流
- 内存存储站内内容（PostStore）
- 提供站内内容查询

**运行方式**:
```bash
# 开发模式
go run cmd/thunder/main.go --grpc_port=50052

# 编译后运行
go build -o bin/thunder cmd/thunder/main.go
./bin/thunder --grpc_port=50052
```

**默认端口**: 50052

**依赖**:
- Kafka（事件流）
- Strato 服务（获取关注列表）

## 服务间通信

```
┌─────────────────┐
│  Home Mixer     │
│  (cmd/server)   │
│  Port: 50051    │
└────────┬────────┘
         │ gRPC
         │ GetInNetworkPosts()
         ↓
┌─────────────────┐
│  Thunder        │
│  (cmd/thunder)  │
│  Port: 50052    │
└─────────────────┘
```

Home Mixer 通过 gRPC 调用 Thunder 服务获取站内内容。

## 独立部署

两个服务可以：

1. **独立编译**:
   ```bash
   go build -o bin/home-mixer cmd/server/main.go
   go build -o bin/thunder cmd/thunder/main.go
   ```

2. **独立运行**:
   ```bash
   # 终端 1: 启动 Thunder 服务
   ./bin/thunder --grpc_port=50052
   
   # 终端 2: 启动 Home Mixer 服务
   ./bin/home-mixer --grpc_port=50051
   ```

3. **独立扩展**: 根据负载独立扩展每个服务

4. **独立监控**: 分别监控各自的指标和日志

## 目录结构

```
go/
├── cmd/
│   ├── server/              # Home Mixer 服务入口
│   │   ├── main.go
│   │   └── README.md
│   └── thunder/             # Thunder 服务入口
│       ├── main.go
│       └── README.md
│
├── internal/
│   ├── mixer/              # Home Mixer 业务逻辑
│   │   ├── server.go       # ScoredPostsService 实现
│   │   └── pipeline.go    # Pipeline 配置
│   │
│   └── thunder/            # Thunder 业务逻辑
│       ├── service/         # InNetworkPostsService 实现
│       ├── poststore/       # 内存存储
│       ├── kafka/          # Kafka 监听
│       ├── strato/         # Strato 客户端
│       └── deserializer/   # 事件反序列化
│
└── pkg/
    └── proto/
        ├── scored_posts.proto      # Home Mixer proto
        └── thunder/
            └── in_network_posts.proto  # Thunder proto
```

## 与 Rust 项目的对应关系

| Rust 服务 | Go 服务 | 入口文件 |
|-----------|---------|----------|
| `home-mixer/main.rs` | `cmd/server/main.go` | Home Mixer |
| `thunder/main.rs` | `cmd/thunder/main.go` | Thunder |

两个项目的服务结构完全一致，便于理解和迁移。
