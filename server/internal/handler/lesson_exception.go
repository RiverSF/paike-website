package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// exceptionReq 单次课程调整请求：只影响指定的那一节课，不改变订单的每周固定频次。
type exceptionReq struct {
	OrderID         uint    `json:"orderId"`
	SourceDate      string  `json:"sourceDate"`      // 原定上课日期 yyyy-mm-dd
	SlotIndex       int     `json:"slotIndex"`       // 对应每周时段下标，加课传 -1
	Type            string  `json:"type"`            // cancel / move / time / extra
	NewDate         string  `json:"newDate"`         // 改期 / 加课的目标日期
	NewStart        string  `json:"newStart"`        // 调整后开始时间 HH:MM
	NewEnd          string  `json:"newEnd"`          // 调整后结束时间 HH:MM
	DurationMinutes int     `json:"durationMinutes"` // 本次课时长度，0 按起止时间 / 沿用订单
	Income          float64 `json:"income"`          // 本次课时费（元），0 按订单标准计费
	Note            string  `json:"note"`            // 调整原因
}

// adjustStep 调课路径中的一步：from 为调整前位置（首步=原定课时），to 为调整后位置。
// 停课步骤的 to 为空；路径最后一步的 to 即课程当前生效位置。
type adjustStep struct {
	FromDate  string  `json:"fromDate"`
	FromStart string  `json:"fromStart"`
	FromEnd   string  `json:"fromEnd"`
	ToDate    string  `json:"toDate"`
	ToStart   string  `json:"toStart"`
	ToEnd     string  `json:"toEnd"`
	Duration  int     `json:"duration"`
	Income    float64 `json:"income"` // 本次调整后的课时费；0 按订单标准计费
	Type      string  `json:"type"`   // move / cancel（首步类型）
	Note      string  `json:"note"`
	At        int64   `json:"at"` // 操作时间（秒级时间戳）
}

func marshalSteps(steps []adjustStep) string {
	b, err := json.Marshal(steps)
	if err != nil {
		logger.Error("marshal adjust steps failed, err=%s", err.Error())
		return ""
	}
	return string(b)
}

// slotStartClock 取订单每周时段的开始时间（下标非法时返回空串）。
func slotStartClock(o *model.Order, slotIndex int) string {
	slots := o.Slots()
	if slotIndex >= 0 && slotIndex < len(slots) {
		return slots[slotIndex].Start
	}
	return ""
}

// stepEndClock 按开始时间 + 课时长度推算结束时间（时长 0 时沿用订单时段结束）。
func stepEndClock(o *model.Order, slotIndex int, start string, duration int) string {
	if start == "" {
		return ""
	}
	startMin := common.ClockToMinutes(start)
	end := startMin + 60
	if duration > 0 {
		end = startMin + duration
	} else if slotIndex >= 0 && slotIndex < len(o.Slots()) {
		// 时段结束时间需晚于开始时间才可用（存量数据时段可能已变更或缺失）
		if slotEnd := common.ClockToMinutes(o.Slots()[slotIndex].End); slotEnd > startMin {
			end = slotEnd
		}
	}
	return minutesToClock(end)
}

