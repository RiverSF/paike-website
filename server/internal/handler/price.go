package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// peakMonths 寒暑假（补课高峰）：1、2、7、8 月；其余为开学平季。
var peakMonths = map[time.Month]bool{1: true, 2: true, 7: true, 8: true}

// GetPrice 公开接口：返回当前会员价格配置，供首页套餐与支付弹窗展示。
func GetPrice(c *gin.Context) {
	p, err := model.NewPriceModel().Get()
	if err != nil {
		ServerError(c, "读取价格失败")
		return
	}
	// 平季（开学）续费打折，寒暑假（高峰）原价
	season := "offpeak"
	if peakMonths[time.Now().Month()] {
		season = "peak"
	}
	// 生效折扣 = 启用的优惠叠加：平季开关 × 活动开关（未启用 / 高峰期不加折扣）
	disc := 1.0
	renewalOn := p.RenewalEnabled && season == "offpeak" && p.RenewalDiscount > 0 && p.RenewalDiscount < 1
	if renewalOn {
		disc *= p.RenewalDiscount
	}
	activityOn := p.ActivityEnabled && p.ActivityDiscount > 0 && p.ActivityDiscount < 1
	if activityOn {
		disc *= p.ActivityDiscount
	}
	// 折后价统一以「专职老师标准价」为基础：专职 = 原价 × 平季折扣；学生 = 原价 × 平季折扣 × 学生折扣
	renewPro := func(v float64) float64 { return math.Ceil(v * disc) }
	renewStu := func(v float64) float64 { return math.Ceil(v * p.StudentDiscount * disc) }

	OK(c, gin.H{
		"proMonthlyAmount":   p.ProMonthlyAmount,
		"proQuarterlyAmount": p.ProQuarterlyAmount,
		"proYearlyAmount":    p.ProYearlyAmount,
		"studentDiscount":    p.StudentDiscount,
		"renewalDiscount":    p.RenewalDiscount,
		"renewalEnabled":     p.RenewalEnabled,
		"season":             season,
		"activityEnabled":    p.ActivityEnabled,
		"activityName":       p.ActivityName,
		"activityDiscount":   p.ActivityDiscount,
		// 学生单享学生折扣后的价格（不含平季折扣），供参考
		"studentMonthlyAmount":   math.Ceil(p.ProMonthlyAmount * p.StudentDiscount),
		"studentQuarterlyAmount": math.Ceil(p.ProQuarterlyAmount * p.StudentDiscount),
		"studentYearlyAmount":    math.Ceil(p.ProYearlyAmount * p.StudentDiscount),
		"proMonthlyRenew":        renewPro(p.ProMonthlyAmount),
		"proQuarterlyRenew":      renewPro(p.ProQuarterlyAmount),
		"proYearlyRenew":         renewPro(p.ProYearlyAmount),
		"studentMonthlyRenew":    renewStu(p.ProMonthlyAmount),
		"studentQuarterlyRenew":  renewStu(p.ProQuarterlyAmount),
		"studentYearlyRenew":     renewStu(p.ProYearlyAmount),
	})
}

// roundCent 四舍五入到分
func roundCent(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

type priceUpdateInput struct {
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

// AdminUpdatePrice 管理员调整会员价格（仅站点拥有者 / 管理员可调用）。
func AdminUpdatePrice(c *gin.Context) {
	var in priceUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, CodeBadRequest, "参数错误")
		return
	}
	if in.ProMonthlyAmount < 0 || in.ProQuarterlyAmount < 0 || in.ProYearlyAmount < 0 {
		Fail(c, CodeBadRequest, "价格不能为负数")
		return
	}
	if in.StudentDiscount < 0.1 || in.StudentDiscount > 0.99 {
		Fail(c, CodeBadRequest, "学生折扣需在 0.1~0.99 之间")
		return
	}
	if in.RenewalDiscount < 0.1 || in.RenewalDiscount > 0.99 {
		Fail(c, CodeBadRequest, "平季续费折扣需在 0.1~0.99 之间")
		return
	}
	if in.ActivityEnabled && (in.ActivityDiscount < 0.1 || in.ActivityDiscount > 0.99) {
		Fail(c, CodeBadRequest, "活动折扣需在 0.1~0.99 之间")
		return
	}
	p := &model.PriceConfig{
		ProMonthlyAmount:   in.ProMonthlyAmount,
		ProQuarterlyAmount: in.ProQuarterlyAmount,
		ProYearlyAmount:    in.ProYearlyAmount,
		StudentDiscount:    in.StudentDiscount,
		RenewalDiscount:    in.RenewalDiscount,
		RenewalEnabled:     in.RenewalEnabled,
		ActivityName:       strings.TrimSpace(in.ActivityName),
		ActivityDiscount:   in.ActivityDiscount,
		ActivityEnabled:    in.ActivityEnabled,
	}
	if len([]rune(p.ActivityName)) > 32 {
		Fail(c, CodeBadRequest, "活动名称最长 32 个字")
		return
	}
	if err := model.NewPriceModel().Save(p); err != nil {
		ServerError(c, "保存价格失败")
		return
	}
	OK(c, p)
}

