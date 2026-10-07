package handler

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

type orderReq struct {
	Source           string             `json:"source"`
	PublishedAt      string             `json:"publishedAt"`
	Publisher        string             `json:"publisher"`
	Grade            string             `json:"grade"`
	OrderNo          string             `json:"orderNo"`
	Address          string             `json:"address"`
	Subject          string             `json:"subject"`
	Subjects         []string           `json:"subjects"`
	Content          string             `json:"content"`
	StudentName      string             `json:"studentName"`
	StartDate        string             `json:"startDate"`
	EndDate          string             `json:"endDate"`
	TotalLessons     int                `json:"totalLessons"` // 计划总课时（节），选填；按课时购买、不明确结束日期时填写
	WeeklySlots      []model.WeeklySlot `json:"weeklySlots"`
	BillMode         string             `json:"billMode"`
	HourlyRate       float64            `json:"hourlyRate"`
	LessonPrice      float64            `json:"lessonPrice"`
	TotalAmount      float64            `json:"totalAmount"` // 整期总费用（元），billMode=total 生效
	Direction        string             `json:"direction"`   // 方向：income=收入 expense=开支；空则按注册身份推导
	StudentSituation string             `json:"studentSituation"`
	StudentGender    string             `json:"studentGender"`
	ContactPhone     string             `json:"contactPhone"`
	Remark           string             `json:"remark"`
	RemarkFlag       bool               `json:"remarkFlag"`
	Status           string             `json:"status"`
}

// generateOrderNo 自动生成订单编号：前缀-YYYYMMDD-4位随机数。
// 前缀按角色取值：站长=OWNER、管理员=ADMIN、普通用户=ORD。通过 ExistsOrderNo 去重，最多重试 8 次。
func generateOrderNo(userID uint) string {
	prefix := "ORD"
	if u, err := model.NewUserModel().GetByID(userID); err == nil {
		switch u.Role {
		case "owner":
			prefix = "OWNER"
		case "admin":
			prefix = "ADMIN"
		}
	}
	date := time.Now().Format("20060102")
	for i := 0; i < 8; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			n = big.NewInt(int64(time.Now().Nanosecond() % 10000))
		}
		cand := fmt.Sprintf("%s-%s-%04d", prefix, date, n.Int64())
		if !model.NewOrderModel().ExistsOrderNo(userID, cand) {
			return cand
		}
	}
	// 极端碰撞情况下叠加时间戳后缀确保唯一
	return fmt.Sprintf("%s-%s-%d", prefix, date, time.Now().UnixNano()%10000)
}

// formatOrderDate 时间戳转 yyyy-mm-dd，0 或空返回空串。
func formatOrderDate(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).Format(common.DateLayout)
}

// todayTs 返回今天 00:00 的时间戳。
func todayTs() int64 {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
}

