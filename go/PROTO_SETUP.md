# Proto 代码生成指南

## 前置要求

1. 安装 `protoc`（Protocol Buffers 编译器）
   ```bash
   # macOS
   brew install protobuf
   
   # Linux
   sudo apt-get install protobuf-compiler
   
   # 或从源码编译
   # https://github.com/protocolbuffers/protobuf/releases
   ```

2. 安装 Go 的 protoc 插件
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

3. 确保 `$GOPATH/bin` 或 `$GOBIN` 在 `$PATH` 中

## 生成代码

在项目根目录（`go/`）运行：

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       pkg/proto/scored_posts.proto
```

这会在 `pkg/proto/` 目录下生成：
- `scored_posts.pb.go` - 消息类型定义
- `scored_posts_grpc.pb.go` - gRPC 服务定义

## 验证

生成后，运行：

```bash
go build ./pkg/proto/...
```

如果没有错误，说明生成成功。

## 注意事项

- 如果修改了 `.proto` 文件，需要重新运行 `protoc` 命令
- 生成的代码不应该手动修改
- 确保 `go.mod` 中的依赖版本正确
