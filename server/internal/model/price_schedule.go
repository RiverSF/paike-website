package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"tutoring_server/pkg/logger"
)

// 价格卡片标识：每个卡片独立设置折扣 / 价格，并独立生效与通知。
const (
	PriceCardPro      = "pro"      // 专职老师套餐价格
	PriceCardStudent  = "student"  // 大学生家教 / 学生折扣
	PriceCardRenewal  = "renewal"  // 平季续费优惠
	PriceCardActivity = "activity" // 限时活动
)

// PriceSchedule 价格调整预约：管理员按卡片设置新值，并指定立即生效或未来某时刻生效。
// 到达生效时间后由后台调度器写入 price_config 并给已登录用户发送对应卡片的站内信（仅调整的卡片通知）。
type PriceSchedule struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Card        string `gorm:"size:16;not null;index" json:"card"`             // 卡片标识
	Payload     string `gorm:"type:text;not null" json:"payload"`              // 卡片字段 JSON（与 price_card_input 对应）
	EffectiveAt int64  `gorm:"not null;index" json:"effectiveAt"`              // 生效时间（秒级）；0 表示立即生效
	Status      string `gorm:"size:16;not null;default:pending" json:"status"` // pending | applied
	CreatedAt   int64  `gorm:"autoCreateTime;not null" json:"createdAt"`
}

func (PriceSchedule) TableName() string { return "price_schedule" }

// PriceCardInput 价格卡片输入（与前端逐卡片保存对应）。
type PriceCardInput struct {
	Card               string  `json:"card"`
	EffectiveAt        int64   `json:"effectiveAt"` // 0 = 立即生效
	ProMonthlyAmount   float64 `json:"proMonthlyAmount"`
	ProQuarterlyAmount float64 `json:"proQuarterlyAmount"`
	ProYearlyAmount    float64 `json:"proYearlyAmount"`
	StudentDiscount    float64 `json:"studentDiscount"`
	RenewalDiscount    float64 `json:"renewalDiscount"`
	RenewalEnabled     bool    `json:"renewalEnabled"`
	ActivityName       string  `json:"activityName"`
	ActivityDiscount   float64 `json:"activityDiscount"`
	ActivityEnabled    bool    `json:"activityEnabled"`
}

// MarshalPayload 将输入序列化为 JSON 存储。
func (in PriceCardInput) MarshalPayload() (string, error) {
	b, err := json.Marshal(in)
	return string(b), err
}

// priceScheduleModel 价格预约数据访问。
type priceScheduleModel struct{}

func NewPriceScheduleModel() *priceScheduleModel { return &priceScheduleModel{} }

// Create 写入一条价格预约（pending）。
func (m *priceScheduleModel) Create(s *PriceSchedule) error {
	return db.Create(s).Error
}

// PendingDue 返回已到生效时间（effectiveAt<=now）且未应用的预约，按时间升序。
func (m *priceScheduleModel) PendingDue(now int64) ([]PriceSchedule, error) {
	var list []PriceSchedule
	err := db.Where("status = ? AND effective_at <= ?", "pending", now).
		Order("effective_at ASC, id ASC").Find(&list).Error
	return list, err
}

// PendingByCard 返回某卡片尚未生效的预约（用于前端展示「待生效」）。
func (m *priceScheduleModel) PendingByCard(card string) ([]PriceSchedule, error) {
	var list []PriceSchedule
	err := db.Where("status = ? AND card = ?", "pending", card).
		Order("effective_at ASC, id ASC").Find(&list).Error
	return list, err
}

// PendingAll 返回全部尚未生效的预约（用于前端展示「待生效」）。
func (m *priceScheduleModel) PendingAll() ([]PriceSchedule, error) {
	var list []PriceSchedule
	err := db.Where("status = ?", "pending").
		Order("effective_at ASC, id ASC").Find(&list).Error
	return list, err
}

// PendingUpcoming 返回未来（effective_at > now）且未应用的预约，按生效时间升序；用于首页提前预告。
func (m *priceScheduleModel) PendingUpcoming(now int64) ([]PriceSchedule, error) {
	var list []PriceSchedule
	err := db.Where("status = ? AND effective_at > ?", "pending", now).
		Order("effective_at ASC, id ASC").Find(&list).Error
	return list, err
}