// dateToTs 把 yyyy-mm-dd 转换为当天 00:00 的时间戳，空/非法返回 0。
func dateToTs(s string) int64 {
	t := common.ParseDate(s)
	if t.IsZero() {
		return 0
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}

// orderVO 订单输出：日期统一为 yyyy-mm-dd，发布时间为年月日。
func orderVO(o *model.Order) gin.H {
	pub := ""
	if !o.PublishedAt.IsZero() { // 零值时间表示「未填写」，该列不允许 NULL
		pub = o.PublishedAt.Format(common.DateLayout)
	}
	return gin.H{
		"id":               o.ID,
		"userId":           o.UserID,
		"source":           o.Source,
		"publishedAt":      pub,
		"publisher":        o.Publisher,
		"grade":            o.Grade,
		"orderNo":          o.OrderNo,
		"address":          o.Address,
		"subject":          o.Subject,
		"content":          o.Content,
		"studentName":      o.StudentName,
		"startDate":        formatOrderDate(o.StartDate),
		"endDate":          formatOrderDate(o.EndDate),
		"totalLessons":     o.TotalLessons,
		"weeklySlots":      o.WeeklySlots,
		"hourlyRate":       o.HourlyRate,
		"billMode":         o.BillMode,
		"lessonPrice":      o.LessonPrice,
		"totalAmount":      o.TotalAmount,
		"direction":        o.Direction,
		"directionName":    model.DirectionName[o.Direction],
		"subjects":         o.SubjectsList(),
		"studentSituation": o.StudentSituation,
		"studentGender":    o.StudentGender,
		"contactPhone":     o.ContactPhone,
		"remark":           o.Remark,
		"remarkFlag":       o.RemarkFlag,
		"status":           o.Status,
		"createdAt":        o.CreatedAt,
		"updatedAt":        o.UpdatedAt,
	}
}

// OrderList 订单列表（分页 + 状态/关键词筛选）。
func OrderList(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	om := model.NewOrderModel()
	// 状态同步：过期订单自动结束；未开始订单（开始日期晚于今日）按日期判定
	if err := om.SyncOrderStatus(user.ID); err != nil {
		logger.Error("finish expired orders failed, err=%s", err.Error())
	}

	list, total, err := om.List(model.OrderQuery{
		UserID:    user.ID,
		Status:    c.Query("status"),
		Keyword:   c.Query("keyword"),
		Grade:     c.Query("grade"),
		Subject:   c.Query("subject"),
		Direction: c.Query("direction"), // 收支筛选：income / expense
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		logger.Error("list order failed, err=%s", err.Error())
		ServerError(c, "查询订单失败")
		return
	}
	// 课程进度：已上 / 预计节数（批量计算，避免逐单查询）
	progress := buildOrderProgress(user.ID, list)
	items := make([]gin.H, 0, len(list))
	for i := range list {
		vo := orderVO(&list[i])
		if p, ok := progress[list[i].ID]; ok {
			vo["progress"] = p
		}
		items = append(items, vo)
	}
	OK(c, PageData{List: items, Total: total, Page: page, PageSize: pageSize})
}

// OrderCreate 新增订单。
func OrderCreate(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req orderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}
	if msg := validateOrder(&req); msg != "" {
		BadRequest(c, msg)
		return
	}

	// 免费试用期间最多可添加 3 门课程（付费会员不限），促进试用转化
	if user.MemberType == model.MemberTypeTrial && !user.IsStaff() {
		const trialCourseLimit = 3
		cnt, err := model.NewOrderModel().CountByUser(user.ID)
		if err != nil {
			logger.Error("count user orders failed, err=%s", err.Error())
			ServerError(c, "新增订单失败")
			return
		}
		if cnt >= trialCourseLimit {
			Fail(c, CodeForbidden, "免费试用期间最多可添加 3 门课程，开通会员后不限课程数量")
			return
		}
	}

	// 方向未指定时按注册身份推导默认值（老师 / 机构=收入，家长 / 个人=开支）
	if strings.TrimSpace(req.Direction) == "" {
		req.Direction = model.DefaultDirection(user.UserRole)
	}

	// 订单编号未填写时自动生成：前缀-YYYYMMDD-4位随机数
	if strings.TrimSpace(req.OrderNo) == "" {
		req.OrderNo = generateOrderNo(user.ID)
	}

	o := buildOrder(user.ID, &req)
	if err := model.NewOrderModel().Create(o); err != nil {
		logger.Error("create order failed, err=%s", err.Error())
		ServerError(c, "新增订单失败")
		return
	}
	// 兜底物化：把该订单截至昨日的历史课次落库（新订单一般无历史，幂等无副作用）
	MaterializeOrderLessons(o, time.Now().AddDate(0, 0, -1))
	OK(c, orderVOWithProgress(user.ID, o))
}

// OrderUpdate 编辑订单（全量字段）。
func OrderUpdate(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "订单 ID 有误")
		return
	}

	old, err := model.NewOrderModel().Get(uint(id), user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}

	var req orderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}
	// 已结束订单仅可查看，不允许再编辑修改
	if old.Status == model.OrderStatusFinished {
		Fail(c, CodeBadRequest, "已结束的订单不可再编辑")
		return
	}
	if msg := validateOrder(&req); msg != "" {
		BadRequest(c, msg)
		return
	}
	// 订单编号被清空时保留原始编号，不重新生成（避免改变既有业务编号）
	if strings.TrimSpace(req.OrderNo) == "" {
		req.OrderNo = old.OrderNo
	}
	// 方向：未指定时沿用原值，仍为空则按注册身份推导
	if strings.TrimSpace(req.Direction) == "" {
		req.Direction = old.Direction
	}
	if strings.TrimSpace(req.Direction) == "" {
		req.Direction = model.DefaultDirection(user.UserRole)
	}
	// 状态联动：切换为已结束时把结束日期同步为今日
	if req.Status == model.OrderStatusFinished {
		req.EndDate = time.Now().Format(common.DateLayout)
	}

	// 兜底物化：保存前先把该订单截至昨日的历史课次落库，使本次改动只影响今日及以后，历史与订单彻底解耦
	MaterializeOrderLessons(old, time.Now().AddDate(0, 0, -1))

	updated := buildOrder(user.ID, &req)
	updated.ID = old.ID
	updated.UserID = user.ID
	updated.CreatedAt = old.CreatedAt
	if err := model.NewOrderModel().Save(updated); err != nil {
		logger.Error("update order failed, err=%s", err.Error())
		ServerError(c, "修改订单失败")
		return
	}
	OK(c, orderVOWithProgress(user.ID, updated))
}