// parseSteps 解析例外中的调课路径；补全每步的起止时间，
// 使每一步都能完整看出该次课时的调整前后的时间（含结束时间）。
func parseSteps(e *model.LessonException, o *model.Order) []adjustStep {
	steps := make([]adjustStep, 0)
	if e.History != "" {
		if err := json.Unmarshal([]byte(e.History), &steps); err != nil {
			steps = nil
		}
	}
	if len(steps) == 0 {
		// 兼容存量记录：按生效字段合成单步
		step := adjustStep{
			FromDate:  tsToDate(e.SourceDate),
			FromStart: slotStartClock(o, e.SlotIndex),
			Type:      e.Type,
			Note:      e.Note,
			At:        e.CreatedAt,
		}
		if e.Type != model.ExceptionTypeCancel {
			step.ToDate = tsToDate(e.NewDate)
			step.ToStart = e.NewStart
			if step.ToStart == "" {
				step.ToStart = step.FromStart
			}
			step.ToEnd = e.NewEnd
			step.Duration = e.DurationMinutes
		}
		steps = []adjustStep{step}
	}

	// 补全起止结束时间：from = 上一步的 to（首步为原定时段），to = 开始时间 + 时长 / 订单时段
	prevEnd := stepEndClock(o, e.SlotIndex, slotStartClock(o, e.SlotIndex), 0)
	for i := range steps {
		s := &steps[i]
		if s.FromStart == "" {
			s.FromStart = slotStartClock(o, e.SlotIndex)
		}
		if s.FromEnd == "" {
			s.FromEnd = prevEnd
		}
		if s.ToDate != "" {
			if s.ToStart == "" {
				s.ToStart = s.FromStart
			}
			if s.ToEnd == "" {
				s.ToEnd = stepEndClock(o, e.SlotIndex, s.ToStart, s.Duration)
			}
			// from 端仍缺结束时间时，按本次调整前后课时时长相同推算
			if s.FromEnd == "" && s.FromStart != "" {
				if d := common.ClockToMinutes(s.ToEnd) - common.ClockToMinutes(s.ToStart); d > 0 {
					s.FromEnd = minutesToClock(common.ClockToMinutes(s.FromStart) + d)
				}
			}
			prevEnd = s.ToEnd
		} else if s.FromEnd != "" {
			prevEnd = s.FromEnd
		}
	}
	// 兜底：仍缺结束时间的步骤按 60 分钟补全（存量数据 / 时段规则已变更的记录）
	for i := range steps {
		s := &steps[i]
		if s.FromStart != "" && s.FromEnd == "" {
			s.FromEnd = minutesToClock(common.ClockToMinutes(s.FromStart) + 60)
		}
		if s.ToDate != "" && s.ToStart != "" && s.ToEnd == "" {
			s.ToEnd = minutesToClock(common.ClockToMinutes(s.ToStart) + 60)
		}
	}
	return steps
}

// effectivePosition 例外的当前生效位置（最后一个非停课步骤的 to；仅停课时回落到原定课时）。
func effectivePosition(steps []adjustStep, e *model.LessonException) (string, string, string) {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].Type != model.ExceptionTypeCancel && steps[i].ToDate != "" {
			return steps[i].ToDate, steps[i].ToStart, steps[i].ToEnd
		}
	}
	return tsToDate(e.SourceDate), "", ""
}

func exceptionVO(e *model.LessonException, o *model.Order) gin.H {
	steps := parseSteps(e, o)
	return gin.H{
		"id":              e.ID,
		"orderId":         e.OrderID,
		"sourceDate":      tsToDate(e.SourceDate),
		"slotIndex":       e.SlotIndex,
		"type":            e.Type,
		"typeName":        model.ExceptionTypeName[e.Type],
		"newDate":         tsToDate(e.NewDate),
		"newStart":        e.NewStart,
		"newEnd":          e.NewEnd,
		"durationMinutes": e.DurationMinutes,
		"note":            e.Note,
		"history":         steps,
		"createdAt":       e.CreatedAt,
		"updatedAt":       e.UpdatedAt,
	}
}

// LessonExceptionList 某订单的调课记录（?orderId=）。
func LessonExceptionList(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	orderID, err := strconv.ParseUint(c.Query("orderId"), 10, 64)
	if err != nil || orderID == 0 {
		BadRequest(c, "订单 ID 有误")
		return
	}
	order, err := model.NewOrderModel().Get(uint(orderID), user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}

	list, err := model.NewLessonExceptionModel().ListByOrder(uint(orderID), user.ID)
	if err != nil {
		logger.Error("list lesson exceptions failed, err=%s", err.Error())
		ServerError(c, "查询调课记录失败")
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		items = append(items, exceptionVO(&list[i], order))
	}
	OK(c, gin.H{"list": items})
}

