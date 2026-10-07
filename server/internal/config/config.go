package config

import (
	"os"
	"strconv"
	"strings"

	"tutoring_server/pkg/path"

	"gopkg.in/ini.v1"
)

type appConfig struct {
	RunMode   string
	Host      string
	LocalHost string
	// StaticDir 前端构建产物目录，为空则不托管前端（由 nginx 托管）。
	StaticDir string
	// UploadDir 头像等上传文件目录。
	UploadDir string
}

type serverConfig struct {
	ServerPort   int
	ReadTimeout  int
	WriteTimeout int
}

type postgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Db       string
	SslMode  string
	TimeZone string
}

type jwtConfig struct {
	Secret      string
	ExpireHours int
}

// inviteConfig 注册准入与邀请推广开关。
// Mode 是全站唯一的准入降级开关：出现刷号时改回 manual 重启即可回到人工发码，无需改代码。
type inviteConfig struct {
	// Mode：manual=仅管理员人工发码（现行，最严）；member=开放会员自助发码/邀请链接
	Mode string
	// MemberMonthlyQuota：会员每月可生成的邀请凭证数量（mode=member 生效）
	MemberMonthlyQuota int
	// MemberCodeExpireDays：会员生成的邀请码有效期（天），过期未注册自动失效回收
	MemberCodeExpireDays int
	// WechatLoginEnabled：是否启用微信授权注册/登录（需先具备开放平台或公众号资质）
	WechatLoginEnabled bool
	// TrialDaysInvite / TrialDaysWechat：不同来源注册的初始试用天数
	TrialDaysInvite int
	TrialDaysWechat int
}

var (
	AppConfig      *appConfig
	ServerConfig   *serverConfig
	PostgresConfig *postgresConfig
	JwtConfig      *jwtConfig
	InviteConfig   *inviteConfig
)

func Init() error {
	cfg, err := ini.Load(configPath())
	if err != nil {
		return err
	}

	AppConfig = &appConfig{
		RunMode:   envOrDefault("RUN_MODE", cfg.Section("").Key("RUN_MODE").MustString("debug")),
		StaticDir: envOrDefault("STATIC_DIR", cfg.Section("app").Key("STATIC_DIR").String()),
		UploadDir: envOrDefault("UPLOAD_DIR", cfg.Section("app").Key("UPLOAD_DIR").MustString("data/uploads")),
	}

	ServerConfig = &serverConfig{
		ServerPort:   envIntOrDefault("HTTP_PORT", cfg.Section("server").Key("HTTP_PORT").MustInt(9090)),
		ReadTimeout:  envIntOrDefault("READ_TIMEOUT", cfg.Section("server").Key("READ_TIMEOUT").MustInt(60)),
		WriteTimeout: envIntOrDefault("WRITE_TIMEOUT", cfg.Section("server").Key("WRITE_TIMEOUT").MustInt(60)),
	}

	PostgresConfig = &postgresConfig{
		Host:     envOrDefault("PG_HOST", cfg.Section("postgres").Key("PG_HOST").MustString("127.0.0.1")),
		Port:     envIntOrDefault("PG_PORT", cfg.Section("postgres").Key("PG_PORT").MustInt(5432)),
		User:     envOrDefault("PG_USER", cfg.Section("postgres").Key("PG_USER").MustString("postgres")),
		Password: envOrDefault("PG_PASSWORD", cfg.Section("postgres").Key("PG_PASSWORD").String()),
		Db:       envOrDefault("PG_DB", cfg.Section("postgres").Key("PG_DB").MustString("tutoring")),
		SslMode:  envOrDefault("PG_SSLMODE", cfg.Section("postgres").Key("PG_SSLMODE").MustString("disable")),
		TimeZone: envOrDefault("PG_TIMEZONE", cfg.Section("postgres").Key("PG_TIMEZONE").MustString("Asia/Shanghai")),
	}

	JwtConfig = &jwtConfig{
		Secret:      envOrDefault("JWT_SECRET", cfg.Section("jwt").Key("JWT_SECRET").MustString("tutoring-website-secret")),
		ExpireHours: envIntOrDefault("JWT_EXPIRE_HOURS", cfg.Section("jwt").Key("JWT_EXPIRE_HOURS").MustInt(168)),
	}

	AppConfig.Host = envOrDefault("APP_HOST", cfg.Section("host").Key("HOST_"+strings.ToUpper(AppConfig.RunMode)).String())
	if AppConfig.Host == "" {
		AppConfig.Host = "http://127.0.0.1:" + strconv.Itoa(ServerConfig.ServerPort)
	}
	AppConfig.LocalHost = AppConfig.Host

	InviteConfig = &inviteConfig{
		Mode:                 envOrDefault("INVITE_MODE", cfg.Section("invite").Key("MODE").MustString("manual")),
		MemberMonthlyQuota:   envIntOrDefault("INVITE_MEMBER_MONTHLY_QUOTA", cfg.Section("invite").Key("MEMBER_MONTHLY_QUOTA").MustInt(3)),
		MemberCodeExpireDays: envIntOrDefault("INVITE_MEMBER_CODE_EXPIRE_DAYS", cfg.Section("invite").Key("MEMBER_CODE_EXPIRE_DAYS").MustInt(7)),
		WechatLoginEnabled:   envBoolOrDefault("WECHAT_LOGIN_ENABLED", cfg.Section("invite").Key("WECHAT_LOGIN_ENABLED").MustBool(false)),
		TrialDaysInvite:      envIntOrDefault("TRIAL_DAYS_INVITE", cfg.Section("invite").Key("TRIAL_DAYS_INVITE").MustInt(30)),
		TrialDaysWechat:      envIntOrDefault("TRIAL_DAYS_WECHAT", cfg.Section("invite").Key("TRIAL_DAYS_WECHAT").MustInt(7)),
	}

	return nil
}

func configPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return path.Join("internal", "conf", "app.ini")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envBoolOrDefault(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
