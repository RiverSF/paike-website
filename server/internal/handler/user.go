package handler

import (
	"errors"
	"fmt"
	"regexp"
	
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"tutoring_server/internal/middleware"
	"tutoring_server/internal/model"
	"tutoring_server/pkg/logger"
)

// 注册赠送的免费会员天数
const trialDays = 30

// errInviteRace 邀请码在本次注册过程中被并发抢先核销，事务需整体回滚。
var errInviteRace = errors.New("invite code already used")

// 中国大陆手机号
var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// 邮箱
var emailRegexp = regexp.MustCompile(`^[\w.+-]+@[\w-]+(\.[\w-]+)+$`)

type registerReq struct {
	Username   string `json:"username" binding:"required,min=2,max=32"`
	Password   string `json:"password" binding:"required,min=6,max=32"`
	UserRole   string `json:"userRole"`   // 使用身份：teacher / parent / personal / org
	InviteCode string `json:"inviteCode"` // 邀请码（必填，与注册手机号关联校验）
	// 手机号必填（用于联系与账号找回），格式在下方单独校验以返回友好提示
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// updateProfileReq 站点内可自助修改的字段。
// 用户名：不可为空、2-32 字符、不含空格，需全局唯一；
// 手机号：支持自助变更（每自然月仅一次），需格式与全局唯一校验，变更后同步关联的邀请码记录；
// 字段留空表示「不修改」。
type updateProfileReq struct {
	Avatar   string `json:"avatar"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// sameMonth 判断时间戳与当前时间是否落在同一自然月（用于手机号每月仅可改一次的限制）。
func sameMonth(ts int64, now time.Time) bool {
	if ts <= 0 {
		return false
	}
	t := time.Unix(ts, 0)
	return t.Year() == now.Year() && t.Month() == now.Month()
}

type changePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=32"`
}

// memberReq 自助续费入参（该入口已关闭，仅保留结构声明以备后续开放）。
type memberReq struct {
	Plan   string  `json:"plan" binding:"required"` // monthly | yearly（兼容 month / year / 1 / 12）
	Amount float64 `json:"amount"`                  // 支付金额，用于累计充值统计
}

// Register 注册：注册成功即赠送 30 天免费会员。
func Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if strings.ContainsAny(req.Username, " \t\r\n") {
		BadRequest(c, "用户名不能包含空格")
		return
	}

	um := model.NewUserModel()

	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		BadRequest(c, "请填写手机号")
		return
	}
	if !phoneRegexp.MatchString(req.Phone) {
		BadRequest(c, "请填写正确的 11 位手机号")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email != "" && !emailRegexp.MatchString(req.Email) {
		BadRequest(c, "邮箱格式不正确")
		return
	}

	// 邀请码：必填，与注册手机号关联查询（存在、未使用、且手机号匹配）
	inviteCode := strings.TrimSpace(req.InviteCode)
	if inviteCode == "" {
		BadRequest(c, "请填写邀请码")
		return
	}
	invite, err := model.NewInviteCodeModel().GetByCode(inviteCode)
	if err != nil {
		BadRequest(c, "邀请码无效，请核对后重新输入")
		return
	}
	if invite.Invalid == 1 {
		BadRequest(c, "该邀请码已作废")
		return
	}
	if invite.Used == 1 {
		BadRequest(c, "该邀请码已被使用")
		return
	}
	if invite.Phone != req.Phone {
		BadRequest(c, "邀请码与注册手机号不匹配")
		return
	}

	// 使用身份：默认老师，非空时需合法（仅影响课程表单与文案，不参与计费）
	userRole := model.UserRoleTeacher
	if r := strings.TrimSpace(req.UserRole); r != "" {
		if !model.ValidUserRole(r) {
			BadRequest(c, "使用身份不合法")
			return
		}
		userRole = r
	}
	// 手机号与邮箱查重
	if exists, err := um.ExistsPhone(req.Phone, 0); err != nil {
		logger.Error("check phone failed, err=%s", err.Error())
		ServerError(c, "注册失败，请稍后重试")
		return
	} else if exists {
		BadRequest(c, "该手机号已被注册")
		return
	}
	if req.Email != "" {
		if exists, err := um.ExistsEmail(req.Email, 0); err != nil {
			logger.Error("check email failed, err=%s", err.Error())
			ServerError(c, "注册失败，请稍后重试")
			return
		} else if exists {
			BadRequest(c, "该邮箱已被注册")
			return
		}
	}

	// 用户名必填（binding 已校验非空与长度）且全局唯一
	if exists, err := um.ExistsUsername(req.Username, 0); err != nil {
		logger.Error("check username failed, err=%s", err.Error())
		ServerError(c, "注册失败，请稍后重试")
		return
	} else if exists {
		BadRequest(c, "用户名已被注册")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ServerError(c, "注册失败，请稍后重试")
		return
	}

	now := time.Now()
	user := &model.User{
		Username:     req.Username,
		Password:     string(hash),
		Phone:        strings.TrimSpace(req.Phone),
		Email:        strings.TrimSpace(req.Email),
		UserRole:     userRole,
		TeacherType:  model.TeacherTypeProfessional, // 师资身份已下线，统一按专职标准价计费
		MemberType:   model.MemberTypeTrial,
		MemberStart:  now.Unix(),
		MemberExpire: now.AddDate(0, 0, trialDays).Unix(),
		Role:         model.RoleUser,
		RegisterIP:   c.ClientIP(),
		RegisterSrc:  model.RegisterSrcInvite,
	}

	// 创建用户 + 核销邀请码必须在同一事务内完成：
	// 任一步失败整体回滚，杜绝「用户已注册但邀请码仍可用」导致的一码多注册。
	txErr := model.DB().Transaction(func(tx *gorm.DB) error {
		if err := um.CreateTx(tx, user); err != nil {
			return err
		}
		ok, err := model.NewInviteCodeModel().MarkUsedTx(tx, inviteCode, user.ID)
		if err != nil {
			return err
		}
		if !ok {
			return errInviteRace
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errInviteRace) {
			BadRequest(c, "该邀请码已被使用")
			return
		}
		if model.IsDuplicateEntry(txErr) {
			BadRequest(c, "用户名已被注册")
			return
		}
		logger.Error("create user failed, err=%s", txErr.Error())
		ServerError(c, "注册失败，请稍后重试")
		return
	}

	token, err := middleware.GenerateToken(user)
	if err != nil {
		ServerError(c, "注册失败，请稍后重试")
		return
	}
	OK(c, gin.H{"token": token, "user": userVO(user)})
}

// Login 登录。连续失败 5 次后锁定 30 分钟，之后可重试。
func Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}

	key := loginKey(req.Username)
	// 锁定期间直接拒绝
	if remain, locked := loginLockRemaining(key); locked {
		Fail(c, CodeForbidden, fmt.Sprintf("登录失败次数过多，账号已锁定，请 %.0f 分钟后再试", remain.Minutes()))
		return
	}

	// 登录仅支持手机号 / 邮箱（管理员同样使用手机号登录，不支持用户名）
	user, err := model.NewUserModel().GetByPhoneOrEmail(req.Username)
	if err != nil {
		respondLoginFail(c, key)
		return
	}
	if user.Status == model.UserStatusFrozen {
		Fail(c, CodeForbidden, "账号已被冻结，请联系管理员")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		respondLoginFail(c, key)
		return
	}

	loginClear(key)
	token, err := middleware.GenerateToken(user)
	if err != nil {
		ServerError(c, "登录失败，请稍后重试")
		return
	}
	OK(c, gin.H{"token": token, "user": userVO(user)})
}