// LessonExceptionCreate 新建或追加一次课程调整。
// 同一节原定课程可连续调整：每次调整在调课路径上追加一步（保留完整路径），
// 生效字段始终等于路径最后一步；原课时不再单独展示，仅保留调整后的课时。
func LessonExceptionCreate(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req exceptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}
	if msg := validateException(&req); msg != "" {
		BadRequest(c, msg)
		return
	}

	order, err := model.NewOrderModel().Get(req.OrderID, user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}
	if order.Status == model.OrderStatusFinished {
		Fail(c, CodeBadRequest, "已结束的订单不可再调整课程")
		return
	}

	em := model.NewLessonExceptionModel()
	sourceTs := dateToTs(req.SourceDate)
	newTs := dateToTs(req.NewDate)

	e := &model.LessonException{
		UserID:          user.ID,
		OrderID:         req.OrderID,
		SourceDate:      sourceTs,
		SlotIndex:       req.SlotIndex,
		Type:            req.Type,
		NewDate:         newTs,
		NewStart:        req.NewStart,
		NewEnd:          req.NewEnd,
		DurationMinutes: req.DurationMinutes,
		Income:          req.Income,
		Note:            req.Note,
	}

	if req.Type == model.ExceptionTypeExtra {
		// 临时加课：不关联固定时段，可重复添加
		e.SourceDate = newTs
		e.SlotIndex = model.ExtraSlotIndex
		if err := em.Create(e); err != nil {
			logger.Error("create lesson exception failed, err=%s", err.Error())
			ServerError(c, "保存调整失败")
			return
		}
		syncMaterializedLessons(order, e, false)
		OK(c, gin.H{"item": exceptionVO(e, order), "conflicts": []gin.H{}})
		return
	}

	// 停课 / 调整时间：定位同一节原定课时的例外，在路径上追加一步
	old, err := em.FindSlot(req.OrderID, user.ID, sourceTs, req.SlotIndex)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.Error("find lesson exception failed, err=%s", err.Error())
		ServerError(c, "保存调整失败")
		return
	}

	var steps []adjustStep
	if old != nil {
		steps = parseSteps(old, order)
	}
	effDate, effStart, effEnd := effectivePosition(steps, e)
	if effStart == "" {
		effStart = slotStartClock(order, req.SlotIndex)
		effEnd = stepEndClock(order, req.SlotIndex, effStart, 0)
	}

	if req.Type == model.ExceptionTypeMove {
		// 改期与改时间已合并：日期可改为任意日期（含同日），起止时间可改为任意时间；至少调整一项
		sameDate := req.NewDate == effDate
		sameStart := req.NewStart == "" || req.NewStart == effStart
		sameEnd := req.NewEnd == "" || req.NewEnd == effEnd
		if sameDate && sameStart && sameEnd {
			BadRequest(c, "请调整日期或上课时间")
			return
		}
	}
	if old == nil && req.Type == model.ExceptionTypeMove && req.NewDate == req.SourceDate && req.NewStart == "" && req.NewEnd == "" {
		BadRequest(c, "请调整日期或上课时间")
		return
	}

	// 构造本次调整步骤：from=当前生效位置（含完整起止时间），to=本次调整后位置
	step := adjustStep{
		FromDate:  effDate,
		FromStart: effStart,
		FromEnd:   effEnd,
		Type:      req.Type,
		Note:      req.Note,
		At:        time.Now().Unix(),
	}
	if req.Type == model.ExceptionTypeMove {
		start := req.NewStart
		if start == "" {
			start = effStart
		}
		startMin := common.ClockToMinutes(start)
		// 结束时间优先：由起止差推算课时时长；时长未指定时沿用当前课时的时长（起止差），
		// 保证仅改日期时结束时间随开始时间平移
		dur := 0
		if req.NewEnd != "" {
			if endMin := common.ClockToMinutes(req.NewEnd); endMin > startMin {
				dur = endMin - startMin
			}
		}
		if dur <= 0 && req.DurationMinutes > 0 {
			dur = req.DurationMinutes
		}
		if dur <= 0 && effStart != "" && effEnd != "" {
			dur = common.ClockToMinutes(effEnd) - common.ClockToMinutes(effStart)
		}
		if dur <= 0 {
			dur = 60
		}
		step.ToDate = req.NewDate
		step.ToStart = start
		step.ToEnd = minutesToClock(startMin + dur)
		step.Duration = dur
		step.Income = req.Income
	}

	if old != nil {
		// 同一节课继续调整：追加路径步骤，生效字段更新为本次结果
		old.Type = e.Type
		old.NewDate = e.NewDate
		old.NewStart = e.NewStart
		old.NewEnd = e.NewEnd
		old.DurationMinutes = e.DurationMinutes
		old.Income = e.Income
		old.Note = e.Note
		old.History = marshalSteps(append(steps, step))
		if err := em.Save(old); err != nil {
			logger.Error("update lesson exception failed, err=%s", err.Error())
			ServerError(c, "保存调整失败")
			return
		}
		e = old
	} else {
		e.History = marshalSteps([]adjustStep{step})
		if err := em.Create(e); err != nil {
			logger.Error("create lesson exception failed, err=%s", err.Error())
			ServerError(c, "保存调整失败")
			return
		}
	}

	// 同步已物化的历史课次：原课次隐藏，仅保留调整后的课时
	syncMaterializedLessons(order, e, false)

	// 冲突提示：调整后的时段与该用户同一天（含历史与将来）的其它课程重叠时返回，仅提示不阻断
	conflicts := make([]gin.H, 0)
	if e.NewDate > 0 {
		startMin, endMin := exceptionMinutes(order, e)
		conflicts = detectConflicts(user.ID, order.ID, time.Unix(e.NewDate, 0), startMin, endMin)
	}

	OK(c, gin.H{"item": exceptionVO(e, order), "conflicts": conflicts})
}

