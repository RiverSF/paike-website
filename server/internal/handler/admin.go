package handler

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/common"
	"tutoring_server/pkg/logger"
)

// AdminUsers 后台用户列表：拥有者看全部账号，管理员只看普通会员。
func AdminUsers(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	onlyActive := c.Query("onlyActive") == "1"

	list, total, err := model.NewUserModel().AdminList(model.AdminUserQuery{
		Keyword:       c.Query("keyword"),
		Role:          c.Query("role"),
		Status:        c.Query("status"),
		MemberType:    c.Query("memberType"),
		InviteCode:    c.Query("inviteCode"),
		OnlyActive:    onlyActive,
		Page:          page,
		PageSize:      pageSize,
		ViewerIsOwner: viewer.Role == model.RoleOwner,
	})
	if err != nil {
		logger.Error("admin list users failed, err=%s", err.Error())
		ServerError(c, "查询用户失败")
		return
	}

	// 批量取邀请码，附到每条用户记录
	ids := make([]uint, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	codeMap, cerr := model.NewInviteCodeModel().CodeMapByUserIDs(ids)
	if cerr != nil {
		logger.Error("load invite codes failed, err=%s", cerr.Error())
		codeMap = map[uint]string{}
	}

	items := make([]gin.H, 0, len(list))
	for i := range list {
		vo := adminUserVO(&list[i])
		vo["inviteCode"] = codeMap[list[i].ID]
		items = append(items, vo)
	}
	OK(c, PageData{List: items, Total: total, Page: page, PageSize: pageSize})
}

// AdminCreateAdmin 站点拥有者添加管理员：仅需填写登录信息（用户名 + 密码）。
// 管理员无需邀请码与手机号，创建后即为永久会员，站点功能不受会员有效期限制。
func AdminCreateAdmin(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	if viewer.Role != model.RoleOwner {
		Fail(c, CodeForbidden, "仅站长可添加管理员")
		return
	}
	var req struct {
		Username string `json:"username" binding:"required,min=2,max=32"`
		Password string `json:"password" binding:"required,min=6,max=32"`
		Phone    string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "请完整填写用户名、手机号与密码（密码 6-32 位）")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if strings.ContainsAny(req.Username, " \t\r\n") {
		BadRequest(c, "用户名不能包含空格")
		return
	}
	req.Phone = strings.TrimSpace(req.Phone)
	if !phoneRegexp.MatchString(req.Phone) {
		BadRequest(c, "请填写正确的 11 位手机号")
		return
	}

	um := model.NewUserModel()
	if _, err := um.GetByUsername(req.Username); err == nil {
		BadRequest(c, "用户名已被使用")
		return
	} else if !model.IsNotFound(err) && !errors.Is(err, model.ErrUserNotFound) {
		logger.Error("query user failed, err=%s", err.Error())
		ServerError(c, "创建失败，请稍后重试")
		return
	}
	// 手机号是管理员的登录账号之一，需全局唯一
	if exists, err := um.ExistsPhone(req.Phone, 0); err != nil {
		logger.Error("check phone failed, err=%s", err.Error())
		ServerError(c, "创建失败，请稍后重试")
		return
	} else if exists {
		BadRequest(c, "该手机号已被使用")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ServerError(c, "创建失败，请稍后重试")
		return
	}
	now := time.Now()
	u := &model.User{
		Username:       req.Username,
		Password:       string(hash),
		Phone:          req.Phone,
		TeacherType:    model.TeacherTypeProfessional,
		RegTeacherType: model.TeacherTypeProfessional,
		MemberType:     model.MemberTypePermanent,
		MemberStart:    now.Unix(),
		MemberExpire:   model.PermanentExpireUnix(),
		Role:           model.RoleAdmin,
		UserRole:       "", // 管理员不绑定使用身份，统一按老师渲染
		RegisterSrc:    model.RegisterSrcAdmin, // 人工开通
	}
	if err := um.Create(u); err != nil {
		if model.IsDuplicateEntry(err) {
			BadRequest(c, "用户名已被使用")
			return
		}
		logger.Error("create admin failed, err=%s", err.Error())
		ServerError(c, "创建失败，请稍后重试")
		return
	}
	OK(c, adminUserVO(u))
}

// AdminResetPassword 后台为账号重置登录密码：无需原密码。
// 目标范围：普通用户与管理员（仅站长可操作），站长仅能重置自己的密码。
// password 留空时由服务端随机生成 12 位密码（大小写字母 + 数字，去除易混淆字符），
// 并在响应中返回明文，便于管理员直接转告用户。
func AdminResetPassword(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	target, errMsg := loadPwdTarget(c)
	if errMsg != "" {
		Fail(c, CodeBadRequest, errMsg)
		return
	}
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "新密码需为 6-32 位")
		return
	}

	plain := strings.TrimSpace(req.Password)
	generated := false
	if plain == "" {
		pwd, err := generateTempPassword()
		if err != nil {
			logger.Error("generate temp password failed, err=%s", err.Error())
			ServerError(c, "重置失败，请稍后重试")
			return
		}
		plain, generated = pwd, true
	}
	if len(plain) < 6 || len(plain) > 32 {
		BadRequest(c, "新密码需为 6-32 位")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		ServerError(c, "重置失败，请稍后重试")
		return
	}
	target.Password = string(hash)
	if err := model.NewUserModel().Save(target); err != nil {
		logger.Error("admin reset password failed, err=%s", err.Error())
		ServerError(c, "重置失败，请稍后重试")
		return
	}

	res := gin.H{"user": adminUserVO(target)}
	if generated {
		// 仅服务端随机生成时回传明文；管理员自定义的密码本就只有管理员自己知道
		res["password"] = plain
	}
	OK(c, res)
}

