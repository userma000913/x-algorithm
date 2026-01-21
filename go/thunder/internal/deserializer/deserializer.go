package deserializer

import (
	"time"

	"github.com/x-algorithm/go/pkg/proto/thunder"
	"google.golang.org/protobuf/proto"
)

// InNetworkEvent represents an in-network event (for internal use)
type InNetworkEvent struct {
	EventVariant interface{} // Can be *TweetCreateEvent or *TweetDeleteEvent
}

// DeserializeTweetEventV2 deserializes a proto binary message into InNetworkEvent
func DeserializeTweetEventV2(payload []byte) (*InNetworkEvent, error) {
	// Try to decode as InNetworkEvent proto message
	// Since we're using placeholder proto, we'll create a mock implementation
	// In production, this would use: proto.Unmarshal(payload, &thunder.InNetworkEvent{})
	
	// For now, return a placeholder
	// TODO: Implement actual proto decoding when proto files are properly generated
	return &InNetworkEvent{
		EventVariant: nil, // Will be set in ExtractPostsFromEvents
	}, nil
}

// DeserializeKafkaMessages deserializes a batch of Kafka messages
func DeserializeKafkaMessages(messages [][]byte, deserializeFunc func([]byte) (*thunder.InNetworkEvent, error)) ([]*thunder.InNetworkEvent, error) {
	results := make([]*thunder.InNetworkEvent, 0, len(messages))

	for _, msg := range messages {
		event, err := deserializeFunc(msg)
		if err != nil {
			// Log error but continue processing other messages
			continue
		}
		if event != nil {
			results = append(results, event)
		}
	}

	return results, nil
}

// ExtractPostsFromEvents extracts LightPost and TweetDeleteEvent from InNetworkEvent
func ExtractPostsFromEvents(events []*InNetworkEvent) ([]*thunder.LightPost, []*thunder.TweetDeleteEvent) {
	createTweets := make([]*thunder.LightPost, 0, len(events))
	deleteTweets := make([]*thunder.TweetDeleteEvent, 0, 10)

	for _, event := range events {
		if event == nil {
			continue
		}

		// For local learning, generate mock posts from events
		// In production, this would check event.EventVariant type from proto
		
		// Generate a mock LightPost for testing
		// In real implementation, this would extract from event.EventVariant
		currentTime := time.Now().Unix()
		authorID := int64(1000 + len(createTweets)%100) // Vary author IDs
		postID := int64(authorID)*1000000 + currentTime + int64(len(createTweets))
		
		createTweets = append(createTweets, &thunder.LightPost{
			PostID:    postID,
			AuthorID:  authorID,
			CreatedAt: currentTime,
			IsReply:   len(createTweets)%3 == 0, // Some are replies
			IsRetweet: len(createTweets)%5 == 0, // Some are retweets
		})
	}

	return createTweets, deleteTweets
}

// Placeholder function to decode proto message
func decodeProtoMessage(payload []byte, msg proto.Message) error {
	return proto.Unmarshal(payload, msg)
}
