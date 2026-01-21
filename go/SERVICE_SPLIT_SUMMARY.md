# 服务拆分总结

## ✅ 服务拆分完成

Go 项目已按照 Rust 项目的方式拆分为两个完全独立的服务：

### 1. Home Mixer 服务（推荐服务）

**入口**: `cmd/server/main.go`

**功能**:
- 提供 `ScoredPostsService` gRPC 接口
- 执行推荐系统管道
- 返回排序后的帖子列表

**独立运行**:
```bash
go run cmd/server/main.go --grpc_port=50051
# 或
go build -o bin/home-mixer cmd/server/main.go
./bin/home-mixer --grpc_port=50051
```

**依赖**:
- 通过 gRPC 客户端调用 Thunder 服务（`internal/sources/thunder.go`）
- 调用 Phoenix 检索/排序服务
- 调用其他外部服务（TES, Gizmoduck, Strato, UAS, VF）

### 2. Thunder 服务（站内内容服务）

**入口**: `cmd/thunder/main.go`

**功能**:
- 提供 `InNetworkPostsService` gRPC 接口
- 监听 Kafka 事件流
- 内存存储站内内容（PostStore）
- 提供站内内容查询

**独立运行**:
```bash
go run cmd/thunder/main.go --grpc_port=50052
# 或
go build -o bin/thunder cmd/thunder/main.go
./bin/thunder --grpc_port=50052
```

**依赖**:
- Kafka（事件流）
- Strato 服务（获取关注列表）

## 服务间通信

```
┌─────────────────────┐
│   Home Mixer        │
│   (cmd/server)      │
│   Port: 50051       │
│                     │
│   - Pipeline        │
│   - Sources         │
│   - Filters         │
│   - Scorers         │
└──────────┬──────────┘
           │ gRPC Client
           │ (ThunderSource)
           ↓
┌─────────────────────┐
│   Thunder           │
│   (cmd/thunder)     │
│   Port: 50052       │
│                     │
│   - PostStore       │
│   - Kafka Listener  │
│   - gRPC Server     │
└─────────────────────┘
```

## 目录结构对比

### Rust 项目
```
x-algorithm/
├── home-mixer/
│   └── main.rs          # Home Mixer 服务入口
└── thunder/
    └── main.rs          # Thunder 服务入口
```

### Go 项目（已拆分）
```
go/
├── cmd/
│   ├── server/
│   │   └── main.go      # Home Mixer 服务入口 ✅
│   └── thunder/
│       └── main.go      # Thunder 服务入口 ✅
│
├── internal/
│   ├── mixer/           # Home Mixer 业务逻辑
│   └── thunder/         # Thunder 业务逻辑
│
└── pkg/
    └── proto/
        ├── scored_posts.proto      # Home Mixer proto
        └── thunder/
            └── in_network_posts.proto  # Thunder proto
```

## 关键点

1. **完全独立**: 两个服务可以分别编译、运行、部署
2. **通过 gRPC 通信**: Home Mixer 通过 gRPC 客户端调用 Thunder
3. **不同端口**: 
   - Home Mixer: 50051
   - Thunder: 50052
4. **独立依赖**: 每个服务有自己的依赖和配置

## 验证

```bash
# 编译两个服务
go build -o bin/home-mixer cmd/server/main.go
go build -o bin/thunder cmd/thunder/main.go

# 分别运行
./bin/thunder --grpc_port=50052 &
./bin/home-mixer --grpc_port=50051
```

## 与 Rust 项目的一致性

| Rust | Go | 状态 |
|------|-----|------|
| `home-mixer/main.rs` | `cmd/server/main.go` | ✅ |
| `thunder/main.rs` | `cmd/thunder/main.go` | ✅ |
| 独立服务 | 独立服务 | ✅ |
| gRPC 通信 | gRPC 通信 | ✅ |

**服务拆分完成！** 🎉
