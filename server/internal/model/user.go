package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 会员类型（user.member_type）：数据库保留细分套餐用于计费与订单记录，
// 对外展示统一收敛为三类：免费试用 / 付费会员 / 永久会员。
const (
	MemberTypeTrial     = "trial"     // 免费试用（注册即送 1 个月）
	MemberTypeDaily     = "daily"     // 付费会员：按天购买（自定义天数）
	MemberTypeMonthly   = "monthly"   // 付费会员：包月
	MemberTypeYearly    = "yearly"    // 付费会员：包年
	MemberTypeQuarterly = "quarterly" // 包季
	MemberTypePermanent = "permanent" // 永久会员（站长 / 管理员）
)

// 会员类型展示分组（前端筛选与展示口径，与细分套餐解耦）
const (
	MemberGroupTrial     = "trial"     // 免费试用
	MemberGroupPaid      = "paid"      // 付费会员（按天 / 包月 / 包季 / 包年）
	MemberGroupPermanent = "permanent" // 永久会员
)

// MemberTypeName 会员类型中文名（展示口径只有三类，套餐明细见价格配置与充值记录）
var MemberTypeName = map[string]string{
	MemberTypeTrial:     "免费试用",
	MemberTypeDaily:     "付费会员",
	MemberTypeMonthly:   "付费会员",
	MemberTypeYearly:    "付费会员",
	MemberTypeQuarterly: "付费会员",
	MemberTypePermanent: "永久会员",
}

// MemberTypeGroup 把细分会员类型归并为展示分组。
func MemberTypeGroup(t string) string {
	switch t {
	case MemberTypePermanent:
		return MemberGroupPermanent
	case MemberTypeDaily, MemberTypeMonthly, MemberTypeQuarterly, MemberTypeYearly:
		return MemberGroupPaid
	default:
		return MemberGroupTrial
	}
}

// 用户角色（user.role）：决定后台权限，与用户 ID 无关。
// 列表展示顺序即 站长 → 管理员 → 普通用户。
const (
	RoleUser  = "user"  // 普通用户：注册账号，可选师资身份并参与会员计费
	RoleAdmin = "admin" // 管理员：后台运营账号，永久会员，不参与师资身份与计费
	RoleOwner = "owner" // 站长：站点拥有者（唯一），超级管理员 + 永久会员
)

// 账号状态
const (
	UserStatusNormal = "normal" // 正常
	UserStatusFrozen = "frozen" // 已冻结（禁止登录）
)

// 账号状态中文名
var UserStatusName = map[string]string{
	UserStatusNormal: "正常",
	UserStatusFrozen: "已冻结",
}

// 角色中文名
var RoleName = map[string]string{
	RoleUser:  "普通用户",
	RoleAdmin: "管理员",
	RoleOwner: "站长",
}

// StaffIdentityName 后台身份中文名：站长与管理员不区分师资身份，
// 统一按后台角色展示（普通用户的身份列展示师资身份）。
var StaffIdentityName = map[string]string{
	RoleOwner: "超级管理员",
	RoleAdmin: "运营管理员",
}

// PermanentExpireUnix 永久会员到期时间（秒级时间戳），用于把账号设为永久会员。
func PermanentExpireUnix() int64 {
	return time.Date(2099, 12, 31, 23, 59, 59, 0, time.Local).Unix()
}

// IsStaff 是否为管理员或站点拥有者。
func (u *User) IsStaff() bool {
	return u.Role == RoleAdmin || u.Role == RoleOwner
}

// 注册来源（user.register_src），用于区分注册方式并做统计归因：
// phone=手机号自助注册 wechat=微信授权注册 admin=管理员人工开通
const (
	RegisterSrcPhone  = "phone"
	RegisterSrcWechat = "wechat"
	RegisterSrcAdmin  = "admin"
)

// 师资身份（user.teacher_type）：普通用户的计费身份，决定会员价格档位；
// 站长与管理员为后台账号，不区分师资身份（展示见 StaffIdentityName）。
const (
	TeacherTypeStudent      = "student"      // 大学生家教（在校大学生），按学生价计费
	TeacherTypeProfessional = "professional" // 专职老师（在职 / 毕业），按标准价计费
)

// TeacherTypeName 师资身份中文名
var TeacherTypeName = map[string]string{
	TeacherTypeStudent:      "大学生家教",
	TeacherTypeProfessional: "专职老师",
}

// ValidTeacherType 校验师资身份合法性
func ValidTeacherType(t string) bool {
	_, ok := TeacherTypeName[t]
	return ok
}

