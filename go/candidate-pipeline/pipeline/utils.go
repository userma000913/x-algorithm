package pipeline

// 辅助函数：处理可能为 nil 的指针值

// PtrOrZero 返回指针的值，如果为 nil 则返回零值
func PtrOrZero[T ~int64 | ~uint64](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}

// FloatOrZero 返回 float64 指针的值，如果为 nil 则返回 0
func FloatOrZero(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// BoolOrFalse 返回 bool 指针的值，如果为 nil 则返回 false
func BoolOrFalse(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

// IntOrZero 返回 int32 指针的值，如果为 nil 则返回 0
func IntOrZero(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// ToU64Slice 将 []int64 转换为 []uint64
func ToU64Slice(xs []int64) []uint64 {
	out := make([]uint64, 0, len(xs))
	for _, x := range xs {
		out = append(out, uint64(x))
	}
	return out
}

// StringOrEmpty 返回 string 指针的值，如果为 nil 则返回空字符串
func StringOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
