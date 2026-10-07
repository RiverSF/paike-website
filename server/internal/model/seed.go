package model

import (
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tutoring_server/pkg/logger"
)

// initOwnerPhoneRegexp 与注册校验保持一致的中国大陆手机号规则。
var initOwnerPhoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// SeedOwner 冷启动初始化站点拥有者：
//   - 若库中已存在 owner 则跳过（幂等）；
//   - 否则读取 INIT_OWNER_USERNAME / INIT_OWNER_PHONE / INIT_OWNER_PASSWORD
//     三个环境变量自动创建首个 owner（永久会员），避免「无 owner 则无法创建管理员」的死锁；
//   - 若未配置这些变量，仅打印告警日志，系统处于待初始化状态，
//     需手工执行 server/migrations/002_role.sql 指定 owner。
//
// 安全约束：仅当三件套齐全且格式合法时才创建，密码不足 8 位或用户名/手机号已存在则跳过创建，
// 绝不回退到任何默认账号/默认密码。
func SeedOwner() {
	um := NewUserModel()
	var cnt int64
	if err := db.Model(&User{}).Where("role = ?", RoleOwner).Count(&cnt).Error; err != nil {
		logger.Error("check owner failed, err=%s", err.Error())
		return
	}
	if cnt > 0 {
		return
	}

	username := strings.TrimSpace(os.Getenv("INIT_OWNER_USERNAME"))
	phone := strings.TrimSpace(os.Getenv("INIT_OWNER_PHONE"))
	password := os.Getenv("INIT_OWNER_PASSWORD")

	if username == "" || phone == "" || password == "" {
		logger.Info("未检测到站点拥有者，且未配置 INIT_OWNER_USERNAME/INIT_OWNER_PHONE/INIT_OWNER_PASSWORD，" +
			"系统将无法创建管理员。请在首次启动前配置上述环境变量，或手工执行 server/migrations/002_role.sql 指定 owner。")
		return
	}
	if strings.ContainsAny(username, " \t\r\n") || len([]rune(username)) < 2 || len([]rune(username)) > 32 {
		logger.Error("INIT_OWNER_USERNAME 不合法（2-32 字符、不含空格），跳过初始化")
		return
	}
	if !initOwnerPhoneRegexp.MatchString(phone) {
		logger.Error("INIT_OWNER_PHONE 不是合法的 11 位手机号，跳过初始化")
		return
	}
	if len(password) < 8 {
		logger.Error("INIT_OWNER_PASSWORD 长度不足 8 位，跳过初始化")
		return
	}

	// 用户名或手机号已存在时避免重复创建
	if exists, err := um.ExistsUsername(username, 0); err == nil && exists {
		logger.Info("INIT_OWNER_USERNAME 已存在，跳过初始化")
		return
	}
	if exists, err := um.ExistsPhone(phone, 0); err == nil && exists {
		logger.Info("INIT_OWNER_PHONE 已存在，跳过初始化")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("hash owner password failed, err=%s", err.Error())
		return
	}

	now := time.Now()
	owner := &User{
		Username:       username,
		Password:       string(hash),
		Phone:          phone,
		UserRole:       "", // 站长不绑定使用身份，统一按老师渲染
		TeacherType:    TeacherTypeProfessional,
		RegTeacherType: TeacherTypeProfessional,
		MemberType:     MemberTypePermanent,
		MemberStart:    now.Unix(),
		MemberExpire:   PermanentExpireUnix(),
		Role:           RoleOwner,
		Status:         UserStatusNormal,
		RegisterSrc:    RegisterSrcAdmin,
	}
	if err := um.Create(owner); err != nil {
		logger.Error("create owner failed, err=%s", err.Error())
		return
	}
	logger.Info("已初始化站点拥有者账号: %s（手机号 %s），请尽快登录后台并修改密码", username, phone)
}
