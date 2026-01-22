package kafka

import (
	"context"
	"log"
	"time"

	"x-algorithm-go/thunder/internal/deserializer"
	"x-algorithm-go/thunder/internal/poststore"
	"golang.org/x/sync/semaphore"
)

// KafkaListener 处理 Kafka 消息消费和处理
type KafkaListener struct {
	postStore *poststore.PostStore
	batchSize int
	semaphore *semaphore.Weighted
}

// NewKafkaListener 创建新的 KafkaListener
func NewKafkaListener(postStore *poststore.PostStore, batchSize int) *KafkaListener {
	return &KafkaListener{
		postStore: postStore,
		batchSize: batchSize,
		semaphore: semaphore.NewWeighted(3), // 限制并发批次处理
	}
}

// StartTweetEventProcessing 启动推文事件处理循环
func (kl *KafkaListener) StartTweetEventProcessing(ctx context.Context) error {
	log.Println("Starting Kafka tweet event processing")

	// 使用 kafka_utils 实现
	// 这是一个简化的包装器
	go kl.processMessages(ctx)

	return nil
}

// processMessages 模拟消息处理（占位符）
func (kl *KafkaListener) processMessages(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 模拟批次处理
			// 在实际实现中，这将轮询 Kafka 获取消息
			log.Println("Kafka listener: simulating message batch processing")
		}
	}
}

// ProcessBatch 处理一批 Kafka 消息
func (kl *KafkaListener) ProcessBatch(messages [][]byte) error {
	// 获取信号量许可
	if !kl.semaphore.TryAcquire(1) {
		// 如果已达到容量，跳过此批次
		return nil
	}
	defer kl.semaphore.Release(1)

	// 使用反序列化包反序列化消息
	events := make([]*deserializer.InNetworkEvent, 0, len(messages))
	for _, msg := range messages {
		event, err := deserializer.DeserializeTweetEventV2(msg)
		if err != nil {
			log.Printf("Error deserializing message: %v", err)
			continue
		}
		if event != nil {
			events = append(events, event)
		}
	}

	// 从事件中提取帖子
	lightPosts, deletePosts := deserializer.ExtractPostsFromEvents(events)

	// 将帖子插入存储
	if len(lightPosts) > 0 {
		kl.postStore.InsertPosts(lightPosts)
	}

	// 将帖子标记为已删除
	if len(deletePosts) > 0 {
		kl.postStore.MarkAsDeleted(deletePosts)
	}

	return nil
}

// StartPartitionLagMonitor 开始监控分区延迟（占位符）
func (kl *KafkaListener) StartPartitionLagMonitor(ctx context.Context, topic string, intervalSecs int) {
	// TODO: 实现分区延迟监控
	log.Printf("Partition lag monitor started for topic %s (interval: %ds)", topic, intervalSecs)
}
