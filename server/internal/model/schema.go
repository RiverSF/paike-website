package model

import (
	"fmt"

	"tutoring_server/pkg/logger"
)

// 本文件集中维护三项库表规范，均由服务启动时幂等执行：
//  1. EnsureNotNull  —— 所有字段不允许 NULL（字符串默认 ''、数值默认 0、布尔默认 false、枚举给业务默认值）
//  2. EnsureIndexes  —— 按查询路径建立索引，并清理重复 / 失效索引
// 表与字段注释见 comments.go 的 EnsureComments。
//
// 新库：AutoMigrate 按 model 的 gorm tag 直接建出 NOT NULL 列，EnsureNotNull 会跳过（表尚不存在）。
// 已有库：EnsureNotNull 先把存量 NULL 回填为默认值，再补默认值与 NOT NULL 约束。

// columnRule 单列的「默认值 + 非空」规则。def 为 SQL 字面量（字符串需带单引号）。
type columnRule struct {
	table  string
	column string
	def    string
}

// notNullRules 全库不允许为 NULL 的列清单（按表分组，顺序与 model 字段一致）。
var notNullRules = []columnRule{
	// ---------- user ----------
	{"user", "username", "''"},
	{"user", "password", "''"},
	{"user", "avatar", "''"},
	{"user", "phone", "''"},
	{"user", "email", "''"},
	{"user", "user_role", "'teacher'"},
	{"user", "teacher_type", "'professional'"},
	{"user", "reg_teacher_type", "''"},
	{"user", "verified", "0"},
	{"user", "student_card_url", "''"},
	{"user", "id_card_url", "''"},
	{"user", "student_verified_at", "0"},
	{"user", "student_expire_at", "0"},
	{"user", "student_expire_notified", "false"},
	{"user", "member_expire_notified", "false"},
	{"user", "member_type", "'trial'"},
	{"user", "member_start", "0"},
	{"user", "member_expire", "0"},
	{"user", "phone_changed_at", "0"},
	{"user", "role", "'user'"},
	{"user", "status", "'normal'"},
	{"user", "total_recharge", "0"},
	{"user", "created_at", "0"},
	{"user", "updated_at", "0"},

	// ---------- order ----------
	{"order", "user_id", "0"},
	{"order", "source", "''"},
	// 发布时间：业务选填字段，用零值时间表示「未填写」，避免 NULL
	{"order", "published_at", "TIMESTAMP '0001-01-01 00:00:00'"},
	{"order", "publisher", "''"},
	{"order", "grade", "''"},
	{"order", "order_no", "''"},
	{"order", "address", "''"},
	{"order", "subject", "''"},
	{"order", "content", "''"},
	{"order", "student_name", "''"},
	{"order", "start_date", "0"},
	{"order", "end_date", "0"},
	{"order", "total_lessons", "0"},
	{"order", "weekly_slots", "''"},
	{"order", "hourly_rate", "0"},
	{"order", "bill_mode", "'hourly'"},
	{"order", "lesson_price", "0"},
	{"order", "total_amount", "0"},
	{"order", "direction", "''"},
	{"order", "subjects", "''"},
	{"order", "student_situation", "''"},
	{"order", "student_gender", "''"},
	{"order", "contact_phone", "''"},
	{"order", "remark", "''"},
	{"order", "remark_flag", "false"},
	{"order", "status", "'running'"},
	{"order", "created_at", "0"},
	{"order", "updated_at", "0"},

	// ---------- lesson_exception ----------
	{"lesson_exception", "user_id", "0"},
	{"lesson_exception", "order_id", "0"},
	{"lesson_exception", "source_date", "0"},
	{"lesson_exception", "slot_index", "0"},
	{"lesson_exception", "type", "''"},
	{"lesson_exception", "new_date", "0"},
	{"lesson_exception", "new_start", "''"},
	{"lesson_exception", "new_end", "''"},
	{"lesson_exception", "duration_minutes", "0"},
	{"lesson_exception", "note", "''"},
	{"lesson_exception", "created_at", "0"},
	{"lesson_exception", "updated_at", "0"},

	// ---------- lesson（历史课次快照）----------
	{"lesson", "user_id", "0"},
	{"lesson", "order_id", "0"},
	{"lesson", "order_no", "''"},
	{"lesson", "date", "''"},
	{"lesson", "weekday", "0"},
	{"lesson", "slot_index", "0"},
	{"lesson", "start_time", "''"},
	{"lesson", "end_time", "''"},
	{"lesson", "duration_minutes", "0"},
	{"lesson", "grade", "''"},
	{"lesson", "student_name", "''"},
	{"lesson", "subject", "''"},
	{"lesson", "address", "''"},
	{"lesson", "content", "''"},
	{"lesson", "remark", "''"},
	{"lesson", "remark_flag", "false"},
	{"lesson", "hourly_rate", "0"},
	{"lesson", "status", "''"},
	{"lesson", "voided", "false"},
	{"lesson", "income", "0"},
	{"lesson", "exception_id", "0"},
	{"lesson", "origin_date", "''"},
	{"lesson", "origin_weekday", "0"},
	{"lesson", "moved_to_date", "''"},
	{"lesson", "moved_to_start", "''"},
	{"lesson", "adjust_note", "''"},
	{"lesson", "created_at", "now()"},
	{"lesson", "updated_at", "now()"},

	// ---------- feedback ----------
	{"feedback", "user_id", "0"},
	{"feedback", "username", "''"},
	{"feedback", "content", "''"},
	{"feedback", "contact", "''"},
	{"feedback", "reply", "''"},
	{"feedback", "status", "'pending'"},
	{"feedback", "created_at", "0"},
	{"feedback", "updated_at", "0"},

	// ---------- recharge ----------
	{"recharge", "user_id", "0"},
	{"recharge", "username", "''"},
	{"recharge", "amount", "0"},
	{"recharge", "period", "''"},
	{"recharge", "period_name", "''"},
	{"recharge", "days", "0"},
	{"recharge", "member_type", "''"},
	{"recharge", "before_expire", "0"},
	{"recharge", "after_expire", "0"},
	{"recharge", "operator_id", "0"},
	{"recharge", "operator_name", "''"},
	{"recharge", "source", "''"},
	{"recharge", "remark", "''"},
	{"recharge", "created_at", "0"},
	{"recharge", "updated_at", "0"},

	// ---------- price_config ----------
	{"price_config", "pro_monthly_amount", "69"},
	{"price_config", "pro_quarterly_amount", "175"},
	{"price_config", "pro_yearly_amount", "529"},
	{"price_config", "student_discount", "0.85"},
	{"price_config", "renewal_discount", "0.9"},
	{"price_config", "updated_at", "0"},

	// ---------- invite_code ----------
	{"invite_code", "code", "''"},
	{"invite_code", "phone", "''"},
	{"invite_code", "used", "0"},
	{"invite_code", "used_by_id", "0"},
	{"invite_code", "used_at", "0"},
	{"invite_code", "created_at", "0"},

	// ---------- message ----------
	{"message", "user_id", "0"},
	{"message", "sender_id", "0"},
	{"message", "sender_name", "''"},
	{"message", "title", "''"},
	{"message", "content", "''"},
	{"message", "type", "'system'"},
	{"message", "read_at", "0"},
	{"message", "created_at", "0"},

	// ---------- student_application ----------
	{"student_application", "user_id", "0"},
	{"student_application", "username", "''"},
	{"student_application", "phone", "''"},
	{"student_application", "invite_code", "''"},
	{"student_application", "student_card_url", "''"},
	{"student_application", "id_card_url", "''"},
	{"student_application", "status", "'pending'"},
	{"student_application", "expire_at", "0"},
	{"student_application", "reviewer_id", "0"},
	{"student_application", "reviewer_name", "''"},
	{"student_application", "remark", "''"},
	{"student_application", "created_at", "0"},
	{"student_application", "reviewed_at", "0"},
}

