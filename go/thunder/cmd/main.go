package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/x-algorithm/go/thunder/internal/kafka"
	"github.com/x-algorithm/go/thunder/internal/metrics"
	"github.com/x-algorithm/go/thunder/internal/poststore"
	"github.com/x-algorithm/go/thunder/internal/service"
	"github.com/x-algorithm/go/thunder/internal/strato"
	"github.com/x-algorithm/go/pkg/proto/thunder"
	"google.golang.org/grpc"
)

var (
	grpcPort              = flag.Int("grpc_port", 50052, "gRPC server port (default: 50052, different from Home Mixer)")
	httpPort              = flag.Int("http_port", 8080, "HTTP server port")
	postRetentionSeconds  = flag.Uint64("post_retention_seconds", 2*24*60*60, "Post retention period in seconds (default: 2 days)")
	requestTimeoutMs      = flag.Uint64("request_timeout_ms", 0, "Request timeout in milliseconds (0 = no timeout)")
	maxConcurrentRequests = flag.Int64("max_concurrent_requests", 100, "Maximum concurrent requests")
	kafkaBatchSize       = flag.Int("kafka_batch_size", 100, "Kafka batch size")
	isServing            = flag.Bool("is_serving", true, "Whether to start in serving mode")
	enableProfiling      = flag.Bool("enable_profiling", false, "Enable profiling")
	
	// Kafka configuration
	kafkaBrokers          = flag.String("kafka_brokers", "localhost:9092", "Kafka broker addresses (comma-separated)")
	kafkaTopic            = flag.String("kafka_topic", "tweet_events", "Kafka topic name")
	kafkaGroupID          = flag.String("kafka_group_id", "thunder", "Kafka consumer group ID")
	kafkaPartitions       = flag.String("kafka_partitions", "", "Kafka partitions to consume (comma-separated, empty = all)")
	kafkaNumThreads       = flag.Int("kafka_num_threads", 1, "Number of Kafka processing threads")
	kafkaLagMonitorSecs   = flag.Int("kafka_lag_monitor_secs", 60, "Partition lag monitor interval in seconds")
	kafkaSkipToLatest     = flag.Bool("kafka_skip_to_latest", false, "Skip to latest offset on startup")
	
	// SSL/SASL configuration
	kafkaSecurityProtocol = flag.String("kafka_security_protocol", "", "Kafka security protocol (SSL, SASL_PLAINTEXT, etc.)")
	kafkaSASLMechanism    = flag.String("kafka_sasl_mechanism", "", "Kafka SASL mechanism")
	kafkaSASLUsername      = flag.String("kafka_sasl_username", "", "Kafka SASL username")
	kafkaSASLPassword      = flag.String("kafka_sasl_password", "", "Kafka SASL password")
)

func main() {
	flag.Parse()

	log.Printf("Starting Thunder service with gRPC port: %d, HTTP port: %d, retention: %d seconds (%.1f days), request_timeout: %dms",
		*grpcPort, *httpPort, *postRetentionSeconds,
		float64(*postRetentionSeconds)/86400.0, *requestTimeoutMs)

	// Initialize PostStore
	postStore := poststore.NewPostStore(*postRetentionSeconds, *requestTimeoutMs)
	log.Printf("Initialized PostStore for in-memory post storage (retention: %d seconds / %.1f days, request_timeout: %dms)",
		*postRetentionSeconds, float64(*postRetentionSeconds)/86400.0, *requestTimeoutMs)

	// Initialize StratoClient
	stratoClient := strato.NewStratoClient()

	// Create ThunderService
	thunderService := service.NewThunderService(postStore, stratoClient, *maxConcurrentRequests)
	log.Printf("Initialized ThunderService with max_concurrent_requests=%d", *maxConcurrentRequests)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", *grpcPort, err)
	}

	// Register the service
	thunder.RegisterInNetworkPostsServiceServer(grpcServer, thunderService)

	go func() {
		log.Printf("gRPC server listening on port %d", *grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Initialize metrics
	metrics.InitMetrics()

	// Start HTTP server for health checks and metrics
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Expose Prometheus metrics
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Metrics endpoint - Prometheus integration pending\n"))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: httpMux,
	}

	go func() {
		log.Printf("HTTP server listening on port %d", *httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Initialize Kafka listener if in serving mode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *isServing {
		// Parse Kafka partitions
		var partitions []int32
		if *kafkaPartitions != "" {
			parts := strings.Split(*kafkaPartitions, ",")
			for _, p := range parts {
				if partID, err := strconv.ParseInt(strings.TrimSpace(p), 10, 32); err == nil {
					partitions = append(partitions, int32(partID))
				}
			}
		}

		// Parse Kafka brokers
		brokers := strings.Split(*kafkaBrokers, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}

		// Create Kafka config
		kafkaConfig := kafka.KafkaConfig{
			Brokers:                brokers,
			Topic:                  *kafkaTopic,
			GroupID:                *kafkaGroupID,
			Partitions:             partitions,
			AutoOffsetReset:        "earliest",
			FetchTimeoutMs:         1000,
			MaxPartitionFetchBytes: 1048576,
			SkipToLatest:           *kafkaSkipToLatest,
			SecurityProtocol:      *kafkaSecurityProtocol,
			SASLMechanism:          *kafkaSASLMechanism,
			SASLUsername:           *kafkaSASLUsername,
			SASLPassword:           *kafkaSASLPassword,
		}

		// Start Kafka processing
		catchupChan := make(chan int64, *kafkaNumThreads)
		go func() {
			if err := kafka.StartKafka(
				ctx,
				kafkaConfig,
				postStore,
				"thunder",
				catchupChan,
				*isServing,
				*kafkaNumThreads,
				*kafkaBatchSize,
				*kafkaLagMonitorSecs,
			); err != nil {
				log.Printf("Kafka processing error: %v", err)
			}
		}()

		// Finalize initialization
		if err := postStore.FinalizeInit(context.Background()); err != nil {
			log.Printf("Failed to finalize PostStore initialization: %v", err)
		}

		// Start auto-trim task
		postStore.StartAutoTrim(context.Background(), 2) // Run every 2 minutes
		log.Printf("Started PostStore auto-trim task (interval: 2 minutes, retention: %.1f days)",
			float64(*postRetentionSeconds)/86400.0)

		// Start stats logger
		postStore.StartStatsLogger(context.Background())
		log.Println("Started PostStore stats logger")
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Thunder service is ready")
	<-sigChan

	log.Println("Shutting down Thunder service...")
	
	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel() // Cancel Kafka context

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("Thunder service shutdown complete")
}
