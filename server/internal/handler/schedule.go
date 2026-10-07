package handler

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// 课表纵轴默认展示的小时区间（06:00 - 23:00）；当天课程超出该区间时会自动扩展。
const (
	dayStartHour = 6
	dayEndHour   = 23
)

// maxRangeDays 自定义日期区间的最大天数（防止一次拉取过多数据）。
const maxRangeDays = 62

// 课程调整类型：由「课程例外」产生，用于前端区分常规课程与单次变动。
const (
	adjustCanceled = "canceled" // 本次停课（不计课时与收入）
	adjustMovedOut = "movedOut" // 本次已调至其它日期（原位仅作提示）
	adjustMovedIn  = "movedIn"  // 由其它日期调入本次
	adjustTime     = "time"     // 仅调整本次起止时间
	adjustExtra    = "extra"    // 临时加课
)

// 不计入课时 / 收入统计的调整类型。
func isExcludedFromSummary(adjustType string) bool {
	return adjustType == adjustCanceled || adjustType == adjustMovedOut
}

var weekdayLabel = map[int]string{1: "周一", 2: "周二", 3: "周三", 4: "周四", 5: "周五", 6: "周六", 7: "周日"}

type scheduleEvent struct {
	Key       string `json:"key"`
	OrderID   uint   `json:"orderId"`
	OrderNo   string `json:"orderNo"`
	Date      string `json:"date"`
	Weekday   int    `json:"weekday"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	// TimeState 课程相对于当前时间的状态：past=已结束历史课程 ongoing=进行中 future=未开始
	TimeState       string  `json:"timeState"`
	StartMinute     int     `json:"startMinute"`
	EndMinute       int     `json:"endMinute"`
	Grade           string  `json:"grade"`
	StudentName     string  `json:"studentName"`
	Subject         string  `json:"subject"`
	Address         string  `json:"address"`
	Content         string  `json:"content"`
	Remark          string  `json:"remark"`
	RemarkFlag      bool    `json:"remarkFlag"`
	HourlyRate      float64 `json:"hourlyRate"`
	Income          float64 `json:"income"` // 本次课时费（元）；例外指定时为定制值，否则按订单标准计费
	DurationMinutes int     `json:"durationMinutes"`
	Status          string  `json:"status"`
	// Direction 课次归属方向：income=收入（我收的课）/ expense=开支（我付的课），
	// 决定课表金额按收入还是开支口径汇总（订单可单独设置，默认跟随注册身份）
	Direction string `json:"direction"`
	LessonID  uint   `json:"lessonId"` // 历史课次（已物化）的主键；未来/当期课次为 0
	// 单次调整相关
	SlotIndex     int    `json:"slotIndex"`     // 对应的每周时段下标，加课为 -1
	AdjustType    string `json:"adjustType"`    // 空=常规课程；canceled / movedOut / movedIn / time / extra
	ExceptionID   uint   `json:"exceptionId"`   // 关联的课程例外 ID，0 表示无
	OriginDate    string `json:"originDate"`    // movedIn：原定上课日期
	OriginWeekday int    `json:"originWeekday"` // movedIn：原定星期（1-7）
	OriginStart   string `json:"originStart"`   // movedIn：原定开始时间
	OriginEnd     string `json:"originEnd"`     // movedIn：原定结束时间
	MovedToDate   string `json:"movedToDate"`   // movedOut：调整后的日期
	MovedToStart  string `json:"movedToStart"`  // movedOut：调整后的开始时间
	AdjustNote    string `json:"adjustNote"`    // 调整原因
}

// Schedule 生成课表：x 轴为日期区间（默认当前自然周，支持自定义起止日期、可跨周），y 轴按小时。
func Schedule(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	// 自定义日期区间（可跨周）；未提供或非法时回退为所在自然周
	from := common.ParseDate(c.Query("start"))
	to := common.ParseDate(c.Query("end"))
	if from.IsZero() || to.IsZero() || to.Before(from) {
		base := time.Now()
		if d := c.Query("date"); d != "" {
			if t := common.ParseDate(d); !t.IsZero() {
				base = t
			}
		}
		from = common.MondayOf(base)
		to = from.AddDate(0, 0, 6)
	} else if n := int(to.Sub(from).Hours()/24) + 1; n > maxRangeDays {
		to = from.AddDate(0, 0, maxRangeDays-1)
	}

	today := time.Now().Format(common.DateLayout)

	// 过期检查：已超过结束日期的进行中订单自动置为已结束，保证课表与订单状态一致
	if err := model.NewOrderModel().SyncOrderStatus(user.ID); err != nil {
		logger.Error("finish expired orders failed, err=%s", err.Error())
	}

	status := c.Query("status")
	orders, exMap, err := loadOrdersForRange(user.ID, from.Unix(), to.Unix(), status)
	if err != nil {
		logger.Error("list orders for schedule failed, err=%s", err.Error())
		ServerError(c, "生成课表失败")
		return
	}

	now := time.Now()
	// 日期边界统一取本地零点：历史 = 今天之前（不含今天），今天起按订单配置实时展开
	todayStart := common.StartOfDay(now)
	yesterday := todayStart.AddDate(0, 0, -1)
	histEnd := to
	if histEnd.After(yesterday) {
		histEnd = yesterday
	}

	events := make([]scheduleEvent, 0)
	totalHours := 0.0
	totalIncome := 0.0
	incomeAmount := 0.0  // 收入方向课时费合计
	expenseAmount := 0.0 // 开支方向课时费合计
	adjusted := 0

	// 订单快照：历史课次费用兜底补算用（物化时订单未配价会冻结 income=0）；
	// 按总费用计费还需要订单的计划总节数才能分摊，故直接存订单本身。
	feeMap := make(map[uint]*model.Order, len(orders))
	// 订单状态快照：历史课次不含订单状态，需按订单回填，避免已结束订单的课在详情里显示为「进行中」
	statusMap := make(map[uint]string, len(orders))
	for i := range orders {
		feeMap[orders[i].ID] = &orders[i]
		statusMap[orders[i].ID] = orders[i].Status
	}

	// 历史部分（<= 昨日）：直接读已物化的 lesson 表，与订单配置解耦，冻结不再随配置改动而漂移。
	// 先兜底物化（定时任务为主力），保证即便定时任务尚未跑过、视图也无缺口。
	if !from.After(histEnd) {
		// 兜底物化范围收窄为「当前自然周」：课表为高频页面，避免每次打开都全量扫描历史。
		// 历史课次主要由每日 01:00 定时任务（仅物化前一日）落库，这里只补当前周，
		// 保证打开课表时本周历史课次无缺口；更早的历史依赖定时任务与既有数据。
		weekStart := common.MondayOf(now)
		weekEnd := weekStart.AddDate(0, 0, 6)
		if weekEnd.After(yesterday) {
			weekEnd = yesterday
		}
		for i := range orders {
			MaterializeOrderRange(&orders[i], weekStart, weekEnd)
		}
		histLessons, lerr := model.NewLessonModel().ListByUserRange(user.ID, from.Format(common.DateLayout), histEnd.Format(common.DateLayout))
		if lerr != nil {
			logger.Error("list lessons failed, err=%s", lerr.Error())
		}
		// 补齐区间外订单的状态（订单起止日期被改过时可能不在本次加载范围内）
		missing := make([]uint, 0, 4)
		for i := range histLessons {
			if _, ok := statusMap[histLessons[i].OrderID]; !ok {
				missing = append(missing, histLessons[i].OrderID)
			}
		}
		if len(missing) > 0 {
			if more, merr := model.NewOrderModel().ListByIDs(user.ID, missing); merr == nil {
				for i := range more {
					statusMap[more[i].ID] = more[i].Status
				}
			}
		}
		for _, l := range histLessons {
			if l.Voided || l.Status == "movedOut" || lessonReplaced(l, exMap[l.OrderID]) {
				// 已删除的历史课次从课表隐藏；已被单次调整替代的原课次不再展示（仅保留调整后的课时）
				continue
			}
			// 兜底：历史课次费用为冻结快照，若物化时订单尚未配价会留下 income=0；
			// 展示时按订单当前计费方式补算，避免课表与调整弹窗显示 ¥0.00
			if l.Income <= 0 {
				if o, ok := feeMap[l.OrderID]; ok {
					planned := orderPlannedLessons(o, exMap[l.OrderID])
					if income := lessonIncomeWithPlan(o, l.DurationMinutes, planned); income > 0 {
						l.Income = income
						if l.HourlyRate <= 0 {
							l.HourlyRate = o.HourlyRate
						}
					}
				}
			}
			ev := lessonToEvent(l, statusMap[l.OrderID])
			if o, ok := feeMap[l.OrderID]; ok {
				ev.Direction = o.Direction
			}
			events = append(events, ev)
			if isExcludedFromSummary(l.Status) {
				adjusted++
				continue // 停课 / 已调出的课不产生课时与收入
			}
			dur := float64(l.DurationMinutes) / 60
			if dur <= 0 {
				dur = 1
			}
			totalHours += dur
			totalIncome += l.Income
			if directionOf(feeMap[l.OrderID]) == model.DirectionExpense {
				expenseAmount += l.Income
			} else {
				incomeAmount += l.Income
			}
		}
	}

	// 未来部分（今日 ~ to）：按当前配置实时展开，改动即时生效
	if !todayStart.After(to) {
		for i := range orders {
			o := orders[i]
			for _, ev := range expandOrder(&o, exMap[o.ID], todayStart, to, now) {
				events = append(events, ev)
				if isExcludedFromSummary(ev.AdjustType) {
					adjusted++
					continue // 停课 / 已调出的课不产生课时与收入
				}
				dur := float64(ev.EndMinute-ev.StartMinute) / 60
				if ev.DurationMinutes > 0 {
					dur = float64(ev.DurationMinutes) / 60
				}
				totalHours += dur
				totalIncome += ev.Income
				if directionOf(&o) == model.DirectionExpense {
					expenseAmount += ev.Income
				} else {
					incomeAmount += ev.Income
				}
			}
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		return events[i].StartMinute < events[j].StartMinute
	})

	// 纵轴小时区间：默认 06:00-23:00，区间内课程更早 / 更晚时自动扩展，避免课程被截断或错位
	startHour, endHour := hourRangeOf(events)

	// x 轴按所选日期区间逐日生成（可跨周）
	days := make([]gin.H, 0, maxRangeDays)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		days = append(days, gin.H{
			"date":    d.Format(common.DateLayout),
			"weekday": common.WeekdayISO(d),
			"label":   weekdayLabel[common.WeekdayISO(d)],
			"dayNum":  d.Day(),
			"isToday": d.Format(common.DateLayout) == today,
		})
	}

	monthLabel := from.Format("2006年01月")
	if from.Format("200601") != to.Format("200601") {
		monthLabel = from.Format("2006年01月") + " - " + to.Format("2006年01月")
	}

	hours := make([]int, 0, endHour-startHour)
	for h := startHour; h < endHour; h++ {
		hours = append(hours, h)
	}

	OK(c, gin.H{
		"rangeStart":   from.Format(common.DateLayout),
		"rangeEnd":     to.Format(common.DateLayout),
		"weekStart":    from.Format(common.DateLayout), // 兼容旧字段：当前展示区间起始
		"weekEnd":      to.Format(common.DateLayout),   // 兼容旧字段：当前展示区间结束
		"monthLabel":   monthLabel,
		"days":         days,
		"hours":        hours,
		"dayStartHour": startHour,
		"dayEndHour":   endHour - 1,
		"events":       events,
		"summary": gin.H{
			"count":    countIncluded(events),
			"hours":    round2(totalHours),
			"income":   round2(totalIncome),
			"adjusted": adjusted,
			// 分方向金额：课表按订单方向区分「我收的课」与「我付的课」
			"incomeAmount":  round2(incomeAmount),
			"expenseAmount": round2(expenseAmount),
		},
	})
}

// directionOf 订单方向：未设置时按「收入」处理（与存量回填口径一致）。
func directionOf(o *model.Order) string {
	if o == nil || o.Direction == "" {
		return model.DirectionIncome
	}
	return o.Direction
}

// lessonReplaced 该已物化课次是否已被单次调整替代（存在匹配「原定日期 + 槽位」的非加课例外）。
// 历史数据兜底：即便存量课时表中「原课次与调整后课时并存」，也只展示调整后的课时。
func lessonReplaced(l model.Lesson, exs []model.LessonException) bool {
	if l.ExceptionID != 0 {
		return false // 例外生成的课次本身就是调整后的课时
	}
	for i := range exs {
		e := &exs[i]
		if e.Type == model.ExceptionTypeExtra {
			continue
		}
		if e.SourceDate == dateToTs(l.Date) && e.SlotIndex == l.SlotIndex {
			return true
		}
	}
	return false
}

// lessonIncomeWithPlan 按计费方式计算单节课程金额：
//   - lesson：单次课时价（一口价）
//   - total ：整期总费用 ÷ 计划总节数（plannedLessons 为 0 时无法分摊，返回 0）
//   - hourly：时长（小时）× 时薪
//
// 兼容历史数据：bill_mode 为空或未知时按「单次价优先、其次时薪」的旧口径计算。
func lessonIncomeWithPlan(o *model.Order, durationMinutes, plannedLessons int) float64 {
	switch o.BillMode {
	case model.BillModeLesson:
		if o.LessonPrice > 0 {
			return o.LessonPrice
		}
		return 0
	case model.BillModeTotal:
		if o.TotalAmount > 0 && plannedLessons > 0 {
			return round2(o.TotalAmount / float64(plannedLessons))
		}
		return 0
	case model.BillModeHourly:
		return hourlyIncome(o, durationMinutes)
	}
	// 未指定计费方式（历史数据）
	if o.LessonPrice > 0 {
		return o.LessonPrice
	}
	return hourlyIncome(o, durationMinutes)
}

// lessonAmountExact 报表口径的单节金额：按总费用计费时返回「未取整」的分摊值，
// 避免逐节四舍五入（如 6000÷9=666.67，×9=6000.03）导致区间总额与课程总费用出现偏差；
// 展示与落库仍使用 lessonIncomeWithPlan 的取整值。
func lessonAmountExact(o *model.Order, durationMinutes, plannedLessons int) float64 {
	if o.BillMode == model.BillModeTotal && o.TotalAmount > 0 && plannedLessons > 0 {
		return o.TotalAmount / float64(plannedLessons)
	}
	return lessonIncomeWithPlan(o, durationMinutes, plannedLessons)
}

// hourlyIncome 按时长（小时）× 时薪计算单节金额。
func hourlyIncome(o *model.Order, durationMinutes int) float64 {
	if o.HourlyRate <= 0 {
		return 0
	}
	dur := float64(durationMinutes) / 60
	if dur <= 0 {
		dur = 1
	}
	return dur * o.HourlyRate
}

// countIncluded 统计实际计入的课程数（停课 / 已调出的占位卡片不计入）。
func countIncluded(events []scheduleEvent) int {
	n := 0
	for _, ev := range events {
		if !isExcludedFromSummary(ev.AdjustType) {
			n++
		}
	}
	return n
}

// hourRangeOf 根据本周课程的时间分布计算纵轴展示区间 [startHour, endHour)。
func hourRangeOf(events []scheduleEvent) (int, int) {
	minMin := dayStartHour * 60
	maxMin := (dayEndHour + 1) * 60
	for _, ev := range events {
		if ev.StartMinute < minMin {
			minMin = ev.StartMinute
		}
		if ev.EndMinute > maxMin {
			maxMin = ev.EndMinute
		}
	}
	startHour := minMin / 60
	if startHour < 0 {
		startHour = 0
	}
	if startHour > 23 {
		startHour = 23
	}
	endHour := (maxMin + 59) / 60
	if endHour <= startHour+1 {
		endHour = startHour + 1
	}
	return startHour, endHour
}

// loadOrdersForRange 查询与 [fromTs,toTs] 有交集的订单，并返回按订单分组的课程例外。
// 除订单周期覆盖该区间的订单外，还会补充「区间内存在调课 / 加课」的订单，
// 否则课程被调到其他周时，目标周会漏掉这一节课。
func loadOrdersForRange(userID uint, fromTs, toTs int64, status string) ([]model.Order, map[uint][]model.LessonException, error) {
	om := model.NewOrderModel()
	// 不按状态过滤：只要订单的补课周期与该周有交集，该周内的课程（含历史课程）都会展开，由 timeState 着色区分
	orders, err := om.ListInRange(userID, fromTs, toTs, status)
	if err != nil {
		return nil, nil, err
	}

	em := model.NewLessonExceptionModel()
	exs, err := em.ListInRange(userID, fromTs, toTs)
	if err != nil {
		return nil, nil, err
	}

	exMap := make(map[uint][]model.LessonException, len(exs))
	exist := make(map[uint]bool, len(orders))
	for _, o := range orders {
		exist[o.ID] = true
	}
	extraIDs := make([]uint, 0)
	for _, e := range exs {
		exMap[e.OrderID] = append(exMap[e.OrderID], e)
		if !exist[e.OrderID] {
			extraIDs = append(extraIDs, e.OrderID)
		}
	}

	if len(extraIDs) > 0 {
		more, err := om.ListByIDs(userID, extraIDs)
		if err != nil {
			return nil, nil, err
		}
		for _, o := range more {
			if status != "" && o.Status != status {
				continue
			}
			orders = append(orders, o)
		}
	}
	return orders, exMap, nil
}

// expandOrder 把订单按时间范围 + 每周频次展开成区间内的课程，再套用课程例外。
func expandOrder(o *model.Order, exs []model.LessonException, weekStart, weekEnd time.Time, now time.Time) []scheduleEvent {
	slots := o.Slots()
	// 计划总节数：按总费用计费时用于把整期费用分摊到每节课（其余计费方式不影响）
	planned := orderPlannedLessons(o, exs)

	// 例外按「原定日期 + 时段下标」索引，临时加课不参与（加课可重复）
	type slotKey struct {
		date int64
		idx  int
	}
	exBySlot := make(map[slotKey]model.LessonException, len(exs))
	for _, e := range exs {
		if e.Type == model.ExceptionTypeExtra {
			continue
		}
		exBySlot[slotKey{e.SourceDate, e.SlotIndex}] = e
	}

	events := make([]scheduleEvent, 0)

	// ① 按周期展开：仅在订单有开始日期与每周时段时生成
	if len(slots) > 0 && o.StartDate > 0 {
		from := time.Unix(o.StartDate, 0)
		to := time.Time{}
		if o.EndDate > 0 {
			to = time.Unix(o.EndDate, 0)
		}
		if from.Before(weekStart) {
			from = weekStart
		}
		if to.IsZero() || to.After(weekEnd) {
			to = weekEnd
		}

		for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
			wd := common.WeekdayISO(d)
			dTs := dayTs(d)
			for si, slot := range slots {
				if !common.ContainsInt(slot.Days, wd) {
					continue
				}
				// 规则版本化：该时段在此日期不生效（生效区间之外）则不生成课程
				if !slot.Active(d) {
					continue
				}
				ev := buildEvent(o, d, si, slot.Start, slot.End, 0, now, planned)
				if ex, ok := exBySlot[slotKey{dTs, si}]; ok {
					ev.ExceptionID = ex.ID
					ev.AdjustNote = ex.Note
					switch ex.Type {
					case model.ExceptionTypeCancel:
						ev.AdjustType = adjustCanceled
						ev.Key += "-c"
					case model.ExceptionTypeTime:
						start := ex.NewStart
						if start == "" {
							start = slot.Start
						}
						ev = buildEvent(o, d, si, start, ex.NewEnd, ex.DurationMinutes, now, planned)
						if ex.Income > 0 {
							ev.Income = ex.Income // 本次课时费按调整时指定的金额计
						}
						ev.ExceptionID = ex.ID
						ev.AdjustNote = ex.Note
						ev.AdjustType = adjustTime
						ev.Key += "-t"
					case model.ExceptionTypeMove:
						// 调整后仅保留新课时：原位置不再生成「已调至」占位，
						// 新课时由下方第②步按例外的生效日期生成
						continue
					}
				}
				events = append(events, ev)
			}
		}
	}

	// ② 追加「调入本节」与「临时加课」：目标日期落在当前区间内才展示
	for _, ex := range exs {
		if ex.NewDate <= 0 {
			continue
		}
		if ex.Type != model.ExceptionTypeMove && ex.Type != model.ExceptionTypeExtra {
			continue
		}
		nd := time.Unix(ex.NewDate, 0)
		if nd.Before(weekStart) || nd.After(weekEnd) {
			continue
		}
		start := ex.NewStart
		if start == "" {
			if ex.SlotIndex >= 0 && ex.SlotIndex < len(slots) {
				start = slots[ex.SlotIndex].Start
			} else if len(slots) > 0 {
				start = slots[0].Start
			}
		}
		if start == "" {
			start = "08:00"
		}
		ev := buildEvent(o, nd, ex.SlotIndex, start, ex.NewEnd, ex.DurationMinutes, now, planned)
		if ex.Income > 0 {
			ev.Income = ex.Income // 本次课时费按调整时指定的金额计
		}
		ev.ExceptionID = ex.ID
		ev.AdjustNote = ex.Note
		if ex.Type == model.ExceptionTypeMove {
			ev.AdjustType = adjustMovedIn
			ev.OriginDate = tsToDate(ex.SourceDate)
			ev.OriginWeekday = weekdayOfTs(ex.SourceDate)
			// 原课时完整起止时间：取调课路径首步的调整前位置（即最初的原定课时）
			if originSteps := parseSteps(&ex, o); len(originSteps) > 0 {
				ev.OriginStart = originSteps[0].FromStart
				ev.OriginEnd = originSteps[0].FromEnd
			}
			ev.Key = fmt.Sprintf("%d-ex%d-in", o.ID, ex.ID)
		} else {
			ev.AdjustType = adjustExtra
			ev.Key = fmt.Sprintf("%d-ex%d-extra", o.ID, ex.ID)
		}
		events = append(events, ev)
	}

	return events
}

// buildEvent 构造单节课程：课时长由本次对应的周时间段起止推算；
// 单次调整（如临时调时间/加课）可携带各自 durationOverride 覆盖，否则按时段起止计算。
func buildEvent(o *model.Order, day time.Time, slotIndex int, startClock, endClock string, durationOverride int, now time.Time, plannedLessons int) scheduleEvent {
	startMin := common.ClockToMinutes(startClock)
	endMin := common.ClockToMinutes(endClock)
	if endMin <= startMin {
		endMin = startMin + 60
	}
	duration := durationOverride
	if duration <= 0 {
		// 无覆盖时，按时段起止推算本次课时长（不同周时间段可不同）
		duration = endMin - startMin
	}
	endMin = startMin + duration
	return scheduleEvent{
		// key 必须全局唯一（含订单 ID 与时段序号），否则前端同 key 会被去重导致课程不显示
		Key:             fmt.Sprintf("%d-%s-%d-%s", o.ID, day.Format(common.DateLayout), slotIndex, startClock),
		OrderID:         o.ID,
		OrderNo:         o.OrderNo,
		Date:            day.Format(common.DateLayout),
		Weekday:         common.WeekdayISO(day),
		StartTime:       minutesToClock(startMin),
		EndTime:         minutesToClock(endMin),
		StartMinute:     startMin,
		EndMinute:       endMin,
		TimeState:       timeStateOf(day, startMin, endMin, now),
		Grade:           o.Grade,
		StudentName:     o.StudentName,
		Subject:         o.Subject,
		Address:         o.Address,
		Content:         o.Content,
		Remark:          o.Remark,
		RemarkFlag:      o.RemarkFlag,
		HourlyRate:      o.HourlyRate,
		Income:          lessonIncomeWithPlan(o, duration, plannedLessons),
		DurationMinutes: duration,
		Status:          o.Status,
		Direction:       o.Direction,
		SlotIndex:       slotIndex,
	}
}

// timeStateOf 判断课程相对当前时间的状态：past / ongoing / future。
func timeStateOf(day time.Time, startMin, endMin int, now time.Time) string {
	start := time.Date(day.Year(), day.Month(), day.Day(), startMin/60, startMin%60, 0, 0, day.Location())
	end := time.Date(day.Year(), day.Month(), day.Day(), endMin/60, endMin%60, 0, 0, day.Location())

	switch {
	case now.After(end):
		return "past"
	case now.Before(start):
		return "future"
	default:
		return "ongoing"
	}
}

// dayTs 返回当天 00:00 的时间戳。
func dayTs(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}

// tsToDate 时间戳转 yyyy-mm-dd，0 返回空串。
func tsToDate(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).Format(common.DateLayout)
}

// weekdayOfTs 返回时间戳对应星期（1=周一 ... 7=周日）。
func weekdayOfTs(ts int64) int {
	if ts <= 0 {
		return 0
	}
	return common.WeekdayISO(time.Unix(ts, 0))
}

// minutesToClock 分钟数转 HH:MM，超过当日 24:00 按 23:59 截断展示。
func minutesToClock(m int) string {
	if m < 0 {
		m = 0
	}
	if m > 1439 {
		m = 1439
	}
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// dayEventsOf 生成某用户在指定日期当天的全部课程（已套用例外），用于调课时的冲突检测。
// 历史日期（<= 昨日）以已物化的课次为准，未来 / 当天按订单配置实时展开，
// 因此无论是历史还是将来的课程，与调整后时段重叠时都会被检出。
func dayEventsOf(userID uint, day time.Time, excludeOrderID uint) ([]scheduleEvent, error) {
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayStr := day.Format(common.DateLayout)
	events := make([]scheduleEvent, 0)

	// 历史：直接读已物化课次（已删除 / 停课 / 已调出 / 已被调整替代的课次不参与冲突）
	from := dayTs(day)
	orders, exMap, err := loadOrdersForRange(userID, from, from, "")
	if err != nil {
		return nil, err
	}
	if day.Before(todayStart) {
		lessons, err := model.NewLessonModel().ListByUserRange(userID, dayStr, dayStr)
		if err != nil {
			return nil, err
		}
		for _, l := range lessons {
			if l.Voided || l.OrderID == excludeOrderID {
				continue
			}
			if l.Status == "canceled" || l.Status == "movedOut" || lessonReplaced(l, exMap[l.OrderID]) {
				continue
			}
			events = append(events, lessonToEvent(l, ""))
		}
		return events, nil
	}

	// 今天 / 未来：按订单配置 + 例外实时展开
	for i := range orders {
		o := orders[i]
		if o.ID == excludeOrderID {
			continue
		}
		for _, ev := range expandOrder(&o, exMap[o.ID], day, day, now) {
			if isExcludedFromSummary(ev.AdjustType) {
				continue
			}
			events = append(events, ev)
		}
	}
	return events, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// LessonDelete 软删除一节已物化的历史课次（如老师实际未上课）。仅标记 voided，不物理删除，
// 以免每日物化定时任务因去重识别为「缺失」而重新生成该课次。
func LessonDelete(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "课次 ID 有误")
		return
	}
	if _, err := model.NewLessonModel().Get(uint(id), user.ID); err != nil {
		Fail(c, CodeNotFound, "课次不存在")
		return
	}
	if err := model.NewLessonModel().SetVoided(uint(id), user.ID, true); err != nil {
		logger.Error("void lesson failed, err=%s", err.Error())
		ServerError(c, "删除课次失败")
		return
	}
	Success(c, "已删除该课次")
}
