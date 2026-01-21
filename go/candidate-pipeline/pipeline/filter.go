package pipeline

import "context"

// Filter 表示过滤器接口
// Filters 顺序执行，每个 filter 基于前一个 filter 的结果
type Filter interface {
	// Filter 过滤候选列表
	// 根据某些条件评估每个候选，返回保留的候选和被移除的候选
	Filter(ctx context.Context, query *Query, candidates []*Candidate) (*FilterResult, error)
	
	// Name 返回 Filter 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 Filter 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
}