// generateTempPassword 生成 12 位临时密码：大小写字母 + 数字，去除易混淆字符（0/O/1/l/I）。
func generateTempPassword() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789"
	out := make([]byte, 12)
	max := big.NewInt(int64(len(chars)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = chars[n.Int64()]
	}
	return string(out), nil
}

// AdminStats 后台概览：用户统计 + 每日注册人数 / 有效会员人数趋势。
// 支持两种时间选择：自定义 start/end 起止日期，或 range 预设天数（默认 30）。
func AdminStats(c *gin.Context) {
	summary, err := model.NewUserModel().StatsSummary()
	if err != nil {
		logger.Error("admin stats summary failed, err=%s", err.Error())
		ServerError(c, "统计失败")
		return
	}

	// 优先使用自定义起止日期（来自趋势图的时间选择器）
	from := common.ParseDate(c.Query("start"))
	to := common.ParseDate(c.Query("end"))
	var daily []model.DailyStat
	if !from.IsZero() && !to.IsZero() && !to.Before(from) {
		daily, err = model.NewUserModel().DailyStatsRange(from, to)
	} else {
		days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
		if days <= 0 {
			days = 30
		}
		daily, err = model.NewUserModel().DailyStats(days)
	}
	if err != nil {
		logger.Error("admin daily stats failed, err=%s", err.Error())
		ServerError(c, "统计失败")
		return
	}
	OK(c, gin.H{
		"summary":    summary,
		"daily":      daily,
		"rangeStart": from.Format(common.DateLayout),
		"rangeEnd":   to.Format(common.DateLayout),
	})
}