// 使用身份（user.user_role）：注册时选择的「你是谁」，只决定课程表单字段与文案，
// 不参与会员计费（计费仍由 teacher_type 决定：大学生享折扣，其余走标准价）。
// 家长（parent）与个人（personal）已融合为「学员」单一身份，统一显示为「学员」。
const (
	UserRoleTeacher  = "teacher"  // 家教老师（含专职 / 大学生，需再选师资身份）
	UserRoleParent   = "parent"   // 学员：家长 / 个人融合（给孩子或自己排课）
	UserRolePersonal = "personal" // 学员：历史个人账号（与新注册 parent 同义，统一显示「学员」）
	UserRoleOrg      = "org"      // 机构 / 工作室（多学员排课）
)

// UserRoleName 使用身份中文名（parent 与 personal 统一显示为「学员」）
var UserRoleName = map[string]string{
	UserRoleTeacher:  "老师",
	UserRoleParent:   "学员",
	UserRolePersonal: "学员",
	UserRoleOrg:      "机构",
}

// ValidUserRole 校验使用身份合法性
func ValidUserRole(r string) bool {
	_, ok := UserRoleName[r]
	return ok
}

// User 站点账号表（user）。字段分组：A 账号信息 / B 角色与状态 / C 师资身份 / D 会员信息 / E 注册归因。
// 会员信息不单拆一表的原因见 docs/用户表设计说明.md。
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"size:64;uniqueIndex;not null" json:"username"` // 用户名，唯一
	Password string `gorm:"size:128;not null" json:"-"`                   // 登录密码，bcrypt 加密
	Avatar   string `gorm:"size:255;not null" json:"avatar"`
	Phone    string `gorm:"size:32;not null" json:"phone"` // 手机号，唯一的登录账号

	// ---- B. 角色与状态 ----
	Role   string `gorm:"size:16;not null;default:user" json:"role"`     // 角色：owner=站长 admin=管理员 user=普通用户
	Status string `gorm:"size:16;not null;default:normal" json:"status"` // 账号状态：normal=正常 frozen=已冻结

	// ---- C. 师资身份（仅普通用户）----
	UserRole              string `gorm:"size:16;not null;default:teacher" json:"userRole"`         // 使用身份：teacher 老师 / parent+personal 学员 / org 机构；站长与管理员留空，统一按老师渲染
	TeacherType           string `gorm:"size:16;not null;default:professional" json:"teacherType"` // 师资身份：student=大学生家教 professional=专职老师
	RegTeacherType        string `gorm:"size:16;not null" json:"regTeacherType"`                   // 注册时选择的师资身份
	Verified              int    `gorm:"not null;default:0" json:"verified"`                       // 认证：0=未认证 1=已认证
	StudentCardURL        string `gorm:"size:255;not null" json:"studentCardUrl"`                  // 学生证照片地址
	IdCardURL             string `gorm:"size:255;not null" json:"idCardUrl"`                       // 身份证照片地址
	StudentVerifiedAt     int64  `gorm:"not null" json:"studentVerifiedAt"`                        // 学生身份审核通过时间（Unix 秒）
	StudentExpireAt       int64  `gorm:"not null" json:"studentExpireAt"`                          // 学生身份有效期 / 毕业时间（Unix 秒）
	StudentExpireNotified bool   `gorm:"not null;default:false" json:"studentExpireNotified"`      // 学生身份到期提醒是否已发送

	// ---- D. 会员信息（展示口径：免费试用 / 付费会员 / 永久会员）----
	MemberType           string  `gorm:"size:16;not null;default:trial" json:"memberType"`                             // 会员类型：trial=免费试用 daily / monthly / quarterly / yearly=付费会员 permanent=永久会员
	MemberStart          int64   `gorm:"not null" json:"memberStart"`                                                  // 会员生效时间（Unix 秒）
	MemberExpire         int64   `gorm:"not null" json:"memberExpire"`                                                 // 会员到期时间（Unix 秒）
	MemberExpireNotified bool    `gorm:"not null;default:false" json:"memberExpireNotified"`                           // 会员到期提醒是否已发送
	TotalPaid            float64 `gorm:"type:decimal(12,2);column:total_recharge;not null;default:0" json:"totalPaid"` // 累计充值金额（元）

	PhoneChangedAt int64 `gorm:"not null;default:0" json:"phoneChangedAt"` // 最近修改手机号的时间（Unix 秒）

	// ---- E. 注册风控（新增列必须带 default，否则存量库 AutoMigrate 会报 contains null values）----
	RegisterIP  string `gorm:"size:64;not null;default:''" json:"-"`              // 注册 IP
	RegisterSrc string `gorm:"size:16;not null;default:phone" json:"registerSrc"` // 注册来源：phone / wechat / admin
	CreatedAt   int64  `gorm:"autoCreateTime;not null" json:"createdAt"`          // 注册时间（Unix 秒）
	UpdatedAt   int64  `gorm:"autoUpdateTime;not null" json:"updatedAt"`          // 更新时间（Unix 秒）
}

func (m User) TableName() string { return "user" }

// MemberActive 会员是否在有效期内。
func (u *User) MemberActive() bool {
	return u.MemberExpire > time.Now().Unix()
}

// StudentExpiringSoon 学生身份是否临近过期（距毕业时间 30 天内）。
func (u *User) StudentExpiringSoon() bool {
	if u.TeacherType != TeacherTypeStudent || u.StudentVerifiedAt == 0 || u.StudentExpireAt == 0 {
		return false
	}
	left := u.StudentExpireAt - time.Now().Unix()
	return left > 0 && left <= 30*86400
}

// ExpireStudentIdentityIfNeeded 学生身份过期超过一年后自动转为专职老师；返回是否发生转换。
// 毕业时间后保留一年缓冲期（可再次认证，如考研考博等），缓冲期内续费按专职老师标准计费。
func (u *User) ExpireStudentIdentityIfNeeded() bool {
	if u.TeacherType != TeacherTypeStudent || u.StudentVerifiedAt == 0 || u.StudentExpireAt == 0 {
		return false
	}
	// 毕业时间后保留一年缓冲期，超过一年才自动转为专职老师
	if time.Now().Unix() >= u.StudentExpireAt+365*86400 {
		u.TeacherType = TeacherTypeProfessional
		u.Verified = 0
		u.StudentVerifiedAt = 0
		u.StudentExpireAt = 0
		return true
	}
	return false
}

// StudentReapplyWindowDays 学生身份申请被驳回后可重新申请的窗口期（天），
// 逾期仍未通过认证的账号自动转为专职老师身份。
const StudentReapplyWindowDays = 30

// ExpireUnverifiedStudentIfNeeded 申请被驳回超过窗口期仍未通过认证，自动转为专职老师；返回是否发生转换。
func (u *User) ExpireUnverifiedStudentIfNeeded(latestRejectedAt int64) bool {
	if u.TeacherType != TeacherTypeStudent || u.Verified != 0 || latestRejectedAt <= 0 {
		return false
	}
	if time.Now().Unix() < latestRejectedAt+int64(StudentReapplyWindowDays)*86400 {
		return false
	}
	u.TeacherType = TeacherTypeProfessional
	u.Verified = 0
	u.StudentVerifiedAt = 0
	u.StudentExpireAt = 0
	return true
}

// MemberLeftDays 会员剩余天数（向下取整，已过期返回 0）。
func (u *User) MemberLeftDays() int {
	left := u.MemberExpire - time.Now().Unix()
	if left <= 0 {
		return 0
	}
	return int(left / 86400)
}

// RegisteredDays 注册至今天数（向上取整，注册当天即算 1 天）。
func (u *User) RegisteredDays() int {
	now := time.Now().Unix()
	if u.CreatedAt <= 0 || u.CreatedAt >= now {
		return 1
	}
	days := int((now - u.CreatedAt + 86399) / 86400)
	if days < 1 {
		days = 1
	}
	return days
}

// MemberStartTime / MemberExpireTime / CreatedTime 便于按 time.Time 使用。
func (u *User) MemberStartTime() time.Time  { return time.Unix(u.MemberStart, 0) }
func (u *User) MemberExpireTime() time.Time { return time.Unix(u.MemberExpire, 0) }
func (u *User) CreatedTime() time.Time      { return time.Unix(u.CreatedAt, 0) }

type UserModel struct{}

func NewUserModel() *UserModel { return &UserModel{} }

var ErrUserNotFound = errors.New("用户不存在")

func (m *UserModel) GetByID(id uint) (*User, error) {
	var u User
	if err := db.Where("id = ?", id).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (m *UserModel) GetByUsername(username string) (*User, error) {
	var u User
	if err := db.Where("username = ?", username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetByPhone 登录仅支持手机号（管理员同样使用手机号登录，不支持用户名）。
func (m *UserModel) GetByPhone(account string) (*User, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return nil, ErrUserNotFound
	}
	var u User
	err := db.Where("phone = ?", account).
		Order("id ASC").First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// ExistsPhone 手机号是否已存在（excludeID>0 时排除该用户）。
func (m *UserModel) ExistsPhone(phone string, excludeID uint) (bool, error) {
	if phone == "" {
		return false, nil
	}
	var count int64
	tx := db.Model(&User{}).Where("phone = ?", phone)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsUsername 用户名是否已存在（excludeID>0 时排除该用户；用户名唯一，用于注册与改名校验）。
func (m *UserModel) ExistsUsername(username string, excludeID uint) (bool, error) {
	if username == "" {
		return false, nil
	}
	var count int64
	tx := db.Model(&User{}).Where("username = ?", username)
	if excludeID > 0 {
		tx = tx.Where("id <> ?", excludeID)
	}
	if err := tx.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type AdminUserQuery struct {
	Keyword       string
	Role          string
	Status        string
	MemberType    string
	OnlyActive    bool
	Page          int
	PageSize      int
	ViewerIsOwner bool
}

// AdminList 后台用户列表：拥有者可见全部账号，管理员仅可见普通会员。
func (m *UserModel) AdminList(q AdminUserQuery) ([]User, int64, error) {
	tx := db.Model(&User{})
	if !q.ViewerIsOwner {
		tx = tx.Where("role = ?", RoleUser)
	}
	if q.Role != "" {
		tx = tx.Where("role = ?", q.Role)
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	// 会员类型筛选：展示口径三类（trial / paid / permanent），paid 覆盖全部付费套餐
	switch q.MemberType {
	case "":
	case MemberGroupPaid:
		tx = tx.Where("member_type IN ?", []string{MemberTypeDaily, MemberTypeMonthly, MemberTypeQuarterly, MemberTypeYearly})
	default:
		tx = tx.Where("member_type = ?", q.MemberType)
	}
	if q.OnlyActive {
		tx = tx.Where("member_expire > ?", time.Now().Unix())
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		// 关键词同时匹配用户 ID（纯数字直接搜 ID 的场景）
		tx = tx.Where("username ILIKE ? OR phone ILIKE ? OR CAST(id AS TEXT) ILIKE ?",
			like, like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	var list []User
	// 角色分组排序：站长 → 管理员 → 普通用户；同角色内按注册先后（ID 升序）。
	// 注意：必须用单次 Order 传入完整排序表达式，多次 Order 调用后者会覆盖前者。
	err := tx.Order(clause.OrderBy{Expression: clause.Expr{
		SQL:  "CASE role WHEN ? THEN 0 WHEN ? THEN 1 ELSE 2 END, id ASC",
		Vars: []interface{}{RoleOwner, RoleAdmin},
	}}).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error
	return list, total, err
}

// PendingStudents 待审核的大学生家教列表（teacherType=student 且 verified=0）。
func (m *UserModel) PendingStudents() ([]User, error) {
	var list []User
	err := db.Where("teacher_type = ? AND verified = ?", TeacherTypeStudent, 0).
		Order("id ASC").Find(&list).Error
	return list, err
}

// DailyStat 每日注册人数、有效会员人数与已开通会员人数。
type DailyStat struct {
	Date          string `json:"date"`
	RegisterCount int64  `json:"registerCount"`
	MemberCount   int64  `json:"memberCount"`
	PaidCount     int64  `gorm:"column:recharge_count" json:"paidCount"`
}

// DailyStats 统计最近 days 天（含今天）的注册人数、有效会员人数与已开通会员人数。
func (m *UserModel) DailyStats(days int) ([]DailyStat, error) {
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	// generate_series 生成日期序列，分别统计当天注册数、当天有效会员数与当天充值会员数（去重）
	const sql = `
