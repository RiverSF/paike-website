package handler

import (
	"strings"
	"sync"
	"time"
)

// 登录失败限制：同一账号连续失败 loginMaxFails 次后，锁定 loginLockDuration，之后可重试。
const (
	loginMaxFails     = 5
	loginLockDuration = 30 * time.Minute
)

type loginAttempt struct {
	fails       int
	lockedUntil time.Time
}

// 进程内登录失败计数（单实例部署足够；重启后计数清零）。
var loginLimiter = struct {
	sync.Mutex
	attempts map[string]*loginAttempt
}{attempts: make(map[string]*loginAttempt)}

// loginKey 归一化账号标识（登录账号为手机号 / 邮箱）。
func loginKey(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}

// loginLockRemaining 查询账号是否处于锁定期；locked 为 true 时 remain 为剩余锁定时间。
func loginLockRemaining(key string) (remain time.Duration, locked bool) {
	loginLimiter.Lock()
	defer loginLimiter.Unlock()
	a := loginLimiter.attempts[key]
	if a == nil {
		return 0, false
	}
	if a.lockedUntil.After(time.Now()) {
		return time.Until(a.lockedUntil), true
	}
	// 锁定期已过，清除记录以允许重新尝试
	if !a.lockedUntil.IsZero() {
		delete(loginLimiter.attempts, key)
	}
	return 0, false
}

// loginRecordFail 记录一次失败：返回剩余可尝试次数 left；若达到上限则返回锁定时长 lockRemain。
func loginRecordFail(key string) (left int, lockRemain time.Duration) {
	loginLimiter.Lock()
	defer loginLimiter.Unlock()
	a := loginLimiter.attempts[key]
	if a == nil {
		a = &loginAttempt{}
		loginLimiter.attempts[key] = a
	}
	a.fails++
	if a.fails >= loginMaxFails {
		a.lockedUntil = time.Now().Add(loginLockDuration)
		return 0, loginLockDuration
	}
	return loginMaxFails - a.fails, 0
}

// loginClear 登录成功后清除失败记录。
func loginClear(key string) {
	loginLimiter.Lock()
	defer loginLimiter.Unlock()
	delete(loginLimiter.attempts, key)
}

// ---- 登录 IP 维度限流 ----
//
// 账号维度（上方）挡的是「盯住一个账号反复试密码」；IP 维度挡的是「同一来源用大量账号各试几次」的
// 横向撞库（这种打法每个账号都不触发账号锁定）。
// 额度刻意放宽（远大于账号维度的 5 次），避免公司 / 学校等共享出口的正常用户被误伤。
const (
	loginIPWindow   = 15 * time.Minute
	loginIPMaxFails = 30
	loginIPMaxKeys  = 20000
)

// loginIPLimiter 进程内 IP 登录失败记录（每个 IP 一份按时间升序的失败时间戳）。
var loginIPLimiter = struct {
	sync.Mutex
	fails map[string][]time.Time
}{fails: make(map[string][]time.Time)}

// loginIPBlocked 查询该 IP 是否已被限流；blocked=true 时 retry 为建议等待时长。
func loginIPBlocked(ip string, now time.Time) (retry time.Duration, blocked bool) {
	if ip == "" {
		return 0, false
	}
	loginIPLimiter.Lock()
	defer loginIPLimiter.Unlock()
	loginIPLimiter.fails[ip] = pruneHits(loginIPLimiter.fails[ip], now.Add(-loginIPWindow))
	return windowExceeded(loginIPLimiter.fails[ip], now, loginIPWindow, loginIPMaxFails)
}

// loginIPRecordFail 记录一次登录失败（密码错误 / 账号不存在）。
// 登录成功不清零该计数：否则攻击者可用自己的账号「重置」IP 计数，限流形同虚设。
func loginIPRecordFail(ip string, now time.Time) {
	if ip == "" {
		return
	}
	loginIPLimiter.Lock()
	defer loginIPLimiter.Unlock()
	loginIPLimiter.fails[ip] = append(pruneHits(loginIPLimiter.fails[ip], now.Add(-loginIPWindow)), now)
	if len(loginIPLimiter.fails) > loginIPMaxKeys {
		loginIPLimiter.fails = pruneLimiter(loginIPLimiter.fails, now.Add(-loginIPWindow))
	}
}
