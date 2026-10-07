package model

import (
	"encoding/json"
	"fmt"
	"time"

	"tutoring_server/pkg/common"

	"gorm.io/gorm"
)

// 订单状态：未开始（开始日期晚于今日）/ 进行中 / 已结束。
// 未开始由开始日期自动判定，仅用于展示与筛选；手动「已结束」后不再受自动状态影响。
const (
	OrderStatusPending  = "pending"  // 未开始
	OrderStatusRunning  = "running"  // 进行中
	OrderStatusFinished = "finished" // 已结束
)

// 计费方式：按小时（时薪 × 课时）、按单次课时（每节固定价）或按整期总费用（按计划总课时分摊到每节）。
const (
	BillModeHourly = "hourly"
	BillModeLesson = "lesson"
	BillModeTotal  = "total"
)

// 订单方向（收 / 支）：默认跟随注册时选择的使用身份，用户可主动调整。
// 仅决定费用口径与报表归属，不影响学员称呼、电话标签等使用身份相关文案。
const (
	DirectionIncome  = "income"  // 收入：我收的课
	DirectionExpense = "expense" // 开支：我付的课
)

// DirectionName 方向中文名。
var DirectionName = map[string]string{
	DirectionIncome:  "收入",
	DirectionExpense: "开支",
}

// DefaultDirection 按使用身份推导默认方向：家长 / 个人为开支，老师 / 机构为收入。
func DefaultDirection(userRole string) string {
	if userRole == UserRoleParent || userRole == UserRolePersonal {
		return DirectionExpense
	}
	return DirectionIncome
}

// BillModeName 计费方式中文名。
var BillModeName = map[string]string{
	BillModeHourly: "按小时（时薪）",
	BillModeLesson: "按单次课时",
	BillModeTotal:  "按总费用",
}

// BillModeUnit 计费方式对应的单位文案（前端单位下拉与列表展示共用）。
var BillModeUnit = map[string]string{
	BillModeHourly: "元/时",
	BillModeLesson: "元/次",
	BillModeTotal:  "总费用",
}

var OrderStatusName = map[string]string{
	OrderStatusPending:  "未开始",
	OrderStatusRunning:  "进行中",
	OrderStatusFinished: "已结束",
}

// WeeklySlot 每周固定时段，days 使用 1=周一 ... 7=周日。
// 例：{"days":[1,2,3,4],"start":"18:00","end":"20:00"} 表示周一到周四 18:00-20:00。
// EffectiveFrom / EffectiveTo 为规则版本化字段：
// 「从某日起改为周日」= 旧时段填 EffectiveTo，新时段填 EffectiveFrom，
// 历史课表按当时的规则重现，不会随规则修改而整体漂移。
type WeeklySlot struct {
	Days          []int  `json:"days"`
	Start         string `json:"start"`
	End           string `json:"end"`
	EffectiveFrom string `json:"effectiveFrom,omitempty"` // 生效起始日期 yyyy-mm-dd，空表示从一开始
	EffectiveTo   string `json:"effectiveTo,omitempty"`   // 生效截止日期 yyyy-mm-dd，空表示长期有效
}

// Active 判断该时段在指定日期是否生效。
// 边界为「含首尾」：EffectiveFrom 当天生效、EffectiveTo 当天仍生效，留空表示不限；
// day 需为自然日（建议用 common.StartOfDay 归一到本地零点），否则带时刻的 day 会在截止当天误判为不生效。
func (s WeeklySlot) Active(day time.Time) bool {
	if s.EffectiveFrom != "" {
		if t := common.ParseDate(s.EffectiveFrom); !t.IsZero() && day.Before(t) {
			return false
		}
	}
	if s.EffectiveTo != "" {
		if t := common.ParseDate(s.EffectiveTo); !t.IsZero() && day.After(t) {
			return false
		}
	}
	return true
}

// IntersectsPeriod 判断该时段的生效区间与订单周期 [from, to] 是否有交集（均为含首尾的自然日）。
// 用于订单保存时校验：无交集的时段永远不会生成课次，属误填。
func (s WeeklySlot) IntersectsPeriod(from, to time.Time) bool {
	if f := common.ParseDate(s.EffectiveFrom); !f.IsZero() && f.After(to) {
		return false // 时段从订单结束之后才开始
	}
	if t := common.ParseDate(s.EffectiveTo); !t.IsZero() && t.Before(from) {
		return false // 时段在订单开始之前就已结束
	}
	return true
}

