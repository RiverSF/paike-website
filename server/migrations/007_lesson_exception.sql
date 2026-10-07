-- 课程例外表：满足「某一次课程的临时变动」（改期 / 停课 / 改时间 / 临时加课）
-- 设计原则：订单的每周固定频次规则（order.weekly_slots）保持不变，单次变动以例外覆盖。
-- 服务启动会 AutoMigrate 自动建表，本文件用于手工执行 / 审阅。

CREATE TABLE IF NOT EXISTS lesson_exception (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT      NOT NULL,
    order_id         BIGINT      NOT NULL,
    source_date      BIGINT      NOT NULL DEFAULT 0,
    slot_index       INTEGER     NOT NULL DEFAULT 0,
    type             VARCHAR(16) NOT NULL DEFAULT '',
    new_date         BIGINT      NOT NULL DEFAULT 0,
    new_start        VARCHAR(8)  DEFAULT '',
    new_end          VARCHAR(8)  DEFAULT '',
    duration_minutes INTEGER     NOT NULL DEFAULT 0,
    note             TEXT        DEFAULT '',
    created_at       BIGINT      NOT NULL DEFAULT 0,
    updated_at       BIGINT      NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_exc_order_date ON lesson_exception (order_id, source_date);
CREATE INDEX IF NOT EXISTS idx_exc_user_date ON lesson_exception (user_id, new_date);

-- 同一「原定课时」只允许一条例外（重复调整走更新，不会叠加）
CREATE UNIQUE INDEX IF NOT EXISTS uk_exc_slot
    ON lesson_exception (order_id, source_date, slot_index)
    WHERE type <> 'extra';

COMMENT ON TABLE lesson_exception IS '课程例外表：单次课程的改期/停课/改时间/加课，覆盖订单的每周固定频次展开结果';
COMMENT ON COLUMN lesson_exception.id IS '主键 ID';
COMMENT ON COLUMN lesson_exception.user_id IS '所属用户 ID，关联 user.id';
COMMENT ON COLUMN lesson_exception.order_id IS '所属订单 ID，关联 order.id';
COMMENT ON COLUMN lesson_exception.source_date IS '原定上课日期（秒级时间戳，当天 00:00）；extra 时等于 new_date';
COMMENT ON COLUMN lesson_exception.slot_index IS '对应 order.weekly_slots 的第几段（从 0 开始）；extra 固定为 -1';
COMMENT ON COLUMN lesson_exception.type IS 'cancel=本次停课 move=本次改期 time=本次仅改时间 extra=临时加课';
COMMENT ON COLUMN lesson_exception.new_date IS '调整后的上课日期（秒级时间戳，当天 00:00）；cancel 时为 0';
COMMENT ON COLUMN lesson_exception.new_start IS '调整后的开始时间 HH:MM，为空沿用订单时段';
COMMENT ON COLUMN lesson_exception.new_end IS '调整后的结束时间 HH:MM，为空按订单单次时长推算';
COMMENT ON COLUMN lesson_exception.duration_minutes IS '本次课时长度（分钟），0 表示沿用订单时长';
COMMENT ON COLUMN lesson_exception.note IS '调整原因备注';
COMMENT ON COLUMN lesson_exception.created_at IS '创建时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN lesson_exception.updated_at IS '更新时间，秒级时间戳（Unix 秒）';