SELECT to_char(d.day, 'YYYY-MM-DD') AS date,
       (SELECT count(*) FROM "user" u
         WHERE u.created_at >= EXTRACT(EPOCH FROM d.day)::bigint
           AND u.created_at < EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint) AS register_count,
       (SELECT count(*) FROM "user" u
         WHERE u.status <> 'frozen'
           AND u.member_start <= EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint
           AND u.member_expire >= EXTRACT(EPOCH FROM d.day)::bigint) AS member_count,
       (SELECT count(DISTINCT r.user_id) FROM recharge r
         WHERE r.created_at >= EXTRACT(EPOCH FROM d.day)::bigint
           AND r.created_at < EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint) AS recharge_count
FROM generate_series(date_trunc('day', now()) - make_interval(days => (?::int) - 1),
                     date_trunc('day', now()),
                     interval '1 day') AS d(day)
ORDER BY d.day`

	var list []DailyStat
	if err := db.Raw(sql, days).Scan(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// DailyStatsRange 统计 [from, to] 闭区间内每天的注册 / 有效会员 / 已开通会员人数（支持自定义起止日期）。
func (m *UserModel) DailyStatsRange(from, to time.Time) ([]DailyStat, error) {
	if to.Before(from) {
		return nil, nil
	}
	const sql = `