// respondLoginFail 记录一次登录失败并返回剩余次数 / 锁定提示。
func respondLoginFail(c *gin.Context, key string) {
	left, lockRemain := loginRecordFail(key)
	if lockRemain > 0 {
		Fail(c, CodeForbidden, fmt.Sprintf("密码错误次数过多，账号已锁定，请 %.0f 分钟后再试", lockRemain.Minutes()))
		return
	}
	Fail(c, CodeBadRequest, fmt.Sprintf("账号或密码错误，还可尝试 %d 次", left))
}

// Profile 个人信息：注册信息 + 会员信息。
func Profile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	OK(c, userVO(user))
}

// UpdateProfile 修改个人信息。
func UpdateProfile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}

	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数有误："+err.Error())
		return
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	// 用户名变更：不可为空、不含空格、2-32 字符，且全局唯一
	if req.Username != "" {
		username := strings.TrimSpace(req.Username)
		if strings.ContainsAny(username, " \t\r\n") {
			BadRequest(c, "用户名不能包含空格")
			return
		}
		if n := len([]rune(username)); n < 2 || n > 32 {
			BadRequest(c, "用户名长度需为 2-32 个字符")
			return
		}
		if username != user.Username {
			if exists, err := model.NewUserModel().ExistsUsername(username, user.ID); err != nil {
				logger.Error("check username failed, err=%s", err.Error())
				ServerError(c, "保存失败，请稍后重试")
				return
			} else if exists {
				BadRequest(c, "该用户名已被使用")
				return
			}
			user.Username = username
		}
	}
	// 手机号变更：格式校验 + 全局唯一（手机号也是登录账号，不能与他人重复）+ 每自然月仅可修改一次
	phoneChanged := false
	if req.Phone != "" {
		phone := strings.TrimSpace(req.Phone)
		if !phoneRegexp.MatchString(phone) {
			BadRequest(c, "请填写正确的 11 位手机号")
			return
		}
		if phone != user.Phone {
			if sameMonth(user.PhoneChangedAt, time.Now()) {
				BadRequest(c, "手机号每月仅可修改一次，请下月再试")
				return
			}
			if exists, err := model.NewUserModel().ExistsPhone(phone, user.ID); err != nil {
				logger.Error("check phone failed, err=%s", err.Error())
				ServerError(c, "保存失败，请稍后重试")
				return
			} else if exists {
				BadRequest(c, "该手机号已被使用")
				return
			}
			user.Phone = phone
			user.PhoneChangedAt = time.Now().Unix()
			phoneChanged = true
		}
	}
	if req.Email != "" {
		email := strings.TrimSpace(strings.ToLower(req.Email))
		if !emailRegexp.MatchString(email) {
			BadRequest(c, "邮箱格式不正确")
			return
		}
		if exists, err := model.NewUserModel().ExistsEmail(email, user.ID); err != nil {
			ServerError(c, "保存失败，请稍后重试")
			return
		} else if exists {
			BadRequest(c, "该邮箱已被使用")
			return
		}
		user.Email = email
	}
	if err := model.NewUserModel().Save(user); err != nil {
		logger.Error("save user failed, err=%s", err.Error())
		ServerError(c, "保存失败，请稍后重试")
		return
	}
	// 手机号变更后同步邀请码关联记录（邀请码与注册手机号强关联，后台按手机号检索）
	if phoneChanged {
		if err := model.NewInviteCodeModel().UpdatePhoneByUser(user.ID, user.Phone); err != nil {
			logger.Error("sync invite code phone failed, err=%s", err.Error())
		}
	}
	OK(c, userVO(user))
}

