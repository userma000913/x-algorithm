package pipeline

import "context"

// SideEffect 表示副作用接口
// Side Effects 异步执行，不阻塞主流程
type SideEffect interface {
	// Run 执行副作用操作
	// 例如：缓存请求信息、记录日志等
	Run(ctx context.Context, query *Query, candidates []*Candidate) error
	
	// Name 返回 SideEffect 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 SideEffect 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
}