// MarkApplied 标记预约已应用。
func (m *priceScheduleModel) MarkApplied(id uint) error {
	return db.Model(&PriceSchedule{}).Where("id = ?", id).Update("status", "applied").Error
}

// Cancel 撤销一条尚未生效的预约；返回是否确实撤销了待生效记录（已生效 / 不存在返回 false）。
func (m *priceScheduleModel) Cancel(id uint) (bool, error) {
	res := db.Where("id = ? AND status = ?", id, "pending").Delete(&PriceSchedule{})
	return res.RowsAffected > 0, res.Error
}

// ApplyCard 将某卡片的预约值合并进当前价格配置（仅覆盖该卡片相关字段），返回保存后的配置。
func (m *priceScheduleModel) ApplyCard(in PriceCardInput) (*PriceConfig, error) {
	p := &PriceConfig{}
	if err := db.Where("id = ?", 1).First(p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			def := defaultPriceConfig()
			p = &def
			p.ID = 1
		} else {
			return nil, err
		}
	}
	switch in.Card {
	case PriceCardPro:
		if in.ProMonthlyAmount >= 0 {
			p.ProMonthlyAmount = in.ProMonthlyAmount
		}
		if in.ProQuarterlyAmount >= 0 {
			p.ProQuarterlyAmount = in.ProQuarterlyAmount
		}
		if in.ProYearlyAmount >= 0 {
			p.ProYearlyAmount = in.ProYearlyAmount
		}
	case PriceCardStudent:
		if in.StudentDiscount > 0 {
			p.StudentDiscount = in.StudentDiscount
		}
	case PriceCardRenewal:
		p.RenewalDiscount = in.RenewalDiscount
		p.RenewalEnabled = in.RenewalEnabled
	case PriceCardActivity:
		p.ActivityName = in.ActivityName
		p.ActivityDiscount = in.ActivityDiscount
		p.ActivityEnabled = in.ActivityEnabled
	}
	if err := NewPriceModel().Save(p); err != nil {
		return nil, err
	}
	return p, nil
}

// CardSnapshot 取某卡片在当前价格配置中的值快照（导出供公开预告接口对比新旧值）。
func CardSnapshot(p *PriceConfig, card string) PriceCardInput { return cardSnapshot(p, card) }

// PriceScheduleMeaningful 判断一条未来生效的价卡预约是否「有意义」：
// 带启用开关的卡片（平季 / 活动），仅当「目标」或「当前」至少一方处于启用状态时才需要排期；
// 当前未启用且目标仍为关闭时，该预约不会产生任何实际价格变化，因此不应创建 / 生效 / 预告。
// 无启用开关的卡片（标准价 / 学生折扣）恒为有意义。
func PriceScheduleMeaningful(in PriceCardInput, cur *PriceConfig) bool {
	switch in.Card {
	case PriceCardRenewal:
		return in.RenewalEnabled || (cur != nil && cur.RenewalEnabled)
	case PriceCardActivity:
		return in.ActivityEnabled || (cur != nil && cur.ActivityEnabled)
	default:
		return true
	}
}

// cardSnapshot 取某卡片在当前价格配置中的值快照（用于价格变更历史的 Before / After）。
func cardSnapshot(p *PriceConfig, card string) PriceCardInput {
	in := PriceCardInput{Card: card}
	switch card {
	case PriceCardPro:
		in.ProMonthlyAmount = p.ProMonthlyAmount
		in.ProQuarterlyAmount = p.ProQuarterlyAmount
		in.ProYearlyAmount = p.ProYearlyAmount
	case PriceCardStudent:
		in.StudentDiscount = p.StudentDiscount
	case PriceCardRenewal:
		in.RenewalDiscount = p.RenewalDiscount
		in.RenewalEnabled = p.RenewalEnabled
	case PriceCardActivity:
		in.ActivityName = p.ActivityName
		in.ActivityDiscount = p.ActivityDiscount
		in.ActivityEnabled = p.ActivityEnabled
	}
	return in
}

