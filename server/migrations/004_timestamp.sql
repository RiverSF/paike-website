-- 创建时间 / 更新时间 / 会员生效时间 改为秒级时间戳（BIGINT）
-- 已有库执行本文件即可完成类型与数据转换（TIMESTAMP -> EXTRACT(EPOCH)）
-- psql -d tutoring -f server/migrations/004_timestamp.sql

ALTER TABLE "user" ALTER COLUMN member_start DROP DEFAULT;
ALTER TABLE "user" ALTER COLUMN member_start TYPE BIGINT
  USING CASE WHEN member_start IS NULL THEN 0 ELSE EXTRACT(EPOCH FROM member_start)::bigint END;

ALTER TABLE "user" ALTER COLUMN member_expire DROP DEFAULT;
ALTER TABLE "user" ALTER COLUMN member_expire TYPE BIGINT
  USING CASE WHEN member_expire IS NULL THEN 0 ELSE EXTRACT(EPOCH FROM member_expire)::bigint END;

ALTER TABLE "user" ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE "user" ALTER COLUMN created_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM created_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);

ALTER TABLE "user" ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE "user" ALTER COLUMN updated_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM updated_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);

ALTER TABLE "order" ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE "order" ALTER COLUMN created_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM created_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);

ALTER TABLE "order" ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE "order" ALTER COLUMN updated_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM updated_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);

ALTER TABLE feedback ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE feedback ALTER COLUMN created_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM created_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);

ALTER TABLE feedback ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE feedback ALTER COLUMN updated_at TYPE BIGINT
  USING COALESCE(EXTRACT(EPOCH FROM updated_at)::bigint, EXTRACT(EPOCH FROM now())::bigint);
