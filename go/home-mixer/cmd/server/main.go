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
	"syscall"
	"time"

	"github.com/x-algorithm/go/home-mixer/internal/clients"
	"github.com/x-algorithm/go/home-mixer/internal/mixer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// 这里假设 proto 生成的代码在 pkg/proto 包中
	// 实际使用时需要先运行 protoc 生成代码
	pb "github.com/x-algorithm/go/pkg/proto"
)

var (
	grpcPort     = flag.Int("grpc_port", 50051, "gRPC server port")
	metricsPort  = flag.Int("metrics_port", 9090, "HTTP metrics server port")
	
	// Service addresses
	thunderAddr         = flag.String("thunder_addr", "localhost:50052", "Thunder service address")
	phoenixRetrievalAddr = flag.String("phoenix_retrieval_addr", "localhost:50053", "Phoenix Retrieval service address")
	phoenixRankingAddr   = flag.String("phoenix_ranking_addr", "localhost:50054", "Phoenix Ranking service address")
	tesAddr             = flag.String("tes_addr", "localhost:50055", "TES service address")
	gizmoduckAddr       = flag.String("gizmoduck_addr", "localhost:50056", "Gizmoduck service address")
	stratoAddr          = flag.String("strato_addr", "localhost:50057", "Strato service address")
	uasAddr             = flag.String("uas_addr", "localhost:50058", "UAS service address")
	vfAddr              = flag.String("vf_addr", "localhost:50059", "VF service address")
)

func main() {
	flag.Parse()

	log.Printf("Starting Home Mixer server with gRPC port: %d, metrics port: %d", *grpcPort, *metricsPort)

	// 1) 初始化客户端
	thunderClient, err := clients.NewThunderClient(*thunderAddr)
	if err != nil {
		log.Fatalf("Failed to create Thunder client: %v", err)
	}
	defer thunderClient.(*clients.ThunderClientImpl).Close()

	phoenixRetrievalClient, err := clients.NewPhoenixRetrievalClient(*phoenixRetrievalAddr)
	if err != nil {
		log.Printf("Warning: Failed to create Phoenix Retrieval client: %v", err)
		phoenixRetrievalClient = nil
	}
	if phoenixRetrievalClient != nil {
		defer phoenixRetrievalClient.(*clients.PhoenixRetrievalClientImpl).Close()
	}

	tesClient, err := clients.NewTESClient(*tesAddr)
	if err != nil {
		log.Printf("Warning: Failed to create TES client: %v", err)
		tesClient = nil
	}
	if tesClient != nil {
		defer tesClient.(*clients.TESClientImpl).Close()
	}

	gizmoduckClient, err := clients.NewGizmoduckClient(*gizmoduckAddr)
	if err != nil {
		log.Printf("Warning: Failed to create Gizmoduck client: %v", err)
		gizmoduckClient = nil
	}
	if gizmoduckClient != nil {
		defer gizmoduckClient.(*clients.GizmoduckClientImpl).Close()
	}

	stratoClient, err := clients.NewStratoClient(*stratoAddr)
	if err != nil {
		log.Printf("Warning: Failed to create Strato client: %v", err)
		stratoClient = nil
	}
	if stratoClient != nil {
		defer stratoClient.(*clients.StratoClientImpl).Close()
	}

	uasFetcher, err := clients.NewUASFetcher(*uasAddr)
	if err != nil {
		log.Printf("Warning: Failed to create UAS fetcher: %v", err)
		uasFetcher = nil
	}
	if uasFetcher != nil {
		defer uasFetcher.(*clients.UASFetcherImpl).Close()
	}

	vfClient, err := clients.NewVFClient(*vfAddr)
	if err != nil {
		log.Printf("Warning: Failed to create VF client: %v", err)
		vfClient = nil
	}
	if vfClient != nil {
		defer vfClient.(*clients.VFClientImpl).Close()
	}

	stratoClientForCache, err := clients.NewStratoClientForCache(*stratoAddr)
	if err != nil {
		log.Printf("Warning: Failed to create Strato client for cache: %v", err)
		stratoClientForCache = nil
	}
	if stratoClientForCache != nil {
		defer stratoClientForCache.(*clients.StratoClientForCacheImpl).Close()
	}

	// 2) 创建 Pipeline 配置
	pipelineConfig := &mixer.PipelineConfig{
		ThunderClient:          thunderClient,
		PhoenixRetrievalClient: phoenixRetrievalClient,
		TESClient:              tesClient,
		GizmoduckClient:        gizmoduckClient,
		VFClient:               vfClient,
		UASFetcher:             uasFetcher,
		StratoClient:           stratoClient,
		StratoClientForCache:   stratoClientForCache,
		ThunderMaxResults:      500,
		PhoenixMaxResults:      500,
		TopK:                   50,
		MaxAge:                 7 * 24 * time.Hour,
	}

	// 3) 创建 Pipeline
	candidatePipeline := mixer.NewPhoenixCandidatePipeline(pipelineConfig)

	// 4) 创建 gRPC 服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 5) 启用 gRPC reflection (for development)
	reflection.Register(grpcServer)

	// 6) 创建服务实现
	homeMixerServer := mixer.NewHomeMixerServer(candidatePipeline.Pipeline)

	// 7) 注册服务
	pb.RegisterScoredPostsServiceServer(grpcServer, homeMixerServer)

	// 8) 启动 HTTP 服务器用于健康检查和指标
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	httpMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Metrics endpoint - Prometheus integration pending\n"))
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: httpMux,
	}

	go func() {
		log.Printf("HTTP server listening on port %d", *metricsPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 9) 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		log.Printf("gRPC server listening on :%d", *grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 10) 优雅关闭
	waitForShutdown(grpcServer, httpServer)
}

// waitForShutdown 等待关闭信号并优雅关闭服务器
func waitForShutdown(grpcServer *grpc.Server, httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 HTTP 服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// 优雅关闭 gRPC 服务器
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	// 等待关闭完成或超时
	select {
	case <-stopped:
		log.Println("Server stopped gracefully")
	case <-ctx.Done():
		log.Println("Server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}
}
