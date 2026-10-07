package handler

import (
	"fmt"
	"time"

	"tutoring_server/internal/model"
	"tutoring_server/pkg/logger"
)

// 试用期转化推送：注册即赠送 30 天免费试用，按注册天数分阶段推送站内信，促进付费转化。
//   - 第 1-3 天  激活：未完成首次排课 → 引导完成第一次排课；
//   - 第 7 天    价值回顾：已完成排课 → 「安排了 X 节课，节省约 Y 小时」；
//   - 第 20 天   预警：剩余 10 天 + 试用成果对比（数据仍可查看，但调课 / 费用统计将受限）；
//   - 第 28-30 天 转化紧迫：限时优惠（首年 99 元）。
//
// 每阶段通过 trial_nudge 表去重，同一用户同一阶段只推送一次；由 main.go 每日调度。
func RunTrialNudges() {
	users, err := model.NewUserModel().TrialingUsers()
	if err != nil {
		logger.Error("load trialing users failed, err=%s", err.Error())
		return
	}
	sent := 0
	for i := range users {
		u := &users[i]
		days := u.RegisteredDays()
		stats := loadTrialStats(u.ID)
		kind, msg := buildTrialNudge(u, days, stats)
		if kind == "" || msg == nil {
			continue
		}
		if model.NewTrialNudgeModel().Sent(u.ID, kind) {
			continue
		}
		if err := model.NewMessageModel().Send(msg, u.ID); err != nil {
			logger.Error("send trial nudge %s to user %d failed, err=%s", kind, u.ID, err.Error())
			continue
		}
		if err := model.NewTrialNudgeModel().Mark(u.ID, kind); err != nil {
			logger.Error("mark trial nudge %s user %d failed, err=%s", kind, u.ID, err.Error())
		}
		sent++
	}
	if sent > 0 {
		logger.Info("trial nudges sent: %d", sent)
	}
}

// trialStats 试用账号的使用数据：课程数、学生数（按学员姓名去重）、已排课次。
type trialStats struct {
	OrderCount   int64
	StudentCount int64
	LessonCount  int64
}

func loadTrialStats(userID uint) trialStats {
	var s trialStats
	_ = model.DB().Raw(
		`SELECT count(*) AS order_count,
		        count(DISTINCT NULLIF(student_name, '')) AS student_count
		 FROM "order" WHERE user_id = ?`,
		userID,
	).Scan(&s)
	// 已排课次：按周时段展开并落库的历史课次（未作废）。
	_ = model.DB().Raw(
		`SELECT count(*) FROM lesson WHERE user_id = ? AND voided = false`,
		userID,
	).Scan(&s.LessonCount)
	return s
}

// buildTrialNudge 按注册天数返回（阶段 kind, 站内信）；未命中阶段返回空。
func buildTrialNudge(u *model.User, days int, stats trialStats) (string, *model.Message) {
	now := time.Now().Unix()
	base := model.Message{
		SenderID:   0,
		SenderName: "系统",
		Type:       model.MessageTypeSystem,
		CreatedAt:  now,
	}
	switch {
	case days >= 1 && days <= 3:
		if stats.OrderCount > 0 {
			return "", nil // 已完成首次排课，无需激活引导
		}
		base.Title = "完成您的第一次排课"
		base.Content = "欢迎试用课满满！录入学生、年级、科目和每周时段，就能自动生成整周课表，课时与费用一目了然。\n\n" +
			"现在就去「课程安排」添加第一门课程，30 秒即可完成。如需帮助可在「使用指南」查看图文教程。"
		return "activate", &base

	case days == 7:
		if stats.OrderCount == 0 {
			return "", nil // 尚未排课的用户留在激活引导阶段，不发价值回顾
		}
		hours := stats.LessonCount * 15 / 60 // 每节课约节省 15 分钟手动排课对表时间
		if hours < 1 {
			hours = 1
		}
		base.Title = "您已安排 " + intStr(int(stats.LessonCount)) + " 节课"
		base.Content = fmt.Sprintf("注册满一周啦！这段时间您已用本工具安排了 %d 节课，节省了约 %d 小时手动排课与对表时间。\n\n"+
			"课表每周自动生成，调课改期随手可调，继续体验吧。", stats.LessonCount, hours)
		return "value", &base

	case days == 20:
		leftDays := u.MemberLeftDays()
		if leftDays > 10 {
			leftDays = 10
		}
		base.Title = "试用期剩余 " + intStr(leftDays) + " 天"
		base.Content = fmt.Sprintf("您的免费试用期还剩约 %d 天。\n\n试用期间您管理了 %d 个学生、%d 门课程、%d 节课——"+
			"试用期结束后，这些已录入的订单和课表仍然可以随时查看，但新增课程、调课和费用统计功能将受限。\n\n"+
			"如需继续完整使用，请及时开通会员。", leftDays, stats.StudentCount, stats.OrderCount, stats.LessonCount)
		return "preview", &base

	case days >= 28 && days <= 30:
		base.Title = "限时优惠：包年首年 99 元"
		base.Type = model.MessageTypePromo
		base.Content = "您的免费试用期即将结束（剩 " + intStr(u.MemberLeftDays()) + " 天）。\n\n" +
			"现在开通包年会员即可享受限时优惠价「首年 99 元」：课程数量不限、课表导出、费用统计与调课提醒全部解锁。\n\n" +
			"请扫码支付后联系管理员开通，优惠名额有限，先到先得。"
		return "urgency", &base
	}
	return "", nil
}

// intStr int → string（包内已有 itoa 工具，此处用独立命名避免冲突）。
func intStr(n int) string {
	return fmt.Sprintf("%d", n)
}
