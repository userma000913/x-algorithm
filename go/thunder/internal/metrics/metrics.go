package metrics

import (
	"sync"
	"time"
)

// Metrics holds all Prometheus metrics
// This is a placeholder implementation - in production, use github.com/prometheus/client_golang
type Metrics struct {
	// PostStore metrics
	PostStoreTotalPosts      *Gauge
	PostStoreUserCount       *Gauge
	PostStoreDeletedPosts    *Gauge
	PostStoreEntityCount     *GaugeVec
	PostStorePostsReturned   *Histogram
	PostStorePostsReturnedRatio *Histogram
	PostStoreRequestTimeouts *Counter
	PostStoreRequests        *Counter

	// GetInNetworkPosts metrics
	GetInNetworkPostsCount                    *Histogram
	GetInNetworkPostsDuration                 *Histogram
	GetInNetworkPostsDurationWithoutStrato    *Histogram
	GetInNetworkPostsFollowingSize            *Histogram
	GetInNetworkPostsExcludedSize             *Histogram
	GetInNetworkPostsMaxResults               *Histogram
	GetInNetworkPostsFoundFreshnessSeconds    *HistogramVec
	GetInNetworkPostsFoundTimeRangeSeconds    *HistogramVec
	GetInNetworkPostsFoundReplyRatio          *HistogramVec
	GetInNetworkPostsFoundUniqueAuthors      *HistogramVec
	GetInNetworkPostsFoundPostsPerAuthor     *HistogramVec
	InFlightRequests                          *Gauge
	RejectedRequests                         *Counter

	// Kafka metrics
	KafkaPartitionLag        *GaugeVec
	KafkaPollErrors          *Counter
	KafkaMessagesFailedParse *Counter
	BatchProcessingTime       *Histogram
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// InitMetrics initializes global metrics
func InitMetrics() {
	metricsOnce.Do(func() {
		globalMetrics = &Metrics{
			PostStoreTotalPosts:          NewGauge("post_store_total_posts"),
			PostStoreUserCount:           NewGauge("post_store_user_count"),
			PostStoreDeletedPosts:        NewGauge("post_store_deleted_posts"),
			PostStoreEntityCount:         NewGaugeVec("post_store_entity_count", []string{"type"}),
			PostStorePostsReturned:       NewHistogram("post_store_posts_returned"),
			PostStorePostsReturnedRatio:   NewHistogram("post_store_posts_returned_ratio"),
			PostStoreRequestTimeouts:     NewCounter("post_store_request_timeouts"),
			PostStoreRequests:            NewCounter("post_store_requests"),
			GetInNetworkPostsCount:       NewHistogram("get_in_network_posts_count"),
			GetInNetworkPostsDuration:   NewHistogram("get_in_network_posts_duration_seconds"),
			GetInNetworkPostsDurationWithoutStrato: NewHistogram("get_in_network_posts_duration_without_strato_seconds"),
			GetInNetworkPostsFollowingSize:        NewHistogram("get_in_network_posts_following_size"),
			GetInNetworkPostsExcludedSize:         NewHistogram("get_in_network_posts_excluded_size"),
			GetInNetworkPostsMaxResults:           NewHistogram("get_in_network_posts_max_results"),
			GetInNetworkPostsFoundFreshnessSeconds: NewHistogramVec("get_in_network_posts_found_freshness_seconds", []string{"stage"}),
			GetInNetworkPostsFoundTimeRangeSeconds: NewHistogramVec("get_in_network_posts_found_time_range_seconds", []string{"stage"}),
			GetInNetworkPostsFoundReplyRatio:      NewHistogramVec("get_in_network_posts_found_reply_ratio", []string{"stage"}),
			GetInNetworkPostsFoundUniqueAuthors:   NewHistogramVec("get_in_network_posts_found_unique_authors", []string{"stage"}),
			GetInNetworkPostsFoundPostsPerAuthor: NewHistogramVec("get_in_network_posts_found_posts_per_author", []string{"stage"}),
			InFlightRequests:                     NewGauge("in_flight_requests"),
			RejectedRequests:                     NewCounter("rejected_requests"),
			KafkaPartitionLag:                     NewGaugeVec("kafka_partition_lag", []string{"topic", "partition"}),
			KafkaPollErrors:                       NewCounter("kafka_poll_errors"),
			KafkaMessagesFailedParse:              NewCounter("kafka_messages_failed_parse"),
			BatchProcessingTime:                   NewHistogram("batch_processing_time_seconds"),
		}
	})
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	InitMetrics()
	return globalMetrics
}

// Gauge represents a Prometheus Gauge
type Gauge struct {
	name  string
	value float64
	mu    sync.RWMutex
}

func NewGauge(name string) *Gauge {
	return &Gauge{name: name}
}

func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

func (g *Gauge) Get() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

func (g *Gauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value++
}

func (g *Gauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value--
}

// GaugeVec represents a Prometheus Gauge with labels
type GaugeVec struct {
	name   string
	labels []string
	values map[string]float64
	mu     sync.RWMutex
}

func NewGaugeVec(name string, labels []string) *GaugeVec {
	return &GaugeVec{
		name:   name,
		labels: labels,
		values: make(map[string]float64),
	}
}

func (gv *GaugeVec) WithLabelValues(labelValues ...string) *Gauge {
	key := gv.buildKey(labelValues)
	return &Gauge{
		name:  gv.name + "{" + key + "}",
		value: gv.get(key),
	}
}

func (gv *GaugeVec) Set(labelValues []string, value float64) {
	gv.mu.Lock()
	defer gv.mu.Unlock()
	key := gv.buildKey(labelValues)
	gv.values[key] = value
}

func (gv *GaugeVec) buildKey(labelValues []string) string {
	key := ""
	for i, label := range gv.labels {
		if i > 0 {
			key += ","
		}
		key += label + "=" + labelValues[i]
	}
	return key
}

func (gv *GaugeVec) get(key string) float64 {
	gv.mu.RLock()
	defer gv.mu.RUnlock()
	return gv.values[key]
}

// Counter represents a Prometheus Counter
type Counter struct {
	name  string
	value float64
	mu    sync.RWMutex
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

func (c *Counter) Get() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Histogram represents a Prometheus Histogram
type Histogram struct {
	name  string
	value float64
	count int64
	mu    sync.RWMutex
}

func NewHistogram(name string) *Histogram {
	return &Histogram{name: name}
}

func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.value += value
	h.count++
}

func (h *Histogram) Get() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.count == 0 {
		return 0
	}
	return h.value / float64(h.count)
}

