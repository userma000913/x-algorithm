package clients

import (
	"github.com/x-algorithm/go/home-mixer/internal/hydrators"
	"github.com/x-algorithm/go/home-mixer/internal/query_hydrators"
	"github.com/x-algorithm/go/home-mixer/internal/side_effects"
	"github.com/x-algorithm/go/home-mixer/internal/sources"
)

// NewMockThunderClient creates a mock Thunder client for local testing
func NewMockThunderClient() sources.ThunderClient {
	return &ThunderClientImpl{
		conn:   nil, // No real connection needed for mock
		client: &thunderClientWrapper{conn: nil},
	}
}

// NewMockPhoenixRetrievalClient creates a mock Phoenix Retrieval client
func NewMockPhoenixRetrievalClient() sources.PhoenixRetrievalClient {
	return &PhoenixRetrievalClientImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockTESClient creates a mock TES client
func NewMockTESClient() hydrators.TweetEntityServiceClient {
	return &TESClientImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockGizmoduckClient creates a mock Gizmoduck client
func NewMockGizmoduckClient() hydrators.GizmoduckClient {
	return &GizmoduckClientImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockVFClient creates a mock VF client
func NewMockVFClient() hydrators.VisibilityFilteringClient {
	return &VFClientImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockStratoClient creates a mock Strato client for query hydrators
func NewMockStratoClient() query_hydrators.StratoClient {
	return &StratoClientImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockStratoClientForCache creates a mock Strato client for side effects
func NewMockStratoClientForCache() side_effects.StratoClient {
	return &StratoClientForCacheImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}

// NewMockUASFetcher creates a mock UAS Fetcher
func NewMockUASFetcher() query_hydrators.UserActionSequenceFetcher {
	return &UASFetcherImpl{
		conn:    nil,
		address: "mock://localhost",
	}
}
