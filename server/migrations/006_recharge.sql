-- 累计充值金额 + 充值记录表
-- 服务启动会 AutoMigrate 自动补列/建表，本文件用于手工执行 / 审阅。

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS total_recharge NUMERIC(12,2) NOT NULL DEFAULT 0;
COMMENT ON COLUMN "user".total_recharge IS '累计充值金额（元），每次续费时累加';

CREATE TABLE IF NOT EXISTS recharge (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL,
    username      VARCHAR(64) DEFAULT '',
    amount        NUMERIC(12,2) DEFAULT 0,
    period        VARCHAR(16) DEFAULT '',
    period_name   VARCHAR(16) DEFAULT '',
    days          INTEGER DEFAULT 0,
    member_type   VARCHAR(16) DEFAULT '',
    before_expire BIGINT DEFAULT 0,
    after_expire  BIGINT DEFAULT 0,
    operator_id   BIGINT DEFAULT 0,
    operator_name VARCHAR(64) DEFAULT '',
    source        VARCHAR(16) DEFAULT '',
    remark        VARCHAR(255) DEFAULT '',
    created_at    BIGINT DEFAULT 0,
    updated_at    BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_recharge_user ON recharge (user_id);

COMMENT ON TABLE recharge IS '会员充值记录表：每次续费（后台充值 / 用户自助）的金额、时长与到期时间';
COMMENT ON COLUMN recharge.id IS '主键 ID';
COMMENT ON COLUMN recharge.user_id IS '充值账号 ID，关联 user.id';
COMMENT ON COLUMN recharge.username IS '充值时的用户名快照';
COMMENT ON COLUMN recharge.amount IS '本次充值金额（元）';
COMMENT ON COLUMN recharge.period IS '续费时长：monthly=一月 yearly=一年';
COMMENT ON COLUMN recharge.period_name IS '续费时长中文名：一月 / 一年';
COMMENT ON COLUMN recharge.days IS '本次实际增加的天数';
COMMENT ON COLUMN recharge.member_type IS '充值后的会员类型';
COMMENT ON COLUMN recharge.before_expire IS '充值前的会员到期时间（秒级时间戳）';
COMMENT ON COLUMN recharge.after_expire IS '充值后的会员到期时间（秒级时间戳）';
COMMENT ON COLUMN recharge.operator_id IS '操作人 ID，0 表示用户自助开通';
COMMENT ON COLUMN recharge.operator_name IS '操作人名称';
COMMENT ON COLUMN recharge.source IS '来源：admin=后台充值 self=用户自助开通';
COMMENT ON COLUMN recharge.remark IS '备注';
COMMENT ON COLUMN recharge.created_at IS '充值时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN recharge.updated_at IS '更新时间，秒级时间戳（Unix 秒）';
