package common

import (
	"fmt"
	"strings"
)

// fmtSscan 解析 "HH:MM" 形式的时钟字符串。
func fmtSscan(s string, h, m *int) error {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid clock: %s", s)
	}
	if _, err := fmt.Sscanf(parts[0], "%d", h); err != nil {
		return err
	}
	if _, err := fmt.Sscanf(parts[1], "%d", m); err != nil {
		return err
	}
	return nil
}