// validateCard 按卡片校验字段并返回错误文案（空串表示通过）。
func validateCard(in model.PriceCardInput) string {
	switch in.Card {
	case model.PriceCardPro:
		if in.ProMonthlyAmount < 0 || in.ProQuarterlyAmount < 0 || in.ProYearlyAmount < 0 {
			return "价格不能为负数"
		}
	case model.PriceCardStudent:
		if in.StudentDiscount < 0.1 || in.StudentDiscount > 0.99 {
			return "学生折扣需在 0.1~0.99 之间"
		}
	case model.PriceCardRenewal:
		if in.RenewalEnabled && (in.RenewalDiscount < 0.1 || in.RenewalDiscount > 0.99) {
			return "平季折扣需在 0.1~0.99 之间"
		}
	case model.PriceCardActivity:
		if in.ActivityEnabled {
			if len([]rune(strings.TrimSpace(in.ActivityName))) == 0 {
				return "请填写活动名称"
			}
			if len([]rune(in.ActivityName)) > 32 {
				return "活动名称最长 32 个字"
			}
			if in.ActivityDiscount < 0.1 || in.ActivityDiscount > 0.99 {
				return "活动折扣需在 0.1~0.99 之间"
			}
		}
	default:
		return "未知的价格卡片"
	}
	return ""
}

// priceWarnings 返回「不阻断保存」的软提醒：套餐价格是否真的形成优惠、折上折后是否过低。
// 仅作运营参考，最终是否保存仍由管理员决定。
func priceWarnings(in model.PriceCardInput, cur *model.PriceConfig) []string {
	warns := []string{}
	pro := *cur
	if in.Card == model.PriceCardPro {
		pro.ProMonthlyAmount = in.ProMonthlyAmount
		pro.ProQuarterlyAmount = in.ProQuarterlyAmount
		pro.ProYearlyAmount = in.ProYearlyAmount
	}
	// 套餐合理性：包季 / 包年应低于包月 ×3 / ×12，否则不成其为优惠
	if pro.ProMonthlyAmount > 0 && pro.ProQuarterlyAmount >= pro.ProMonthlyAmount*3 {
		warns = append(warns, fmt.Sprintf("包季 ¥%.0f 不低于包月 ¥%.0f × 3，未形成套餐优惠", pro.ProQuarterlyAmount, pro.ProMonthlyAmount))
	}
	if pro.ProMonthlyAmount > 0 && pro.ProYearlyAmount >= pro.ProMonthlyAmount*12 {
		warns = append(warns, fmt.Sprintf("包年 ¥%.0f 不低于包月 ¥%.0f × 12，未形成套餐优惠", pro.ProYearlyAmount, pro.ProMonthlyAmount))
	}
	// 折上折护栏：按输入卡片的最新值，取最省组合（学生 × 平季 × 活动）
	stu, renew, act := pro.StudentDiscount, pro.RenewalDiscount, pro.ActivityDiscount
	renewOn, actOn := pro.RenewalEnabled, pro.ActivityEnabled
	switch in.Card {
	case model.PriceCardStudent:
		if in.StudentDiscount > 0 {
			stu = in.StudentDiscount
		}
	case model.PriceCardRenewal:
		renew, renewOn = in.RenewalDiscount, in.RenewalEnabled
	case model.PriceCardActivity:
		act, actOn = in.ActivityDiscount, in.ActivityEnabled
	}
	final := 1.0
	if renewOn && renew > 0 && renew < 1 {
		final *= renew
	}
	if actOn && act > 0 && act < 1 {
		final *= act
	}
	if stu > 0 && stu < 1 {
		final *= stu
	}
	if final < 0.3 {
		warns = append(warns, fmt.Sprintf("折上折后最低约 %.1f 折（学生身份叠加当前优惠），价格明显偏低，请确认是否符合预期", final*10))
	}
	return warns
}

