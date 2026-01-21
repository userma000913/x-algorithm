package pipeline

import "context"

// QueryHydrator 表示查询增强器接口
// QueryHydrators 并行执行，增强查询对象（添加用户上下文）
type QueryHydrator interface {
	// Hydrate 增强查询对象
	// 执行异步操作，返回增强后的查询对象
	Hydrate(ctx context.Context, query *Query) (*Query, error)
	
	// Name 返回 QueryHydrator 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 QueryHydrator 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
	
	// Update 更新查询对象的增强字段
	// 只应该复制这个 hydrator 负责的字段
	Update(query *Query, hydrated *Query)
}
