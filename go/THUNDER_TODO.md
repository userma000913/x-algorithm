# Thunder 服务 TODO 列表

## Thunder 服务缺失功能

### 1. Kafka 工具函数 ❌
**文件**: `thunder/internal/kafka/utils.go`

**功能**:
- [ ] `CreateKafkaConsumer` - 创建和启动 Kafka consumer
- [ ] `DeserializeKafkaMessages` - 批量反序列化 Kafka 消息
- [ ] 错误处理和指标记录

**参考**: `thunder/kafka/utils.rs`

---

### 2. Kafka 启动逻辑 ❌
**文件**: `thunder/internal/kafka/kafka_utils.go` 或整合到 `thunder/cmd/main.go`

**功能**:
- [ ] `StartKafka` - 完整的 Kafka 启动逻辑
- [ ] 处理 serving 模式和非 serving 模式
- [ ] 配置 consumer/producer
- [ ] 多线程处理（partition 分配）
- [ ] Kafka catchup 信号处理（等待所有线程完成初始化）

**参考**: `thunder/kafka_utils.rs`, `thunder/main.rs` (lines 66-90)

---

### 3. Kafka 监听器完善 ❌
**文件**: `thunder/internal/kafka/listener.go`

**当前状态**: 只有占位实现

**需要完善**:
- [ ] 实际的 Kafka consumer 创建和启动
- [ ] 消息轮询（poll）逻辑
- [ ] 批量处理逻辑
- [ ] Partition lag 监控
- [ ] 多线程处理（每个线程处理部分 partitions）
- [ ] Kafka catchup 检测和信号发送
- [ ] 错误处理和重试逻辑

**参考**: `thunder/kafka/tweet_events_listener_v2.rs`

---

### 4. Prometheus 监控指标 ❌
**文件**: `thunder/internal/metrics/metrics.go`

**需要定义的指标**:
- [ ] `POST_STORE_TOTAL_POSTS` - PostStore 总帖子数
- [ ] `POST_STORE_USER_COUNT` - 用户数
- [ ] `POST_STORE_DELETED_POSTS` - 已删除帖子数
- [ ] `POST_STORE_ENTITY_COUNT` - 实体计数（带 labels）
- [ ] `POST_STORE_POSTS_RETURNED` - 返回的帖子数
- [ ] `POST_STORE_POSTS_RETURNED_RATIO` - 返回比例
- [ ] `POST_STORE_REQUEST_TIMEOUTS` - 请求超时数
- [ ] `POST_STORE_REQUESTS` - 请求计数
- [ ] `GET_IN_NETWORK_POSTS_COUNT` - GetInNetworkPosts 返回数
- [ ] `GET_IN_NETWORK_POSTS_DURATION` - 请求耗时
- [ ] `GET_IN_NETWORK_POSTS_DURATION_WITHOUT_STRATO` - 不含 Strato 的耗时
- [ ] `GET_IN_NETWORK_POSTS_FOLLOWING_SIZE` - 关注列表大小
- [ ] `GET_IN_NETWORK_POSTS_EXCLUDED_SIZE` - 排除列表大小
- [ ] `GET_IN_NETWORK_POSTS_FOUND_FRESHNESS_SECONDS` - 帖子新鲜度
- [ ] `GET_IN_NETWORK_POSTS_FOUND_TIME_RANGE_SECONDS` - 时间范围
- [ ] `GET_IN_NETWORK_POSTS_FOUND_REPLY_RATIO` - 回复比例
- [ ] `GET_IN_NETWORK_POSTS_FOUND_UNIQUE_AUTHORS` - 唯一作者数
- [ ] `GET_IN_NETWORK_POSTS_FOUND_POSTS_PER_AUTHOR` - 每个作者的帖子数
- [ ] `GET_IN_NETWORK_POSTS_MAX_RESULTS` - 最大结果数
- [ ] `IN_FLIGHT_REQUESTS` - 进行中的请求数
- [ ] `REJECTED_REQUESTS` - 被拒绝的请求数
- [ ] `KAFKA_PARTITION_LAG` - Kafka partition lag
- [ ] `KAFKA_POLL_ERRORS` - Kafka 轮询错误
- [ ] `KAFKA_MESSAGES_FAILED_PARSE` - 解析失败的消息数
- [ ] `BATCH_PROCESSING_TIME` - 批量处理时间

**参考**: `thunder/thunder_service.rs` (metrics usage), `thunder/posts/post_store.rs` (metrics usage)

---

### 5. PostStore 统计日志 ❌
**文件**: `thunder/internal/poststore/post_store.go`

**功能**:
- [ ] `StartStatsLogger` - 启动后台统计日志任务（每 5 秒记录一次）
- [ ] 统计用户数、帖子数、删除帖子数等
- [ ] 更新 Prometheus 指标

**参考**: `thunder/posts/post_store.rs` (lines 330-390)

---

### 6. ThunderService 统计报告 ❌
**文件**: `thunder/internal/service/service.go`

**功能**:
- [ ] `AnalyzeAndReportPostStatistics` - 分析帖子统计并报告指标
- [ ] 计算帖子新鲜度、时间范围、回复比例、唯一作者数等
- [ ] 更新 Prometheus 指标

**参考**: `thunder/thunder_service.rs` (lines 62-148)

---

### 7. 事件反序列化完善 ❌
**文件**: `thunder/internal/deserializer/deserializer.go`

**功能**:
- [ ] `DeserializeTweetEventV2` - 实际实现 proto 消息反序列化
- [ ] `ExtractPostsFromEvents` - 从事件中提取 LightPost 和 TweetDeleteEvent
- [ ] 处理 `TweetCreateEvent` 和 `TweetDeleteEvent`

