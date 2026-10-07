package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// TrialNudge 试用期转化推送去重表：同一用户在同一试用阶段只推送一次站内信。
// 阶段（kind）：activate=1-3天引导排课 / value=第7天价值回顾 / preview=第20天预警 / urgency=28-30天限时优惠。
type TrialNudge struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"not null;uniqueIndex:uk_trial_nudge_user_kind" json:"userId"`
	Kind      string `gorm:"size:32;not null;uniqueIndex:uk_trial_nudge_user_kind" json:"kind"`
	CreatedAt int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
}

func (TrialNudge) TableName() string { return "trial_nudge" }

type TrialNudgeModel struct{}

func NewTrialNudgeModel() *TrialNudgeModel { return &TrialNudgeModel{} }

// Sent 该用户在该阶段是否已推送过。
func (m *TrialNudgeModel) Sent(userID uint, kind string) bool {
	var n int64
	err := db.Model(&TrialNudge{}).Where("user_id = ? AND kind = ?", userID, kind).Count(&n).Error
	return err == nil && n > 0
}

// Mark 记录一次推送（与发送站在同一调用链，失败仅意味着可能重复推送，可接受）。
func (m *TrialNudgeModel) Mark(userID uint, kind string) error {
	rec := &TrialNudge{UserID: userID, Kind: kind, CreatedAt: time.Now().Unix()}
	if err := db.Create(rec).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil
		}
		return err
	}
	return nil
}

// TrialingUsers 仍处于免费试用期的账号（member_type=trial 且未冻结）。
// 试用期内付费的账号 member_type 会被改为付费套餐，自动退出推送范围。
func (m *UserModel) TrialingUsers() ([]User, error) {
	var list []User
	err := db.Where("member_type = ? AND status <> ?", MemberTypeTrial, UserStatusFrozen).
		Order("id ASC").Find(&list).Error
	return list, err
}
