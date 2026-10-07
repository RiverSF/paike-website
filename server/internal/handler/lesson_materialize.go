package handler

import (
	"fmt"
	"time"

	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// MaterializeOrderRange 把订单在 [from, to] 区间内、按当时配置应产生的课次落库为历史快照。
// 已存在的课次不会被覆盖（保证历史冻结、不受后续配置改动影响）。to 一般传「昨日」，
// 即只物化历史；今天的课仍按配置实时展开，改动即时生效。
//
// 该函数幂等：先拉取区间内已物化的课次键集合，仅补齐缺失的日期，因此可被定时任务反复调用。
func MaterializeOrderRange(o *model.Order, from, to time.Time) {
	if o == nil || o.StartDate <= 0 {
		return
	}
	// 入参统一归一到本地零点：本函数按自然日遍历，若 from/to 携带时刻（如 time.Now() 派生），
	// 会因 from 比订单结束日期（零点）晚而误判 from>to，导致整天课次被跳过。
	from = common.StartOfDay(from)
	to = common.StartOfDay(to)
	if o.EndDate > 0 {
		if e := time.Unix(o.EndDate, 0); e.Before(to) {
			to = e
		}
	}
	if from.After(to) {
		return
	}

	lm := model.NewLessonModel()
	existing, err := lm.ListByOrderRange(o.ID, from.Format(common.DateLayout), to.Format(common.DateLayout))
	if err != nil {
		logger.Error("load existing lessons failed, err=%s", err.Error())
		return
	}
	existSet := make(map[string]bool, len(existing))
	for _, l := range existing {
		existSet[fmt.Sprintf("%s|%d|%d", l.Date, l.SlotIndex, l.ExceptionID)] = true
	}

	exs, _ := model.NewLessonExceptionModel().ListByOrder(o.ID, o.UserID)
	// now=to 仅用于 TimeState，历史课次统一按 past 渲染，此处取值无影响
	events := expandOrder(o, exs, from, to, to)
	for _, ev := range events {
		key := fmt.Sprintf("%s|%d|%d", ev.Date, ev.SlotIndex, ev.ExceptionID)
		if existSet[key] {
			continue
		}
		status := "regular"
		if ev.AdjustType != "" {
			status = ev.AdjustType
		}
		l := model.Lesson{
			UserID:          o.UserID,
			OrderID:         o.ID,
			OrderNo:         o.OrderNo,
			Date:            ev.Date,
			Weekday:         ev.Weekday,
			SlotIndex:       ev.SlotIndex,
			StartTime:       ev.StartTime,
			EndTime:         ev.EndTime,
			DurationMinutes: ev.DurationMinutes,
			Grade:           ev.Grade,
			StudentName:     ev.StudentName,
			Subject:         ev.Subject,
			Address:         ev.Address,
			Content:         ev.Content,
			Remark:          ev.Remark,
			RemarkFlag:      ev.RemarkFlag,
			HourlyRate:      ev.HourlyRate,
			Status:          status,
			Income:          ev.Income,
			ExceptionID:     ev.ExceptionID,
			OriginDate:      ev.OriginDate,
			OriginWeekday:   ev.OriginWeekday,
			MovedToDate:     ev.MovedToDate,
			MovedToStart:    ev.MovedToStart,
			AdjustNote:      ev.AdjustNote,
		}
		if err := lm.Create(&l); err != nil {
			logger.Error("create lesson failed, err=%s", err.Error())
		}
	}
}

// MaterializeOrderLessons 兼容旧调用：物化订单从开始日期到 upTo 的整个历史区间（全量）。
// 供订单新增/编辑、统计等低频场景使用，保留全量自愈能力。
func MaterializeOrderLessons(o *model.Order, upTo time.Time) {
	if o == nil {
		return
	}
	MaterializeOrderRange(o, time.Unix(o.StartDate, 0), upTo)
}

// MaterializeOrderDay 仅物化指定某一天（一般传昨日）的课次。历史课次一旦落库即冻结，
// 不会因后续配置/调课再变化，每日唯一新进入「历史」的只有昨日，故定时任务只需物化前一日即可，
// 避免对全量历史重复扫描。
func MaterializeOrderDay(o *model.Order, day time.Time) {
	MaterializeOrderRange(o, day, day)
}

// MaterializeHistoryUntilYesterday 物化所有订单「昨日」当天的历史课次（每日定时任务调用）。
// 改为仅物化前一日：历史课次是冻结快照，无需每日重扫全量历史。
// 注意：日期必须取「本地零点」，否则时间戳携带时刻会让 MaterializeOrderRange 的
// from>to 判定成立，导致结束日期恰好是昨日的订单被整体跳过、昨日课次永不落库。
func MaterializeHistoryUntilYesterday() {
	yesterday := common.StartOfDay(time.Now()).AddDate(0, 0, -1)
	orders, err := model.NewOrderModel().ListAll()
	if err != nil {
		logger.Error("list orders for materialize failed, err=%s", err.Error())
		return
	}
	for i := range orders {
		MaterializeOrderDay(&orders[i], yesterday)
	}
}

// RepairAdjustedLessons 存量数据修复（服务启动时执行一次）：
// 历史版本中「改期后原课次与调出占位并存」，这里统一清理为「仅保留调整后的课时」——
// 例外生成的课次按例外当前生效状态重建，被调整替代的原生课次软删除。
func RepairAdjustedLessons() {
	orders, err := model.NewOrderModel().ListAll()
	if err != nil {
		logger.Error("list orders for repair failed, err=%s", err.Error())
		return
	}
	count := 0
	for i := range orders {
		o := &orders[i]
		exs, eerr := model.NewLessonExceptionModel().ListByOrder(o.ID, o.UserID)
		if eerr != nil || len(exs) == 0 {
			continue
		}
		for j := range exs {
			if exs[j].Type == model.ExceptionTypeExtra {
				continue
			}
			syncMaterializedLessons(o, &exs[j], false)
			count++
		}
	}
	if count > 0 {
		logger.Info("repair adjusted lessons done, exceptions=%d", count)
	}
}

// RepairLessonIncomes 存量课时费修复（服务启动时执行一次，幂等）：
// 早期版本物化时订单可能尚未填写价格（或计费逻辑缺失），会留下 income=0 / hourly_rate=0 的冻结快照，
// 课表与调整弹窗据此显示 ¥0.00。这里按订单当前计费方式补算回填：
// 仅当订单确实配置了价格（单次课时价或时薪）且补算结果大于 0 时才修复，订单无价格的课次保持 0。
func RepairLessonIncomes() {
	orders, err := model.NewOrderModel().ListAll()
	if err != nil {
		logger.Error("list orders for income repair failed, err=%s", err.Error())
		return
	}
	// 订单快照：按总费用计费时还需计划总节数才能分摊，故保留完整订单
	fees := make(map[uint]*model.Order, len(orders))
	for i := range orders {
		fees[orders[i].ID] = &orders[i]
	}
	lessons, err := model.NewLessonModel().ListZeroIncome()
	if err != nil {
		logger.Error("list zero income lessons failed, err=%s", err.Error())
		return
	}
	fixed := 0
	for _, l := range lessons {
		o, ok := fees[l.OrderID]
		if !ok {
			continue
		}
		// 修复只做历史补算，忽略例外影响（停课 / 改期对总节数的影响很小）
		income := lessonIncomeWithPlan(o, l.DurationMinutes, orderPlannedLessons(o, nil))
		if income <= 0 {
			continue // 订单本身未配价格，0 为合理值
		}
		hourly := l.HourlyRate
		if hourly <= 0 {
			hourly = o.HourlyRate
		}
		if err := model.NewLessonModel().UpdateFee(l.ID, income, hourly); err != nil {
			logger.Error("repair lesson income %d failed, err=%s", l.ID, err.Error())
			continue
		}
		fixed++
	}
	if fixed > 0 {
		logger.Info("repair lesson incomes done, rows=%d", fixed)
	}
}

// lessonToEvent 把已物化的历史课次还原为课表展示事件（历史一律标记为 past）。
// orderStatus 为该课次所属订单的当前状态（课表快照不保存订单状态）：
// 曾硬编码为 running，导致已结束订单的课在详情里显示「进行中」，但调整时被后端以「已结束」拒绝，前后矛盾。
func lessonToEvent(l model.Lesson, orderStatus string) scheduleEvent {
	if orderStatus == "" {
		orderStatus = model.OrderStatusRunning
	}
	sm := common.ClockToMinutes(l.StartTime)
	em := common.ClockToMinutes(l.EndTime)
	if em <= sm {
		em = sm + 60
	}
	dur := l.DurationMinutes
	if dur <= 0 {
		dur = em - sm
	}
	return scheduleEvent{
		Key:             fmt.Sprintf("L-%d", l.ID),
		LessonID:        l.ID,
		OrderID:         l.OrderID,
		OrderNo:         l.OrderNo,
		Date:            l.Date,
		Weekday:         l.Weekday,
		StartTime:       l.StartTime,
		EndTime:         l.EndTime,
		TimeState:       "past",
		StartMinute:     sm,
		EndMinute:       em,
		Grade:           l.Grade,
		StudentName:     l.StudentName,
		Subject:         l.Subject,
		Address:         l.Address,
		Content:         l.Content,
		Remark:          l.Remark,
		RemarkFlag:      l.RemarkFlag,
		HourlyRate:      l.HourlyRate,
		Income:          l.Income,
		DurationMinutes: dur,
		Status:          orderStatus,
		SlotIndex:       l.SlotIndex,
		AdjustType:      l.Status,
		ExceptionID:     l.ExceptionID,
		OriginDate:      l.OriginDate,
		OriginWeekday:   l.OriginWeekday,
		MovedToDate:     l.MovedToDate,
		MovedToStart:    l.MovedToStart,
		AdjustNote:      l.AdjustNote,
	}
}
