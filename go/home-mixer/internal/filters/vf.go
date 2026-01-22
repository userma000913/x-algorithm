package filters

import (
	"context"
	"strings"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// VFFilter 移除可见性过滤（Visibility Filtering）标记为不可见的帖子
// 例如：已删除、垃圾内容、暴力内容等
type VFFilter struct{}

// NewVFFilter 创建新的 VFFilter 实例
func NewVFFilter() *VFFilter {
	return &VFFilter{}
}

// Filter 实现 Filter 接口
func (f *VFFilter) Filter(ctx context.Context, query *pipeline.Query, candidates []*pipeline.Candidate) (*pipeline.FilterResult, error) {
	var kept []*pipeline.Candidate
	var removed []*pipeline.Candidate

	for _, candidate := range candidates {
		// 检查 visibility_reason，如果有且表示应该移除，则移除
		if shouldDrop(candidate.VisibilityReason) {
			removed = append(removed, candidate)
		} else {
			kept = append(kept, candidate)
		}
	}

	return &pipeline.FilterResult{
		Kept:    kept,
		Removed: removed,
	}, nil
}

// shouldDrop 判断是否应该移除
// 简化实现：如果 visibility_reason 不为空且包含特定关键词，则移除
func shouldDrop(reason *string) bool {
	if reason == nil || *reason == "" {
		return false
	}
	
	reasonLower := strings.ToLower(*reason)
	
	// 检查是否包含表示应该移除的关键词
	dropKeywords := []string{
		"drop",
		"deleted",
		"spam",
		"violence",
		"gore",
		"blocked",
		"filtered",
	}
	
	for _, keyword := range dropKeywords {
		if strings.Contains(reasonLower, keyword) {
			return true
		}
	}
	
	return false
}

// Name 返回 Filter 名称
func (f *VFFilter) Name() string {
	return "VFFilter"
}

// Enable 决定是否启用（VFFilter 总是启用）
func (f *VFFilter) Enable(query *pipeline.Query) bool {
	return true
}
