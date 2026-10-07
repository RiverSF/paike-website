package model

import (
	"fmt"
	"time"
)

// 站内信类型
const (
	MessageTypeSystem = "system" // 系统通知
	MessageTypePromo  = "promo"  // 优惠 / 活动通知
)

// Message 站内信：管理员 / 站长发送，面向指定用户或全员广播。
// 广播（user_id=0）会展开为每位用户的独立记录，以便各自维护已读状态。
type Message struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"not null" json:"userId"`   // 接收人；0 表示全员广播（展开后写入各自 user_id）
	SenderID   uint   `gorm:"not null" json:"senderId"` // 发送人（管理员 / 站长）
	SenderName string `gorm:"size:32;not null" json:"senderName"`
	Title      string `gorm:"size:128;not null" json:"title"`
	Content    string `gorm:"type:text;not null" json:"content"`
	Type       string `gorm:"size:16;not null;default:system" json:"type"`
	ReadAt     int64  `gorm:"not null;default:0" json:"readAt"` // 0 表示未读
	CreatedAt  int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
}

func (m Message) TableName() string { return "message" }

// Read 是否已读
func (m Message) Read() bool { return m.ReadAt > 0 }

type MessageModel struct{}

func NewMessageModel() *MessageModel { return &MessageModel{} }

// Send 发送站内信：userID>0 发给指定用户，userID=0 广播给全部用户。
func (m *MessageModel) Send(msg *Message, userID uint) error {
	if userID > 0 {
		msg.UserID = userID
		return db.Create(msg).Error
	}
	// 广播：展开为每位用户的独立记录
	var ids []uint
	if err := db.Model(&User{}).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	list := make([]Message, 0, len(ids))
	for _, id := range ids {
		cp := *msg
		cp.ID = 0
		cp.UserID = id
		list = append(list, cp)
	}
	return db.Create(&list).Error
}

// ListForUser 查询某用户的站内信（按时间倒序）。
func (m *MessageModel) ListForUser(userID uint, page, pageSize int) ([]Message, int64, error) {
	tx := db.Model(&Message{}).Where("user_id = ?", userID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var list []Message
	err := tx.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// UnreadCount 某用户未读站内信数量。
func (m *MessageModel) UnreadCount(userID uint) (int64, error) {
	var n int64
	err := db.Model(&Message{}).Where("user_id = ? AND read_at = 0", userID).Count(&n).Error
	return n, err
}

// MarkRead 标记已读（仅限本人消息）。
func (m *MessageModel) MarkRead(id, userID uint) error {
	return db.Model(&Message{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("read_at", time.Now().Unix()).Error
}

// AllIDs 返回全部用户 ID（用于广播展开）。
func (m *UserModel) AllIDs() ([]uint, error) {
	var ids []uint
	err := db.Model(&User{}).Pluck("id", &ids).Error
	return ids, err
}

// StudentIDs 返回学生身份（teacher_type=student）用户 ID，用于限定接收人群。
func (m *UserModel) StudentIDs() ([]uint, error) {
	var ids []uint
	err := db.Model(&User{}).Where("teacher_type = ?", TeacherTypeStudent).Pluck("id", &ids).Error
	return ids, err
}

// ActiveMemberIDs 返回当前会员有效的账号 ID（含永久会员），供定时提醒按账号扫描；
// 只按账号与会员有效期判断，与是否登录 / 在线无关；已冻结账号不参与。
func (m *UserModel) ActiveMemberIDs() ([]uint, error) {
	now := time.Now().Unix()
	var ids []uint
	err := db.Model(&User{}).
		Where("status <> ?", UserStatusFrozen).
		Where("member_type = ? OR member_expire > ?", MemberTypePermanent, now).
		Pluck("id", &ids).Error
	return ids, err
}

// PriceAudienceCount 价格调整站内信的预计接收人数：studentsOnly=true 时仅统计大学生家教身份账号，
// 否则统计全部注册账号。与登录 / 在线状态无关。
func (m *UserModel) PriceAudienceCount(studentsOnly bool) (int64, error) {
	tx := db.Model(&User{})
	if studentsOnly {
		tx = tx.Where("teacher_type = ?", TeacherTypeStudent)
	}
	var n int64
	err := tx.Count(&n).Error
	return n, err
}

// StaffIDs 返回全部管理账号（管理员 / 站长）的用户 ID；excludeID>0 时排除该用户。
func (m *UserModel) StaffIDs(excludeID uint) ([]uint, error) {
	var ids []uint
	tx := db.Model(&User{}).Where("role IN ?", []string{RoleAdmin, RoleOwner})
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	err := tx.Pluck("id", &ids).Error
	return ids, err
}

// SendToUsers 给指定用户列表展开发送站内信（各自独立记录，维护已读状态）。
func (m *MessageModel) SendToUsers(msg *Message, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}
	list := make([]Message, 0, len(userIDs))
	for _, id := range userIDs {
		cp := *msg
		cp.ID = 0
		cp.UserID = id
		list = append(list, cp)
	}
	return db.Create(&list).Error
}

// ExpiringStudents 返回学生身份在 days 天内即将到期、且尚未发送过提醒的用户。
func (m *UserModel) ExpiringStudents(days int) ([]User, error) {
	now := time.Now().Unix()
	upper := now + int64(days)*86400
	var list []User
	err := db.Where(
		"teacher_type = ? AND student_verified_at > 0 AND student_expire_at > ? AND student_expire_at <= ? AND student_expire_notified = ?",
		TeacherTypeStudent, now, upper, false,
	).Find(&list).Error
	return list, err
}

// NotifyExpiringStudents 扫描即将过期（默认 30 天）的学生身份，逐人发送 promo 站内信并标记已提醒。
// 返回成功发送条数；用于后台定时任务。
func NotifyExpiringStudents(notifyDays int) (int, error) {
	users, err := NewUserModel().ExpiringStudents(notifyDays)
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	sent := 0
	for _, u := range users {
		expireDate := time.Unix(u.StudentExpireAt, 0).Format("2006-01-02")
		msg := &Message{
			SenderID:   0,
			SenderName: "系统",
			Title:      "学生身份即将到期提醒",
			Content: fmt.Sprintf(
				"您的大学生家教身份将于 %s 到期，到期后享有一年缓冲期，缓冲期结束后将自动转为专职老师（按专职老师价格计费）。如需继续享受家教优惠，请提前联系管理员更新毕业时间并续期。",
				expireDate,
			),
			Type:      MessageTypePromo,
			CreatedAt: now,
		}
		if err := NewMessageModel().Send(msg, u.ID); err != nil {
			return sent, err
		}
		if err := db.Model(&User{}).Where("id = ?", u.ID).Update("student_expire_notified", true).Error; err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

// ExpiringMembers 返回会员有效期在 days 天内即将到期、且尚未发送过提醒的活跃用户。
func (m *UserModel) ExpiringMembers(days int) ([]User, error) {
	now := time.Now().Unix()
	upper := now + int64(days)*86400
	var list []User
	err := db.Where(
		"member_expire > ? AND member_expire <= ? AND member_expire_notified = ?",
		now, upper, false,
	).Find(&list).Error
	return list, err
}

// NotifyExpiringMembers 扫描会员有效期即将到期（默认 7 天）的用户，逐人发送 system 站内信提醒并标记已提醒。
func NotifyExpiringMembers(notifyDays int) (int, error) {
	users, err := NewUserModel().ExpiringMembers(notifyDays)
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	sent := 0
	for _, u := range users {
		leftDays := int((u.MemberExpire - now) / 86400)
		if leftDays < 1 {
			leftDays = 1
		}
		expireDate := time.Unix(u.MemberExpire, 0).Format("2006-01-02")
		msg := &Message{
			SenderID:   0,
			SenderName: "系统",
			Title:      "会员即将到期提醒",
			Content: fmt.Sprintf(
				"您的会员将于 %s 到期（剩余约 %d 天）。到期后已录入的订单和课表仍可查看，但新增课程、调课与费用统计将受限。如需继续使用，请尽快在「个人中心」续费包月或包年。",
				expireDate, leftDays,
			),
			Type:      MessageTypeSystem,
			CreatedAt: now,
		}
		if err := NewMessageModel().Send(msg, u.ID); err != nil {
			return sent, err
		}
		if err := db.Model(&User{}).Where("id = ?", u.ID).Update("member_expire_notified", true).Error; err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
