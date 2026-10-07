package handler

import (
	"fmt"
	"time"

	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// lessonReminderLeadSeconds 提前提醒窗口：距开课 2 小时内触发提醒。
const lessonReminderLeadSeconds = 2 * 60 * 60

// NotifyUpcomingLessons 扫描全部有效会员账号的今日 / 明日课次，
// 对「距开始时间不足 2 小时且尚未开始」的课次给对应账号发送一条站内信提醒。
// 通知范围只由账号身份决定，与是否登录 / 在线无关；同一节课只提醒一次
// （lesson_reminder 表 + 唯一索引幂等），因此可安全地反复调用。
func NotifyUpcomingLessons() {
	now := time.Now()
	ids, err := model.NewUserModel().ActiveMemberIDs()
	if err != nil {
		logger.Error("lesson reminder: load active members failed, err=%s", err.Error())
		return
	}
	// 清理 30 天前的去重记录，避免表无限增长
	if err := model.NewLessonReminderModel().CleanupBefore(now.AddDate(0, 0, -30).Unix()); err != nil {
		logger.Error("lesson reminder: cleanup history failed, err=%s", err.Error())
	}
	sent := 0
	for _, uid := range ids {
		n, err := remindUserUpcomingLessons(uid, now)
		if err != nil {
			logger.Error("lesson reminder: user %d failed, err=%s", uid, err.Error())
			continue
		}
		sent += n
	}
	if sent > 0 {
		logger.Info("lesson reminder: sent %d reminder(s)", sent)
	}
}

// remindUserUpcomingLessons 展开某账号今天与明天的课次，为临近开始的课次发送提醒，返回发送条数。
// 未来课次不在 lesson 表（那里只有历史快照），因此按订单周期 + 课程例外实时展开。
func remindUserUpcomingLessons(userID uint, now time.Time) (int, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	from := dayStart
	// 覆盖「今天 + 明天」：保证次日凌晨的课次在跨零点前也能被扫到
	to := dayStart.AddDate(0, 0, 2).Add(-time.Second)
	orders, exMap, err := loadOrdersForRange(userID, from.Unix(), to.Unix(), "")
	if err != nil {
		return 0, err
	}

	nowTs := now.Unix()
	limitTs := nowTs + lessonReminderLeadSeconds
	sent := 0
	for i := range orders {
		o := &orders[i]
		if o.Status == model.OrderStatusFinished {
			continue // 已结束的订单不再提醒
		}
		for _, ev := range expandOrder(o, exMap[o.ID], from, to, now) {
			// 停课 / 已调至其它日期的位置不会实际发生，跳过
			if isExcludedFromSummary(ev.AdjustType) {
				continue
			}
			day, err := time.ParseInLocation(common.DateLayout, ev.Date, now.Location())
			if err != nil {
				continue
			}
			startTs := day.Unix() + int64(ev.StartMinute)*60
			// 仅提醒「尚未开始且距开始不足 2 小时」的课次
			if startTs <= nowTs || startTs > limitTs {
				continue
			}
			fresh, err := model.NewLessonReminderModel().MarkIfNew(userID, o.ID, ev.Date, ev.StartTime)
			if err != nil {
				return sent, err
			}
			if !fresh {
				continue // 该节课已提醒过
			}
			if err := model.NewMessageModel().Send(lessonReminderMessage(ev, startTs-nowTs), userID); err != nil {
				return sent, err
			}
			sent++
		}
	}
	return sent, nil
}

// lessonReminderMessage 组装开课提醒站内信。
func lessonReminderMessage(ev scheduleEvent, leftSeconds int64) *model.Message {
	leftText := fmt.Sprintf("%d 分钟", leftSeconds/60)
	if leftSeconds >= 3600 {
		leftText = fmt.Sprintf("约 %d 小时", (leftSeconds+1800)/3600)
	}
	student := ev.StudentName
	if student == "" {
		student = "学生"
	}
	detail := fmt.Sprintf("%s %s-%s · %s", ev.Date, ev.StartTime, ev.EndTime, student)
	if ev.Subject != "" {
		detail += "（" + ev.Subject + "）"
	}
	if ev.Address != "" {
		detail += " · " + ev.Address
	}
	return &model.Message{
		SenderID:   0,
		SenderName: "系统",
		Title:      "上课提醒：课程即将开始",
		Content: fmt.Sprintf(
			"您有一节课将在 %s后开始（%s），请预留出行与备课时间。可在「课表」中查看课程详情。",
			leftText, detail,
		),
		Type:      model.MessageTypeSystem,
		CreatedAt: time.Now().Unix(),
	}
}
