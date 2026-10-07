package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// 订单进度（课时统计）口径：
//   - 已上（done）：从补课开始日期到今天，按每周时段展开、日期不晚于今天，
//     且未被「停课 / 已改期（原位置）」占用、未被手动作废的课次；改期后的新日期与临时加课一并计入。
//   - 预计（planned）：优先取订单填写的「总课时」；未填时按补课周期 × 每周时段推算。
//     长期订单（无结束日期）且未填总课时时 planned = 0，前端只展示「已上 N 节」。
//
// 说明：停课不计入已上（视为顺延）；作废课次（老师实际未上课）同样剔除。
type orderProgress struct {
	Planned int `json:"planned"` // 预计总节数，0 表示未知
	Done    int `json:"done"`    // 已上节数
}

// buildOrderProgress 批量计算订单进度：一次查询例外与作废课次，避免逐单查询。
func buildOrderProgress(userID uint, orders []model.Order) map[uint]orderProgress {
	out := make(map[uint]orderProgress, len(orders))
	if len(orders) == 0 {
		return out
	}
	ids := make([]uint, 0, len(orders))
	for i := range orders {
		ids = append(ids, orders[i].ID)
	}
	exMap, err := model.NewLessonExceptionModel().ListByOrderIDs(userID, ids)
	if err != nil {
		logger.Error("load lesson exceptions for progress failed, err=%s", err.Error())
		exMap = map[uint][]model.LessonException{}
	}
	voidedMap, err := model.NewLessonModel().CountVoidedByOrders(userID, ids)
	if err != nil {
		logger.Error("count voided lessons for progress failed, err=%s", err.Error())
		voidedMap = map[uint]int{}
	}

	// 今天 00:00：当天已排课程视为「已上」，与课表把当天课程标为已发生/进行中的口径保持一致
	today := time.Unix(todayTs(), 0)
	for i := range orders {
		o := &orders[i]
		out[o.ID] = computeOrderProgress(o, exMap[o.ID], voidedMap[o.ID], today)
	}
	return out
}

// orderVOWithProgress 单条订单输出：附上课程进度（用于创建 / 编辑 / 状态变更的返回）。
func orderVOWithProgress(userID uint, o *model.Order) gin.H {
	vo := orderVO(o)
	if p, ok := buildOrderProgress(userID, []model.Order{*o})[o.ID]; ok {
		vo["progress"] = p
	}
	return vo
}

// computeOrderProgress 计算单个订单的进度。
func computeOrderProgress(o *model.Order, exs []model.LessonException, voided int, today time.Time) orderProgress {
	var p orderProgress
	if len(o.Slots()) == 0 || o.StartDate <= 0 {
		// 无开始日期或每周时段：无法按周期推算，只回显用户填写的总课时
		p.Planned = o.TotalLessons
		return p
	}
	start := time.Unix(o.StartDate, 0)

	// 已上：截到今天（或订单结束日期，取更早者）
	doneTo := today
	if o.EndDate > 0 {
		if end := time.Unix(o.EndDate, 0); end.Before(doneTo) {
			doneTo = end
		}
	}
	p.Done = countLessonsInRange(o, exs, start, doneTo)
	if voided > 0 {
		p.Done -= voided
		if p.Done < 0 {
			p.Done = 0
		}
	}

	// 预计：用户填写的总课时优先，其次按周期推算
	switch {
	case o.TotalLessons > 0:
		p.Planned = o.TotalLessons
	case o.EndDate > 0:
		p.Planned = countLessonsInRange(o, exs, start, time.Unix(o.EndDate, 0))
	}
	if p.Planned > 0 && p.Done > p.Planned {
		p.Done = p.Planned // 加课等导致超出计划时，进度不超过 100%
	}
	return p
}

// orderPlannedLessons 订单的计划总节数：优先取用户填写的「总课时」，
// 未填时按补课周期 × 每周时段推算；长期订单（无结束日期）且未填总课时时返回 0（表示未知）。
// 「按总费用」计费方式依赖该值把整期费用分摊到每节课。
func orderPlannedLessons(o *model.Order, exs []model.LessonException) int {
	if o.TotalLessons > 0 {
		return o.TotalLessons
	}
	if o.StartDate > 0 && o.EndDate > 0 {
		return countLessonsInRange(o, exs, time.Unix(o.StartDate, 0), time.Unix(o.EndDate, 0))
	}
	return 0
}

// countLessonsInRange 统计订单在 [from, to] 区间内按周期与每周时段展开的课次数量。
// 停课与「已改期」的原位置不计入；改期后的新日期与临时加课在目标日期落入区间时计入。
// 与课表展开（expandOrder）保持同一口径：时段生效区间（slot.Active）内才生成课次。
func countLessonsInRange(o *model.Order, exs []model.LessonException, from, to time.Time) int {
	slots := o.Slots()
	if len(slots) == 0 || to.Before(from) {
		return 0
	}

	// 原位置被停课 / 改期的课时不计数（改期后的课次由下方按 newDate 计入）
	type slotKey struct {
		date int64
		idx  int
	}
	blocked := map[slotKey]bool{}
	for _, e := range exs {
		if e.Type == model.ExceptionTypeCancel || e.Type == model.ExceptionTypeMove {
			blocked[slotKey{e.SourceDate, e.SlotIndex}] = true
		}
	}

	n := 0
	from = common.StartOfDay(from)
	to = common.StartOfDay(to)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		wd := common.WeekdayISO(d)
		dTs := dayTs(d)
		for si, slot := range slots {
			if !common.ContainsInt(slot.Days, wd) || !slot.Active(d) {
				continue
			}
			if blocked[slotKey{dTs, si}] {
				continue
			}
			n++
		}
	}

	// 改期后的课次与临时加课：目标日期落在区间内时计入
	fromTs, toTs := from.Unix(), to.Unix()
	for _, e := range exs {
		if e.Type != model.ExceptionTypeMove && e.Type != model.ExceptionTypeExtra {
			continue
		}
		if e.NewDate <= 0 || e.NewDate < fromTs || e.NewDate > toTs {
			continue
		}
		n++
	}
	return n
}
