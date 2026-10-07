package model

import "time"

// Lesson 已物化的单次课次（历史快照）。
// 由「每日定时任务 + 订单保存兜底」按当时配置落库，一旦写入即冻结，
// 后续对订单配置的任意修改都不会影响历史课次，从而使课表与报表的历史部分与订单彻底解耦。
type Lesson struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	UserID          uint    `gorm:"not null" json:"userId"`
	OrderID         uint    `gorm:"not null" json:"orderId"`
	OrderNo         string  `gorm:"size:32;not null" json:"orderNo"`
	Date            string  `gorm:"type:varchar(10);not null" json:"date"`
	Weekday         int     `gorm:"not null" json:"weekday"`
	SlotIndex       int     `gorm:"not null" json:"slotIndex"`
	StartTime       string  `gorm:"size:8;not null" json:"startTime"`
	EndTime         string  `gorm:"size:8;not null" json:"endTime"`
	DurationMinutes int     `gorm:"not null" json:"durationMinutes"`
	Grade           string  `gorm:"size:16;not null" json:"grade"`
	StudentName     string  `gorm:"size:32;not null" json:"studentName"`
	Subject         string  `gorm:"size:32;not null" json:"subject"`
	Address         string  `gorm:"size:128;not null" json:"address"`
	Content         string  `gorm:"size:255;not null" json:"content"`
	Remark          string  `gorm:"size:255;not null" json:"remark"`
	RemarkFlag      bool    `gorm:"not null" json:"remarkFlag"`
	HourlyRate      float64 `gorm:"not null" json:"hourlyRate"`
	// Status：regular / canceled / movedOut / movedIn / time / extra
	Status string `gorm:"size:16;not null" json:"status"`
	// default:false 必需：存量库新增该 NOT NULL 列时，PG 要求带默认值否则报 contains null values
	Voided        bool      `gorm:"not null;default:false" json:"voided"` // 该历史课次已被删除（老师未上课等），仅作隐藏，不物理删除以免被再次物化
	Income        float64   `gorm:"not null" json:"income"`
	ExceptionID   uint      `gorm:"not null" json:"exceptionId"`
	OriginDate    string    `gorm:"size:10;not null" json:"originDate"`
	OriginWeekday int       `gorm:"not null" json:"originWeekday"`
	MovedToDate   string    `gorm:"size:10;not null" json:"movedToDate"`
	MovedToStart  string    `gorm:"size:8;not null" json:"movedToStart"`
	AdjustNote    string    `gorm:"size:255;not null" json:"adjustNote"`
	CreatedAt     time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"not null" json:"updatedAt"`
}

func (Lesson) TableName() string { return "lesson" }

type LessonModel struct{}

func NewLessonModel() *LessonModel { return &LessonModel{} }

// CountVoidedByOrders 批量统计各订单「已作废（老师未上课被删除）」的课次数量，
// 用于订单进度统计：作废课次不计入已上节数。
func (m *LessonModel) CountVoidedByOrders(userID uint, orderIDs []uint) (map[uint]int, error) {
	out := make(map[uint]int, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	type row struct {
		OrderID uint
		Total   int
	}
	var rows []row
	err := db.Model(&Lesson{}).
		Select("order_id, COUNT(*) AS total").
		Where("user_id = ? AND order_id IN ? AND voided = ?", userID, orderIDs, true).
		Group("order_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.OrderID] = r.Total
	}
	return out, nil
}

// ListByUserRange 读取某用户指定日期区间已物化的课次（用于课表 / 报表的历史部分）。
func (m *LessonModel) ListByUserRange(userID uint, from, to string) ([]Lesson, error) {
	var list []Lesson
	err := db.Where("user_id = ? AND date >= ? AND date <= ?", userID, from, to).
		Order("date ASC, start_time ASC").Find(&list).Error
	return list, err
}

// ListByOrderRange 读取某订单指定日期区间的课次（用于物化时去重）。
func (m *LessonModel) ListByOrderRange(orderID uint, from, to string) ([]Lesson, error) {
	var list []Lesson
	err := db.Where("order_id = ? AND date >= ? AND date <= ?", orderID, from, to).
		Find(&list).Error
	return list, err
}

// Create 写入一条课次（幂等由调用方去重保证）。
func (m *LessonModel) Create(l *Lesson) error { return db.Create(l).Error }

// Get 按 ID 查询课次（带用户隔离）。
func (m *LessonModel) Get(id, userID uint) (*Lesson, error) {
	var l Lesson
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// SetVoided 软删除/恢复某条已物化课次（老师未上课等场景）。voided=true 即从课表与报表隐藏，
// 但记录仍保留，定时物化去重时会识别为已存在而不会重新生成。
func (m *LessonModel) SetVoided(id, userID uint, voided bool) error {
	return db.Model(&Lesson{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("voided", voided).Error
}

// DeleteByException 物理删除某条例外生成的、未被用户删除的全部课次。
// 例外课次（改期/调入/加课）由展开逻辑按例外当前状态随时重建，不同于用户删除的原生课次
// （原生课次用 voided 软删除，避免被定时任务重新物化）。voided 行保留作为「已删除」墓碑，
// 防止重建时复活用户已删除的课次。
func (m *LessonModel) DeleteByException(orderID, exceptionID, userID uint) error {
	return db.Where("order_id = ? AND exception_id = ? AND user_id = ? AND voided = ?",
		orderID, exceptionID, userID, false).
		Delete(&Lesson{}).Error
}

// ListZeroIncome 返回课时费为 0 的全部课次（用于存量冻结快照修复：订单后补价格时历史课次仍显示 ¥0）。
func (m *LessonModel) ListZeroIncome() ([]Lesson, error) {
	var list []Lesson
	err := db.Where("income = 0").Find(&list).Error
	return list, err
}

// UpdateFee 回填课次的课时费与时薪快照（仅修复流程使用）。
func (m *LessonModel) UpdateFee(id uint, income, hourlyRate float64) error {
	return db.Model(&Lesson{}).Where("id = ?", id).
		Updates(map[string]interface{}{"income": income, "hourly_rate": hourlyRate}).Error
}
