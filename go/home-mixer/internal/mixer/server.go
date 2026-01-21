package mixer

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// 这里假设 proto 生成的代码在 pkg/proto 包中
	// 实际使用时需要先运行 protoc 生成代码
	pb "github.com/x-algorithm/go/pkg/proto"
)

// HomeMixerServer 实现 gRPC 服务
type HomeMixerServer struct {
	pb.UnimplementedScoredPostsServiceServer
	pipeline *pipeline.CandidatePipeline
}

// NewHomeMixerServer 创建新的 HomeMixerServer 实例
func NewHomeMixerServer(p *pipeline.CandidatePipeline) *HomeMixerServer {
	return &HomeMixerServer{
		pipeline: p,
	}
}

// GetScoredPosts 处理获取排序后帖子的请求
func (s *HomeMixerServer) GetScoredPosts(
	ctx context.Context,
	req *pb.ScoredPostsQuery,
) (*pb.ScoredPostsResponse, error) {
	start := time.Now()

	// 1) 参数校验
	if req.ViewerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "viewer_id must be specified")
	}

	// 2) 构建内部 Query
	query := NewScoredPostsQuery(
		req.ViewerId,
		req.ClientAppId,
		req.CountryCode,
		req.LanguageCode,
		req.SeenIds,
		req.ServedIds,
		req.InNetworkOnly,
		req.IsBottomRequest,
		convertBloomFilterEntries(req.BloomFilterEntries),
	)

	log.Printf("Scored Posts request - request_id %s", query.RequestID)

	// 3) 执行候选管道
	pipelineResult, err := s.pipeline.Execute(ctx, query)
	if err != nil {
		// 根据错误类型决定返回的 gRPC 状态码
		return nil, status.Errorf(codes.Internal, "pipeline execute failed: %v", err)
	}

	// 4) 转换为响应格式
	scoredPosts := make([]*pb.ScoredPost, 0, len(pipelineResult.SelectedCandidates))
	for _, c := range pipelineResult.SelectedCandidates {
		screenNames := c.GetScreenNames()
		
		// 转换为 map[string]string（proto 的 map 键必须是 string）
		screenNamesMap := make(map[string]string)
		for k, v := range screenNames {
			screenNamesMap[uint64ToString(k)] = v
		}

		var retweetedTweetID uint64
		if c.RetweetedTweetID != nil {
			retweetedTweetID = *c.RetweetedTweetID
		}
		var retweetedUserID uint64
		if c.RetweetedUserID != nil {
			retweetedUserID = *c.RetweetedUserID
		}
		var inReplyToTweetID uint64
		if c.InReplyToTweetID != nil {
			inReplyToTweetID = *c.InReplyToTweetID
		}
		var score float32
		if c.Score != nil {
			score = float32(*c.Score)
		}
		var inNetwork bool
		if c.InNetwork != nil {
			inNetwork = *c.InNetwork
		}
		var servedType int32
		if c.ServedType != nil {
			servedType = *c.ServedType
		}
		var lastScoredTimestampMs uint64
		if c.LastScoredAtMs != nil {
			lastScoredTimestampMs = *c.LastScoredAtMs
		}
		var predictionRequestID uint64
		if c.PredictionRequestID != nil {
			predictionRequestID = *c.PredictionRequestID
		}
		var visibilityReason string
		if c.VisibilityReason != nil {
			visibilityReason = *c.VisibilityReason
		}

		scoredPosts = append(scoredPosts, &pb.ScoredPost{
			TweetId:               uint64(c.TweetID),
			AuthorId:              c.AuthorID,
			RetweetedTweetId:      retweetedTweetID,
			RetweetedUserId:       retweetedUserID,
			InReplyToTweetId:      inReplyToTweetID,
			Score:                 score,
			InNetwork:             inNetwork,
			ServedType:            servedType,
			LastScoredTimestampMs: lastScoredTimestampMs,
			PredictionRequestId:   predictionRequestID,
			Ancestors:             c.Ancestors,
			ScreenNames:           screenNamesMap,
			VisibilityReason:      visibilityReason,
		})
	}

	log.Printf(
		"Scored Posts response - request_id %s - %d posts (%d ms)",
		pipelineResult.Query.RequestID,
		len(scoredPosts),
		time.Since(start).Milliseconds(),
	)

	return &pb.ScoredPostsResponse{ScoredPosts: scoredPosts}, nil
}

// NewScoredPostsQuery 从 gRPC 请求构建内部 Query 对象
func NewScoredPostsQuery(
	viewerID int64,
	clientAppID int32,
	countryCode string,
	languageCode string,
	seenIDs []int64,
	servedIDs []int64,
	inNetworkOnly bool,
	isBottomRequest bool,
	bloomFilterEntries []pipeline.BloomFilterEntry,
) *pipeline.Query {
	// 生成 request_id（简化实现，实际应该使用更复杂的生成逻辑）
	requestID := generateRequestID(viewerID)

	return &pipeline.Query{
		UserID:            viewerID,
		ClientAppID:       clientAppID,
		CountryCode:       countryCode,
		LanguageCode:      languageCode,
		SeenIDs:           seenIDs,
		ServedIDs:         servedIDs,
		InNetworkOnly:     inNetworkOnly,
		IsBottomRequest:   isBottomRequest,
		BloomFilterEntries: bloomFilterEntries,
		RequestID:         requestID,
		UserFeatures:      pipeline.UserFeatures{}, // 初始为空，由 Query Hydrators 填充
	}
}

// convertBloomFilterEntries 转换 proto 的 BloomFilterEntry 到内部类型
func convertBloomFilterEntries(entries []*pb.BloomFilterEntry) []pipeline.BloomFilterEntry {
	if entries == nil {
		return nil
	}
	result := make([]pipeline.BloomFilterEntry, len(entries))
	for i, entry := range entries {
		// 将proto的BloomFilterEntry转换为内部类型
		result[i] = pipeline.BloomFilterEntry{
			Data: entry.Data, // proto的BloomFilterEntry包含bytes data字段
		}
	}
	return result
}

// generateRequestID 生成请求 ID
func generateRequestID(userID int64) string {
	return utils.GenerateRequestID(userID)
}

// 辅助函数
func uint64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}
