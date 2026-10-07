-- 下线邀请码机制并移除邮箱：注册改为手机号自助注册，登录仅支持手机号。
-- 原因：邀请码准入已取消（不必再人工发码 / 核销），邮箱也不再作为登录账号或联系方式。
-- 对应的表结构与注释已同步从 model 移除（model/invite.go 删除、User.Email / InvitedByID、
-- StudentApplication.InviteCode 字段删除），启动时不会再自动重建；
-- 本文件用于存量库清理，需手工执行一次，可重复执行。
-- 执行方式：psql -d tutoring -f server/migrations/015_drop_invite_code_and_email.sql

-- 一、删除邀请码表（索引会随表一起消失）
DROP INDEX IF EXISTS uk_invite_phone;
DROP INDEX IF EXISTS idx_invite_used_by;
DROP INDEX IF EXISTS idx_invite_phone;
DROP TABLE IF EXISTS invite_code;

-- 二、删除用户表的邮箱列（唯一索引先显式清理）
DROP INDEX IF EXISTS idx_user_email;
ALTER TABLE "user" DROP COLUMN IF EXISTS email;

-- 三、删除注册邀请归因相关的列
DROP INDEX IF EXISTS idx_user_invited_by;
ALTER TABLE "user" DROP COLUMN IF EXISTS invited_by_id;
ALTER TABLE "user" ALTER COLUMN register_src SET DEFAULT 'phone';
UPDATE "user" SET register_src = 'phone' WHERE register_src = 'invite';

-- 四、删除学生申请表里的邀请码快照列
ALTER TABLE student_application DROP COLUMN IF EXISTS invite_code;