// LessonExceptionDelete 撤销一次调整：仅回退到最后一次调整时间点（调课路径 pop 一步）。
// 路径只剩一步时撤销即删除例外、恢复为订单周期规则生成的原课次。
func LessonExceptionDelete(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "记录 ID 有误")
		return
	}
	e, err := model.NewLessonExceptionModel().Get(uint(id), user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "调整记录不存在")
		return
	}
	order, err := model.NewOrderModel().Get(e.OrderID, user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}

	if e.Type == model.ExceptionTypeExtra {
		// 临时加课：直接整条撤销
		if err := model.NewLessonExceptionModel().Delete(e.ID, user.ID); err != nil {
			logger.Error("delete lesson exception failed, err=%s", err.Error())
			ServerError(c, "撤销调整失败")
			return
		}
		syncMaterializedLessons(order, e, false)
		Success(c, "已撤销临时加课")
		return
	}

	steps := parseSteps(e, order)
	if len(steps) <= 1 {
		// 仅一次调整：撤销后完全恢复原课次
		if err := model.NewLessonExceptionModel().Delete(e.ID, user.ID); err != nil {
			logger.Error("delete lesson exception failed, err=%s", err.Error())
			ServerError(c, "撤销调整失败")
			return
		}
		syncMaterializedLessons(order, e, true)
		Success(c, "已撤销，课程已恢复为原定课时")
		return
	}

	// 多次调整：回退到最后一次调整时间点
	steps = steps[:len(steps)-1]
	last := steps[len(steps)-1]
	e.Type = last.Type
	if last.Type == model.ExceptionTypeCancel || last.ToDate == "" {
		e.Type = model.ExceptionTypeCancel
		e.NewDate = 0
		e.NewStart = ""
		e.NewEnd = ""
		e.DurationMinutes = 0
	} else {
		e.NewDate = dateToTs(last.ToDate)
		e.NewStart = func() string {
			if last.ToStart == last.FromStart {
				return "" // 沿用原时段开始时间
			}
			return last.ToStart
		}()
		e.NewEnd = ""
		e.DurationMinutes = last.Duration
		e.Income = last.Income
	}
	e.Note = last.Note
	e.History = marshalSteps(steps)
	if err := model.NewLessonExceptionModel().Save(e); err != nil {
		logger.Error("revert lesson exception failed, err=%s", err.Error())
		ServerError(c, "撤销调整失败")
		return
	}
	syncMaterializedLessons(order, e, false)
	Success(c, "已恢复到上一次调整的课时")
}

// syncMaterializedLessons 调整 / 撤销后同步已物化的历史课次，保证同一节课只保留调整后的那条：
//   - 例外生成的课次（改期调入/加课/停课占位）：物理删除，随后按例外当前生效状态重新物化
//     （这类课次可由展开逻辑随时重建，不存在「用户删除需保留」的语义）；
//   - 原生课次（exception_id=0，原定日期 + 槽位）：originKept=false 时软删除（由调整后的课次替代），
//     true（撤销回原位）时恢复；
//   - 最后对涉及的日期区间（仅历史）做一次物化兜底，补齐缺失的课次。
func syncMaterializedLessons(o *model.Order, ex *model.LessonException, originKept bool) {
	if o == nil || ex == nil {
		return
	}
	lm := model.NewLessonModel()
	srcDate := tsToDate(ex.SourceDate)
	effDate := tsToDate(ex.NewDate)
	if effDate == "" {
		effDate = srcDate // 停课：生效位置在原定日期（canceled 行）
	}

	// 例外生成的课次：全部物理删除（含历史各次调整产生的行），由下方物化按当前状态重建
	if ex.ID > 0 {
		if err := lm.DeleteByException(o.ID, ex.ID, o.UserID); err != nil {
			logger.Error("delete lessons by exception failed, err=%s", err.Error())
		}
	}

	// 原生课次：按是否回到原位恢复 / 隐藏
	list, err := lm.ListByOrderRange(o.ID, srcDate, srcDate)
	if err != nil {
		logger.Error("list lessons for sync failed, err=%s", err.Error())
		return
	}
	for _, l := range list {
		if l.ExceptionID != 0 || l.SlotIndex != ex.SlotIndex {
			continue
		}
		want := !originKept
		if l.Voided != want {
			if err := lm.SetVoided(l.ID, o.UserID, want); err != nil {
				logger.Error("sync void lesson failed, err=%s", err.Error())
			}
		}
	}

	// 物化兜底：仅覆盖历史区间（未来课次实时展开，无需落库）
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -1)
	from, to := srcDate, effDate
	if from > to {
		from, to = to, from
	}
	fromT := dateToTs(from)
	toT := dateToTs(to)
	if toT > yesterday.Unix() {
		toT = yesterday.Unix()
	}
	if fromT > toT {
		return
	}
	MaterializeOrderRange(o, time.Unix(fromT, 0), time.Unix(toT, 0))
}

