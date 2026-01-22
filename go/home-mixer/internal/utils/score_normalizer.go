package utils

import (
	"math"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// NormalizeScore 归一化加权分数
// 这是推荐系统中常用的分数归一化方法，使用对数变换来压缩分数范围
// 参考 Rust 版本的实现逻辑
func NormalizeScore(candidate *pipeline.Candidate, score float64) float64 {
	// 如果分数为0或负数，返回0
	if score <= 0.0 {
		return 0.0
	}

	// 使用对数变换来归一化分数
	// log1p(x) = log(1 + x) 对于小值更稳定，对于大值有压缩效果
	// 这样可以将分数映射到更合理的范围
	normalized := math.Log1p(score)

	// 可选：进一步缩放以适应预期的分数范围
	// 这里使用一个缩放因子，可以根据实际数据分布调整
	// 假设原始分数范围大致在 [0, 10] 左右，归一化后的范围应该在 [0, 2.4] 左右
	// 如果需要保持类似的尺度，可以应用缩放
	// scaled := normalized / 2.4 * 10.0

	// 或者使用另一种常见的归一化方法：tanh归一化
	// tanh可以将任何实数映射到 [-1, 1]，对于正数映射到 [0, 1]
	// normalized := math.Tanh(score)

	// 使用 sigmoid 函数也是一个选项，但它会将所有值映射到 (0, 1)
	// normalized := 1.0 / (1.0 + math.Exp(-score))

	// 这里使用 log1p，因为它在推荐系统中被广泛使用
	// 它能够：
	// 1. 压缩大值的影响
	// 2. 保持小值相对线性
	// 3. 避免数值溢出
	return normalized
}

// NormalizeScoreWithBounds 使用边界值归一化分数（min-max归一化）
// minScore: 分数的最小值
// maxScore: 分数的最大值
func NormalizeScoreWithBounds(score, minScore, maxScore float64) float64 {
	if maxScore <= minScore {
		return score // 如果范围无效，返回原始分数
	}

	// min-max归一化: (x - min) / (max - min)
	normalized := (score - minScore) / (maxScore - minScore)

	// 确保结果在 [0, 1] 范围内
	if normalized < 0.0 {
		return 0.0
	}
	if normalized > 1.0 {
		return 1.0
	}

	return normalized
}

// NormalizeScoreZScore Z-score归一化（标准化）
// mean: 分数的均值
// stdDev: 分数的标准差
func NormalizeScoreZScore(score, mean, stdDev float64) float64 {
	if stdDev <= 0.0 {
		return score // 如果标准差无效，返回原始分数
	}

	// Z-score: (x - mean) / stdDev
	return (score - mean) / stdDev
}
