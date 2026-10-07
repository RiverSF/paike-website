package common

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// GetEnv 读取环境变量，为空时返回默认值。
func GetEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// GetEnvInt 读取整型环境变量。
func GetEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

const DateLayout = "2006-01-02"

// ParseDate 解析 yyyy-mm-dd，失败返回零值。
func ParseDate(s string) time.Time {
	t, err := time.ParseInLocation(DateLayout, strings.TrimSpace(s), time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseClock 解析 HH:MM，失败返回 false。
func ParseClock(s string) (int, int, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return 0, 0, false
	}
	var h, m int
	if err := fmtSscan(s, &h, &m); err != nil {
		return 0, 0, false
	}
	if h < 0 || h > 24 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// StartOfDay 返回 t 当天 00:00（本地时区）。
// 全站日期边界统一使用「本地零点」：订单周期、时段生效区间、历史课次切分均按自然日比较，
// 避免时间戳携带时刻导致边界判定出现 1 天误差（例如 end_date 当天被判定为已过期）。
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// Today 返回今天的本地零点。
func Today() time.Time { return StartOfDay(time.Now()) }

// MondayOf 返回所在自然周的周一 00:00。
func MondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, t.Location())
}

// WeekdayISO 返回 1=周一 ... 7=周日。
func WeekdayISO(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// ContainsInt 判断切片是否包含指定整数。
func ContainsInt(list []int, v int) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// ClockToMinutes 把 HH:MM 转为分钟数。
func ClockToMinutes(clock string) int {
	h, m, ok := ParseClock(clock)
	if !ok {
		return 0
	}
	return h*60 + m
}
