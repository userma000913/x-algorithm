# Rust vs Go 详细差异分析

> **深度检查日期**: 2024年  
> **状态**: 🔴 发现关键差异

---

## 🔴 关键问题（必须修复）

### 1. PhoenixScorer 返回值不一致 ⚠️ **严重**

**Rust版本** (`home-mixer/scorers/phoenix_scorer.rs:74-75`):
```rust
// Return candidates unchanged if no scoring could be done
Ok(candidates.to_vec())
```

**Go版本** (`go/home-mixer/internal/scorers/phoenix.go:86-88`):
```go
// 检查是否有 user_action_sequence
if query.UserActionSequence == nil {
    return nil, nil // 如果没有用户历史，返回空（不更新分数）
}
```

**问题**: 
- Rust版本：当没有`user_action_sequence`时，返回**未改变的候选列表**
- Go版本：当没有`user_action_sequence`时，返回`nil, nil`（**错误！**）

**影响**: 🔴 **严重** - 这会导致当用户没有历史时，Go版本返回空结果，而Rust版本返回原始候选

**修复**: Go版本应该返回未改变的候选列表，而不是`nil`

---

### 2. WeightedScorer 的 Score 字段更新不一致 ⚠️

**Rust版本** (`home-mixer/scorers/weighted_scorer.rs:24-26`):
```rust
PostCandidate {
    weighted_score: Some(normalized_weighted_score),
    ..Default::default()
}
```

**Rust版本的update方法** (`home-mixer/scorers/weighted_scorer.rs:34-36`):
```rust
fn update(&self, candidate: &mut PostCandidate, scored: PostCandidate) {
    candidate.weighted_score = scored.weighted_score;
}
```

**Go版本** (`go/home-mixer/internal/scorers/weighted.go:100-102`):
```go
scored[i].WeightedScore = &normalizedScore
// 同时更新最终分数（用于排序）
scored[i].Score = &normalizedScore
```

**问题**:
- Rust版本：**只更新**`weighted_score`字段
- Go版本：**同时更新**`WeightedScore`和`Score`字段

**影响**: 🟡 **中等** - 可能导致后续Scorer（如AuthorDiversityScorer）的行为不一致

**需要确认**: Rust版本中`score`字段是在哪个Scorer中设置的？

---

### 3. WeightedScorer 权重值不一致 ⚠️

**Rust版本**: 从`params`模块读取权重（实际值未知）

**Go版本**: 使用硬编码的默认权重：
```go
FavoriteWeight:         1.0,
ReplyWeight:            1.0,
RetweetWeight:          1.0,
PhotoExpandWeight:      0.5,
ClickWeight:            0.5,
// ... 等等
```

**问题**: 权重值可能不同

**影响**: 🟡 **中等** - 如果权重值不同，会导致排序结果不同

**建议**: 需要确认Rust版本的`params`模块中的实际权重值

---

### 4. normalize_score 实现可能不一致 ⚠️

**Rust版本**: 使用`crate::util::score_normalizer::normalize_score`（实现未找到）

**Go版本**: 使用`math.Log1p(score)`（对数变换）

**问题**: 无法确认Rust版本的`normalize_score`实现

**影响**: 🟡 **中等** - 如果实现不同，归一化结果会不同

**建议**: 需要找到Rust版本的`normalize_score`实现进行对比

---

## 🟡 次要差异（不影响核心功能）

### 1. AuthorDiversityScorer 默认参数

**Rust版本**: 从`params`模块读取`AUTHOR_DIVERSITY_DECAY`和`AUTHOR_DIVERSITY_FLOOR`

**Go版本**: 硬编码为`DecayFactor: 0.8, Floor: 0.5`

**影响**: 🟢 **低** - 如果参数值不同，衰减效果会略有不同

---

### 2. OONScorer 权重因子

**Rust版本**: 从`params::OON_WEIGHT_FACTOR`读取

**Go版本**: 硬编码为`0.8`

**影响**: 🟢 **低** - 如果因子不同，站外内容调整会不同

---

## ✅ 一致的部分

### 1. AuthorDiversityScorer 算法逻辑
- ✅ 排序逻辑一致
- ✅ 衰减计算一致
- ✅ 位置计数逻辑一致

### 2. AgeFilter 逻辑
- ✅ 雪花ID提取时间一致
- ✅ 年龄检查逻辑一致

### 3. Pipeline 执行流程
- ✅ 执行顺序一致
- ✅ 并行/顺序策略一致

---

## 📋 修复建议

### 高优先级修复

1. **修复 PhoenixScorer 返回值**
   ```go
   // 修改前
   if query.UserActionSequence == nil {
       return nil, nil
   }
   
   // 修改后
   if query.UserActionSequence == nil {
       // 返回未改变的候选（与Rust版本一致）
       scored := make([]*pipeline.Candidate, len(candidates))
       for i, c := range candidates {
           scored[i] = c.Clone()
       }
       return scored, nil
   }
   ```

2. **确认 WeightedScorer 的 Score 字段更新**
   - 检查Rust版本中`score`字段是在哪个Scorer中设置的
   - 如果只在AuthorDiversityScorer中设置，则Go版本不应该在WeightedScorer中设置

### 中优先级修复

3. **确认权重值**
   - 找到Rust版本的`params`模块
   - 对比权重值是否一致

4. **确认 normalize_score 实现**
   - 找到Rust版本的`normalize_score`实现
   - 对比实现是否一致

---

## 🔍 需要进一步检查的地方

1. ✅ Rust版本的`params`模块（权重值）
2. ✅ Rust版本的`normalize_score`实现
3. ✅ Rust版本中`score`字段的设置位置
4. ✅ Rust版本的`util::score_normalizer`模块

---

**最后更新**: 2024年
