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
