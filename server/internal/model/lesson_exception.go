package model

// 课程例外类型：只影响「某一次」课程，不改动订单的每周固定频次规则。
const (
	ExceptionTypeCancel = "cancel" // 本次停课（不计课时与收入）
	ExceptionTypeMove   = "move"   // 本次改期（原位置消失，新日期生成一节）
	ExceptionTypeTime   = "time"   // 本次仅调整起止时间
	ExceptionTypeExtra  = "extra"  // 临时加课（不关联任何固定时段）
)

// ExceptionTypeName 例外类型中文名。
var ExceptionTypeName = map[string]string{
	ExceptionTypeCancel: "停课",
	ExceptionTypeMove:   "改期",
	ExceptionTypeTime:   "调整时间",
	ExceptionTypeExtra:  "临时加课",
}

// ExtraSlotIndex 临时加课的 slot 下标（加课不关联固定时段）。
const ExtraSlotIndex = -1

// LessonException 课程例外：对订单按周期展开出来的某一节课做单次调整。
// 同一「原定课时」（order_id + source_date + slot_index）只允许一条例外；
// 重复调整在 history 路径上追加一步（保留完整调课路径），生效字段始终等于最后一步。
type LessonException struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	UserID          uint    `gorm:"not null" json:"userId"`
	OrderID         uint    `gorm:"not null" json:"orderId"`
	SourceDate      int64   `gorm:"not null" json:"sourceDate"`       // 原定上课日期（秒级时间戳，当天 00:00）
	SlotIndex       int     `gorm:"not null" json:"slotIndex"`        // 对应 weekly_slots 第几段；extra 为 -1
	Type            string  `gorm:"size:16;not null" json:"type"`     // cancel / move / time / extra（生效状态）
	NewDate         int64   `gorm:"not null" json:"newDate"`          // 调整后日期（秒级时间戳）；cancel 为 0
	NewStart        string  `gorm:"size:8;not null" json:"newStart"`  // 调整后开始 HH:MM
	NewEnd          string  `gorm:"size:8;not null" json:"newEnd"`    // 调整后结束 HH:MM
	DurationMinutes int     `gorm:"not null" json:"durationMinutes"`  // 本次课时长度（分钟）；0 表示按调整前后的起止时间推算
	Note            string  `gorm:"type:text;not null" json:"note"`   // 调整原因
	Income          float64 `gorm:"not null;default:0" json:"income"` // 本次课时费（元）；0 表示按订单标准计费
	// History 完整调课路径（JSON 数组，见 handler.adjustStep）：每次调整追加一步，
	// 撤销仅回退最后一步，路径清空后删除例外、恢复原课次。
	History   string `gorm:"type:text;not null;default:''" json:"-"`
	CreatedAt int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
	UpdatedAt int64  `gorm:"autoUpdateTime;not null" json:"updatedAt"`
}

func (LessonException) TableName() string { return "lesson_exception" }

type LessonExceptionModel struct{}

func NewLessonExceptionModel() *LessonExceptionModel { return &LessonExceptionModel{} }

// ListByOrder 查询某订单的全部例外（按原定日期倒序）。
func (m *LessonExceptionModel) ListByOrder(orderID, userID uint) ([]LessonException, error) {
	var list []LessonException
	err := db.Where("order_id = ? AND user_id = ?", orderID, userID).
		Order("source_date DESC, id DESC").Find(&list).Error
	return list, err
}

// ListByOrderIDs 批量查询多个订单的例外（用于订单列表统计进度，避免逐单查询）。
func (m *LessonExceptionModel) ListByOrderIDs(userID uint, orderIDs []uint) (map[uint][]LessonException, error) {
	out := make(map[uint][]LessonException, len(orderIDs))
	if len(orderIDs) == 0 {
		return out, nil
	}
	var list []LessonException
	if err := db.Where("user_id = ? AND order_id IN ?", userID, orderIDs).Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		out[list[i].OrderID] = append(out[list[i].OrderID], list[i])
	}
	return out, nil
}

// ListInRange 查询与 [fromTs,toTs] 有交集的例外：
// 原定日期落在区间内（本周被调走 / 停掉的课）或调整后日期落在区间内（调到本周的课）。
func (m *LessonExceptionModel) ListInRange(userID uint, fromTs, toTs int64) ([]LessonException, error) {
	var list []LessonException
	err := db.Where("user_id = ?", userID).
		Where("(source_date BETWEEN ? AND ?) OR (new_date > 0 AND new_date BETWEEN ? AND ?)", fromTs, toTs, fromTs, toTs).
		Order("id ASC").Find(&list).Error
	return list, err
}

// OrderIDsInRange 返回在区间内存在例外的订单 ID（用于把订单补充进课表查询）。
func (m *LessonExceptionModel) OrderIDsInRange(userID uint, fromTs, toTs int64) ([]uint, error) {
	var ids []uint
	err := db.Model(&LessonException{}).
		Where("user_id = ?", userID).
		Where("(source_date BETWEEN ? AND ?) OR (new_date > 0 AND new_date BETWEEN ? AND ?)", fromTs, toTs, fromTs, toTs).
		Distinct().Pluck("order_id", &ids).Error
	return ids, err
}

// FindSlot 查找某「原定课时」已有的例外（extra 不参与，允许重复加课）。
func (m *LessonExceptionModel) FindSlot(orderID, userID uint, sourceDate int64, slotIndex int) (*LessonException, error) {
	var e LessonException
	err := db.Where("order_id = ? AND user_id = ? AND source_date = ? AND slot_index = ? AND type <> ?",
		orderID, userID, sourceDate, slotIndex, ExceptionTypeExtra).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// Get 按 ID 查询（带用户隔离）。
func (m *LessonExceptionModel) Get(id, userID uint) (*LessonException, error) {
	var e LessonException
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

// Create 新增例外。
func (m *LessonExceptionModel) Create(e *LessonException) error { return db.Create(e).Error }

// Save 更新例外。
func (m *LessonExceptionModel) Save(e *LessonException) error { return db.Save(e).Error }

// Delete 撤销例外（恢复为周期规则生成的课程）。
func (m *LessonExceptionModel) Delete(id, userID uint) error {
	return db.Where("id = ? AND user_id = ?", id, userID).Delete(&LessonException{}).Error
}

// DeleteByOrder 删除某订单的全部例外（订单删除时清理）。
func (m *LessonExceptionModel) DeleteByOrder(orderID, userID uint) error {
	return db.Where("order_id = ? AND user_id = ?", orderID, userID).Delete(&LessonException{}).Error
}
