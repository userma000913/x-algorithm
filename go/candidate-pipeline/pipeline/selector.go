package pipeline

import "context"

// Selector 表示选择器接口
// Selector 在打分后执行，选择最终的候选
type Selector interface {
	// Select 选择候选列表
	// 根据分数排序并选择 Top-K 候选
	Select(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate
	
	// Name 返回 Selector 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 Selector 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
	
	// Score 从候选对象中提取分数用于排序
	Score(candidate *Candidate) float64
	
	// Sort 按分数降序排序候选列表
	Sort(candidates []*Candidate) []*Candidate
	
	// Size 返回要选择的候选数量（可选）
	// 如果不覆盖，默认不截断
	Size() *int
}
