-- 使用身份（user.user_role）：teacher=老师 parent=家长 personal=个人 org=机构
-- 与 teacher_type（师资身份，决定会员价格档位）正交：
-- user_role 只驱动课程表单字段与文案，不参与计费；仅「老师 + 大学生」需要证件认证。
ALTER TABLE "user" ADD COLUMN IF NOT EXISTS user_role VARCHAR(16) NOT NULL DEFAULT 'teacher';

COMMENT ON COLUMN "user".user_role IS '使用身份：teacher=老师 parent=家长 personal=个人 org=机构；仅影响课程表单与文案，不参与计费';
