-- 表与字段注释补全（PostgreSQL，可重复执行）
-- 服务启动时由 model.EnsureComments() 自动执行，本文件用于手工执行 / 审阅。
-- 已有库可直接运行：psql -d tutoring -f server/migrations/003_comments.sql
-- 写法约定：只写「字段是什么 + 取值枚举 + 必要格式」，业务规则与流程说明留在代码注释里。

COMMENT ON TABLE "user" IS '用户表：账号信息、角色、师资身份与会员信息';
COMMENT ON COLUMN "user".id IS '主键 ID';
COMMENT ON COLUMN "user".username IS '用户名，唯一，2-32 位';
COMMENT ON COLUMN "user".password IS '登录密码，bcrypt 加密';
COMMENT ON COLUMN "user".avatar IS '头像地址';
COMMENT ON COLUMN "user".phone IS '手机号，登录账号之一';
COMMENT ON COLUMN "user".email IS '邮箱，选填';
COMMENT ON COLUMN "user".member_type IS '会员类型：trial=免费试用 daily / monthly / quarterly / yearly=付费会员 permanent=永久会员';
COMMENT ON COLUMN "user".member_start IS '会员生效时间（Unix 秒）';
COMMENT ON COLUMN "user".member_expire IS '会员到期时间（Unix 秒）';
COMMENT ON COLUMN "user".role IS '角色：owner=站长 admin=管理员 user=普通用户';
COMMENT ON COLUMN "user".status IS '账号状态：normal=正常 frozen=已冻结';
COMMENT ON COLUMN "user".created_at IS '注册时间（Unix 秒）';
COMMENT ON COLUMN "user".updated_at IS '更新时间（Unix 秒）';

COMMENT ON TABLE "order" IS '家教订单表：一次辅导需求';
COMMENT ON COLUMN "order".id IS '主键 ID';
COMMENT ON COLUMN "order".user_id IS '所属用户 ID，关联 user.id';
COMMENT ON COLUMN "order".source IS '信息来源';
COMMENT ON COLUMN "order".published_at IS '订单发布时间，零值表示未填写';
COMMENT ON COLUMN "order".publisher IS '发布人';
COMMENT ON COLUMN "order".grade IS '年级';
COMMENT ON COLUMN "order".order_no IS '订单编号，留空自动生成';
COMMENT ON COLUMN "order".address IS '上课地址';
COMMENT ON COLUMN "order".subject IS '辅导科目（单科目）';
COMMENT ON COLUMN "order".content IS '辅导内容';
COMMENT ON COLUMN "order".student_name IS '学生姓名';
COMMENT ON COLUMN "order".start_date IS '补课开始日期（Unix 秒）';
COMMENT ON COLUMN "order".end_date IS '补课结束日期（Unix 秒），0 表示长期';
COMMENT ON COLUMN "order".weekly_slots IS '每周上课时段 JSON 数组，例：[{"days":[1,2,3,4],"start":"18:00","end":"20:00"}]，days: 1=周一 ... 7=周日';
COMMENT ON COLUMN "order".duration_minutes IS '单次课时时长（分钟）';
COMMENT ON COLUMN "order".hourly_rate IS '时薪（元/小时）';
COMMENT ON COLUMN "order".student_situation IS '学生情况';
COMMENT ON COLUMN "order".student_gender IS '学生性别：男 / 女';
COMMENT ON COLUMN "order".remark IS '备注';
COMMENT ON COLUMN "order".remark_flag IS '特殊备注标记，true 时课表显示红点';
COMMENT ON COLUMN "order".status IS '订单状态：pending=未开始 running=进行中 finished=已结束 closed=已关闭';
COMMENT ON COLUMN "order".created_at IS '创建时间（Unix 秒）';
COMMENT ON COLUMN "order".updated_at IS '更新时间（Unix 秒）';

COMMENT ON TABLE feedback IS '问题反馈表：用户提交的问题与建议';
COMMENT ON COLUMN feedback.id IS '主键 ID';
COMMENT ON COLUMN feedback.user_id IS '提交用户 ID，关联 user.id';
COMMENT ON COLUMN feedback.username IS '提交时用户名快照';
COMMENT ON COLUMN feedback.content IS '反馈内容，最多 1000 字';
COMMENT ON COLUMN feedback.contact IS '联系方式，选填';
COMMENT ON COLUMN feedback.reply IS '官方回复内容';
COMMENT ON COLUMN feedback.status IS '处理状态：pending=待处理 resolved=已回复';
COMMENT ON COLUMN feedback.created_at IS '提交时间（Unix 秒）';
COMMENT ON COLUMN feedback.updated_at IS '更新时间（Unix 秒）';
