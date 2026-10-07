-- 010: 课程例外增加「本次课时费」
-- 服务启动时会执行 AutoMigrate（缺失字段自动新增），本文件用于手工执行 / 审阅。

ALTER TABLE lesson_exception ADD COLUMN IF NOT EXISTS income DOUBLE PRECISION NOT NULL DEFAULT 0;
COMMENT ON COLUMN lesson_exception.income IS '本次课时费（元）；0 表示按订单标准计费';