type Order struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null" json:"userId"`
	Source string `gorm:"size:128;not null" json:"source"` // 信息来源
	// 发布时间：业务选填字段，用零值时间表示「未填写」，避免 NULL
	PublishedAt      time.Time `gorm:"not null" json:"publishedAt"`
	Publisher        string    `gorm:"size:64;not null" json:"publisher"`               // 发布人
	Grade            string    `gorm:"size:64;not null" json:"grade"`                   // 年级
	OrderNo          string    `gorm:"size:64;not null" json:"orderNo"`                 // 订单编号
	Address          string    `gorm:"size:255;not null" json:"address"`                // 地址
	Subject          string    `gorm:"size:64;not null" json:"subject"`                 // 科目
	Content          string    `gorm:"type:text;not null" json:"content"`               // 辅导内容
	StudentName      string    `gorm:"size:64;not null" json:"studentName"`             // 学生姓名
	StartDate        int64     `gorm:"not null" json:"-"`                               // 补课开始日期（秒级时间戳，当天 00:00）
	EndDate          int64     `gorm:"not null" json:"-"`                               // 补课结束日期（秒级时间戳，当天 00:00）；0 表示长期
	TotalLessons     int       `gorm:"not null;default:0" json:"totalLessons"`          // 计划总课时（节）：按课时购买时填写；0 表示未填，按补课周期推算
	WeeklySlots      string    `gorm:"type:text;not null" json:"weeklySlots"`           // 每周时段 JSON 数组
	HourlyRate       float64   `gorm:"not null" json:"hourlyRate"`                      // 时薪（billMode=hourly 生效）
	BillMode         string    `gorm:"size:16;not null;default:hourly" json:"billMode"` // hourly=按小时 lesson=按单次课时 total=按整期总费用
	LessonPrice      float64   `gorm:"not null" json:"lessonPrice"`                     // 单次课时价格（billMode=lesson 生效）
	TotalAmount      float64   `gorm:"not null;default:0" json:"totalAmount"`           // 整期总费用（元，billMode=total 生效）：按计划总课时分摊到每节
	Direction        string    `gorm:"size:16;not null;default:''" json:"direction"`    // 方向：income=收入 expense=开支；空表示按注册身份推导默认值
	Subjects         string    `gorm:"type:text;not null" json:"subjects"`              // 科目（多选，JSON 数组）
	StudentSituation string    `gorm:"type:text;not null" json:"studentSituation"`      // 学生情况
	StudentGender    string    `gorm:"size:8;not null" json:"studentGender"`            // 学生性别
	ContactPhone     string    `gorm:"size:32;not null" json:"contactPhone"`            // 对方联系电话（收入=学员家长，开支=授课老师）
	Remark           string    `gorm:"type:text;not null" json:"remark"`                // 备注
	RemarkFlag       bool      `gorm:"not null" json:"remarkFlag"`                      // 特殊备注（课表红点标记）
	Status           string    `gorm:"size:16;not null;default:running" json:"status"`
	CreatedAt        int64     `gorm:"autoCreateTime;not null" json:"createdAt"` // 创建时间，秒级时间戳
	UpdatedAt        int64     `gorm:"autoUpdateTime;not null" json:"updatedAt"` // 更新时间，秒级时间戳
}

func (m Order) TableName() string { return "order" }

// SubjectsList 解析多选科目；为空时回退到单科目字段 subject，保证历史数据可用。
func (o *Order) SubjectsList() []string {
	if o.Subjects != "" {
		var list []string
		if err := json.Unmarshal([]byte(o.Subjects), &list); err == nil && len(list) > 0 {
			return list
		}
	}
	if o.Subject != "" {
		return []string{o.Subject}
	}
	return nil
}

// Slots 解析每周时段 JSON。
func (o *Order) Slots() []WeeklySlot {
	if o.WeeklySlots == "" {
		return nil
	}
	var slots []WeeklySlot
	if err := json.Unmarshal([]byte(o.WeeklySlots), &slots); err != nil {
		return nil
	}
	return slots
}

type OrderModel struct{}

func NewOrderModel() *OrderModel { return &OrderModel{} }

// ListAll 返回全部订单（用于每日课次物化任务遍历）。
func (m *OrderModel) ListAll() ([]Order, error) {
	var list []Order
	err := db.Order("id ASC").Find(&list).Error
	return list, err
}

type OrderQuery struct {
	UserID    uint
	Status    string
	Keyword   string
	Grade     string
	Subject   string
	Direction string // income / expense；空表示全部
	Page      int
	PageSize  int
}

