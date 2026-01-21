package pipeline

import "context"

// Scorer 表示打分器接口
// Scorers 顺序执行，每个 scorer 基于前一个 scorer 的结果
type Scorer interface {
	// Score 为候选列表打分
	// 执行异步操作，返回打分后的候选列表
	//
	// 重要：返回的切片必须与输入的候选数量相同且顺序一致
	// 不允许在 scorer 中删除候选，应该使用 filter 阶段
	Score(ctx context.Context, query *Query, candidates []*Candidate) ([]*Candidate, error)
	
	// Name 返回 Scorer 的名称（用于日志和监控）
	Name() string
	
	// Enable 决定这个 Scorer 是否应该为给定的查询执行
	// 默认返回 true，子类可以覆盖以实现条件执行
	Enable(query *Query) bool
	
	// Update 更新单个候选的打分字段
	// 只应该复制这个 scorer 负责的字段
	Update(candidate *Candidate, scored *Candidate)
	
	// UpdateAll 批量更新候选的打分字段
	// 默认实现遍历并调用 Update 方法
	UpdateAll(candidates []*Candidate, scored []*Candidate)
}

// DefaultScorerUpdateAll 提供 UpdateAll 的默认实现
func DefaultScorerUpdateAll(scorer Scorer, candidates []*Candidate, scored []*Candidate) {
	if len(candidates) != len(scored) {
		return
	}
	for i := 0; i < len(candidates); i++ {
		scorer.Update(candidates[i], scored[i])
	}
}
