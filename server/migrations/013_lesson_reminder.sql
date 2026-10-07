-- 开课提醒去重表（lesson_reminder）：同一账号的同一节课（订单 + 日期 + 开始时间）只提醒一次。
-- 依赖唯一索引保证定时任务重复 / 并发扫描时不会重复推送站内信。
-- 业务代码也会在 AutoMigrate 中自动建表；本文件供手工执行 / 存量库对账使用，可重复执行。
CREATE TABLE IF NOT EXISTS lesson_reminder (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,
  date VARCHAR(10) NOT NULL,
  start_time VARCHAR(8) NOT NULL,
  created_at BIGINT NOT NULL DEFAULT 0
);

COMMENT ON TABLE lesson_reminder IS '开课提醒去重记录：每节课只发送一次提醒站内信';
COMMENT ON COLUMN lesson_reminder.date IS '课程日期 yyyy-mm-dd';
COMMENT ON COLUMN lesson_reminder.start_time IS '课程开始时间 HH:MM';

CREATE UNIQUE INDEX IF NOT EXISTS idx_lesson_reminder_once
  ON lesson_reminder (user_id, order_id, date, start_time);

CREATE INDEX IF NOT EXISTS idx_lesson_reminder_created_at ON lesson_reminder (created_at);
