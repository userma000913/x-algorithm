# Thunder 服务

这是站内内容服务，提供 `InNetworkPostsService` gRPC 接口。

## 运行方式

```bash
# 开发模式
go run cmd/thunder/main.go

# 编译后运行
go build -o bin/thunder cmd/thunder/main.go
./bin/thunder

# 带参数运行
go run cmd/thunder/main.go \
  --grpc_port=50052 \
  --http_port=8080 \
  --post_retention_seconds=172800 \
  --max_concurrent_requests=100 \
  --kafka_batch_size=100 \
  --is_serving=true
```

## 服务说明

- **服务名称**: Thunder
- **gRPC 端口**: 默认 50052
- **HTTP 端口**: 默认 8080（用于健康检查）
- **功能**: 
  - 监听 Kafka 事件流
  - 内存存储站内内容（PostStore）
  - 提供站内内容查询接口
- **依赖**: 
  - Kafka（事件流）
  - Strato 服务（获取关注列表）

## 与 Home Mixer 的关系

Thunder 服务被 Home Mixer 调用：

```go
// Home Mixer 中的 ThunderSource 调用本服务
// internal/sources/thunder.go
```

## 主要组件

1. **PostStore**: 内存存储，使用 sync.Map 实现并发安全
2. **Kafka Listener**: 监听 tweet 事件流
3. **ThunderService**: gRPC 服务实现
4. **Strato Client**: 获取用户关注列表
