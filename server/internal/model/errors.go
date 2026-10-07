package model

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

// IsNotFound 判断是否为记录不存在错误。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsDuplicateEntry 判断是否为 PostgreSQL 唯一约束冲突（错误码 23505，如用户名重复）。
func IsDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key")
}