// dropIndexSQLs 需要清理的重复 / 失效索引：
// 早期各迁移文件建的单列索引与 GORM 按 tag 自动建的索引大量重复，
// 且已被下面的组合索引覆盖（组合索引的左前缀可等价服务单列查询）。
var dropIndexSQLs = []string{
	// order：单列索引全部由 (user_id, ...) 组合索引的左前缀覆盖
	`DROP INDEX IF EXISTS idx_order_user`,
	`DROP INDEX IF EXISTS idx_order_user_id`,
	`DROP INDEX IF EXISTS idx_order_status`,
	`DROP INDEX IF EXISTS idx_order_no`,
	`DROP INDEX IF EXISTS idx_order_order_no`,
	`DROP INDEX IF EXISTS idx_order_start_date`,
	`DROP INDEX IF EXISTS idx_order_end_date`,

	// lesson：order_id 单列索引被 (order_id, date) 覆盖
	`DROP INDEX IF EXISTS idx_lesson_order`,

	// feedback / recharge：旧的 user_id 单列索引与 GORM 建的重名重复
	`DROP INDEX IF EXISTS idx_feedback_user`,
	`DROP INDEX IF EXISTS idx_feedback_user_id`,
	`DROP INDEX IF EXISTS idx_feedback_status`,
	`DROP INDEX IF EXISTS idx_recharge_user`,
	`DROP INDEX IF EXISTS idx_recharge_user_id`,

	// message：user_id 单列与 read_at 单列（read_at 单独无查询场景）
	`DROP INDEX IF EXISTS idx_message_user_id`,
	`DROP INDEX IF EXISTS idx_message_read_at`,

	// student_application：单列索引由组合索引覆盖
	`DROP INDEX IF EXISTS idx_student_application_user_id`,
	`DROP INDEX IF EXISTS idx_student_application_status`,

	// lesson_exception：type 单列无独立查询场景
	`DROP INDEX IF EXISTS idx_lesson_exception_type`,

	// user：学生身份两个单列索引合并为 (teacher_type, student_expire_at)
	`DROP INDEX IF EXISTS idx_user_student_verified_at`,
	`DROP INDEX IF EXISTS idx_user_student_expire_at`,

	// invite_code：phone 普通索引升级为部分唯一索引 uk_invite_phone
	`DROP INDEX IF EXISTS idx_invite_phone`,
}