// OrderUpdateStatus 仅修改订单状态（进行中 / 已结束 ...）。
func OrderUpdateStatus(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "订单 ID 有误")
		return
	}
	o, err := model.NewOrderModel().Get(uint(id), user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误")
		return
	}
	if _, ok := model.OrderStatusName[req.Status]; !ok {
		BadRequest(c, "订单状态不合法")
		return
	}
	// 已结束订单不可再切换为其他状态
	if o.Status == model.OrderStatusFinished {
		Fail(c, CodeBadRequest, "已结束的订单不可再修改状态")
		return
	}
	// 进行中 / 已结束不可变更为「未开始」：未开始仅由开始日期自动判定，不可手动回退
	if req.Status == model.OrderStatusPending && o.Status != model.OrderStatusPending {
		Fail(c, CodeBadRequest, "进行中或已结束的订单不可变更为未开始")
		return
	}

	// 未开始 → 进行中：开始日期可能仍在未来，做时间校验并自动校正为今日，
	// 使状态与日期一致（否则会被 SyncOrderStatus 再次判定为未开始）。
	if o.Status == model.OrderStatusPending && req.Status == model.OrderStatusRunning && o.StartDate > todayTs() {
		o.StartDate = todayTs()
	}
	o.Status = req.Status
	// 置为已结束时，同步把补课结束日期更新为今日
	if req.Status == model.OrderStatusFinished {
		o.EndDate = todayTs()
	}
	if err := model.NewOrderModel().Save(o); err != nil {
		logger.Error("update order status failed, err=%s", err.Error())
		ServerError(c, "修改状态失败")
		return
	}
	OK(c, orderVOWithProgress(user.ID, o))
}

// OrderDelete 删除订单。
func OrderDelete(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "订单 ID 有误")
		return
	}
	o, err := model.NewOrderModel().Get(uint(id), user.ID)
	if err != nil {
		Fail(c, CodeNotFound, "订单不存在")
		return
	}
	// 进行中 / 已结束的订单保留历史记录，不可删除；仅「未开始」订单（尚无历史课次）可删除
	if o.Status == model.OrderStatusRunning || o.Status == model.OrderStatusFinished {
		Fail(c, CodeBadRequest, "进行中或已结束的订单不可删除，可保留历史记录")
		return
	}
	if err := model.NewOrderModel().Delete(uint(id), user.ID); err != nil {
		logger.Error("delete order failed, err=%s", err.Error())
		ServerError(c, "删除订单失败")
		return
	}
	// 订单删除后其调课记录已无意义，一并清理
	if err := model.NewLessonExceptionModel().DeleteByOrder(uint(id), user.ID); err != nil {
		logger.Error("delete lesson exceptions failed, err=%s", err.Error())
	}
	Success(c, "删除成功")
}

// OrderOptions 订单可选的枚举（状态、年级、科目），便于前端下拉复用。
func OrderOptions(c *gin.Context) {
	OK(c, gin.H{
		"status":   model.OrderStatusName,
		"grades":   defaultGrades(),
		"subjects": defaultSubjects(),
	})
}

