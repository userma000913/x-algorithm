package kafka

import (
	"context"
	"log"
	"time"

	"github.com/x-algorithm/go/thunder/internal/metrics"
)

// GetMetrics returns the metrics instance
func GetMetrics() *metrics.Metrics {
	return metrics.GetMetrics()
}

// IncKafkaPollErrors increments Kafka poll errors counter
func IncKafkaPollErrors() {
	metrics.IncKafkaPollErrors()
}

// KafkaMessage represents a Kafka message
type KafkaMessage struct {
	Payload []byte
	Topic   string
	Partition int32
	Offset   int64
}

// KafkaConsumer represents a Kafka consumer interface
type KafkaConsumer interface {
	Poll(ctx context.Context, batchSize int) ([][]byte, error)
	GetPartitionLags(ctx context.Context) ([]PartitionLag, error)
	Start(ctx context.Context) error
	Close() error
}

// PartitionLag represents partition lag information
type PartitionLag struct {
	PartitionID int32
	Lag         int64
}

// KafkaConsumerConfig represents Kafka consumer configuration
type KafkaConsumerConfig struct {
	Brokers        []string
	Topic          string
	GroupID        string
	Partitions     []int32
	AutoOffsetReset string
	FetchTimeoutMs int
	MaxPartitionFetchBytes int
	SkipToLatest   bool
}

// CreateKafkaConsumer creates and starts a Kafka consumer with the given configuration
// This is a placeholder - in real implementation, this would create an actual Kafka consumer
func CreateKafkaConsumer(ctx context.Context, config KafkaConsumerConfig) (KafkaConsumer, error) {
	// TODO: Implement actual Kafka consumer creation
	// This would use a Kafka library like sarama or confluent-kafka-go
	// For now, return a mock consumer
	
	log.Printf("Creating Kafka consumer for topic %s, group %s", config.Topic, config.GroupID)
	
	// Placeholder: return a mock consumer
	return &MockKafkaConsumer{
		config: config,
	}, nil
}

// MockKafkaConsumer is a placeholder implementation for local learning
type MockKafkaConsumer struct {
	config     KafkaConsumerConfig
	messageCount int64
	startTime    time.Time
}

// NewMockKafkaConsumer creates a new mock Kafka consumer
func NewMockKafkaConsumer(config KafkaConsumerConfig) *MockKafkaConsumer {
	return &MockKafkaConsumer{
		config:     config,
		messageCount: 0,
		startTime:    time.Now(),
	}
}

func (m *MockKafkaConsumer) Poll(ctx context.Context, batchSize int) ([][]byte, error) {
	// Mock implementation for local learning - generates test messages periodically
	// In production, this would poll real Kafka
	
	// For local learning, return empty most of the time, but occasionally generate test messages
	// This simulates a real Kafka stream
	
	// Check if we should generate messages (simulate periodic message arrival)
	if m.messageCount == 0 || time.Since(m.startTime).Seconds() > 10 {
		// Generate a few test messages
		messages := make([][]byte, 0, batchSize)
		for i := 0; i < batchSize && i < 5; i++ {
			// Generate a mock message (will be deserialized by deserializer)
			// For now, just return empty - the deserializer will handle it
			messages = append(messages, []byte{})
		}
		m.messageCount++
		m.startTime = time.Now()
		return messages, nil
	}
	
	// Return empty most of the time to simulate idle periods
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return [][]byte{}, nil
	}
}

func (m *MockKafkaConsumer) GetPartitionLags(ctx context.Context) ([]PartitionLag, error) {
	// Mock implementation - return zero lag for local learning
	lags := make([]PartitionLag, 0, len(m.config.Partitions))
	for _, partition := range m.config.Partitions {
		lags = append(lags, PartitionLag{
			PartitionID: partition,
			Lag:         0, // Zero lag in mock mode
		})
	}
	return lags, nil
}

func (m *MockKafkaConsumer) Start(ctx context.Context) error {
	log.Printf("Mock Kafka consumer started for topic %s, group %s", m.config.Topic, m.config.GroupID)
	return nil
}

func (m *MockKafkaConsumer) Close() error {
	log.Printf("Mock Kafka consumer closed")
	return nil
}

// DeserializeKafkaMessages processes a batch of Kafka messages and deserializes them using the provided deserializer function
func DeserializeKafkaMessages(
	messages []KafkaMessage,
	deserializer func([]byte) (interface{}, error),
) ([]interface{}, error) {
	startTime := time.Now()
	defer func() {
		metrics.RecordBatchProcessingTime(time.Since(startTime))
	}()

	kafkaData := make([]interface{}, 0, len(messages))

	for _, msg := range messages {
		if len(msg.Payload) == 0 {
			continue
		}

		deserializedMsg, err := deserializer(msg.Payload)
		if err != nil {
			log.Printf("Failed to parse Kafka message: %v", err)
			metrics.IncKafkaMessagesFailedParse()
			continue
		}

		if deserializedMsg != nil {
			kafkaData = append(kafkaData, deserializedMsg)
		}
	}

	return kafkaData, nil
}

// ConvertKafkaMessages converts [][]byte to []KafkaMessage
func ConvertKafkaMessages(rawMessages [][]byte) []KafkaMessage {
	messages := make([]KafkaMessage, len(rawMessages))
	for i, payload := range rawMessages {
		messages[i] = KafkaMessage{
			Payload: payload,
		}
	}
	return messages
}