// AdminUpdatePriceCard 按卡片保存价格 / 折扣，支持立即生效或指定未来时间生效。
// 立即生效：写入价格配置、记录变更历史并给对应账号发送站内信（仅调整的卡片通知）。
// 指定时间：写入 price_schedule，由后台调度器在到达时间后应用并通知。
// 响应中的 warnings 为软提醒，不影响保存结果。
func AdminUpdatePriceCard(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	var in model.PriceCardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		Fail(c, CodeBadRequest, "参数错误")
		return
	}
	in.ActivityName = strings.TrimSpace(in.ActivityName)
	if msg := validateCard(in); msg != "" {
		Fail(c, CodeBadRequest, msg)
		return
	}
	// 生效时间精确到「日」：统一归一到所选日期的 00:00，避免出现半天生效的歧义
	if in.EffectiveAt > 0 {
		in.EffectiveAt = common.StartOfDay(time.Unix(in.EffectiveAt, 0)).Unix()
	}
	cur, err := model.NewPriceModel().Get()
	if err != nil {
		ServerError(c, "读取价格失败")
		return
	}
	warns := priceWarnings(in, cur)
	now := time.Now().Unix()
	immediate := in.EffectiveAt <= 0 || in.EffectiveAt <= now
	// 未启用的价卡（当前与目标均为关闭）不会有实际价格变化，禁止为其创建未来预约
	if !immediate && !model.PriceScheduleMeaningful(in, cur) {
		Fail(c, CodeBadRequest, "「"+priceCardName(in.Card)+"」当前未启用且保持关闭，无需指定生效日期")
		return
	}
	if immediate {
		if _, err := model.NewPriceScheduleModel().ApplyCardWithHistory(in, "immediate", viewer.ID, viewer.Username); err != nil {
			ServerError(c, "保存价格失败")
			return
		}
		if err := model.NotifyPriceCard(in); err != nil {
			logger.Error("notify price card %s failed, err=%s", in.Card, err.Error())
		}
		OK(c, gin.H{"applied": true, "effectiveAt": now, "warnings": warns})
		return
	}
	payload, err := in.MarshalPayload()
	if err != nil {
		ServerError(c, "保存价格失败")
		return
	}
	sch := &model.PriceSchedule{Card: in.Card, Payload: payload, EffectiveAt: in.EffectiveAt, Status: "pending"}
	if err := model.NewPriceScheduleModel().Create(sch); err != nil {
		ServerError(c, "保存价格失败")
		return
	}
	OK(c, gin.H{"applied": false, "scheduleId": sch.ID, "effectiveAt": in.EffectiveAt, "warnings": warns})
}

// AdminListPriceSchedules 管理员查看尚未生效的价格调整预约（按卡片 + 时间）。
// 未启用价卡的无效预约（当前与目标均为关闭）会被过滤，避免展示不会生效的「待生效」。
func AdminListPriceSchedules(c *gin.Context) {
	list, err := model.NewPriceScheduleModel().PendingAll()
	if err != nil {
		ServerError(c, "读取预约失败")
		return
	}
	cur, err := model.NewPriceModel().Get()
	if err != nil {
		ServerError(c, "读取价格失败")
		return
	}
	out := make([]model.PriceSchedule, 0, len(list))
	for _, s := range list {
		var in model.PriceCardInput
		if json.Unmarshal([]byte(s.Payload), &in) != nil || model.PriceScheduleMeaningful(in, cur) {
			out = append(out, s) // 解析失败时保留原样，交由管理员处理
		}
	}
	OK(c, out)
}

