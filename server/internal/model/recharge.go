package model

// 续费时长
const (
	RenewPeriodDaily     = "daily"     // 按天（自定义天数，用于补偿 / 短期开通）
	RenewPeriodMonthly   = "monthly"   // 一月
	RenewPeriodQuarterly = "quarterly" // 一季（三个月）
	RenewPeriodYearly    = "yearly"    // 一年
)

// 续费时长中文名
var RenewPeriodName = map[string]string{
	RenewPeriodDaily:     "按天",
	RenewPeriodMonthly:   "一月",
	RenewPeriodQuarterly: "一季",
	RenewPeriodYearly:    "一年",
}

// RenewDaysMax 按天续费的显示上限（约 10 年），防止误填极端值。
const RenewDaysMax = 3650

// Payment 会员支付 / 开通记录：每次续费（管理员操作或用户自助）都会留痕。
type Payment struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	UserID       uint    `gorm:"not null" json:"userId"`
	Username     string  `gorm:"size:64;not null" json:"username"`     // 账号快照（开通时用户名）
	Amount       float64 `gorm:"type:decimal(12,2);not null" json:"amount"` // 支付金额
	Period       string  `gorm:"size:16;not null" json:"period"`       // monthly / quarterly / yearly
	PeriodName   string  `gorm:"size:16;not null" json:"periodName"`   // 一月 / 一季 / 一年
	Days         int     `gorm:"not null" json:"days"`                 // 实际增加天数
	MemberType   string  `gorm:"size:16;not null" json:"memberType"`  // 开通后的会员类型
	BeforeExpire int64   `gorm:"not null" json:"beforeExpire"`         // 开通前到期时间（秒级时间戳）
	AfterExpire  int64   `gorm:"not null" json:"afterExpire"`          // 开通后到期时间（秒级时间戳）
	OperatorID   uint    `gorm:"not null" json:"operatorId"`           // 操作人，0 表示用户自助
	OperatorName string  `gorm:"size:64; not null" json:"operatorName"` // 操作人名称
	Source       string  `gorm:"size:16; not null" json:"source"`      // admin=后台开通 self=自助开通
	Remark       string  `gorm:"size:255; not null" json:"remark"`     // 备注
	CreatedAt    int64   `gorm:"autoCreateTime;not null" json:"createdAt"`
	UpdatedAt    int64   `gorm:"autoUpdateTime;not null" json:"updatedAt"`
}

// TableName 物理表名沿用 recharge（历史数据兼容，不额外迁移）
func (m Payment) TableName() string { return "recharge" }

type PaymentModel struct{}

func NewPaymentModel() *PaymentModel { return &PaymentModel{} }

func (m *PaymentModel) Create(r *Payment) error { return db.Create(r).Error }

// ListByUser 某用户的支付记录（按时间倒序）。
func (m *PaymentModel) ListByUser(userID uint, limit int) ([]Payment, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []Payment
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&list).Error
	return list, err
}

// LastByUser 最近一次支付记录。
func (m *PaymentModel) LastByUser(userID uint) (*Payment, error) {
	var r Payment
	err := db.Where("user_id = ?", userID).Order("created_at DESC").First(&r).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}