// HistogramVec represents a Prometheus Histogram with labels
type HistogramVec struct {
	name   string
	labels []string
	hists  map[string]*Histogram
	mu     sync.RWMutex
}

func NewHistogramVec(name string, labels []string) *HistogramVec {
	return &HistogramVec{
		name:   name,
		labels: labels,
		hists:  make(map[string]*Histogram),
	}
}

func (hv *HistogramVec) WithLabelValues(labelValues ...string) *Histogram {
	key := hv.buildKey(labelValues)
	hv.mu.Lock()
	defer hv.mu.Unlock()
	if hv.hists[key] == nil {
		hv.hists[key] = NewHistogram(hv.name + "{" + key + "}")
	}
	return hv.hists[key]
}

func (hv *HistogramVec) buildKey(labelValues []string) string {
	key := ""
	for i, label := range hv.labels {
		if i > 0 {
			key += ","
		}
		key += label + "=" + labelValues[i]
	}
	return key
}

// Timer represents a timing measurement
type Timer struct {
	histogram *Histogram
	start     time.Time
}

func NewTimer(histogram *Histogram) *Timer {
	return &Timer{
		histogram: histogram,
		start:     time.Now(),
	}
}

func (t *Timer) ObserveDuration() {
	if t.histogram != nil {
		t.histogram.Observe(time.Since(t.start).Seconds())
	}
}

// Helper functions for global metrics access
func RecordBatchProcessingTime(duration time.Duration) {
	GetMetrics().BatchProcessingTime.Observe(duration.Seconds())
}

func IncKafkaMessagesFailedParse() {
	GetMetrics().KafkaMessagesFailedParse.Inc()
}

func IncKafkaPollErrors() {
	GetMetrics().KafkaPollErrors.Inc()
}