// ApplyCardWithHistory 应用某卡片的新值并记录一条价格变更历史。
// Before = 应用前该卡片的快照，After = 应用后的快照；source 区分即时保存 / 预约到点 / 回滚。
// 历史写入失败只记日志，不阻断价格生效（价格本身已保存成功）。
func (m *priceScheduleModel) ApplyCardWithHistory(in PriceCardInput, source string, operatorID uint, operatorName string) (*PriceConfig, error) {
	cur, err := NewPriceModel().Get()
	if err != nil {
		return nil, err
	}
	before := cardSnapshot(cur, in.Card)
	p, err := m.ApplyCard(in)
	if err != nil {
		return nil, err
	}
	after := cardSnapshot(p, in.Card)
	bj, err := json.Marshal(before)
	if err != nil {
		logger.Error("marshal price history before failed, err=%s", err.Error())
		return p, nil
	}
	aj, err := json.Marshal(after)
	if err != nil {
		logger.Error("marshal price history after failed, err=%s", err.Error())
		return p, nil
	}
	h := &PriceHistory{
		Card:         in.Card,
		Before:       string(bj),
		After:        string(aj),
		Source:       source,
		OperatorID:   operatorID,
		OperatorName: operatorName,
	}
	if err := NewPriceHistoryModel().Create(h); err != nil {
		logger.Error("save price history failed, card=%s err=%s", in.Card, err.Error())
	}
	return p, nil
}

// zheText 折扣小数转中文「X 折」展示，如 0.85 → "8.5 折"。
func zheText(d float64) string {
	return fmt.Sprintf("%.1f 折", math.Round(d*100)/10)
}

// NotifyPriceCard 价格卡片生效时给用户发送站内信：每个卡片对应一条通知，仅调整的卡片通知。
// 通知范围只由账号身份决定，与登录 / 在线状态无关：
// 学生折扣卡片仅通知大学生家教（teacher_type=student）账号；其余卡片通知全部注册账号。
// 未注册的游客没有账号，因此天然收不到站内信。
func NotifyPriceCard(in PriceCardInput) error {
	var title, content, msgType string
	toStudentsOnly := false
	switch in.Card {
	case PriceCardPro:
		title = "会员套餐价格调整"
		content = fmt.Sprintf(
			"会员套餐价格已更新：专职老师包月 ¥%.0f、包季 ¥%.0f、包年 ¥%.0f。新价格现已生效。",
			in.ProMonthlyAmount, in.ProQuarterlyAmount, in.ProYearlyAmount,
		)
		msgType = MessageTypeSystem
	case PriceCardStudent:
		title = "学生专属折扣更新"
		content = fmt.Sprintf(
			"大学生家教专属折扣已调整为 %s。该优惠仅大学生家教身份用户享受，学生价 = 专职标准价 × 折扣（向上取整）。",
			zheText(in.StudentDiscount),
		)
		msgType = MessageTypePromo
		toStudentsOnly = true
	case PriceCardRenewal:
		title = "平季续费优惠调整"
		if in.RenewalEnabled {
			content = fmt.Sprintf("开学平季续费优惠已开启，续费享 %s（寒暑假高峰维持原价）。", zheText(in.RenewalDiscount))
		} else {
			content = "平季续费优惠已关闭，续费按标准价计费。"
		}
		msgType = MessageTypePromo
	case PriceCardActivity:
		title = "限时活动通知"
		if in.ActivityEnabled {
			content = fmt.Sprintf("限时活动「%s」已上线，活动期间享 %s 优惠，可在首页套餐与支付时选用。", in.ActivityName, zheText(in.ActivityDiscount))
		} else {
			content = fmt.Sprintf("限时活动「%s」已结束，价格恢复标准计费。", in.ActivityName)
		}
		msgType = MessageTypePromo
	default:
		return nil
	}
	msg := &Message{
		SenderID:   0,
		SenderName: "系统",
		Title:      title,
		Content:    content,
		Type:       msgType,
		CreatedAt:  time.Now().Unix(),
	}
	if toStudentsOnly {
		ids, err := NewUserModel().StudentIDs()
		if err != nil {
			return err
		}
		return NewMessageModel().SendToUsers(msg, ids)
	}
	// 广播给全部已登录（注册）用户；未登录访客不提示
	return NewMessageModel().Send(msg, 0)
}