func (m *OrderModel) List(q OrderQuery) ([]Order, int64, error) {
	tx := db.Model(&Order{}).Where("user_id = ?", q.UserID)
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	// 收支筛选：income / expense
	if q.Direction != "" {
		tx = tx.Where("direction = ?", q.Direction)
	}
	if q.Grade != "" {
		tx = tx.Where("grade = ?", q.Grade)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		// 模糊匹配：学生姓名 / 地址 / 备注 / 来源
		tx = tx.Where(
			"student_name ILIKE ? OR address ILIKE ? OR remark ILIKE ? OR source ILIKE ?",
			like, like, like, like,
		)
	}
	if q.Subject != "" {
		// 科目筛选：兼容单科目字段与多选科目（JSON 数组字符串，如 ["数学","英语"]）
		tx = tx.Where("subject = ? OR subjects ILIKE ?", q.Subject, `%"`+q.Subject+`"%`)
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

	// 进行中订单排在最上方，已结束订单在下方；同组内按创建时间倒序
	var list []Order
	err := tx.
		Order(gorm.Expr("CASE WHEN status = ? THEN 0 ELSE 1 END", OrderStatusRunning)).
		Order("created_at DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

func (m *OrderModel) Get(id, userID uint) (*Order, error) {
	var o Order
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ListInRange 查询与 [fromTs,toTs] 时间戳区间有交集的订单（用于生成课表）。
func (m *OrderModel) ListInRange(userID uint, fromTs, toTs int64, status string) ([]Order, error) {
	tx := db.Where("user_id = ?", userID).
		Where("start_date > 0 AND start_date <= ?", toTs).
		Where("end_date = 0 OR end_date >= ?", fromTs)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	var list []Order
	err := tx.Order("created_at DESC").Find(&list).Error
	return list, err
}

// ListByIDs 按 ID 批量查询（带用户隔离，用于把「本周有调课」的订单补充进课表）。
func (m *OrderModel) ListByIDs(userID uint, ids []uint) ([]Order, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []Order
	err := db.Where("user_id = ? AND id IN ?", userID, ids).Find(&list).Error
	return list, err
}

// ExistsOrderNo 判断订单编号是否已存在（自动生成编号时去重）。
func (m *OrderModel) ExistsOrderNo(userID uint, no string) bool {
	var n int64
	db.Model(&Order{}).Where("user_id = ? AND order_no = ?", userID, no).Count(&n)
	return n > 0
}

func (m *OrderModel) Create(o *Order) error { return db.Create(o).Error }

// CountByUser 统计用户的课程数量（试用期间限制课程数用）。
func (m *OrderModel) CountByUser(userID uint) (int64, error) {
	var n int64
	err := db.Model(&Order{}).Where("user_id = ?", userID).Count(&n).Error
	return n, err
}

func (m *OrderModel) Save(o *Order) error { return db.Save(o).Error }

func (m *OrderModel) Delete(id, userID uint) error {
	return db.Where("id = ? AND user_id = ?", id, userID).Delete(&Order{}).Error
}

// FinishExpired 自动结束：补课结束日期早于「今天零点」且仍为「进行中」的订单，自动置为「已结束」。
// 边界：结束日期当天订单仍为「进行中」（周期含首尾），次日 00:00 起才算已结束，
// 与课表「结束日期当天仍生成课次」的语义保持一致。
func (m *OrderModel) FinishExpired(userID uint) error {
	todayStart := common.StartOfDay(time.Now()).Unix()
	return db.Model(&Order{}).
		Where("user_id = ?", userID).
		Where("status = ?", OrderStatusRunning).
		Where("end_date > 0 AND end_date < ?", todayStart).
		Update("status", OrderStatusFinished).Error
}

// SyncOrderStatus 同步订单状态（每次进入列表/课表/仪表盘前调用，保证状态与日期一致）：
//  1. 已过结束日期的进行中订单 → 已结束（FinishExpired）；
//  2. 其余未结束订单按开始日期判定：开始日期晚于今日 → 未开始，否则 → 进行中。
//
// 已手动置为「已结束」的订单不再被自动改写；「未开始」是纯展示态，到达开始日期后自动转「进行中」。
// 物化历史课次只依赖开始/结束日期，状态切换不会删除已落库的课时（历史冻结、与订单解耦）。
func (m *OrderModel) SyncOrderStatus(userID uint) error {
	if err := m.FinishExpired(userID); err != nil {
		return err
	}
	todayStart := common.StartOfDay(time.Now()).Unix()
	// 未结束且开始日期晚于今日 → 未开始
	if err := db.Model(&Order{}).
		Where("user_id = ?", userID).
		Where("status <> ?", OrderStatusFinished).
		Where("start_date > ?", todayStart).
		Update("status", OrderStatusPending).Error; err != nil {
		return err
	}
	// 未结束且开始日期不晚于今日 → 进行中
	if err := db.Model(&Order{}).
		Where("user_id = ?", userID).
		Where("status <> ?", OrderStatusFinished).
		Where("start_date <= ?", todayStart).
		Update("status", OrderStatusRunning).Error; err != nil {
		return err
	}
	return nil
}

// MigrateOrderContactPhone 把历史列 parent_phone 改名为中性的 contact_phone。
// 该列在收入订单存学员家长电话、在开支订单存授课老师电话，「家长」命名已不准确。
// 必须在 AutoMigrate 之前执行：否则 AutoMigrate 会先建出空的 contact_phone 列，
// 老数据会留在旧列上；本函数同时兼容该情况（先搬值、再删旧列）。
func MigrateOrderContactPhone() error {
	if !db.Migrator().HasTable("order") {
		return nil
	}
	if !db.Migrator().HasColumn("order", "parent_phone") {
		return nil // 新库或已迁移完成
	}
	if !db.Migrator().HasColumn("order", "contact_phone") {
		return db.Exec(`ALTER TABLE "order" RENAME COLUMN parent_phone TO contact_phone`).Error
	}
	// 新列已被 AutoMigrate 建出（空列）：先把旧值搬过来，再删掉旧列
	if err := db.Exec(`UPDATE "order" SET contact_phone = parent_phone
	                    WHERE (contact_phone IS NULL OR contact_phone = '') AND parent_phone <> ''`).Error; err != nil {
		return err
	}
	return db.Exec(`ALTER TABLE "order" DROP COLUMN parent_phone`).Error
}

// MigrateOrderDateToTimestamp 把 start_date / end_date 由 'yyyy-mm-dd' 字符串列迁移为秒级时间戳（bigint）。
// 需在 AutoMigrate 之前执行；已迁移则跳过。
func MigrateOrderDateToTimestamp() error {
	var dataType string
	if err := db.Raw(
		`SELECT data_type FROM information_schema.columns WHERE table_name = 'order' AND column_name = 'start_date'`,
	).Scan(&dataType).Error; err != nil || dataType == "" {
		return nil // 表/列尚不存在，交由 AutoMigrate 建表
	}
	if dataType == "bigint" {
		return nil // 已迁移
	}
	for _, col := range []string{"start_date", "end_date"} {
		// 兼容两种存量文本：'yyyy-mm-dd' 日期串，或已是秒级时间戳的数字串
		// （列被回退成 varchar 后再次启动时会遇到后者，直接 to_date 会报 date out of range）。
		sql := fmt.Sprintf(
			`ALTER TABLE "order" ALTER COLUMN %q TYPE bigint USING CASE`+
				` WHEN %[1]q IS NULL OR btrim(%[1]q) = '' THEN 0`+
				` WHEN btrim(%[1]q) ~ '^[0-9]{1,18}$' THEN btrim(%[1]q)::bigint`+
				` ELSE EXTRACT(EPOCH FROM to_date(btrim(%[1]q), 'YYYY-MM-DD'))::bigint END`,
			col,
		)
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

// MigrateOrderStatus 历史订单状态收敛为「进行中 / 已结束」：待开始→进行中，已关闭→已结束。
// MigrateOrderStatus 历史订单状态收敛：早期数据使用 closed 表示已结束，统一为 finished。
// 注：早期 pending（待开始）已在此前多次启动中收敛为 running；现 pending 为「未开始」正式状态，
// 由 SyncOrderStatus 按开始日期维护，故此处不再对 pending 做任何转换。
func MigrateOrderStatus() error {
	return db.Model(&Order{}).Where("status = ?", "closed").Update("status", OrderStatusFinished).Error
}

// MigrateDropOrderDuration 删除订单表废弃的「单次时长」列：课时长现由各周时间段起止时间推算，
// 不再由订单级固定字段保存。存在才删，已删除则跳过（幂等）。
func MigrateDropOrderDuration() error {
	var cnt int
	if err := db.Raw(
		`SELECT 1 FROM information_schema.columns WHERE table_name = 'order' AND column_name = 'duration_minutes'`,
	).Scan(&cnt).Error; err != nil || cnt == 0 {
		return nil // 列不存在，已迁移或尚未建表
	}
	return db.Exec(`ALTER TABLE "order" DROP COLUMN duration_minutes`).Error
}
