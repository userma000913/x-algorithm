package utils

import (
	"hash/fnv"
	"math"

	"x-algorithm-go/candidate-pipeline/pipeline"
)

// BloomFilter 表示一个布隆过滤器
// 用于高效地检查元素是否可能存在于集合中
type BloomFilter struct {
	bits      []byte // 位数组
	numBits   uint64 // 位数
	numHashes uint32 // 哈希函数数量
}

// NewBloomFilter 从 BloomFilterEntry 创建 BloomFilter
// 根据实际实现，entry.Data 应该包含序列化的布隆过滤器数据
// 这里假设 entry.Data 包含位数组数据，以及可选的元数据
func NewBloomFilterFromEntry(entry pipeline.BloomFilterEntry) *BloomFilter {
	// 默认参数：假设从字节数据中解析
	// 如果 entry.Data 为空，创建一个空的布隆过滤器
	if len(entry.Data) == 0 {
		return nil
	}

	// 假设数据格式：前8字节是numHashes (uint32小端)，其余是位数组
	// 或者更简单的假设：data直接是位数组，使用默认的哈希函数数量
	// 这里使用默认值，实际应该根据序列化格式解析
	defaultNumHashes := uint32(7) // 常用的哈希函数数量
	numBits := uint64(len(entry.Data)) * 8

	return &BloomFilter{
		bits:      entry.Data,
		numBits:   numBits,
		numHashes: defaultNumHashes,
	}
}

// MayContain 检查元素是否可能存在于布隆过滤器中
// 返回 true 表示可能存在（可能是误报），false 表示肯定不存在
func (bf *BloomFilter) MayContain(postID int64) bool {
	if bf == nil || len(bf.bits) == 0 {
		return false
	}

	// 使用双哈希方法生成多个哈希值
	h1, h2 := bf.hash(postID)

	// 检查所有 k 个哈希位置是否都被设置
	for i := uint32(0); i < bf.numHashes; i++ {
		// 双哈希：hash_i = (h1 + i * h2) mod numBits
		pos := (h1 + uint64(i)*h2) % bf.numBits
		
		// 检查对应位的值
		byteIndex := pos / 8
		bitIndex := pos % 8
		
		if byteIndex >= uint64(len(bf.bits)) {
			return false
		}

		// 检查位是否被设置
		if (bf.bits[byteIndex] & (1 << bitIndex)) == 0 {
			return false // 如果任何一个位为0，肯定不存在
		}
	}

	return true // 所有位都为1，可能存在
}

// hash 对 postID 计算两个独立的哈希值
// 使用 FNV-1a 哈希算法，这是一种快速的非加密哈希算法
func (bf *BloomFilter) hash(postID int64) (h1 uint64, h2 uint64) {
	// 计算第一个哈希值
	hasher1 := fnv.New64a()
	hasher1.Write([]byte{byte(postID), byte(postID >> 8), byte(postID >> 16), byte(postID >> 24),
		byte(postID >> 32), byte(postID >> 40), byte(postID >> 48), byte(postID >> 56)})
	h1 = hasher1.Sum64()

	// 计算第二个哈希值（使用不同的种子）
	hasher2 := fnv.New64a()
	hasher2.Write([]byte{byte(postID >> 56), byte(postID >> 48), byte(postID >> 40), byte(postID >> 32),
		byte(postID >> 24), byte(postID >> 16), byte(postID >> 8), byte(postID)})
	hasher2.Write([]byte{0x42, 0x5A, 0x7E, 0x1C}) // 额外的种子
	h2 = hasher2.Sum64()

	// 确保哈希值在有效范围内
	h1 = h1 % bf.numBits
	h2 = h2 % bf.numBits

	return h1, h2
}

// CalculateOptimalParameters 根据预期元素数量和误报率计算最优参数
// n: 预期元素数量
// p: 期望的误报率 (0 < p < 1)
// 返回: (numBits, numHashes)
func CalculateOptimalParameters(n uint64, p float64) (uint64, uint32) {
	if n == 0 || p <= 0 || p >= 1 {
		// 使用默认值
		return 1024 * 8, 7 // 1KB, 7个哈希函数
	}

	// 最优位数: m = -n * ln(p) / (ln(2)^2)
	m := -float64(n) * math.Log(p) / (math.Log(2) * math.Log(2))
	numBits := uint64(math.Ceil(m))

	// 最优哈希函数数: k = (m/n) * ln(2)
	k := (m / float64(n)) * math.Log(2)
	numHashes := uint32(math.Ceil(k))

	// 确保合理的最小值
	if numBits < 64 {
		numBits = 64
	}
	if numHashes < 1 {
		numHashes = 1
	}
	if numHashes > 32 {
		numHashes = 32 // 限制最大哈希函数数量
	}

	return numBits, numHashes
}
