package model

import (
	"gorm.io/gorm"

	"tutoring_server/pkg/logger"
)

// PriceConfig 会员价格配置（单行，id=1），金额单位：元。
// 由站点拥有者 / 管理员在后台调整，前端首页与支付弹窗实时读取。
type PriceConfig struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// 专职老师（标准价），管理员配置
	ProMonthlyAmount   float64 `gorm:"type:decimal(10,2);not null;default:69" json:"proMonthlyAmount"`
	ProQuarterlyAmount float64 `gorm:"type:decimal(10,2);not null;default:175" json:"proQuarterlyAmount"`
	ProYearlyAmount    float64 `gorm:"type:decimal(10,2);not null;default:529" json:"proYearlyAmount"`
	// 大学生家教折扣（相对专职标准价），默认 0.85 即 8.5 折；学生价 = ceil(专职价 × 折扣)
	StudentDiscount float64 `gorm:"type:decimal(5,4);not null;default:0.85" json:"studentDiscount"`
	// 平季（开学）续费折扣，默认 0.9 即 9 折；寒暑假（高峰）续费原价
	RenewalDiscount float64 `gorm:"type:decimal(3,2);not null;default:0.9" json:"renewalDiscount"`
	// 平季优惠启用开关：启用后首页与续费才按平季折扣计价
	RenewalEnabled bool `gorm:"not null;default:true" json:"renewalEnabled"`
	// 限时活动：自定义活动名称与折扣，启用后叠加到续费 / 购买价上
	ActivityName     string  `gorm:"size:64;not null;default:''" json:"activityName"`
	ActivityDiscount float64 `gorm:"type:decimal(5,4);not null;default:1" json:"activityDiscount"`
	ActivityEnabled  bool    `gorm:"not null;default:false" json:"activityEnabled"`
	UpdatedAt        int64   `gorm:"autoUpdateTime;not null" json:"updatedAt"`
}

func (m PriceConfig) TableName() string { return "price_config" }

type PriceModel struct{}

func NewPriceModel() *PriceModel { return &PriceModel{} }

// Get 读取价格配置（id=1）；不存在则写入默认配置后返回。
// 关键约定：任何「兜底 / 迁移」得到的值都会**立即持久化**，保证「后台所见 = 库中所存 = 首页所示」，
// 否则库里为 0、接口用默认值渲染，会导致价格设置页与首页/支付弹窗口径漂移。
func (m *PriceModel) Get() (*PriceConfig, error) {
	var p PriceConfig
	err := db.First(&p, 1).Error
	if err == gorm.ErrRecordNotFound {
		p = defaultPriceConfig()
		if cerr := db.Create(&p).Error; cerr != nil {
			return nil, cerr
		}
		return &p, nil
	}
	if err != nil {
		return nil, err
	}
	def := defaultPriceConfig()
	changed := false
	// 折扣缺失（历史数据）按默认折扣补全
	if p.StudentDiscount <= 0 {
		p.StudentDiscount = def.StudentDiscount
		changed = true
	}
	// 专职标准价缺失（历史数据/异常写入 0）：补默认并落库
	if p.ProMonthlyAmount == 0 && p.ProYearlyAmount == 0 {
		p.ProMonthlyAmount = def.ProMonthlyAmount
		p.ProQuarterlyAmount = def.ProQuarterlyAmount
		p.ProYearlyAmount = def.ProYearlyAmount
		changed = true
	}
	// 活动折扣缺失（历史数据为 0）时归 1：1 表示不打折
	if p.ActivityDiscount <= 0 {
		p.ActivityDiscount = 1
		changed = true
	}
	// 旧版默认价一键迁移到新版分析价（79/225/790），不覆盖管理员自定义
	if isLegacyDefaultPrice(&p) {
		p.ProMonthlyAmount = def.ProMonthlyAmount
		p.ProQuarterlyAmount = def.ProQuarterlyAmount
		p.ProYearlyAmount = def.ProYearlyAmount
		changed = true
	}
	if changed {
		if serr := db.Save(&p).Error; serr != nil {
			logger.Error("repair price config failed, err=%s", serr.Error())
		}
	}
	return &p, nil
}

// isLegacyDefaultPrice 判断当前配置是否仍为历史默认价（未做过自定义），用于平滑迁移到新价。
func isLegacyDefaultPrice(p *PriceConfig) bool {
	// 历史默认价一：39/99/299；历史默认价二：69/175/529
	if p.ProMonthlyAmount == 39 && p.ProQuarterlyAmount == 99 && p.ProYearlyAmount == 299 {
		return true
	}
	if p.ProMonthlyAmount == 69 && p.ProQuarterlyAmount == 175 && p.ProYearlyAmount == 529 {
		return true
	}
	return false
}

// defaultPriceConfig 会员价格默认配置：专职老师包月 79 元，学生在此基础上享 8.5 折。
func defaultPriceConfig() PriceConfig {
	return PriceConfig{
		ID:                 1,
		ProMonthlyAmount:   79,
		ProQuarterlyAmount: 225,
		ProYearlyAmount:    790,
		StudentDiscount:    0.85,
		RenewalDiscount:    0.9,
		RenewalEnabled:     true,
		ActivityDiscount:   1,
	}
}

// Save 覆盖保存价格配置（固定 id=1）。
// 注意：这里不再对折扣做静默修正（旧代码把缺失的学生折扣改成 0.9，与默认 0.85 不一致，
// 会让「保存后设置页显示的值」与管理员填写值不符），取值合法性由 handler 校验保证。
func (m *PriceModel) Save(p *PriceConfig) error {
	p.ID = 1
	return db.Save(p).Error
}
