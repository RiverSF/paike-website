package model

import (
	"time"

	"gorm.io/gorm/clause"
)

// LessonReminder 开课提醒去重记录：同一账号的同一节课（订单 + 日期 + 开始时间）只提醒一次。
// 用唯一索引兜底，保证定时任务重启 / 重复扫描 / 多实例并发时都不会重复推送站内信。
type LessonReminder struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"not null;uniqueIndex:idx_lesson_reminder_once,priority:1" json:"userId"`
	OrderID   uint   `gorm:"not null;uniqueIndex:idx_lesson_reminder_once,priority:2" json:"orderId"`
	Date      string `gorm:"size:10;not null;uniqueIndex:idx_lesson_reminder_once,priority:3" json:"date"`
	StartTime string `gorm:"size:8;not null;uniqueIndex:idx_lesson_reminder_once,priority:4" json:"startTime"`
	CreatedAt int64  `gorm:"autoCreateTime;not null;index" json:"createdAt"`
}

func (LessonReminder) TableName() string { return "lesson_reminder" }

type LessonReminderModel struct{}

func NewLessonReminderModel() *LessonReminderModel { return &LessonReminderModel{} }

// MarkIfNew 标记某节课已提醒，返回是否为首次（true = 之前未提醒过，调用方应发送站内信）。
// 依赖唯一索引做幂等：重复 / 并发调用时只有一个调用方能拿到 true。
func (m *LessonReminderModel) MarkIfNew(userID, orderID uint, date, startTime string) (bool, error) {
	r := &LessonReminder{
		UserID:    userID,
		OrderID:   orderID,
		Date:      date,
		StartTime: startTime,
		CreatedAt: time.Now().Unix(),
	}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(r)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// CleanupBefore 清理过期提醒记录（一般传 30 天前的时间戳），防止表无限增长。
func (m *LessonReminderModel) CleanupBefore(before int64) error {
	return db.Where("created_at < ?", before).Delete(&LessonReminder{}).Error
}