func validateOrder(req *orderReq) string {
	if strings.TrimSpace(req.Grade) == "" {
		return "请填写年级"
	}
	if strings.TrimSpace(req.Address) == "" {
		return "请填写地址"
	}
	if strings.TrimSpace(req.StartDate) == "" {
		return "请填写补课开始时间"
	}
	if common.ParseDate(req.StartDate).IsZero() {
		return "补课开始时间格式有误，应为 yyyy-mm-dd"
	}
	if strings.TrimSpace(req.EndDate) != "" && common.ParseDate(req.EndDate).IsZero() {
		return "补课结束时间格式有误，应为 yyyy-mm-dd"
	}
	start := common.ParseDate(req.StartDate)
	end := common.ParseDate(req.EndDate)
	if !end.IsZero() && end.Before(start) {
		return "补课结束时间不能早于开始时间"
	}
	if len(req.WeeklySlots) == 0 {
		return "请添加周时间段"
	}
	for _, s := range req.WeeklySlots {
		if _, _, ok := common.ParseClock(s.Start); !ok {
			return "周时间段开始时间格式有误，应为 HH:MM"
		}
		if _, _, ok := common.ParseClock(s.End); !ok {
			return "周时间段结束时间格式有误，应为 HH:MM"
		}
		if len(s.Days) == 0 {
			return "周时间段请选择星期"
		}
		for _, d := range s.Days {
			if d < 1 || d > 7 {
				return "星期取值应在 1-7 之间"
			}
		}
		if s.EffectiveFrom != "" && common.ParseDate(s.EffectiveFrom).IsZero() {
			return "生效起始日期格式有误，应为 yyyy-mm-dd"
		}
		if s.EffectiveTo != "" && common.ParseDate(s.EffectiveTo).IsZero() {
			return "生效截止日期格式有误，应为 yyyy-mm-dd"
		}
		if s.EffectiveFrom != "" && s.EffectiveTo != "" && common.ParseDate(s.EffectiveTo).Before(common.ParseDate(s.EffectiveFrom)) {
			return "生效截止日期不能早于生效起始日期"
		}
		// 生效区间与补课周期（均为含首尾的自然日）若无交集，则该时段永远不会生成课次，属明显误填
		if !end.IsZero() && !s.IntersectsPeriod(start, end) {
			return "时间段的生效日期与补课周期没有交集，请调整生效日期或补课周期"
		}
	}

	// 同一订单多个时间段：同一星期在相同生效周期内不可重复安排（避免同一周出现两种上课安排）
	for i := 0; i < len(req.WeeklySlots); i++ {
		for j := i + 1; j < len(req.WeeklySlots); j++ {
			a, b := req.WeeklySlots[i], req.WeeklySlots[j]
			shared := false
			for _, da := range a.Days {
				for _, db := range b.Days {
					if da == db {
						shared = true
						break
					}
				}
				if shared {
					break
				}
			}
			if !shared {
				continue
			}
			// 生效日期留空表示不设限：起始留空视为最早生效，截止留空视为一直生效
			af, at, bf, bt := a.EffectiveFrom, a.EffectiveTo, b.EffectiveFrom, b.EffectiveTo
			if (af == "" || bt == "" || af <= bt) && (bf == "" || at == "" || bf <= at) {
				return "多个时间段的生效周期在同一星期上重叠，同一周内同一星期只能有一种安排，请调整生效日期"
			}
		}
	}
	if req.Status == "" {
		req.Status = model.OrderStatusRunning
	}
	if _, ok := model.OrderStatusName[req.Status]; !ok {
		return "订单状态不合法"
	}
	if req.StudentGender != "" && req.StudentGender != "男" && req.StudentGender != "女" {
		return "学生性别仅支持 男 / 女"
	}
	// 总课时为选填：按课时购买（买了多少节）但不明确结束日期时填写，用于统计课程进度
	if req.TotalLessons < 0 || req.TotalLessons > 2000 {
		return "总课时需为 0-2000 节（留空表示按补课周期推算）"
	}
	// 方向：允许留空（由服务端按注册身份推导默认值），填写时必须是收入 / 开支
	if d := strings.TrimSpace(req.Direction); d != "" && d != model.DirectionIncome && d != model.DirectionExpense {
		return "订单方向仅支持收入或开支"
	}
	// 课程费用：三种计费方式各自校验对应金额
	switch strings.TrimSpace(req.BillMode) {
	case model.BillModeLesson:
		if req.LessonPrice <= 0 {
			return "请填写单次课时价"
		}
	case model.BillModeTotal:
		if req.TotalAmount <= 0 {
			return "请填写课程总费用"
		}
		// 总费用需要分摊到每节课，必须能确定计划总节数
		if req.TotalLessons <= 0 && strings.TrimSpace(req.EndDate) == "" {
			return "按总费用计费需填写总课时或结束日期，才能把费用分摊到每节课"
		}
	default: // hourly（默认）
		if req.HourlyRate <= 0 {
			return "请填写时薪"
		}
	}
	return ""
}

