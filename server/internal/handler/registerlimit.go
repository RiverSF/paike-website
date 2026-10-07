package handler

import (
	"sync"
	"time"
)

// 注册防刷：在同一客户端 IP 维度做「双窗口」限流。
//
// 背景：注册接口无短信验证码、无邀请码，是刷号的主要入口。这里用频率限制把脚本批量注册的
// 成本显著抬高，同时不影响正常用户（正常用户一个人只会注册一次）。
//
//   - 短窗口：同一 IP 在 registerBurstWindow 内最多 registerBurstMax 次注册请求（拦脚本连续打）；
//   - 长窗口：同一 IP 在 registerDailyWindow 内最多 registerDailyMax 次注册请求（拦长期低频刷号）。
//
// 设计取舍：
//   - 只统计「请求次数」而非「成功注册数」：失败的请求同样计入，避免被拿去当手机号/用户名的探测工具；
//   - 触发时不回显剩余次数，只给等待时间，防止被用于探测阈值；
//   - 客户端 IP 由 middleware 侧的 SetTrustedProxies 配置保证不可伪造（见 router.New）。
const (
	registerBurstWindow = 10 * time.Minute
	registerBurstMax    = 3
	registerDailyWindow = 24 * time.Hour
	registerDailyMax    = 10
	// 记录上限：IP 数量超过该值时整体清理一轮，防止 map 无限增长（内存保护）
	registerLimitMaxIPs = 20000
)

// registerLimiter 进程内 IP 请求记录（每个 IP 一份按时间升序的请求时间戳）。
var registerLimiter = struct {
	sync.Mutex
	hits map[string][]time.Time
}{hits: make(map[string][]time.Time)}

// registerLimitCheck 记录一次注册请求并判断是否超限；limited=true 时 retryAfter 为建议等待时长。
func registerLimitCheck(ip string, now time.Time) (retryAfter time.Duration, limited bool) {
	if ip == "" {
		return 0, false
	}
	registerLimiter.Lock()
	defer registerLimiter.Unlock()

	// 只保留长窗口内的记录：同时覆盖短窗口判定
	hits := pruneHits(registerLimiter.hits[ip], now.Add(-registerDailyWindow))
	if d, exceeded := windowExceeded(hits, now, registerBurstWindow, registerBurstMax); exceeded {
		return d, true
	}
	if d, exceeded := windowExceeded(hits, now, registerDailyWindow, registerDailyMax); exceeded {
		return d, true
	}
	registerLimiter.hits[ip] = append(hits, now)
	// 内存保护：IP 记录过多时清理掉已全部滑出窗口的条目
	if len(registerLimiter.hits) > registerLimitMaxIPs {
		registerLimiter.hits = pruneLimiter(registerLimiter.hits, now.Add(-registerDailyWindow))
	}
	return 0, false
}