func validateException(req *exceptionReq) string {
	if req.OrderID == 0 {
		return "请选择订单"
	}
	if _, ok := model.ExceptionTypeName[req.Type]; !ok {
		return "调整类型不合法"
	}
	if common.ParseDate(req.SourceDate).IsZero() {
		return "原定日期格式有误，应为 yyyy-mm-dd"
	}
	if req.NewStart != "" {
		if _, _, ok := common.ParseClock(req.NewStart); !ok {
			return "开始时间格式有误，应为 HH:MM"
		}
	}
	if req.NewEnd != "" {
		if _, _, ok := common.ParseClock(req.NewEnd); !ok {
			return "结束时间格式有误，应为 HH:MM"
		}
	}
	if req.DurationMinutes < 0 || req.DurationMinutes > 600 {
		return "课时长度应在 0-600 分钟之间"
	}
	if req.Income < 0 {
		req.Income = 0
	}

	switch req.Type {
	case model.ExceptionTypeCancel:
		req.NewDate = ""
		req.NewStart = ""
		req.NewEnd = ""
	case model.ExceptionTypeMove:
		if common.ParseDate(req.NewDate).IsZero() {
			return "请选择调整后的日期"
		}
		// 改期与改时间合并：日期可改为任意日期（含同日仅改时间）
	case model.ExceptionTypeExtra:
		if common.ParseDate(req.NewDate).IsZero() {
			return "请选择加课日期"
		}
		req.SlotIndex = model.ExtraSlotIndex
	case model.ExceptionTypeTime:
		if req.NewStart == "" && req.NewEnd == "" {
			return "请填写调整后的开始或结束时间"
		}
		req.NewDate = req.SourceDate
	}
	return ""
}

// exceptionMinutes 计算本次调整后的起止分钟数（与课表展开逻辑保持一致）。
func exceptionMinutes(o *model.Order, e *model.LessonException) (int, int) {
	start := common.ClockToMinutes(e.NewStart)
	if e.NewStart == "" {
		slots := o.Slots()
		if e.SlotIndex >= 0 && e.SlotIndex < len(slots) {
			start = common.ClockToMinutes(slots[e.SlotIndex].Start)
		}
	}
	duration := e.DurationMinutes
	if duration <= 0 {
		// 无显式时长时，按该周时间段自身的起止推算本次课时长
		if e.SlotIndex >= 0 {
			slots := o.Slots()
			if e.SlotIndex < len(slots) {
				duration = common.ClockToMinutes(slots[e.SlotIndex].End) - common.ClockToMinutes(slots[e.SlotIndex].Start)
			}
		}
	}
	end := common.ClockToMinutes(e.NewEnd)
	if duration > 0 {
		end = start + duration
	}
	if end <= start {
		end = start + 60
	}
	return start, end
}

// detectConflicts 检测目标日期同一时段是否已有其它订单的课程，用于提示（不阻断）。
func detectConflicts(userID, orderID uint, day time.Time, startMin, endMin int) []gin.H {
	events, err := dayEventsOf(userID, day, orderID)
	if err != nil {
		logger.Error("detect conflicts failed, err=%s", err.Error())
		return []gin.H{}
	}
	list := make([]gin.H, 0)
	for _, ev := range events {
		if startMin < ev.EndMinute && endMin > ev.StartMinute {
			list = append(list, gin.H{
				"date":        ev.Date,
				"startTime":   ev.StartTime,
				"endTime":     ev.EndTime,
				"studentName": ev.StudentName,
				"grade":       ev.Grade,
				"orderNo":     ev.OrderNo,
			})
		}
	}
	return list
}