// PublicPriceSchedules 公开接口：返回尚未生效（生效时间在未来）的价格调整预约，
// 供首页「价格调整预告」横幅提前告知用户；含当前值(before)与目标值(target)便于前端对比展示。
// 已到点 / 已应用的预约不再返回（由站内信通知）。
func PublicPriceSchedules(c *gin.Context) {
	now := time.Now().Unix()
	list, err := model.NewPriceScheduleModel().PendingUpcoming(now)
	if err != nil {
		ServerError(c, "读取价格预约失败")
		return
	}
	cur, err := model.NewPriceModel().Get()
	if err != nil {
		ServerError(c, "读取价格失败")
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, s := range list {
		var in model.PriceCardInput
		if err := json.Unmarshal([]byte(s.Payload), &in); err != nil {
			continue
		}
		// 未启用价卡的无效预约（当前与目标均为关闭）不预告
		if !model.PriceScheduleMeaningful(in, cur) {
			continue
		}
		out = append(out, gin.H{
			"id":          s.ID,
			"card":        s.Card,
			"cardName":    priceCardName(s.Card),
			"effectiveAt": s.EffectiveAt,
			"before":      model.CardSnapshot(cur, s.Card),
			"target":      in,
		})
	}
	OK(c, out)
}

// priceCardName 价格卡片中文名（与前端 cardLabel 保持一致）。
func priceCardName(card string) string {
	switch card {
	case model.PriceCardPro:
		return "标准价"
	case model.PriceCardStudent:
		return "学生折扣"
	case model.PriceCardRenewal:
		return "平季续费优惠"
	case model.PriceCardActivity:
		return "限时活动"
	default:
		return card
	}
}

// AdminCancelPriceSchedule 撤销一条尚未生效的价格预约（已生效的记录不可撤销）。
func AdminCancelPriceSchedule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "预约 ID 有误")
		return
	}
	ok, err := model.NewPriceScheduleModel().Cancel(uint(id))
	if err != nil {
		ServerError(c, "撤销预约失败")
		return
	}
	if !ok {
		Fail(c, CodeBadRequest, "该预约已生效或不存在，无法撤销")
		return
	}
	OK(c, gin.H{"canceled": true})
}

// AdminPriceImpact 价格调整站内信的预计触达人数：all = 全部注册账号，students = 大学生家教账号。
// 仅按账号身份统计，与是否在线 / 当前登录状态无关。
func AdminPriceImpact(c *gin.Context) {
	um := model.NewUserModel()
	all, err := um.PriceAudienceCount(false)
	if err != nil {
		ServerError(c, "统计触达人数失败")
		return
	}
	students, err := um.PriceAudienceCount(true)
	if err != nil {
		ServerError(c, "统计触达人数失败")
		return
	}
	OK(c, gin.H{"all": all, "students": students})
}

// AdminPriceHistory 价格变更历史（按时间倒序），用于审计与回滚。
func AdminPriceHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := model.NewPriceHistoryModel().List(limit)
	if err != nil {
		ServerError(c, "读取变更历史失败")
		return
	}
	OK(c, list)
}

// AdminRollbackPrice 按历史记录把某卡片回滚到变更前的值。
// 回滚本身也会写入一条历史（source=rollback），因此可以再次回滚。
func AdminRollbackPrice(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "历史记录 ID 有误")
		return
	}
	h, err := model.NewPriceHistoryModel().GetByID(uint(id))
	if err != nil {
		Fail(c, CodeBadRequest, "历史记录不存在")
		return
	}
	var in model.PriceCardInput
	if err := json.Unmarshal([]byte(h.Before), &in); err != nil {
		ServerError(c, "历史记录解析失败")
		return
	}
	in.Card = h.Card
	in.EffectiveAt = 0
	if msg := validateCard(in); msg != "" {
		Fail(c, CodeBadRequest, "历史值已不可用："+msg)
		return
	}
	if _, err := model.NewPriceScheduleModel().ApplyCardWithHistory(in, "rollback", viewer.ID, viewer.Username); err != nil {
		ServerError(c, "回滚失败")
		return
	}
	if err := model.NotifyPriceCard(in); err != nil {
		logger.Error("notify price rollback failed, err=%s", err.Error())
	}
	OK(c, gin.H{"rolledBack": true, "card": h.Card})
}
