package handler

import (
	"fmt"
	"time"
)

// 进程内限流的通用工具：注册（registerlimit.go）与登录 IP 维度（loginlimit.go）共用。
//
// 统一口径：
//   - 每个 key（注册按 IP、登录按 IP）保存一份按时间升序的命中时间戳切片；
//   - 查询时按窗口统计命中次数，达到上限即拒绝，并给出「最早一次命中滑出窗口」的等待时间；
//   - 记录随查询原地裁剪，避免长期运行内存增长。
//
// 注意：计数保存在进程内，仅适用于单实例部署（与原有 loginlimit.go 一致）；
// 多副本部署时应改为 Redis 等共享存储，否则实际额度会随副本数放大。

// windowExceeded 判断窗口内命中次数是否达到上限；
// 超限时返回「最早一次命中滑出窗口」所需的等待时间（至少 1 秒）。
func windowExceeded(hits []time.Time, now time.Time, window time.Duration, max int) (time.Duration, bool) {
	from := now.Add(-window)
	count := 0
	var earliest time.Time
	for _, t := range hits {
		if t.After(from) {
			if count == 0 {
				earliest = t
			}
			count++
		}
	}
	if count < max {
		return 0, false
	}
	retry := earliest.Add(window).Sub(now)
	if retry < time.Second {
		retry = time.Second
	}
	return retry, true
}

// pruneHits 丢弃 keepAfter 之前的命中记录（原地过滤，避免频繁分配）。
func pruneHits(hits []time.Time, keepAfter time.Time) []time.Time {
	out := hits[:0]
	for _, t := range hits {
		if t.After(keepAfter) {
			out = append(out, t)
		}
	}
	return out
}

// pruneLimiter 清理整体记录，只保留窗口内仍有命中的 key（内存保护）。
func pruneLimiter(m map[string][]time.Time, keepAfter time.Time) map[string][]time.Time {
	out := make(map[string][]time.Time, len(m))
	for key, hits := range m {
		if h := pruneHits(hits, keepAfter); len(h) > 0 {
			out[key] = h
		}
	}
	return out
}

// limitMessage 限流提示：不透露剩余次数或阈值，只告知大约何时可重试。
func limitMessage(action string, retry time.Duration) string {
	if retry >= time.Minute {
		minutes := int((retry + time.Minute - 1) / time.Minute)
		return fmt.Sprintf("%s过于频繁，请 %d 分钟后再试", action, minutes)
	}
	return fmt.Sprintf("%s过于频繁，请稍后再试", action)
}
