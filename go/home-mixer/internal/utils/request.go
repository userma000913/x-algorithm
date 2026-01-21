package utils

import (
	"fmt"
	"time"
)

// GenerateRequestID 生成请求 ID
// 格式: timestamp-userID
func GenerateRequestID(userID int64) string {
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s-%d", timestamp, userID)
}