// createIndexSQLs 按实际查询路径建立的索引（全部 IF NOT EXISTS，可重复执行）。
//
// 设计原则：
//   - 用户维度查询一律以 user_id 打头（课表 / 订单 / 报表都是按用户隔离的）；
//   - 课次物化的去重查询是 (order_id, date) 范围扫描，单独建组合索引；
//   - 分页列表的排序字段尽量并入索引，避免额外排序。
var createIndexSQLs = []string{
	// ---------- user ----------
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_phone ON "user" (phone) WHERE phone <> ''`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email ON "user" (email) WHERE email <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_user_created_at ON "user" (created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_user_member_expire ON "user" (member_expire)`,
	`CREATE INDEX IF NOT EXISTS idx_user_role_status ON "user" (role, status)`,
	`CREATE INDEX IF NOT EXISTS idx_user_teacher_verified ON "user" (teacher_type, verified)`,
	`CREATE INDEX IF NOT EXISTS idx_user_student_expire ON "user" (teacher_type, student_expire_at)`,
	// 邀请归因与风控：按推荐人统计、按注册 IP 查聚集
	`CREATE INDEX IF NOT EXISTS idx_user_invited_by ON "user" (invited_by_id)`,
	`CREATE INDEX IF NOT EXISTS idx_user_register_ip ON "user" (register_ip)`,

	// ---------- order ----------
	`CREATE INDEX IF NOT EXISTS idx_order_user_status ON "order" (user_id, status)`,
	`CREATE INDEX IF NOT EXISTS idx_order_user_range ON "order" (user_id, start_date, end_date)`,
	`CREATE INDEX IF NOT EXISTS idx_order_user_no ON "order" (user_id, order_no)`,

	// ---------- lesson_exception ----------
	`CREATE INDEX IF NOT EXISTS idx_exc_order_date ON lesson_exception (order_id, source_date)`,
	`CREATE INDEX IF NOT EXISTS idx_exc_user_date ON lesson_exception (user_id, new_date)`,
	`CREATE INDEX IF NOT EXISTS idx_exc_user_source ON lesson_exception (user_id, source_date)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uk_exc_slot ON lesson_exception (order_id, source_date, slot_index) WHERE type <> 'extra'`,

	// ---------- lesson ----------
	`CREATE INDEX IF NOT EXISTS idx_lesson_user_date ON lesson (user_id, date)`,
	`CREATE INDEX IF NOT EXISTS idx_lesson_order_date ON lesson (order_id, date)`,
	// 物化去重依据「订单+日期+时段+例外」，加唯一索引兜底，杜绝重复落库
	`CREATE UNIQUE INDEX IF NOT EXISTS uk_lesson_slot ON lesson (order_id, date, slot_index, exception_id)`,

	// ---------- feedback ----------
	`CREATE INDEX IF NOT EXISTS idx_feedback_user_created ON feedback (user_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_feedback_status_created ON feedback (status, created_at DESC)`,

	// ---------- recharge ----------
	`CREATE INDEX IF NOT EXISTS idx_recharge_user_created ON recharge (user_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_recharge_created ON recharge (created_at)`,

	// ---------- message ----------
	`CREATE INDEX IF NOT EXISTS idx_message_user_id ON message (user_id, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_message_unread ON message (user_id) WHERE read_at = 0`,

	// ---------- student_application ----------
	`CREATE INDEX IF NOT EXISTS idx_app_user_status ON student_application (user_id, status, id DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_app_status_id ON student_application (status, id DESC)`,

	// ---------- invite_code ----------
	// 手机号改为唯一索引（WHERE phone <> ''）：会员分享码不绑手机号，允许多条空 phone，
	// 但人工发码必须保证「一个手机号一个码」，应用层查重挡不住并发。
	`CREATE UNIQUE INDEX IF NOT EXISTS uk_invite_phone ON invite_code (phone) WHERE phone <> ''`,
	`CREATE INDEX IF NOT EXISTS idx_invite_used_by ON invite_code (used_by_id)`,
}

