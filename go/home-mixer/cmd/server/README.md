# Home Mixer 服务

这是推荐系统的主服务，提供 `ScoredPostsService` gRPC 接口。

## 运行方式

```bash
# 开发模式
go run cmd/server/main.go

# 编译后运行
go build -o bin/home-mixer cmd/server/main.go
./bin/home-mixer

# 带参数运行
go run cmd/server/main.go --grpc_port=50051 --metrics_port=9090
```

## 服务说明

- **服务名称**: Home Mixer
- **gRPC 端口**: 默认 50051
- **功能**: 执行推荐系统管道，返回排序后的帖子列表
- **依赖**: 
  - Thunder 服务（通过 gRPC 调用）
  - Phoenix 检索/排序服务（通过 gRPC 调用）
  - 其他外部服务（TES, Gizmoduck, Strato, UAS, VF）

## 与 Thunder 服务的关系

Home Mixer 通过 gRPC 调用 Thunder 服务获取站内内容：

```go
// internal/sources/thunder.go
// ThunderSource 调用 Thunder 服务的 GetInNetworkPosts 接口
```
