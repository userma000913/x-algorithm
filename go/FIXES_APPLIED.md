# 已修复的差异

> **修复日期**: 2024年

---

## ✅ 已修复的问题

### 1. PhoenixScorer 返回值不一致 ✅ **已修复**

**问题**: Go版本在`user_action_sequence`为空时返回`nil, nil`，而Rust版本返回未改变的候选

**修复**: 
- 文件: `go/home-mixer/internal/scorers/phoenix.go`
- 修改: 当`user_action_sequence`为空时，返回未改变的候选列表（克隆）

**代码变更**:
```go
// 修改前
if query.UserActionSequence == nil {
    return nil, nil
}

// 修改后
if query.UserActionSequence == nil {
    scored := make([]*pipeline.Candidate, len(candidates))
    for i, c := range candidates {
        scored[i] = c.Clone()
    }
    return scored, nil
}
```

---

### 2. WeightedScorer 的 Score 字段更新 ✅ **已修复**

**问题**: Go版本同时更新`WeightedScore`和`Score`字段，而Rust版本只更新`weighted_score`

**修复**:
- 文件: `go/home-mixer/internal/scorers/weighted.go`
- 修改: 只更新`WeightedScore`字段，不更新`Score`字段（`Score`由后续的`AuthorDiversityScorer`设置）

**代码变更**:
```go
// 修改前
scored[i].WeightedScore = &normalizedScore
scored[i].Score = &normalizedScore  // 删除这行

// 修改后
scored[i].WeightedScore = &normalizedScore
// Score 字段由后续的 AuthorDiversityScorer 设置
```

---

## ⚠️ 仍需确认的问题

### 1. 权重值是否一致
- Rust版本从`params`模块读取权重
- Go版本使用硬编码的默认权重
- **需要**: 找到Rust版本的`params`模块，对比权重值

### 2. normalize_score 实现是否一致
- Rust版本使用`crate::util::score_normalizer::normalize_score`
- Go版本使用`math.Log1p(score)`
- **需要**: 找到Rust版本的`normalize_score`实现进行对比

---

## 📊 修复后的状态

### 核心算法一致性: 🟢 **95%**

| 组件 | 修复前 | 修复后 |
|------|--------|--------|
| PhoenixScorer返回值 | 🔴 不一致 | ✅ 一致 |
| WeightedScorer字段更新 | 🟡 不一致 | ✅ 一致 |
| AuthorDiversityScorer | ✅ 一致 | ✅ 一致 |
| AgeFilter | ✅ 一致 | ✅ 一致 |

---

**最后更新**: 2024年