**参考**: `thunder/kafka/tweet_events_listener_v2.rs` (lines 118-167)

---

### 8. Thunder 参数解析 ❌
**文件**: `thunder/cmd/main.go`

**需要添加的参数**:
- [ ] `kafka_num_threads` - Kafka 处理线程数
- [ ] `kafka_group_id` - Kafka group ID
- [ ] `kafka_batch_size` - Kafka 批量大小（已有）
- [ ] `kafka_tweet_events_v2_num_partitions` - Partition 数量
- [ ] `lag_monitor_interval_secs` - Lag 监控间隔
- [ ] `auto_offset_reset` - Offset 重置策略
- [ ] `fetch_timeout_ms` - Fetch 超时
- [ ] `skip_to_latest` - 是否跳到最新
- [ ] `security_protocol` - 安全协议
- [ ] `sasl_mechanism` - SASL 机制
- [ ] `sasl_username` - SASL 用户名
- [ ] `sasl_password` - SASL 密码
- [ ] `producer_sasl_mechanism` - Producer SASL 机制
- [ ] `producer_sasl_username` - Producer SASL 用户名
- [ ] `producer_sasl_password` - Producer SASL 密码
- [ ] `in_network_events_consumer_dest` - Consumer 目标地址

**参考**: `thunder/main.rs`, `thunder/kafka_utils.rs`

---

### 9. HTTP 服务器支持 ❌
**文件**: `thunder/cmd/main.go`

**功能**:
- [ ] HTTP 服务器（用于健康检查）
- [ ] gRPC 和 HTTP 同时支持
- [ ] 优雅关闭

**参考**: `thunder/main.rs` (lines 51-60)

---

### 10. PostStore 完善 ❌
**文件**: `thunder/internal/poststore/post_store.go`

**需要完善**:
- [ ] `start_stats_logger` 方法（已有 `StartAutoTrim`，但缺少 `StartStatsLogger`）
- [ ] 确保所有方法实现完整

---

## Home Mixer 服务缺失功能

### 1. 客户端实现 ❌
**文件**: `home-mixer/internal/clients/`

**需要实现的客户端**:
- [ ] `ThunderClient` - Thunder 服务客户端（用于 ThunderSource）
  - [ ] 连接池管理
  - [ ] 随机选择 channel（cluster 支持）
- [ ] `PhoenixRetrievalClient` - Phoenix 检索服务客户端
- [ ] `PhoenixPredictionClient` - Phoenix 排序服务客户端
- [ ] `TESClient` - Tweet Entity Service 客户端
- [ ] `GizmoduckClient` - Gizmoduck 服务客户端
- [ ] `StratoClient` - Strato 服务客户端（用于 query hydrator）
- [ ] `VFClient` - Visibility Filtering 客户端
- [ ] `UASFetcher` - User Action Sequence Fetcher

**当前状态**: 只有接口定义和 Mock 实现

**参考**: `home-mixer/clients/` (未开源，但可以从使用处推断接口)

---

### 2. Pipeline 配置完善 ❌
**文件**: `home-mixer/internal/mixer/pipeline.go`

**功能**:
- [ ] `NewPhoenixCandidatePipeline` - 完整的生产环境配置
- [ ] 创建所有客户端实例
- [ ] 组装所有组件（与 Rust 版本一致）

**参考**: `home-mixer/candidate_pipeline/phoenix_candidate_pipeline.rs` (lines 162-212)

---

### 3. 参数配置 ❌
**文件**: `home-mixer/internal/params/params.go` 或整合到配置中

**需要定义的常量**:
- [ ] `THUNDER_MAX_RESULTS` - Thunder 最大结果数
- [ ] `PHOENIX_MAX_RESULTS` - Phoenix 最大结果数
- [ ] `MAX_GRPC_MESSAGE_SIZE` - gRPC 消息最大大小
- [ ] 其他配置常量

**参考**: `home-mixer/params/` (未开源，但从使用处可以推断)

---

### 4. Home Mixer 服务入口完善 ❌
**文件**: `home-mixer/cmd/server/main.go`

**功能**:
- [ ] Metrics port 支持
- [ ] gRPC reflection 支持
- [ ] 压缩支持（Gzip, Zstd）
- [ ] HTTP 服务器（用于健康检查）
- [ ] 优雅关闭

**参考**: `home-mixer/main.rs`

---

## 通用缺失功能

### 1. 导入路径修复 ❌
**问题**: 拆分服务后，所有 import 路径需要更新

**需要修复**:
- [ ] 修复所有 `github.com/x-algorithm/go/home-mixer/internal/thunder/` 为 `github.com/x-algorithm/go/thunder/internal/`
- [ ] 确保所有文件使用正确的 import 路径
- [ ] 修复编译错误

---

### 2. Proto 代码生成 ❌
**文件**: `pkg/proto/`

**需要**:
- [ ] 实际运行 `protoc` 生成代码（或完善占位实现）
- [ ] 确保 proto 定义完整

---

## 优先级

### 高优先级
1. ✅ 服务拆分（已完成）
2. ❌ 修复 import 路径和编译错误
3. ❌ Kafka 工具函数和启动逻辑
4. ❌ 事件反序列化完善
5. ❌ PostStore 统计日志

### 中优先级
6. ❌ Prometheus 监控指标
7. ❌ ThunderService 统计报告
8. ❌ Thunder 参数解析完善
9. ❌ HTTP 服务器支持

### 低优先级
10. ❌ Home Mixer 客户端实现（需要外部服务）
11. ❌ Pipeline 配置完善
12. ❌ 参数配置

---

**最后更新**: 2024年
