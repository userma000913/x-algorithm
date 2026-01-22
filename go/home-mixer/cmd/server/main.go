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

	"x-algorithm-go/home-mixer/internal/clients"
	"x-algorithm-go/home-mixer/internal/mixer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// 这里假设 proto 生成的代码在 proto 包中
	// 实际使用时需要先运行 protoc 生成代码
	pb "x-algorithm-go/proto"
)

var (
	grpcPort     = flag.Int("grpc_port", 50051, "gRPC 服务器端口")
	metricsPort  = flag.Int("metrics_port", 9090, "HTTP 指标服务器端口")
	
	// 服务地址
	thunderAddr         = flag.String("thunder_addr", "localhost:50052", "Thunder 服务地址")
	phoenixRetrievalAddr = flag.String("phoenix_retrieval_addr", "localhost:50053", "Phoenix 检索服务地址")
	phoenixRankingAddr   = flag.String("phoenix_ranking_addr", "localhost:50054", "Phoenix 排序服务地址")
	tesAddr             = flag.String("tes_addr", "localhost:50055", "TES 服务地址")
	gizmoduckAddr       = flag.String("gizmoduck_addr", "localhost:50056", "Gizmoduck 服务地址")
	stratoAddr          = flag.String("strato_addr", "localhost:50057", "Strato 服务地址")
	uasAddr             = flag.String("uas_addr", "localhost:50058", "UAS 服务地址")
	vfAddr              = flag.String("vf_addr", "localhost:50059", "VF 服务地址")
)

func main() {
	flag.Parse()

	log.Printf("启动 Home Mixer 服务器，gRPC 端口: %d，指标端口: %d", *grpcPort, *metricsPort)

	// 1) 初始化客户端
	thunderClient, err := clients.NewThunderClient(*thunderAddr)
	if err != nil {
		log.Fatalf("创建 Thunder 客户端失败: %v", err)
	}
	defer thunderClient.(*clients.ThunderClientImpl).Close()

	phoenixRetrievalClient, err := clients.NewPhoenixRetrievalClient(*phoenixRetrievalAddr)
	if err != nil {
		log.Printf("警告: 创建 Phoenix 检索客户端失败: %v", err)
		phoenixRetrievalClient = nil
	}
	if phoenixRetrievalClient != nil {
		defer phoenixRetrievalClient.(*clients.PhoenixRetrievalClientImpl).Close()
	}

	tesClient, err := clients.NewTESClient(*tesAddr)
	if err != nil {
		log.Printf("警告: 创建 TES 客户端失败: %v", err)
		tesClient = nil
	}
	if tesClient != nil {
		defer tesClient.(*clients.TESClientImpl).Close()
	}

	gizmoduckClient, err := clients.NewGizmoduckClient(*gizmoduckAddr)
	if err != nil {
		log.Printf("警告: 创建 Gizmoduck 客户端失败: %v", err)
		gizmoduckClient = nil
	}
	if gizmoduckClient != nil {
		defer gizmoduckClient.(*clients.GizmoduckClientImpl).Close()
	}

	stratoClient, err := clients.NewStratoClient(*stratoAddr)
	if err != nil {
		log.Printf("警告: 创建 Strato 客户端失败: %v", err)
		stratoClient = nil
	}
	if stratoClient != nil {
		defer stratoClient.(*clients.StratoClientImpl).Close()
	}

	uasFetcher, err := clients.NewUASFetcher(*uasAddr)
	if err != nil {
		log.Printf("警告: 创建 UAS 获取器失败: %v", err)
		uasFetcher = nil
	}
	if uasFetcher != nil {
		defer uasFetcher.(*clients.UASFetcherImpl).Close()
	}

	vfClient, err := clients.NewVFClient(*vfAddr)
	if err != nil {
		log.Printf("警告: 创建 VF 客户端失败: %v", err)
		vfClient = nil
	}
	if vfClient != nil {
		defer vfClient.(*clients.VFClientImpl).Close()
	}

	stratoClientForCache, err := clients.NewStratoClientForCache(*stratoAddr)
	if err != nil {
		log.Printf("警告: 创建用于缓存的 Strato 客户端失败: %v", err)
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
		log.Fatalf("监听失败: %v", err)
	}

	grpcServer := grpc.NewServer()

	// 5) 启用 gRPC 反射（用于开发）
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
		log.Printf("HTTP 服务器监听端口 %d", *metricsPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动 HTTP 服务器失败: %v", err)
		}
	}()

	// 9) 启动 gRPC 服务器（在 goroutine 中）
	go func() {
		log.Printf("gRPC 服务器监听端口 :%d", *grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("服务启动失败: %v", err)
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

	log.Println("正在关闭服务器...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 HTTP 服务器
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("关闭 HTTP 服务器时出错: %v", err)
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
		log.Println("服务器已优雅关闭")
	case <-ctx.Done():
		log.Println("服务器关闭超时，强制停止")
		grpcServer.Stop()
	}
}
