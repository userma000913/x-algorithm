package pipeline

import (
	"context"
	"log"
	"sync"
)

// CandidatePipeline 表示候选管道
// 协调整个推荐流程，包括查询增强、候选获取、增强、过滤、打分、选择等阶段
type CandidatePipeline struct {
	// 组件列表
	QueryHydrators        []QueryHydrator
	Sources               []Source
	Hydrators             []Hydrator
	Filters               []Filter
	Scorers               []Scorer
	Selector              Selector
	PostSelectionHydrators []Hydrator
	PostSelectionFilters   []Filter
	SideEffects           []SideEffect
	
	// 配置
	ResultSize            int // 最终返回的候选数量，0 表示不限制
}

// Execute 执行完整的管道流程
// 这是管道的主入口方法，协调各个阶段的执行
func (p *CandidatePipeline) Execute(ctx context.Context, query *Query) (*PipelineResult, error) {
	// 1) Query Hydration（并行）
	hydratedQuery := p.hydrateQuery(ctx, query)
	
	// 2) Candidate Sourcing（并行）
	candidates := p.fetchCandidates(ctx, hydratedQuery)
	
	// 3) Candidate Hydration（并行）
	hydratedCandidates := p.hydrateCandidates(ctx, hydratedQuery, candidates)
	
	// 4) Pre-Scoring Filtering（顺序）
	keptCandidates, filteredCandidates := p.filterCandidates(ctx, hydratedQuery, hydratedCandidates)
	
	// 5) Scoring（顺序）
	scoredCandidates := p.scoreCandidates(ctx, hydratedQuery, keptCandidates)
	
	// 6) Selection（排序/截断）
	selectedCandidates := p.selectCandidates(ctx, hydratedQuery, scoredCandidates)
	
	// 7) Post-Selection Hydration（并行）
	postHydrated := p.hydratePostSelection(ctx, hydratedQuery, selectedCandidates)
	
	// 8) Post-Selection Filtering（顺序）
	finalCandidates, postFiltered := p.filterPostSelection(ctx, hydratedQuery, postHydrated)
	filteredCandidates = append(filteredCandidates, postFiltered...)
	
	// 9) 截断到结果大小
	if p.ResultSize > 0 && len(finalCandidates) > p.ResultSize {
		finalCandidates = finalCandidates[:p.ResultSize]
	}
	
	// 10) Side Effects（异步，不阻塞主链路）
	// 使用 context.Background() 确保 side effects 不会因为主请求取消而中断
	go p.runSideEffects(context.Background(), hydratedQuery, finalCandidates)
	
	return &PipelineResult{
		RetrievedCandidates: hydratedCandidates,
		FilteredCandidates:  filteredCandidates,
		SelectedCandidates:  finalCandidates,
		Query:               hydratedQuery,
	}, nil
}

// hydrateQuery 并行执行所有 Query Hydrators，并合并结果到 query
func (p *CandidatePipeline) hydrateQuery(ctx context.Context, query *Query) *Query {
	hydrated := query.Clone()
	
	// 筛选启用的 hydrators
	hydrators := make([]QueryHydrator, 0, len(p.QueryHydrators))
	for _, h := range p.QueryHydrators {
		if h.Enable(query) {
			hydrators = append(hydrators, h)
		}
	}
	
	if len(hydrators) == 0 {
		return hydrated
	}
	
	// 并行执行
	type item struct {
		h QueryHydrator
		r *Query
		e error
	}
	ch := make(chan item, len(hydrators))
	var wg sync.WaitGroup
	
	for _, h := range hydrators {
		wg.Add(1)
		go func(hydrator QueryHydrator) {
			defer wg.Done()
			res, err := hydrator.Hydrate(ctx, query)
			ch <- item{h: hydrator, r: res, e: err}
		}(h)
	}
	
	wg.Wait()
	close(ch)
	
	// 合并结果
	for it := range ch {
		if it.e != nil {
			log.Printf("request_id=%s stage=QueryHydrator component=%s failed: %v",
				query.RequestID, it.h.Name(), it.e)
			continue
		}
		it.h.Update(hydrated, it.r)
	}
	
	return hydrated
}

// fetchCandidates 并行执行所有 Sources，并收集所有候选
func (p *CandidatePipeline) fetchCandidates(ctx context.Context, query *Query) []*Candidate {
	// 筛选启用的 sources
	sources := make([]Source, 0, len(p.Sources))
	for _, s := range p.Sources {
		if s.Enable(query) {
			sources = append(sources, s)
		}
	}
	
	if len(sources) == 0 {
		return []*Candidate{}
	}
	
	// 并行执行
	type item struct {
		s Source
		c []*Candidate
		e error
	}
	ch := make(chan item, len(sources))
	var wg sync.WaitGroup
	
	for _, s := range sources {
		wg.Add(1)
		go func(source Source) {
			defer wg.Done()
			cs, err := source.GetCandidates(ctx, query)
			ch <- item{s: source, c: cs, e: err}
		}(s)
	}
	
	wg.Wait()
	close(ch)
	
	// 收集结果
	var collected []*Candidate
	for it := range ch {
		if it.e != nil {
			log.Printf("request_id=%s stage=Source component=%s failed: %v",
				query.RequestID, it.s.Name(), it.e)
			continue
		}
		log.Printf("request_id=%s stage=Source component=%s fetched %d candidates",
			query.RequestID, it.s.Name(), len(it.c))
		collected = append(collected, it.c...)
	}
	
	return collected
}

