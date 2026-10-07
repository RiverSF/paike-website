package model

// 反馈状态
const (
	FeedbackPending  = "pending"
	FeedbackResolved = "resolved"
)

var FeedbackStatusName = map[string]string{
	FeedbackPending:  "待处理",
	FeedbackResolved: "已回复",
}

type Feedback struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"not null" json:"userId"`
	Username  string `gorm:"size:64;not null" json:"username"`
	Content   string `gorm:"type:text;not null" json:"content"`
	Contact   string `gorm:"size:64;not null" json:"contact"`
	Reply     string `gorm:"type:text;not null" json:"reply"`
	Status    string `gorm:"size:16;not null;default:pending" json:"status"`
	CreatedAt int64  `gorm:"autoCreateTime;not null" json:"createdAt"` // 提交时间，秒级时间戳
	UpdatedAt int64  `gorm:"autoUpdateTime;not null" json:"updatedAt"` // 更新时间，秒级时间戳
}

func (m Feedback) TableName() string { return "feedback" }

type FeedbackModel struct{}

func NewFeedbackModel() *FeedbackModel { return &FeedbackModel{} }

func (m *FeedbackModel) List(page, pageSize int) ([]Feedback, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	if err := db.Model(&Feedback{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []Feedback
	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (m *FeedbackModel) ListByUser(userID uint) ([]Feedback, error) {
	var list []Feedback
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&list).Error
	return list, err
}

func (m *FeedbackModel) Create(f *Feedback) error { return db.Create(f).Error }

// FeedbackQuery 后台反馈查询条件。
type FeedbackQuery struct {
	Status         string
	OnlyNormalUser bool // 仅看普通会员提交的内容
	Page           int
	PageSize       int
}

// AdminList 后台反馈列表（可按状态过滤，默认只看普通会员提交）。
func (m *FeedbackModel) AdminList(q FeedbackQuery) ([]Feedback, int64, error) {
	tx := db.Model(&Feedback{})
	if q.OnlyNormalUser {
		tx = tx.Where("user_id IN (SELECT id FROM \"user\" WHERE role = ?)", RoleUser)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	var list []Feedback
	err := tx.Order("created_at DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// LastByUser 该用户最近一次反馈，用于提交频率限制。
func (m *FeedbackModel) LastByUser(userID uint) (*Feedback, error) {
	var f Feedback
	err := db.Where("user_id = ?", userID).Order("created_at DESC").First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}