// AdminFreeze 冻结 / 解冻账号（普通用户与管理员均可，站长账号受保护）。
func AdminFreeze(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	target, err := loadStaffTarget(c)
	if err != "" {
		Fail(c, CodeBadRequest, err)
		return
	}
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req struct {
		Frozen bool `json:"frozen"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误")
		return
	}

	if req.Frozen {
		target.Status = model.UserStatusFrozen
	} else {
		target.Status = model.UserStatusNormal
	}
	if err := model.NewUserModel().Save(target); err != nil {
		logger.Error("admin freeze user failed, err=%s", err.Error())
		ServerError(c, "操作失败")
		return
	}
	Success(c, map[bool]string{true: "已冻结", false: "已解冻"}[req.Frozen])
}

// AdminRenew 为普通会员续费：填写支付金额 + 续费时长（一月 / 一年）。
// 自动累加累计充值金额、计算新的会员到期时间，并生成支付记录。
func AdminRenew(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	target, errMsg := loadTarget(c)
	if errMsg != "" {
		Fail(c, CodeBadRequest, errMsg)
		return
	}
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Period string  `json:"period"`
		Days   int     `json:"days"` // 仅 period=daily 时生效
		Remark string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误")
		return
	}
	if req.Amount < 0 {
		BadRequest(c, "支付金额不能为负数")
		return
	}

	period := strings.ToLower(req.Period)
	record, errMsg := renewMember(target, renewInput{
		Amount:       req.Amount,
		Period:       period,
		Days:         req.Days,
		OperatorID:   viewer.ID,
		OperatorName: viewer.Username,
		Source:       "admin",
		Remark:       req.Remark,
	})
	if errMsg != "" {
		BadRequest(c, errMsg)
		return
	}

	OK(c, gin.H{"user": adminUserVO(target), "payment": record})
}

// AdminSetRole 站点拥有者设置 / 取消管理员（仅操作普通会员与管理员账号）。
func AdminSetRole(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	if viewer.Role != model.RoleOwner {
		Fail(c, CodeForbidden, "仅站点拥有者可以设置管理员")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "用户 ID 有误")
		return
	}
	if uint(id) == viewer.ID {
		BadRequest(c, "不能修改自己的角色")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误")
		return
	}
	if req.Role != model.RoleAdmin && req.Role != model.RoleUser {
		BadRequest(c, "角色仅支持 admin（管理员）或 user（普通用户）")
		return
	}

	target, err := model.NewUserModel().GetByID(uint(id))
	if err != nil {
		Fail(c, CodeNotFound, "用户不存在")
		return
	}
	if target.Role == model.RoleOwner {
		BadRequest(c, "不能修改站点拥有者的角色")
		return
	}

	target.Role = req.Role
	// 站长 / 管理员不绑定使用身份（身份字段留空，统一按老师渲染）；退回普通用户时给默认「老师」
	if target.Role == model.RoleAdmin {
		target.UserRole = ""
	} else {
		target.UserRole = model.UserRoleTeacher
	}
	if err := model.NewUserModel().Save(target); err != nil {
		logger.Error("admin set role failed, err=%s", err.Error())
		ServerError(c, "操作失败")
		return
	}
	OK(c, adminUserVO(target))
}

// AdminDeleteUser 站长删除账号（用于清理误加的管理员等）。
// 仅站点拥有者可删；不能删自己与站长账号；存在课程订单 / 充值流水的账号一律拒绝，
// 避免历史教学与财务记录失去归属，此类账号请改用「取消管理员」。
func AdminDeleteUser(c *gin.Context) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	if viewer.Role != model.RoleOwner {
		Fail(c, CodeForbidden, "仅站点拥有者可以删除账号")
		return
	}
	target, errMsg := loadStaffTarget(c)
	if errMsg != "" {
		Fail(c, CodeBadRequest, errMsg)
		return
	}

	um := model.NewUserModel()
	has, err := um.HasTeachingData(target.ID)
	if err != nil {
		logger.Error("check teaching data failed, err=%s", err.Error())
		ServerError(c, "删除失败，请稍后重试")
		return
	}
	if has {
		BadRequest(c, "该账号存在课程订单或充值记录，为保留历史数据不能删除，请改为取消其管理员身份")
		return
	}

	if err := um.DeleteAccount(target.ID); err != nil {
		logger.Error("admin delete user failed, id=%d err=%s", target.ID, err.Error())
		ServerError(c, "删除失败，请稍后重试")
		return
	}
	Success(c, "账号已删除")
}

// AdminFeedbacks 后台反馈列表（默认只看普通会员提交的内容）。
func AdminFeedbacks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	list, total, err := model.NewFeedbackModel().AdminList(model.FeedbackQuery{
		Status:         c.Query("status"),
		OnlyNormalUser: c.Query("all") != "1",
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		logger.Error("admin list feedback failed, err=%s", err.Error())
		ServerError(c, "查询反馈失败")
		return
	}
	OK(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

// AdminFeedbackReply 答复反馈并设置审核状态。
func AdminFeedbackReply(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		BadRequest(c, "反馈 ID 有误")
		return
	}

	var req struct {
		Reply  string `json:"reply"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误")
		return
	}
	if req.Status != "" {
		if _, ok := model.FeedbackStatusName[req.Status]; !ok {
			BadRequest(c, "状态不合法")
			return
		}
	}

	var fb model.Feedback
	if err := model.DB().Where("id = ?", id).First(&fb).Error; err != nil {
		Fail(c, CodeNotFound, "反馈不存在")
		return
	}

	fb.Reply = strings.TrimSpace(req.Reply)
	if req.Status != "" {
		fb.Status = req.Status
	} else if fb.Reply != "" {
		fb.Status = model.FeedbackResolved
	}
	if err := model.DB().Save(&fb).Error; err != nil {
		logger.Error("admin reply feedback failed, err=%s", err.Error())
		ServerError(c, "保存失败")
		return
	}
	OK(c, fb)
}

