-- 账号状态字段（冻结 / 正常）+ 登录账号唯一性辅助索引
-- 服务启动时会 AutoMigrate 自动补列，本文件用于手工执行 / 审阅。

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'normal';
COMMENT ON COLUMN "user".status IS '账号状态：normal=正常 frozen=已冻结（冻结后禁止登录）';

-- 手机号 / 邮箱查重（仅对非空值建索引，避免空串冲突）
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_phone ON "user" (phone) WHERE phone <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email ON "user" (email) WHERE email <> '';
