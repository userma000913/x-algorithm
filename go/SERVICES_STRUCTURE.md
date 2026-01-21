# 服务拆分说明

## Rust 项目结构

Rust 项目中有两个完全独立的服务：

1. **home-mixer** - 推荐服务
   - 入口：`home-mixer/main.rs`
   - 提供：`ScoredPostsService` gRPC 服务
   - 功能：推荐系统管道执行

2. **thunder** - 站内内容服务
   - 入口：`thunder/main.rs`
   - 提供：`InNetworkPostsService` gRPC 服务
   - 功能：Kafka 事件监听、内存存储、站内内容查询

## Go 项目结构（重构后）

Go 项目应该按照相同的方式拆分：

```
go/
├── cmd/
│   ├── server/              # Home Mixer 服务入口
│   │   └── main.go         # 推荐服务主程序
│   └── thunder/             # Thunder 服务入口
│       └── main.go         # 站内内容服务主程序
│
├── internal/
│   ├── mixer/              # Home Mixer 业务逻辑
│   │   ├── server.go      # ScoredPostsService 实现
│   │   └── pipeline.go    # Pipeline 配置
│   │
│   └── thunder/            # Thunder 业务逻辑
│       ├── service/        # InNetworkPostsService 实现
│       ├── poststore/      # 内存存储
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

## 服务独立性

### Home Mixer 服务
- **可独立运行**：`go run cmd/server/main.go`
- **可独立编译**：`go build -o bin/home-mixer cmd/server/main.go`
- **依赖**：调用 Thunder 服务的 gRPC 接口（通过 `internal/sources/thunder.go`）

### Thunder 服务
- **可独立运行**：`go run cmd/thunder/main.go`
- **可独立编译**：`go build -o bin/thunder cmd/thunder/main.go`
- **依赖**：Kafka、Strato 服务（外部依赖）

## 服务间通信

Home Mixer 通过 gRPC 调用 Thunder 服务：

```
Home Mixer (cmd/server)
    ↓ gRPC
Thunder Service (cmd/thunder)
    ↓ gRPC
InNetworkPostsService.GetInNetworkPosts()
```

## 部署方式

两个服务可以：
1. **独立部署**：分别部署在不同的机器/容器中
2. **独立扩展**：根据负载独立扩展
3. **独立监控**：分别监控各自的指标
