package utils

import (
	"time"
)

// Twitter 雪花ID的时间戳部分从 2006-03-21 20:50:14 UTC 开始
// 这是 Twitter 的 epoch 时间
const twitterEpoch int64 = 1142974214000 // 2006-03-21 20:50:14 UTC 的毫秒时间戳

// DurationSinceCreation 从雪花ID提取创建时间，返回距离现在的时间
// 如果提取失败，返回 nil
func DurationSinceCreation(snowflakeID int64) *time.Duration {
	timestamp := CreationTime(snowflakeID)
	if timestamp == nil {
		return nil
	}
	
	now := time.Now()
	duration := now.Sub(*timestamp)
	return &duration
}

// CreationTime 从雪花ID提取创建时间
// 雪花ID的结构：41位时间戳 + 10位机器ID + 12位序列号
// 时间戳部分在最高41位
func CreationTime(snowflakeID int64) *time.Time {
	if snowflakeID <= 0 {
		return nil
	}
	
	// 提取时间戳部分（右移22位，去掉机器ID和序列号）
	timestamp := (snowflakeID >> 22) + twitterEpoch
	
	// 转换为 time.Time
	creationTime := time.Unix(0, timestamp*int64(time.Millisecond))
	return &creationTime
}

// IsWithinAge 检查雪花ID对应的帖子是否在指定年龄内
func IsWithinAge(snowflakeID int64, maxAge time.Duration) bool {
	duration := DurationSinceCreation(snowflakeID)
	if duration == nil {
		return false
	}
	return *duration <= maxAge
}
