-- 家教教培站点 初始化表结构（PostgreSQL）
-- 说明：服务启动时会自动执行 AutoMigrate + 字段注释，本文件用于手工建库 / 审阅结构。

CREATE TABLE IF NOT EXISTS "user" (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL UNIQUE,
    password      VARCHAR(128) NOT NULL,
    avatar        VARCHAR(255) DEFAULT '',
    phone         VARCHAR(32)  DEFAULT '',
    email         VARCHAR(128) DEFAULT '',
    member_type   VARCHAR(16)  DEFAULT 'trial',
    member_start  BIGINT DEFAULT 0,
    member_expire BIGINT DEFAULT 0,
    created_at    BIGINT DEFAULT 0,
    updated_at    BIGINT DEFAULT 0
);

COMMENT ON TABLE "user" IS '用户表：站点注册用户，含会员与角色信息';
COMMENT ON COLUMN "user".id IS '主键 ID（权限由 role 字段判定，与 ID 无关）';
COMMENT ON COLUMN "user".username IS '用户名，唯一，2-32 位，不含空格';
COMMENT ON COLUMN "user".password IS '密码，bcrypt 加密存储';
COMMENT ON COLUMN "user".avatar IS '头像地址，支持上传或外部 URL';
COMMENT ON COLUMN "user".phone IS '手机号，注册必填，11 位中国大陆号码';
COMMENT ON COLUMN "user".email IS '邮箱，选填，用于找回账号';
COMMENT ON COLUMN "user".member_type IS '会员类型：trial=免费试用 monthly=包月 yearly=包年 permanent=永久会员';
COMMENT ON COLUMN "user".member_start IS '会员生效时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN "user".member_expire IS '会员到期时间，秒级时间戳（Unix 秒），过期后订单与课表不可用（管理员/拥有者除外）';
COMMENT ON COLUMN "user".created_at IS '注册时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN "user".updated_at IS '更新时间，秒级时间戳（Unix 秒）';

CREATE TABLE IF NOT EXISTS "order" (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    source            VARCHAR(128) DEFAULT '',
    published_at      TIMESTAMP,
    publisher         VARCHAR(64)  DEFAULT '',
    grade             VARCHAR(64)  DEFAULT '',
    order_no          VARCHAR(64)  DEFAULT '',
    address           VARCHAR(255) DEFAULT '',
    subject           VARCHAR(64)  DEFAULT '',
    content           TEXT         DEFAULT '',
    student_name      VARCHAR(64)  DEFAULT '',
    start_date        VARCHAR(16)  DEFAULT '',
    end_date          VARCHAR(16)  DEFAULT '',
    weekly_slots      TEXT         DEFAULT '',
    duration_minutes  INTEGER      DEFAULT 0,
    hourly_rate       NUMERIC(10,2) DEFAULT 0,
    student_situation TEXT         DEFAULT '',
    student_gender    VARCHAR(8)   DEFAULT '',
    remark            TEXT         DEFAULT '',
    remark_flag       BOOLEAN      DEFAULT FALSE,
    status            VARCHAR(16)  DEFAULT 'running',
    created_at        BIGINT DEFAULT 0,
    updated_at        BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_order_user ON "order" (user_id);
CREATE INDEX IF NOT EXISTS idx_order_status ON "order" (status);
CREATE INDEX IF NOT EXISTS idx_order_no ON "order" (order_no);

COMMENT ON TABLE "order" IS '家教订单表：一次辅导需求，含上课周期与每周频次';
COMMENT ON COLUMN "order".id IS '主键 ID';
COMMENT ON COLUMN "order".user_id IS '所属用户 ID，关联 user.id';
COMMENT ON COLUMN "order".source IS '信息来源，如微信群名、中介渠道';
COMMENT ON COLUMN "order".published_at IS '订单发布时间（信息发布时间）';
COMMENT ON COLUMN "order".publisher IS '发布人，如发布订单的老师或中介';
COMMENT ON COLUMN "order".grade IS '年级，如初二、高一';
COMMENT ON COLUMN "order".order_no IS '订单编号，群内编号或自定义编号';
COMMENT ON COLUMN "order".address IS '上课地址';
COMMENT ON COLUMN "order".subject IS '辅导科目，如数学、英语';
COMMENT ON COLUMN "order".content IS '辅导内容说明';
COMMENT ON COLUMN "order".student_name IS '学生姓名';
COMMENT ON COLUMN "order".start_date IS '补课开始日期 yyyy-mm-dd';
COMMENT ON COLUMN "order".end_date IS '补课结束日期 yyyy-mm-dd，留空表示长期';
COMMENT ON COLUMN "order".weekly_slots IS '每周上课时段 JSON 数组，例：[{"days":[1,2,3,4],"start":"18:00","end":"20:00"}]，days: 1=周一 ... 7=周日';
COMMENT ON COLUMN "order".duration_minutes IS '单次课时时长（分钟），课表展示优先按此值计算';
COMMENT ON COLUMN "order".hourly_rate IS '时薪（元/小时），用于课表预计收入统计';
COMMENT ON COLUMN "order".student_situation IS '学生情况：成绩、性格、薄弱环节等';
COMMENT ON COLUMN "order".student_gender IS '学生性别：男 / 女';
COMMENT ON COLUMN "order".remark IS '备注，课表中悬浮可查看完整内容';
COMMENT ON COLUMN "order".remark_flag IS '特殊备注标记，true 时课表课程右上角显示红点';
COMMENT ON COLUMN "order".status IS '订单状态：pending=待开始 running=进行中 finished=已结束 closed=已关闭';
COMMENT ON COLUMN "order".created_at IS '创建时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN "order".updated_at IS '更新时间，秒级时间戳（Unix 秒）';

CREATE TABLE IF NOT EXISTS feedback (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT  NOT NULL,
    username   VARCHAR(64) DEFAULT '',
    content    TEXT    NOT NULL,
    contact    VARCHAR(64) DEFAULT '',
    reply      TEXT    DEFAULT '',
    status     VARCHAR(16) DEFAULT 'pending',
    created_at BIGINT DEFAULT 0,
    updated_at BIGINT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_feedback_user ON feedback (user_id);

COMMENT ON TABLE feedback IS '问题反馈表：用户在使用指南页提交的问题与建议';
COMMENT ON COLUMN feedback.id IS '主键 ID';
COMMENT ON COLUMN feedback.user_id IS '提交用户 ID，关联 user.id';
COMMENT ON COLUMN feedback.username IS '提交时的用户名快照';
COMMENT ON COLUMN feedback.content IS '反馈内容，最多 1000 字';
COMMENT ON COLUMN feedback.contact IS '联系方式（微信 / 手机号），选填';
COMMENT ON COLUMN feedback.reply IS '官方回复内容';
COMMENT ON COLUMN feedback.status IS '处理状态：pending=待处理 resolved=已回复';
COMMENT ON COLUMN feedback.created_at IS '提交时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN feedback.updated_at IS '更新时间，秒级时间戳（Unix 秒）';
