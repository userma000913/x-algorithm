package clients

import (
	"context"
	"fmt"

	"github.com/x-algorithm/go/candidate-pipeline/pipeline"
	"github.com/x-algorithm/go/home-mixer/internal/sources"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PhoenixRetrievalClientImpl implements PhoenixRetrievalClient interface
type PhoenixRetrievalClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewPhoenixRetrievalClient creates a new Phoenix Retrieval client
func NewPhoenixRetrievalClient(address string) (sources.PhoenixRetrievalClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Phoenix Retrieval service: %w", err)
	}

	return &PhoenixRetrievalClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// Retrieve implements PhoenixRetrievalClient interface
func (c *PhoenixRetrievalClientImpl) Retrieve(
	ctx context.Context,
	userID uint64,
	sequence *pipeline.UserActionSequence,
	maxResults int,
) (*sources.RetrievalResponse, error) {
	// Mock implementation for local learning/testing
	// Returns test out-of-network posts based on user action sequence
	
	_ = ctx
	
	// Generate mock out-of-network candidates
	candidates := make([]sources.ScoredCandidate, 0)
	currentTime := int64(1704067200) // 2024-01-01 00:00:00 UTC
	
	// Create test posts from random authors (out-of-network)
	for i := 0; i < maxResults && i < 50; i++ {
		// Generate author ID (different from userID to ensure out-of-network)
		authorID := uint64(1000000 + i)
		tweetID := int64(authorID)*1000000 + currentTime + int64(i)
		
		candidates = append(candidates, sources.ScoredCandidate{
			Candidate: &sources.TweetInfo{
				TweetID:         tweetID,
				AuthorID:        authorID,
				InReplyToTweetID: 0,
			},
		})
	}
	
	return &sources.RetrievalResponse{
		TopKCandidates: []sources.ScoredCandidatesGroup{
			{
				Candidates: candidates,
			},
		},
	}, nil
}

// Close closes the gRPC connection
func (c *PhoenixRetrievalClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PhoenixRankingClientImpl implements PhoenixRankingClient interface
type PhoenixRankingClientImpl struct {
	conn   *grpc.ClientConn
	address string
}

// NewPhoenixRankingClient creates a new Phoenix Ranking client
func NewPhoenixRankingClient(address string) (*PhoenixRankingClientImpl, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Phoenix Ranking service: %w", err)
	}

	return &PhoenixRankingClientImpl{
		conn:    conn,
		address: address,
	}, nil
}

// Rank implements PhoenixRankingClient interface
func (c *PhoenixRankingClientImpl) Rank(
	ctx context.Context,
	req interface{}, // TODO: Define proper request type
) (interface{}, error) {
	// Mock implementation - this would normally call the Phoenix ranking service
	// For local learning, we return mock predictions
	_ = ctx
	_ = req
	return nil, fmt.Errorf("Phoenix Ranking requires proper request type - use scorers.NewPhoenixScorer with mock client")
}


// Close closes the gRPC connection
func (c *PhoenixRankingClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
