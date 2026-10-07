-- 用户角色字段 + 站点拥有者（ID=1）初始化
-- 服务启动时会执行 AutoMigrate 与 SeedOwner，本文件用于手工执行 / 审阅。

ALTER TABLE "user" ADD COLUMN IF NOT EXISTS role VARCHAR(16) NOT NULL DEFAULT 'user';
COMMENT ON COLUMN "user".role IS 'user=普通用户 admin=管理员 owner=站点拥有者';

-- 指定站点拥有者（把 username 换成实际账号即可；权限一律以 role 字段为准）
-- 拥有者具备管理员权限，通常会同时设为永久会员。
UPDATE "user"
   SET role = 'owner',
       member_type = 'permanent',
       member_start = COALESCE(member_start, created_at),
       member_expire = EXTRACT(EPOCH FROM TIMESTAMP '2099-12-31 23:59:59')::bigint
 WHERE username = '改成你的账号';
