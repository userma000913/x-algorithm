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

	"x-algorithm-go/thunder/internal/kafka"
	"x-algorithm-go/thunder/internal/metrics"
	"x-algorithm-go/thunder/internal/poststore"
	"x-algorithm-go/thunder/internal/service"
	"x-algorithm-go/thunder/internal/strato"
	"x-algorithm-go/proto/thunder"
	"google.golang.org/grpc"
)

var (
	grpcPort              = flag.Int("grpc_port", 50052, "gRPC 服务器端口（默认: 50052，与 Home Mixer 不同）")
	httpPort              = flag.Int("http_port", 8080, "HTTP 服务器端口")
	postRetentionSeconds  = flag.Uint64("post_retention_seconds", 2*24*60*60, "帖子保留期（秒，默认: 2 天）")
	requestTimeoutMs      = flag.Uint64("request_timeout_ms", 0, "请求超时（毫秒，0 = 无超时）")
	maxConcurrentRequests = flag.Int64("max_concurrent_requests", 100, "最大并发请求数")
	kafkaBatchSize       = flag.Int("kafka_batch_size", 100, "Kafka 批次大小")
	isServing            = flag.Bool("is_serving", true, "是否以服务模式启动")
	enableProfiling      = flag.Bool("enable_profiling", false, "启用性能分析")
	
	// Kafka 配置
	kafkaBrokers          = flag.String("kafka_brokers", "localhost:9092", "Kafka 代理地址（逗号分隔）")
	kafkaTopic            = flag.String("kafka_topic", "tweet_events", "Kafka 主题名称")
	kafkaGroupID          = flag.String("kafka_group_id", "thunder", "Kafka 消费者组 ID")
	kafkaPartitions       = flag.String("kafka_partitions", "", "要消费的 Kafka 分区（逗号分隔，空 = 全部）")
	kafkaNumThreads       = flag.Int("kafka_num_threads", 1, "Kafka 处理线程数")
	kafkaLagMonitorSecs   = flag.Int("kafka_lag_monitor_secs", 60, "分区延迟监控间隔（秒）")
	kafkaSkipToLatest     = flag.Bool("kafka_skip_to_latest", false, "启动时跳转到最新偏移量")
	
	// SSL/SASL 配置
	kafkaSecurityProtocol = flag.String("kafka_security_protocol", "", "Kafka 安全协议（SSL、SASL_PLAINTEXT 等）")
	kafkaSASLMechanism    = flag.String("kafka_sasl_mechanism", "", "Kafka SASL 机制")
	kafkaSASLUsername      = flag.String("kafka_sasl_username", "", "Kafka SASL 用户名")
	kafkaSASLPassword      = flag.String("kafka_sasl_password", "", "Kafka SASL 密码")
)

func main() {
	flag.Parse()

	log.Printf("启动 Thunder 服务，gRPC 端口: %d，HTTP 端口: %d，保留期: %d 秒 (%.1f 天)，请求超时: %d 毫秒",
		*grpcPort, *httpPort, *postRetentionSeconds,
		float64(*postRetentionSeconds)/86400.0, *requestTimeoutMs)

	// 初始化 PostStore
	postStore := poststore.NewPostStore(*postRetentionSeconds, *requestTimeoutMs)
	log.Printf("已初始化 PostStore 用于内存帖子存储（保留期: %d 秒 / %.1f 天，请求超时: %d 毫秒）",
		*postRetentionSeconds, float64(*postRetentionSeconds)/86400.0, *requestTimeoutMs)

	// 初始化 StratoClient
	stratoClient := strato.NewStratoClient()

	// 创建 ThunderService
	thunderService := service.NewThunderService(postStore, stratoClient, *maxConcurrentRequests)
	log.Printf("已初始化 ThunderService，最大并发请求数=%d", *maxConcurrentRequests)

	// 创建 gRPC 服务器
	grpcServer := grpc.NewServer()

	// 启动 gRPC 服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("监听端口 %d 失败: %v", *grpcPort, err)
	}

	// 注册服务
	thunder.RegisterInNetworkPostsServiceServer(grpcServer, thunderService)

	go func() {
		log.Printf("gRPC 服务器监听端口 %d", *grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("启动 gRPC 服务器失败: %v", err)
		}
	}()

	// 初始化指标
	metrics.InitMetrics()

	// 启动 HTTP 服务器用于健康检查和指标
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// TODO: 暴露 Prometheus 指标
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Metrics endpoint - Prometheus integration pending\n"))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: httpMux,
	}

	go func() {
		log.Printf("HTTP 服务器监听端口 %d", *httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动 HTTP 服务器失败: %v", err)
		}
	}()

	// 如果在服务模式下，初始化 Kafka 监听器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *isServing {
		// 解析 Kafka 分区
		var partitions []int32
		if *kafkaPartitions != "" {
			parts := strings.Split(*kafkaPartitions, ",")
			for _, p := range parts {
				if partID, err := strconv.ParseInt(strings.TrimSpace(p), 10, 32); err == nil {
					partitions = append(partitions, int32(partID))
				}
			}
		}

		// 解析 Kafka 代理
		brokers := strings.Split(*kafkaBrokers, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}

		// 创建 Kafka 配置
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

		// 启动 Kafka 处理
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
				log.Printf("Kafka 处理错误: %v", err)
			}
		}()

		// 完成初始化
		if err := postStore.FinalizeInit(context.Background()); err != nil {
			log.Printf("完成 PostStore 初始化失败: %v", err)
		}

		// 启动自动清理任务
		postStore.StartAutoTrim(context.Background(), 2) // 每 2 分钟运行一次
		log.Printf("已启动 PostStore 自动清理任务（间隔: 2 分钟，保留期: %.1f 天）",
			float64(*postRetentionSeconds)/86400.0)

		// 启动统计日志记录器
		postStore.StartStatsLogger(context.Background())
		log.Println("已启动 PostStore 统计日志记录器")
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Thunder 服务已就绪")
	<-sigChan

	log.Println("正在关闭 Thunder 服务...")
	
	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	cancel() // 取消 Kafka 上下文

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭 HTTP 服务器时出错: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("Thunder 服务关闭完成")
}
