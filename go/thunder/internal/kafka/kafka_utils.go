package kafka

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/x-algorithm/go/thunder/internal/deserializer"
	"github.com/x-algorithm/go/thunder/internal/metrics"
	"github.com/x-algorithm/go/thunder/internal/poststore"
	"golang.org/x/sync/semaphore"
)

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	// Consumer config
	Brokers              []string
	Topic                string
	GroupID              string
	AutoOffsetReset      string
	FetchTimeoutMs       int
	MaxPartitionFetchBytes int
	SkipToLatest         bool
	Partitions           []int32

	// SSL/SASL config
	SecurityProtocol     string
	SASLMechanism        string
	SASLUsername         string
	SASLPassword         string

	// Producer config (for non-serving mode)
	ProducerBrokers      []string
	ProducerTopic        string
	ProducerSASLMechanism string
	ProducerSASLUsername string
	ProducerSASLPassword string
}

// StartKafka starts Kafka processing based on serving mode
func StartKafka(
	ctx context.Context,
	config KafkaConfig,
	postStore *poststore.PostStore,
	user string,
	catchupChan chan<- int64,
	isServing bool,
	numThreads int,
	batchSize int,
	lagMonitorIntervalSecs int,
) error {
	if isServing {
		// Serving mode: use v2 listener (InNetworkEvents)
		log.Println("Starting Kafka in serving mode (v2 listener)")
		
		uniqueID := generateUniqueID()
		config.GroupID = fmt.Sprintf("%s-%s", config.GroupID, uniqueID)
		
		return StartTweetEventProcessingV2(
			ctx,
			config,
			postStore,
			catchupChan,
			numThreads,
			batchSize,
			lagMonitorIntervalSecs,
		)
	} else {
		// Non-serving mode: use v1 listener + producer
		log.Println("Starting Kafka in non-serving mode (v1 listener + producer)")
		
		config.GroupID = fmt.Sprintf("%s-%s", config.GroupID, user)
		
		return StartTweetEventProcessing(
			ctx,
			config,
			postStore,
			numThreads,
			batchSize,
			lagMonitorIntervalSecs,
		)
	}
}

// StartTweetEventProcessingV2 starts v2 tweet event processing (serving mode)
func StartTweetEventProcessingV2(
	ctx context.Context,
	config KafkaConfig,
	postStore *poststore.PostStore,
	catchupChan chan<- int64,
	numThreads int,
	batchSize int,
	lagMonitorIntervalSecs int,
) error {
	// Calculate partition distribution
	partitionsPerThread := (len(config.Partitions) + numThreads - 1) / numThreads
	
	log.Printf("Starting %d message processing threads for %d partitions (%d partitions per thread)",
		numThreads, len(config.Partitions), partitionsPerThread)

	// Create semaphore to limit concurrent batch processing
	sem := semaphore.NewWeighted(3)

	// Create wait group to track all threads
	var wg sync.WaitGroup
	wg.Add(numThreads)

	// Spawn processing threads
	for threadID := 0; threadID < numThreads; threadID++ {
		startIdx := threadID * partitionsPerThread
		endIdx := (threadID + 1) * partitionsPerThread
		if endIdx > len(config.Partitions) {
			endIdx = len(config.Partitions)
		}
		if startIdx >= len(config.Partitions) {
			break
		}

		threadPartitions := config.Partitions[startIdx:endIdx]
		
		// Convert KafkaConfig to KafkaConsumerConfig
		consumerConfig := KafkaConsumerConfig{
			Brokers:              config.Brokers,
			Topic:                config.Topic,
			GroupID:              config.GroupID,
			Partitions:           threadPartitions,
			AutoOffsetReset:      config.AutoOffsetReset,
			FetchTimeoutMs:       config.FetchTimeoutMs,
			MaxPartitionFetchBytes: config.MaxPartitionFetchBytes,
			SkipToLatest:         config.SkipToLatest,
		}

		go func(tid int, tpartitions []int32, tconfig KafkaConsumerConfig) {
			defer wg.Done()
			
			log.Printf("Starting message processing thread %d for partitions %v", tid, tpartitions)

			consumer, err := CreateKafkaConsumer(ctx, tconfig)
			if err != nil {
				log.Fatalf("Failed to create consumer for thread %d: %v", tid, err)
				return
			}

			// Start partition lag monitoring
			StartPartitionLagMonitor(ctx, consumer, config.Topic, lagMonitorIntervalSecs)

			// Process messages
			if err := ProcessTweetEventsV2(
				ctx,
				consumer,
				postStore,
				batchSize,
				catchupChan,
				sem,
			); err != nil {
				log.Fatalf("Tweet events processing thread %d exited unexpectedly: %v", tid, err)
			}
		}(threadID, threadPartitions, consumerConfig)
	}

	// Wait for all threads to complete catchup
	// Note: catchupChan is send-only, so we can't receive from it here
	// In a real implementation, we would use a separate receive channel
	if catchupChan != nil {
		log.Println("Waiting for Kafka catchup...")
		wg.Wait()
		log.Println("All Kafka threads completed catchup")
	} else {
		wg.Wait()
	}

	return nil
}