SELECT to_char(d.day, 'YYYY-MM-DD') AS date,
       (SELECT count(*) FROM "user" u
         WHERE u.created_at >= EXTRACT(EPOCH FROM d.day)::bigint
           AND u.created_at < EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint) AS register_count,
       (SELECT count(*) FROM "user" u
         WHERE u.status <> 'frozen'
           AND u.member_start <= EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint
           AND u.member_expire >= EXTRACT(EPOCH FROM d.day)::bigint) AS member_count,
       (SELECT count(DISTINCT r.user_id) FROM recharge r
         WHERE r.created_at >= EXTRACT(EPOCH FROM d.day)::bigint
           AND r.created_at < EXTRACT(EPOCH FROM d.day + interval '1 day')::bigint) AS recharge_count
FROM generate_series(date_trunc('day', $1::timestamptz), date_trunc('day', $2::timestamptz), interval '1 day') AS d(day)
ORDER BY d.day`
	var list []DailyStat
	// 传日期字符串而非 time.Time：参数类型明确，避免 date_trunc(unknown, unknown) 重载歧义（SQLSTATE 42725）；
	// 日期按数据库会话时区（PG_TIMEZONE）解释为当天零点，与 now() 口径一致。
	if err := db.Raw(sql, from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Summary 后台概览数据。
type Summary struct {
	TotalUsers    int64   `json:"totalUsers"`
	ActiveMembers int64   `json:"activeMembers"`
	FrozenUsers   int64   `json:"frozenUsers"`
	TodayRegister int64   `json:"todayRegister"`
	TotalPaid     float64 `gorm:"column:total_recharge" json:"totalPaid"`
}

func (m *UserModel) StatsSummary() (*Summary, error) {
	now := time.Now().Unix()
	// 今日零点：按本地时区取当天 00:00（不能用 Truncate(24h)——它按 UTC 纪元对齐，
	// 在 Asia/Shanghai 会得到当天 08:00，导致「今日注册」漏统计当天 0-8 点的用户）
	n := time.Now()
	todayStart := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location()).Unix()

	s := &Summary{}
	if err := db.Model(&User{}).Count(&s.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&User{}).Where("member_expire > ?", now).Count(&s.ActiveMembers).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&User{}).Where("status = ?", UserStatusFrozen).Count(&s.FrozenUsers).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&User{}).Where("created_at >= ?", todayStart).Count(&s.TodayRegister).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&User{}).Select("COALESCE(SUM(total_recharge), 0) AS total_recharge").Scan(&s.TotalPaid).Error; err != nil {
		return nil, err
	}
	return s, nil
}

func (m *UserModel) Create(u *User) error {
	return db.Create(u).Error
}

// CreateTx 事务版 Create，供「创建用户 + 核销邀请码 + 建学生申请」放在同一事务内调用。
func (m *UserModel) CreateTx(tx *gorm.DB, u *User) error {
	return tx.Create(u).Error
}

func (m *UserModel) Save(u *User) error {
	return db.Save(u).Error
}

// HasTeachingData 判断账号是否存在课程订单或充值流水。
// 订单与流水属于教学 / 财务留档，删除账号前必须先确认没有这类数据。
func (m *UserModel) HasTeachingData(id uint) (bool, error) {
	var cnt int64
	if err := db.Table("order").Where("user_id = ?", id).Limit(1).Count(&cnt).Error; err != nil {
		return false, err
	}
	if cnt > 0 {
		return true, nil
	}
	if err := db.Table("recharge").Where("user_id = ?", id).Limit(1).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// DeleteAccount 删除账号及其附属数据（站内信、反馈、学生申请、上课提醒）。
// 课程订单、课次与充值流水不删除：调用方需先用 HasTeachingData 确认账号没有这类数据。
func (m *UserModel) DeleteAccount(id uint) error {
	// table + WHERE 条件均来自下方固定清单，不含外部输入
	stmts := []struct {
		table string
		where string
		args  []interface{}
	}{
		{table: "message", where: "user_id = ? OR sender_id = ?", args: []interface{}{id, id}},
		{table: "feedback", where: "user_id = ?", args: []interface{}{id}},
		{table: "student_application", where: "user_id = ?", args: []interface{}{id}},
		{table: "lesson_reminder", where: "user_id = ?", args: []interface{}{id}},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, s := range stmts {
			if err := tx.Exec(`DELETE FROM `+quoteIdent(s.table)+` WHERE `+s.where, s.args...).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&User{}, id).Error
	})
}