// ChangePassword 修改登录密码：需校验原密码，新密码 6-32 位。
func ChangePassword(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "新密码需为 6-32 位")
		return
	}
	if req.OldPassword == "" {
		BadRequest(c, "请输入原密码")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		BadRequest(c, "原密码不正确")
		return
	}
	if req.OldPassword == req.NewPassword {
		BadRequest(c, "新密码不能与原密码相同")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		ServerError(c, "修改失败，请稍后重试")
		return
	}
	user.Password = string(hash)
	if err := model.NewUserModel().Save(user); err != nil {
		logger.Error("change password failed, err=%s", err.Error())
		ServerError(c, "修改失败，请稍后重试")
		return
	}
	OK(c, gin.H{"ok": true})
}

// MyPayments 我的支付记录 + 累计充值金额。
func MyPayments(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		Fail(c, CodeUnauthorized, "请先登录")
		return
	}
	list, err := model.NewPaymentModel().ListByUser(user.ID, 50)
	if err != nil {
		logger.Error("list payment failed, err=%s", err.Error())
		ServerError(c, "查询支付记录失败")
		return
	}
	OK(c, gin.H{"list": list, "totalPaid": user.TotalPaid})
}

// ActivateMember 已关闭。
// 会员续费统一由管理员在后台操作（填写支付金额 + 续费时长），
// 用户侧只保留付款码引导，不再支持自助开通，避免自填金额无法核实。
func ActivateMember(c *gin.Context) {
	Fail(c, CodeForbidden, "自助续费已关闭，请扫码支付后联系管理员开通")
}

// userVO 组装返回给前端的用户信息。
func userVO(u *model.User) gin.H {
	typeName, ok := model.MemberTypeName[u.MemberType]
	if !ok {
		typeName = "免费用户"
	}
	roleName, ok := model.RoleName[u.Role]
	if !ok {
		roleName = model.RoleName[model.RoleUser]
	}
	return gin.H{
		"id":                  u.ID,
		"username":            u.Username,
		"avatar":              u.Avatar,
		"phone":               u.Phone,
		"phoneChangedAt":      u.PhoneChangedAt,
		"email":               u.Email,
		"role":                u.Role,
		"roleName":            roleName,
		"isStaff":             u.IsStaff(),
		"status":              u.Status,
		"statusName":          model.UserStatusName[u.Status],
		"memberType":          u.MemberType,
		"memberTypeName":      typeName,
		"totalPaid":          u.TotalPaid, // 累计充值金额
		"memberStart":         u.MemberStart,   // 秒级时间戳
		"memberExpire":        u.MemberExpire,  // 秒级时间戳
		"memberActive":        u.MemberActive(),
		"memberLeftDays":      u.MemberLeftDays(),
		"registeredAt":        u.CreatedAt, // 秒级时间戳
		"registeredDays":      u.RegisteredDays(),
		"userRole":            u.UserRole,
		"userRoleName":        model.UserRoleName[u.UserRole],
	}
}


