package pipeline

import "context"

// Hydrator 表示候选增强器接口
// Hydrators 并行执行，每个 hydrator 补充不同的数据
type Hydrator interface {
	// Hydrate 增强候选列表
	// 执行异步操作，返回增强后的候选列表
	// 
	// 重要：返回的切片必须与输入的候选数量相同且顺序一致
	// 不允许在 hydrator 中删除候选，应该使用 filter 阶段
	Hydrate(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
	
	// Name 返回 Hydrator 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 Hydrator 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
	
	// Update 更新单个候选的增强字段
	// 只应该复制这个 hydrator 负责的字段
	Update(candidate *Candidate, hydrated *Candidate)
	
	// UpdateAll 批量更新候选的增强字段
	// 默认实现遍历并调用 Update 方法
	UpdateAll(candidates []*Candidate, hydrated []*Candidate)
}

// DefaultUpdateAll 提供 UpdateAll 的默认实现
func DefaultUpdateAll(hydrator Hydrator, candidates []*Candidate, hydrated []*Candidate) {
	if len(candidates) != len(hydrated) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		hydrator.Update(candidates[i], hydrated[i])
	}
}
