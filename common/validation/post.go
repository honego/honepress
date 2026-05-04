package validation

import (
	"fmt"
	"strings"
	"time"
)

// DateLayout 是 Front Matter 中约定的发布时间格式，固定格式便于排序和 RSS 输出。
const DateLayout = "2006-01-02 15:04:05"

// ValidateRequiredPostFields 在保存和渲染前统一检查核心字段，避免生成半成品页面。
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

// ParsePostDate 使用本地时区解释文章时间，这样 Docker 部署时可以通过 TZ 控制 RSS 时间。
func ParsePostDate(dateText string) (time.Time, error) {
	parsedPostDate, err := time.ParseInLocation(DateLayout, strings.TrimSpace(dateText), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("发布时间格式必须是 YYYY-MM-DD HH:mm:ss：%w", err)
	}
	return parsedPostDate, nil
}
