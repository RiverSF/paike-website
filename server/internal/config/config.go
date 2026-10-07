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
	// TrustedProxies 可信反向代理地址（IP / CIDR），来自这些地址的 X-Forwarded-For 才会被采信。
	// 留空表示不信任任何代理：此时客户端 IP 取 TCP 连接地址，避免伪造 XFF 绕过 IP 限流。
	// 部署在 nginx / ingress 之后时，必须填入代理地址（如 127.0.0.1、10.0.0.0/8），否则限流会把所有请求视为同一 IP。
	TrustedProxies []string
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

var (
	AppConfig      *appConfig
	ServerConfig   *serverConfig
	PostgresConfig *postgresConfig
	JwtConfig      *jwtConfig
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
		ServerPort:     envIntOrDefault("HTTP_PORT", cfg.Section("server").Key("HTTP_PORT").MustInt(9090)),
		ReadTimeout:    envIntOrDefault("READ_TIMEOUT", cfg.Section("server").Key("READ_TIMEOUT").MustInt(60)),
		WriteTimeout:   envIntOrDefault("WRITE_TIMEOUT", cfg.Section("server").Key("WRITE_TIMEOUT").MustInt(60)),
		TrustedProxies: splitList(envOrDefault("TRUSTED_PROXIES", cfg.Section("server").Key("TRUSTED_PROXIES").String())),
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

// splitList 解析逗号分隔的配置项（如 TRUSTED_PROXIES = 127.0.0.1,10.0.0.0/8），返回非空项。
func splitList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
