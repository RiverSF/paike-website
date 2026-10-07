package handler

import (
	"strings"
	"time"

	"tutoring_server/internal/model"
)

// renewInput 续费入参，后台开通与用户自助开通共用。
type renewInput struct {
	Amount       float64
	Period       string // daily=按天 monthly=一月 quarterly=一季 yearly=一年
	Days         int    // 仅 period=daily 时生效：增加的天数
	OperatorID   uint
	OperatorName string
	Source       string // admin=后台开通 self=自助开通
	Remark       string
}

// renewMember 统一处理会员续费：
// 1. 按「按天 / 一月 / 一季 / 一年」在原到期时间（已过期则从当前时间）上顺延，自动算出新的到期时间
// 2. 累加用户累计充值金额
// 3. 生成一条支付记录
func renewMember(u *model.User, in renewInput) (*model.Payment, string) {
	period := strings.ToLower(in.Period)
	switch period {
	case model.RenewPeriodDaily, model.RenewPeriodMonthly, model.RenewPeriodQuarterly, model.RenewPeriodYearly:
	default:
		return nil, "续费时长仅支持 daily（按天）、monthly（一月）、quarterly（一季）或 yearly（一年）"
	}
	if in.Amount < 0 {
		return nil, "支付金额不能为负数"
	}
	if period == model.RenewPeriodDaily && (in.Days <= 0 || in.Days > model.RenewDaysMax) {
		return nil, "按天续费请填写 1-3650 天"
	}

	now := time.Now().Unix()
	// 未过期则在原到期时间上顺延，已过期则从当前时间开始
	base := u.MemberExpire
	if base < now {
		base = now
	}

	var after int64
	switch period {
	case model.RenewPeriodDaily:
		u.MemberType = model.MemberTypeDaily
		after = time.Unix(base, 0).AddDate(0, 0, in.Days).Unix()
	case model.RenewPeriodMonthly:
		u.MemberType = model.MemberTypeMonthly
		after = time.Unix(base, 0).AddDate(0, 1, 0).Unix()
	case model.RenewPeriodQuarterly:
		u.MemberType = model.MemberTypeQuarterly
		after = time.Unix(base, 0).AddDate(0, 3, 0).Unix()
	default:
		u.MemberType = model.MemberTypeYearly
		after = time.Unix(base, 0).AddDate(1, 0, 0).Unix()
	}
	if u.MemberStart == 0 || u.MemberStart > now {
		u.MemberStart = now
	}

	before := u.MemberExpire
	u.MemberExpire = after
	u.MemberExpireNotified = false // 重置提醒标记，续费后新有效期可再次触发到期前站内信
	u.TotalPaid += in.Amount

	if err := model.NewUserModel().Save(u); err != nil {
		return nil, "保存会员信息失败"
	}

	// 永久会员不做时长顺延（后台不会对其续费），此处仅记录
	days := int((after - before) / 86400)
	record := &model.Payment{
		UserID:       u.ID,
		Username:     u.Username,
		Amount:       in.Amount,
		Period:       period,
		PeriodName:   model.RenewPeriodName[period],
		Days:         days,
		MemberType:   u.MemberType,
		BeforeExpire: before,
		AfterExpire:  after,
		OperatorID:   in.OperatorID,
		OperatorName: in.OperatorName,
		Source:       in.Source,
		Remark:       in.Remark,
	}
	if err := model.NewPaymentModel().Create(record); err != nil {
		return nil, "保存支付记录失败"
	}
	return record, ""
}
