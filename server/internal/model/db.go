package model

import (
	"fmt"
	"strings"
	"time"

	"tutoring_server/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

// DB 返回全局数据库句柄。
func DB() *gorm.DB { return db }

// SetDB 替换数据库句柄，便于测试。
func SetDB(d *gorm.DB) { db = d }

func Init() error {
	// 注意：空值不要拼进 DSN（如 password=），否则后续 key=value 可能被吞掉
	parts := []string{
		fmt.Sprintf("host=%s", config.PostgresConfig.Host),
		fmt.Sprintf("port=%d", config.PostgresConfig.Port),
		fmt.Sprintf("user=%s", config.PostgresConfig.User),
		fmt.Sprintf("dbname=%s", config.PostgresConfig.Db),
		fmt.Sprintf("sslmode=%s", config.PostgresConfig.SslMode),
		fmt.Sprintf("TimeZone=%s", config.PostgresConfig.TimeZone),
	}
	if config.PostgresConfig.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.PostgresConfig.Password))
	}
	dsn := strings.Join(parts, " ")

	logLevel := glogger.Info
	if config.AppConfig.RunMode == "release" {
		logLevel = glogger.Warn
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         glogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	return AutoMigrate()
}

// AutoMigrate 自动建表（缺失字段会自动新增，已存在的字段不会删除）。
func AutoMigrate() error {
	// 历史列改名：parent_phone → contact_phone（须在 AutoMigrate 之前，见函数注释）
	if err := MigrateOrderContactPhone(); err != nil {
		return err
	}
	// 先把订单的 start_date / end_date 字符串列迁移为时间戳，再交给 AutoMigrate 对齐结构
	if err := MigrateOrderDateToTimestamp(); err != nil {
		return err
	}
	// 已有库：先把存量 NULL 回填为默认值并补上 NOT NULL 约束，
	// 必须在 AutoMigrate 之前执行，否则 AutoMigrate 对齐 not null tag 时会因存量 NULL 报错
	if err := EnsureNotNull(); err != nil {
		return err
	}
	if err := db.AutoMigrate(&User{}, &Order{}, &Feedback{}, &Payment{}, &PriceConfig{}, &PriceSchedule{}, &PriceHistory{}, &InviteCode{}, &Message{}, &StudentApplication{}, &LessonException{}, &Lesson{}, &LessonReminder{}, &TrialNudge{}); err != nil {
		return err
	}
	// 索引统一在 EnsureIndexes 维护（建需要的、清冗余的），需在 AutoMigrate 之后执行
	if err := EnsureIndexes(); err != nil {
		return err
	}
	if err := EnsureComments(); err != nil {
		return err
	}
	// 历史订单状态收敛为「进行中 / 已结束」
	if err := MigrateOrderStatus(); err != nil {
		return err
	}
	// 删除订单表废弃的「单次时长」列（课时长改由各周时间段起止推算）
	if err := MigrateDropOrderDuration(); err != nil {
		return err
	}
	return nil
}

func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