// loadTarget 解析并校验被操作的用户：必须是普通会员且不能是自己。
func loadTarget(c *gin.Context) (*model.User, string) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		return nil, "请先登录"
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, "用户 ID 有误"
	}
	if uint(id) == viewer.ID {
		return nil, "不能操作自己的账号"
	}

	target, err := model.NewUserModel().GetByID(uint(id))
	if err != nil {
		return nil, "用户不存在"
	}
	if target.Role != model.RoleUser {
		return nil, "只能对普通会员账号操作"
	}
	return target, ""
}

// loadStaffTarget 解析并校验可冻结 / 可删除的账号：普通用户或管理员，且不能是自己、不能是站长。
func loadStaffTarget(c *gin.Context) (*model.User, string) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		return nil, "请先登录"
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, "用户 ID 有误"
	}
	if uint(id) == viewer.ID {
		return nil, "不能操作自己的账号"
	}

	target, err := model.NewUserModel().GetByID(uint(id))
	if err != nil {
		return nil, "用户不存在"
	}
	if target.Role == model.RoleOwner {
		return nil, "不能操作站点拥有者账号"
	}
	return target, ""
}

// loadPwdTarget 解析并校验可重置密码的账号：普通用户 / 管理员 / 站长本人。
// 管理员只能重置普通用户；管理员与站长账号的密码仅站长可重置，站长仅能重置自己。
func loadPwdTarget(c *gin.Context) (*model.User, string) {
	viewer := middleware.CurrentUser(c)
	if viewer == nil {
		return nil, "请先登录"
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, "用户 ID 有误"
	}
	target, err := model.NewUserModel().GetByID(uint(id))
	if err != nil {
		return nil, "用户不存在"
	}
	if target.ID == viewer.ID {
		return target, ""
	}
	if viewer.Role != model.RoleOwner {
		if target.Role != model.RoleUser {
			return nil, "仅站长可重置管理员账号的密码"
		}
		return target, ""
	}
	if target.Role == model.RoleOwner {
		return nil, "不能重置其他站长账号的密码"
	}
	return target, ""
}

func adminUserVO(u *model.User) gin.H {
	typeName, ok := model.MemberTypeName[u.MemberType]
	if !ok {
		typeName = model.MemberTypeName[model.MemberTypeTrial]
	}
	roleName, ok := model.RoleName[u.Role]
	if !ok {
		roleName = model.RoleName[model.RoleUser]
	}
	// 统一身份口径：站长 / 管理员是后台账号，展示后台身份；
	// 普通用户展示注册时选择的「使用身份」（老师 / 学员 / 机构）。师资身份设计已下线。
	identityType := u.UserRole
	identityName := model.UserRoleName[u.UserRole]
	if identityName == "" {
		identityName = model.UserRoleName[model.UserRoleTeacher]
	}
	if u.IsStaff() {
		identityType = u.Role
		identityName = model.StaffIdentityName[u.Role]
		if identityName == "" {
			identityName = roleName
		}
	}
	return gin.H{
		"id":             u.ID,
		"username":       u.Username,
		"phone":          u.Phone,
		"email":          u.Email,
		"avatar":         u.Avatar,
		"role":           u.Role,
		"roleName":       roleName,
		"isStaff":        u.IsStaff(),
		"userRole":       u.UserRole,
		"identityType":   identityType, // owner / admin / teacher / parent / org
		"identityName":   identityName, // 超级管理员 / 运营管理员 / 老师 / 学员 / 机构
		"status":         u.Status,
		"statusName":     model.UserStatusName[u.Status],
		"memberType":     u.MemberType,
		"memberTypeName": typeName,
		"memberGroup":    model.MemberTypeGroup(u.MemberType),
		"totalPaid":      u.TotalPaid, // 站长与管理员恒为 0，前端按 isStaff 显示为 -
		"memberStart":    u.MemberStart,
		"memberExpire":   u.MemberExpire,
		"memberActive":   u.MemberActive(),
		"memberLeftDays": u.MemberLeftDays(),
		"registeredAt":   u.CreatedAt,
		"registeredDays": u.RegisteredDays(),
	}
}
