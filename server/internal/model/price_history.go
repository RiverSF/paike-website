package model

// PriceHistory 价格变更历史：记录每次价格卡片生效前后的值、操作人与来源，用于审计与一键回滚。
// Before / After 均为该卡片字段的 JSON 快照（结构同 PriceCardInput），便于回滚时直接回放。
type PriceHistory struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Card string `gorm:"size:16;not null;index" json:"card"`
	// 列名避开 SQL 保留字 before / after
	Before       string `gorm:"column:before_value;type:text;not null" json:"before"`
	After        string `gorm:"column:after_value;type:text;not null" json:"after"`
	Source       string `gorm:"size:16;not null" json:"source"` // immediate 立即保存 | schedule 预约到点 | rollback 回滚
	OperatorID   uint   `gorm:"not null;default:0" json:"operatorId"`
	OperatorName string `gorm:"size:32;not null;default:''" json:"operatorName"`
	CreatedAt    int64  `gorm:"autoCreateTime;not null;index" json:"createdAt"`
}

func (PriceHistory) TableName() string { return "price_history" }

type priceHistoryModel struct{}

func NewPriceHistoryModel() *priceHistoryModel { return &priceHistoryModel{} }

// Create 写入一条价格变更历史。
func (m *priceHistoryModel) Create(h *PriceHistory) error {
	return db.Create(h).Error
}

// List 返回最近的价格变更历史（按时间倒序），limit 默认 50，上限 200。
func (m *priceHistoryModel) List(limit int) ([]PriceHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var list []PriceHistory
	err := db.Order("id DESC").Limit(limit).Find(&list).Error
	return list, err
}

// GetByID 读取单条历史记录（回滚时使用）。
func (m *priceHistoryModel) GetByID(id uint) (*PriceHistory, error) {
	var h PriceHistory
	if err := db.First(&h, id).Error; err != nil {
		return nil, err
	}
	return &h, nil
}