// EnsureNotNull 为已有库补齐默认值与非空约束（幂等）。
// 仅当核心表已存在时执行：全新库直接跳过，交由 AutoMigrate 按 model tag 建表，避免首启动产生大量无效报错。
// 已为 NOT NULL 的列会先被 information_schema 判定跳过，稳定状态下几乎无开销。
// colInfo information_schema.columns 中取出的列属性，用于判断是否还需要补默认值 / 非空约束。
// ColumnDefault 用 COALESCE 归一：空串表示「无默认值」，避免在 Go 侧处理 NULL。
type colInfo struct {
	IsNullable    string
	ColumnDefault string
}

func EnsureNotNull() error {
	if !db.Migrator().HasTable("user") {
		return nil
	}
	for _, r := range notNullRules {
		var info colInfo
		if err := db.Raw(
			`SELECT is_nullable, COALESCE(column_default, '') AS column_default
			   FROM information_schema.columns
			  WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`,
			r.table, r.column,
		).Scan(&info).Error; err != nil || info.IsNullable == "" {
			continue // 列不存在（新库尚未建表），跳过
		}
		tbl, col := quoteIdent(r.table), quoteIdent(r.column)

		// 缺默认值就补，保证后续新增列 / 裸 SQL 插入也不会产生 NULL
		if info.ColumnDefault == "" {
			if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s`, tbl, col, r.def)).Error; err != nil {
				logger.Error("set default failed, table=%s column=%s err=%s", r.table, r.column, err.Error())
			}
		}

		// 仍允许 NULL：先把存量 NULL 回填为默认值，再加 NOT NULL 约束
		if info.IsNullable == "YES" {
			if err := db.Exec(fmt.Sprintf(`UPDATE %s SET %s = %s WHERE %s IS NULL`, tbl, col, r.def, col)).Error; err != nil {
				logger.Error("backfill null failed, table=%s column=%s err=%s", r.table, r.column, err.Error())
				continue
			}
			if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET NOT NULL`, tbl, col)).Error; err != nil {
				logger.Error("set not null failed, table=%s column=%s err=%s", r.table, r.column, err.Error())
			}
		}
	}
	return nil
}

// EnsureIndexes 建立索引并清理重复 / 失效索引（幂等，失败仅记日志不影响启动）。
// 索引统一在此维护：model 上不再散落 index tag，避免 AutoMigrate 反复重建被清理掉的索引。
func EnsureIndexes() error {
	for _, sql := range dropIndexSQLs {
		if err := db.Exec(sql).Error; err != nil {
			logger.Error("drop index failed, sql=%s err=%s", sql, err.Error())
		}
	}
	for _, sql := range createIndexSQLs {
		if err := db.Exec(sql).Error; err != nil {
			logger.Error("create index failed, sql=%s err=%s", sql, err.Error())
		}
	}
	return nil
}

// quoteIdent 给标识符加双引号（user / order 等是 PostgreSQL 保留字，必须引用）。
// 入参均来自本文件内的固定清单，不含外部输入。
func quoteIdent(s string) string { return `"` + s + `"` }
