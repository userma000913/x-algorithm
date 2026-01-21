package pipeline

import "context"

// Source 表示候选源接口
// Sources 并行执行，从不同的数据源获取候选
type Source interface {
	// GetCandidates 获取候选列表
	// 根据查询条件从数据源中获取候选帖子
	GetCandidates(ctx context.Context, query *Query) ([]*Candidate, error)
	
	// Name 返回 Source 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 Source 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
}
