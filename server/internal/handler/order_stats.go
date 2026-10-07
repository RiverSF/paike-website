package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// maxDashboardDays 仪表盘时间区间上限（天），防止一次展开过多数据。
const maxDashboardDays = 180

// OrderDashboard 订单数据仪表盘：在所选时间区间内按课程频次展开后统计
// 课程数、预计金额（区间内全部计划课程）、真实金额（已发生且未停课的课程）、课时数，
// 并提供每日趋势（课时数 / 课程数 / 金额）。
// 支持预设区间（range=7|30|90）、自定义起止日期（start/end）与收支筛选（direction=income|expense）。
// 金额口径：按订单方向归属（收入 / 开支），「按总费用」计费时按未取整的分摊值累加，保证区间总额准确。
func OrderDashboard(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	// 过期检查：自动结束已过期订单，保证统计口径与课表一致
	if err := model.NewOrderModel().SyncOrderStatus(user.ID); err != nil {
		logger.Error("finish expired orders failed, err=%s", err.Error())
	}

	from, to := resolveDashboardRange(c)
	now := time.Now()
	// 收支筛选：income / expense / 空（全部）
	direction := strings.TrimSpace(c.Query("direction"))
	if direction != model.DirectionIncome && direction != model.DirectionExpense {
		direction = ""
	}

	orders, exMap, err := loadOrdersForRange(user.ID, from.Unix(), to.Unix(), "")
	if err != nil {
		logger.Error("dashboard load orders failed, err=%s", err.Error())
		ServerError(c, "统计失败")
		return
	}
	// 方向过滤：按订单方向筛掉不参与本次统计的课程（历史课次按所属订单方向归属）
	keptSet := make(map[uint]bool, len(orders))
	if direction != "" {
		kept := make([]model.Order, 0, len(orders))
		for i := range orders {
			if orders[i].Direction == direction {
				kept = append(kept, orders[i])
				keptSet[orders[i].ID] = true
			}
		}
		orders = kept
	}

	type dayAgg struct {
		estimated float64
		real      float64
		lessons   int
		courses   map[uint]bool // 当天有课的不同课程（订单）集合：按「课程」维度统计用
	}
	dailyMap := make(map[string]*dayAgg, 0)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dailyMap[d.Format(common.DateLayout)] = &dayAgg{courses: map[uint]bool{}}
	}
	// 取当天聚合（超区间日期按需创建，保证不丢数据）
	aggOf := func(date string) *dayAgg {
		agg, ok := dailyMap[date]
		if !ok {
			agg = &dayAgg{courses: map[uint]bool{}}
			dailyMap[date] = agg
		}
		return agg
	}

	orderSet := make(map[uint]bool)

	// 历史部分（<= 昨日）：读已物化的 lesson 表（冻结快照），与订单配置解耦
	yesterday := now.AddDate(0, 0, -1)
	histEnd := to
	if histEnd.After(yesterday) {
		histEnd = yesterday
	}
	if !from.After(histEnd) {
		for i := range orders {
			MaterializeOrderLessons(&orders[i], histEnd)
		}
		histLessons, lerr := model.NewLessonModel().ListByUserRange(user.ID, from.Format(common.DateLayout), histEnd.Format(common.DateLayout))
		if lerr != nil {
			logger.Error("list lessons for dashboard failed, err=%s", lerr.Error())
		}
		for _, l := range histLessons {
			if l.Voided || isExcludedFromSummary(l.Status) || lessonReplaced(l, exMap[l.OrderID]) {
				// 已删除 / 停课 / 已调出 / 已被调整替代的课次不计入统计
				continue
			}
			if direction != "" && !keptSet[l.OrderID] {
				continue // 方向筛选：不属于所选方向的课程不参与统计
			}
			orderSet[l.OrderID] = true
			agg := aggOf(l.Date)
			agg.estimated += l.Income
			agg.real += l.Income
			agg.lessons++
			agg.courses[l.OrderID] = true
		}
	}

	// 未来部分（今日 ~ to）：按当前配置实时展开，改动即时生效
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !todayStart.After(to) {
		for i := range orders {
			o := orders[i]
			// 计划总节数：按总费用计费时用于分摊单节金额（其余计费方式不影响）
			planned := orderPlannedLessons(&o, exMap[o.ID])
			for _, ev := range expandOrder(&o, exMap[o.ID], todayStart, to, now) {
				if isExcludedFromSummary(ev.AdjustType) {
					continue // 停课 / 已调出的课不产生课时与收入
				}
				orderSet[o.ID] = true
				// 报表金额用未取整的分摊值，保证区间总额与课程总费用一致
				inc := lessonAmountExact(&o, ev.DurationMinutes, planned)
				agg := aggOf(ev.Date)
				agg.estimated += inc
				agg.lessons++
				agg.courses[o.ID] = true
				if ev.TimeState == "past" {
					agg.real += inc
				}
			}
		}
	}

	daily := make([]gin.H, 0, len(dailyMap))
	totalEstimated := 0.0
	totalReal := 0.0
	totalLessons := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(common.DateLayout)
		agg := dailyMap[key]
		if agg == nil {
			agg = &dayAgg{courses: map[uint]bool{}}
		}
		totalEstimated += agg.estimated
		totalReal += agg.real
		totalLessons += agg.lessons
		daily = append(daily, gin.H{
			"date":            key,
			"estimatedIncome": round2(agg.estimated),
			"realIncome":      round2(agg.real),
			"lessonCount":     agg.lessons,      // 课时维度：当天课时数
			"courseCount":     len(agg.courses), // 课程维度：当天有课的课程数
		})
	}

	OK(c, gin.H{
		"rangeStart": from.Format(common.DateLayout),
		"rangeEnd":   to.Format(common.DateLayout),
		"summary": gin.H{
			"orderCount":      len(orderSet), // 课程维度：区间内涉及课程数
			"estimatedIncome": round2(totalEstimated),
			"realIncome":      round2(totalReal),
			"lessonCount":     totalLessons, // 课时维度：区间内课时数
			"courseCount":     len(orderSet),
		},
		"daily": daily,
	})
}

// resolveDashboardRange 解析仪表盘时间区间：
// 优先自定义 start/end；否则按 range 预设（默认近 30 天，结束日为今天）。
func resolveDashboardRange(c *gin.Context) (time.Time, time.Time) {
	from := common.ParseDate(c.Query("start"))
	to := common.ParseDate(c.Query("end"))
	if !from.IsZero() && !to.IsZero() && !to.Before(from) {
		if n := int(to.Sub(from).Hours()/24) + 1; n > maxDashboardDays {
			to = from.AddDate(0, 0, maxDashboardDays-1)
		}
		return from, to
	}
	days, _ := strconv.Atoi(c.DefaultQuery("range", "30"))
	if days <= 0 {
		days = 30
	}
	if days > maxDashboardDays {
		days = maxDashboardDays
	}
	base := time.Now()
	if d := c.Query("date"); d != "" {
		if t := common.ParseDate(d); !t.IsZero() {
			base = t
		}
	}
	end := base
	start := base.AddDate(0, 0, -(days - 1))
	return start, end
}
