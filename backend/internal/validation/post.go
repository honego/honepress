package validation

import (
	"fmt"
	"strings"
	"time"
)

// 文章发布时间格式
const DateLayout = "2006-01-02 15:04:05"

// 校验文章必填字段
func ValidateRequiredPostFields(title string, dateText string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("标题不能为空")
	}
	if strings.TrimSpace(dateText) == "" {
		return fmt.Errorf("发布时间不能为空")
	}
	if _, err := time.ParseInLocation(DateLayout, strings.TrimSpace(dateText), time.Local); err != nil {
		return fmt.Errorf("发布时间格式必须是 YYYY-MM-DD HH:mm:ss：%w", err)
	}
	return nil
}

// 解析文章发布时间
func ParsePostDate(dateText string) (time.Time, error) {
	parsedPostDate, err := time.ParseInLocation(DateLayout, strings.TrimSpace(dateText), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("发布时间格式必须是 YYYY-MM-DD HH:mm:ss：%w", err)
	}
	return parsedPostDate, nil
}
