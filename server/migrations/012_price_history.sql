-- 价格变更历史（price_history）：记录每次价格卡片生效前后的值、操作人与来源，支持审计与一键回滚。
-- 业务代码也会在 AutoMigrate 中自动建表；本文件供手工执行 / 存量库对账使用，可重复执行。
CREATE TABLE IF NOT EXISTS price_history (
  id BIGSERIAL PRIMARY KEY,
  card VARCHAR(16) NOT NULL,
  before_value TEXT NOT NULL,
  after_value TEXT NOT NULL,
  source VARCHAR(16) NOT NULL,
  operator_id BIGINT NOT NULL DEFAULT 0,
  operator_name VARCHAR(32) NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL DEFAULT 0
);

COMMENT ON TABLE price_history IS '价格变更历史：Before / After 为对应卡片字段的 JSON 快照';
COMMENT ON COLUMN price_history.card IS '价格卡片：pro=标准价 student=学生折扣 renewal=平季优惠 activity=限时活动';
COMMENT ON COLUMN price_history.source IS '变更来源：immediate=立即保存 schedule=预约到点 rollback=回滚';
COMMENT ON COLUMN price_history.operator_id IS '操作人用户 ID；0 表示系统调度';
COMMENT ON COLUMN price_history.operator_name IS '操作人用户名；系统调度为「系统调度」';

CREATE INDEX IF NOT EXISTS idx_price_history_card ON price_history (card);
CREATE INDEX IF NOT EXISTS idx_price_history_created_at ON price_history (created_at);