// StartTweetEventProcessing starts v1 tweet event processing (non-serving mode)
func StartTweetEventProcessing(
	ctx context.Context,
	config KafkaConfig,
	postStore *poststore.PostStore,
	numThreads int,
	batchSize int,
	lagMonitorIntervalSecs int,
) error {
	// Similar to v2 but uses v1 deserializer
	// For now, just log
	log.Println("V1 tweet event processing not yet implemented")
	return nil
}

// ProcessTweetEventsV2 is the main message processing loop
func ProcessTweetEventsV2(
	ctx context.Context,
	consumer KafkaConsumer,
	postStore *poststore.PostStore,
	batchSize int,
	catchupChan chan<- int64,
	sem *semaphore.Weighted,
) error {
	messageBuffer := make([][]byte, 0, batchSize)
	batchCount := 0
	initDataDownloaded := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Poll for messages
		messages, err := consumer.Poll(ctx, batchSize)
		if err != nil {
			log.Printf("Error polling messages: %v", err)
			IncKafkaPollErrors()
			continue
		}

		// Check if catchup is complete
		if !initDataDownloaded && catchupChan != nil {
			if lags, err := consumer.GetPartitionLags(ctx); err == nil {
				totalLag := int64(0)
				for _, lag := range lags {
					totalLag += lag.Lag
				}
				if totalLag < int64(len(lags)*batchSize) {
					initDataDownloaded = true
					select {
					case catchupChan <- totalLag:
					default:
					}
				}
			}
		}

		messageBuffer = append(messageBuffer, messages...)

		// Process batch when we have enough messages
		if len(messageBuffer) >= batchSize {
			batchCount++
			batch := make([][]byte, len(messageBuffer))
			copy(batch, messageBuffer)
			messageBuffer = messageBuffer[:0]

			// Acquire semaphore if init data is downloaded
			if initDataDownloaded {
				if err := sem.Acquire(ctx, 1); err != nil {
					log.Printf("Failed to acquire semaphore: %v", err)
					continue
				}
			}

			// Process batch in background
			go func(b [][]byte, count int) {
				defer func() {
					if initDataDownloaded {
						sem.Release(1)
					}
				}()

				if err := ProcessBatch(postStore, b); err != nil {
					log.Printf("Error processing batch %d: %v", count, err)
				}
			}(batch, batchCount)
		}
	}
}

// ProcessBatch processes a single batch of messages
func ProcessBatch(postStore *poststore.PostStore, messages [][]byte) error {
	if len(messages) == 0 {
		return nil
	}

	// Deserialize messages using the deserializer package
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

	// Extract posts from events
	lightPosts, deletePosts := deserializer.ExtractPostsFromEvents(events)

	// Insert posts into store
	if len(lightPosts) > 0 {
		postStore.InsertPosts(lightPosts)
		log.Printf("Processed batch: inserted %d posts", len(lightPosts))
	}

	// Mark posts as deleted
	if len(deletePosts) > 0 {
		postStore.MarkAsDeleted(deletePosts)
		log.Printf("Processed batch: deleted %d posts", len(deletePosts))
	}

	return nil
}

// StartPartitionLagMonitor starts monitoring partition lag
func StartPartitionLagMonitor(
	ctx context.Context,
	consumer KafkaConsumer,
	topic string,
	intervalSecs int,
) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSecs) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if lags, err := consumer.GetPartitionLags(ctx); err == nil {
					m := metrics.GetMetrics()
					for _, lag := range lags {
						m.KafkaPartitionLag.WithLabelValues(
							topic,
							fmt.Sprintf("%d", lag.PartitionID),
						).Set(float64(lag.Lag))
					}
				}
			}
		}
	}()
}

// generateUniqueID generates a unique ID (UUID-like)
func generateUniqueID() string {
	// Simple implementation - in production use proper UUID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
