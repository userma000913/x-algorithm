package deserializer

import (
	"time"

	"x-algorithm-go/proto/thunder"
	"google.golang.org/protobuf/proto"
)

// InNetworkEvent 表示站内事件（供内部使用）
type InNetworkEvent struct {
	EventVariant interface{} // 可以是 *TweetCreateEvent 或 *TweetDeleteEvent
}

// DeserializeTweetEventV2 将 proto 二进制消息反序列化为 InNetworkEvent
func DeserializeTweetEventV2(payload []byte) (*InNetworkEvent, error) {
	// 尝试解码为 InNetworkEvent proto 消息
	// 由于我们使用的是占位符 proto，我们将创建一个模拟实现
	// 在生产环境中，这将使用: proto.Unmarshal(payload, &thunder.InNetworkEvent{})
	
	// 目前返回一个占位符
	// TODO: 当 proto 文件正确生成时实现实际的 proto 解码
	return &InNetworkEvent{
		EventVariant: nil, // 将在 ExtractPostsFromEvents 中设置
	}, nil
}

// DeserializeKafkaMessages 反序列化一批 Kafka 消息
func DeserializeKafkaMessages(messages [][]byte, deserializeFunc func([]byte) (*thunder.InNetworkEvent, error)) ([]*thunder.InNetworkEvent, error) {
	results := make([]*thunder.InNetworkEvent, 0, len(messages))

	for _, msg := range messages {
		event, err := deserializeFunc(msg)
		if err != nil {
			// 记录错误但继续处理其他消息
			continue
		}
		if event != nil {
			results = append(results, event)
		}
	}

	return results, nil
}

// ExtractPostsFromEvents 从 InNetworkEvent 中提取 LightPost 和 TweetDeleteEvent
func ExtractPostsFromEvents(events []*InNetworkEvent) ([]*thunder.LightPost, []*thunder.TweetDeleteEvent) {
	createTweets := make([]*thunder.LightPost, 0, len(events))
	deleteTweets := make([]*thunder.TweetDeleteEvent, 0, 10)

	for _, event := range events {
		if event == nil {
			continue
		}

		// 用于本地学习，从事件生成模拟帖子
		// 在生产环境中，这将检查来自 proto 的 event.EventVariant 类型
		
		// 生成一个用于测试的模拟 LightPost
		// 在实际实现中，这将从 event.EventVariant 中提取
		currentTime := time.Now().Unix()
		authorID := int64(1000 + len(createTweets)%100) // 变化作者 ID
		postID := int64(authorID)*1000000 + currentTime + int64(len(createTweets))
		
		createTweets = append(createTweets, &thunder.LightPost{
			PostID:    postID,
			AuthorID:  authorID,
			CreatedAt: currentTime,
			IsReply:   len(createTweets)%3 == 0, // 一些是回复
			IsRetweet: len(createTweets)%5 == 0, // 一些是转发
		})
	}

	return createTweets, deleteTweets
}

// 用于解码 proto 消息的占位符函数
func decodeProtoMessage(payload []byte, msg proto.Message) error {
	return proto.Unmarshal(payload, msg)
}