func buildOrder(userID uint, req *orderReq) *model.Order {
	slots := req.WeeklySlots
	if slots == nil {
		slots = []model.WeeklySlot{}
	}
	slotsJSON, _ := json.Marshal(slots)

	// 多选科目：使用去空白后的列表；单科目 subject 作为兼容字段（取首个多选科目）
	subjects := make([]string, 0, len(req.Subjects))
	for _, s := range req.Subjects {
		if t := strings.TrimSpace(s); t != "" {
			subjects = append(subjects, t)
		}
	}
	subjectsJSON, _ := json.Marshal(subjects)
	singleSubject := strings.TrimSpace(req.Subject)
	if singleSubject == "" && len(subjects) > 0 {
		singleSubject = subjects[0]
	}

	billMode := strings.TrimSpace(req.BillMode)
	if _, ok := model.BillModeName[billMode]; !ok {
		billMode = model.BillModeHourly
	}

	return &model.Order{
		UserID:           userID,
		Source:           strings.TrimSpace(req.Source),
		PublishedAt:      parseDateOnly(req.PublishedAt),
		Publisher:        strings.TrimSpace(req.Publisher),
		Grade:            strings.TrimSpace(req.Grade),
		OrderNo:          strings.TrimSpace(req.OrderNo),
		Address:          strings.TrimSpace(req.Address),
		Subject:          singleSubject,
		Subjects:         string(subjectsJSON),
		Content:          strings.TrimSpace(req.Content),
		StudentName:      strings.TrimSpace(req.StudentName),
		StartDate:        dateToTs(req.StartDate),
		EndDate:          dateToTs(req.EndDate),
		TotalLessons:     req.TotalLessons,
		WeeklySlots:      string(slotsJSON),
		BillMode:         billMode,
		HourlyRate:       req.HourlyRate,
		LessonPrice:      req.LessonPrice,
		TotalAmount:      req.TotalAmount,
		Direction:        strings.TrimSpace(req.Direction),
		StudentSituation: strings.TrimSpace(req.StudentSituation),
		StudentGender:    strings.TrimSpace(req.StudentGender),
		ContactPhone:     strings.TrimSpace(req.ContactPhone),
		Remark:           strings.TrimSpace(req.Remark),
		RemarkFlag:       req.RemarkFlag,
		Status:           req.Status,
	}
}

// parseDateOnly 解析发布时间（yyyy-mm-dd），为空或非法返回 nil。
// parseDateOnly 解析 yyyy-mm-dd；无法解析时返回零值时间，
// 对应 order.published_at 的「未填写」（该列不允许 NULL，不再使用 nil 表示）。
func parseDateOnly(s string) time.Time {
	return common.ParseDate(s)
}

func defaultGrades() []string {
	return []string{"小班", "中班", "大班", "一年级", "二年级", "三年级", "四年级", "五年级", "六年级", "初一", "初二", "初三", "高一", "高二", "高三", "成人"}
}

func defaultSubjects() []string {
	return []string{"语文", "数学", "英语", "物理", "化学", "生物", "政治", "历史", "地理", "奥数", "钢琴", "书法", "美术", "编程", "其他"}
}