// hydrateCandidates 并行执行所有 Hydrators，并合并结果到 candidates
func (p *CandidatePipeline) hydrateCandidates(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	return p.runHydrators(ctx, query, candidates, p.Hydrators, "Hydrator")
}

// hydratePostSelection 并行执行所有 Post-Selection Hydrators
func (p *CandidatePipeline) hydratePostSelection(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	return p.runHydrators(ctx, query, candidates, p.PostSelectionHydrators, "PostSelectionHydrator")
}

// runHydrators 执行 hydrators 的共享辅助方法
func (p *CandidatePipeline) runHydrators(
	ctx context.Context,
	query *Query,
	candidates []*Candidate,
	hydrators []Hydrator,
	stageName string,
) []*Candidate {
	// 筛选启用的 hydrators
	enabledHydrators := make([]Hydrator, 0, len(hydrators))
	for _, h := range hydrators {
		if h.Enable(query) {
			enabledHydrators = append(enabledHydrators, h)
		}
	}
	
	if len(enabledHydrators) == 0 {
		return candidates
	}
	
	expectedLen := len(candidates)
	
	// 并行执行
	type item struct {
		h Hydrator
		r []*Candidate
		e error
	}
	ch := make(chan item, len(enabledHydrators))
	var wg sync.WaitGroup
	
	for _, h := range enabledHydrators {
		wg.Add(1)
		go func(hydrator Hydrator) {
			defer wg.Done()
			hydrated, err := hydrator.Hydrate(ctx, query, candidates)
			ch <- item{h: hydrator, r: hydrated, e: err}
		}(h)
	}
	
	wg.Wait()
	close(ch)
	
	// 合并结果
	for it := range ch {
		if it.e != nil {
			log.Printf("request_id=%s stage=%s component=%s failed: %v",
				query.RequestID, stageName, it.h.Name(), it.e)
			continue
		}
		if len(it.r) != expectedLen {
			log.Printf("request_id=%s stage=%s component=%s skipped: length_mismatch expected=%d got=%d",
				query.RequestID, stageName, it.h.Name(), expectedLen, len(it.r))
			continue
		}
		// merge：逐个 candidate update
		for i := 0; i < expectedLen; i++ {
			it.h.Update(candidates[i], it.r[i])
		}
	}
	
	return candidates
}

// filterCandidates 顺序执行所有 Filters
func (p *CandidatePipeline) filterCandidates(ctx context.Context, query *Query, candidates []*Candidate) (kept []*Candidate, removed []*Candidate) {
	return p.runFilters(ctx, query, candidates, p.Filters, "Filter")
}

// filterPostSelection 顺序执行所有 Post-Selection Filters
func (p *CandidatePipeline) filterPostSelection(ctx context.Context, query *Query, candidates []*Candidate) (kept []*Candidate, removed []*Candidate) {
	return p.runFilters(ctx, query, candidates, p.PostSelectionFilters, "PostSelectionFilter")
}

// runFilters 执行 filters 的共享辅助方法
func (p *CandidatePipeline) runFilters(
	ctx context.Context,
	query *Query,
	candidates []*Candidate,
	filters []Filter,
	stageName string,
) (kept []*Candidate, removed []*Candidate) {
	kept = candidates
	removed = []*Candidate{}
	
	for _, f := range filters {
		if !f.Enable(query) {
			continue
		}
		
		// 备份，以防失败
		backup := make([]*Candidate, len(kept))
		for i, c := range kept {
			backup[i] = c.Clone()
		}
		
		res, err := f.Filter(ctx, query, kept)
		if err != nil {
			log.Printf("request_id=%s stage=%s component=%s failed: %v",
				query.RequestID, stageName, f.Name(), err)
			kept = backup // 恢复备份
			continue
		}
		
		kept = res.Kept
		removed = append(removed, res.Removed...)
	}
	
	log.Printf("request_id=%s stage=%s kept %d, removed %d",
		query.RequestID, stageName, len(kept), len(removed))
	
	return kept, removed
}

// scoreCandidates 顺序执行所有 Scorers
func (p *CandidatePipeline) scoreCandidates(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	expectedLen := len(candidates)
	
	for _, s := range p.Scorers {
		if !s.Enable(query) {
			continue
		}
		
		scored, err := s.Score(ctx, query, candidates)
		if err != nil {
			log.Printf("request_id=%s stage=Scorer component=%s failed: %v",
				query.RequestID, s.Name(), err)
			continue
		}
		
		if len(scored) != expectedLen {
			log.Printf("request_id=%s stage=Scorer component=%s skipped: length_mismatch expected=%d got=%d",
				query.RequestID, s.Name(), expectedLen, len(scored))
			continue
		}
		
		// 更新每个 candidate
		for i := 0; i < expectedLen; i++ {
			s.Update(candidates[i], scored[i])
		}
	}
	
	return candidates
}

// selectCandidates 执行 Selector 选择候选
func (p *CandidatePipeline) selectCandidates(ctx context.Context, query *Query, candidates []*Candidate) []*Candidate {
	if !p.Selector.Enable(query) {
		return candidates
	}
	return p.Selector.Select(ctx, query, candidates)
}

// runSideEffects 异步执行所有 Side Effects（不阻塞主链路）
func (p *CandidatePipeline) runSideEffects(ctx context.Context, query *Query, candidates []*Candidate) {
	if len(p.SideEffects) == 0 {
		return
	}
	
	// 在 goroutine 中执行，不阻塞
	go func() {
		for _, se := range p.SideEffects {
			if !se.Enable(query) {
				continue
			}
			// 异步执行，忽略错误（side effect 不应该影响主流程）
			_ = se.Run(ctx, query, candidates)
		}
	}()
}
