-- 库表规范化：清空历史课次 + 全字段非空 + 补全注释 + 索引优化（PostgreSQL，可重复执行）
--
-- 执行方式：psql -d tutoring -f server/migrations/008_schema_optimize.sql
-- 服务启动时 model.EnsureNotNull / EnsureIndexes / EnsureComments 会自动执行同样的逻辑，
-- 本文件用于已有库手工一次性执行 / 审阅结构。
--
-- 约定：所有业务字段均不允许 NULL。
--   字符串 → ''   数值 → 0   布尔 → false   枚举 / 状态 → 业务默认值
--   order.published_at 为零值时间（0001-01-01）表示「未填写」

-- ===========================================================================
-- 1. 清空已物化的历史课次
--    课次由每日定时任务（仅物化前一日）重新生成，清空后会自动按当前订单配置重建。
--    注意：本节为破坏性操作，会删除 lesson 表全部数据，请确认后再执行。
-- ===========================================================================
TRUNCATE TABLE lesson RESTART IDENTITY;

-- ===========================================================================
-- 2. 全字段非空：先把存量 NULL 回填为默认值，再补默认值与 NOT NULL 约束
--    用 DO 块按「列名 + 默认值」清单循环执行，避免上百条重复语句
-- ===========================================================================

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('username',                $$''$$),
        ('password',                $$''$$),
        ('avatar',                  $$''$$),
        ('phone',                   $$''$$),
        ('email',                   $$''$$),
        ('teacher_type',            $$'professional'$$),
        ('reg_teacher_type',        $$''$$),
        ('verified',                $$0$$),
        ('student_card_url',        $$''$$),
        ('id_card_url',             $$''$$),
        ('student_verified_at',     $$0$$),
        ('student_expire_at',       $$0$$),
        ('student_expire_notified', $$false$$),
        ('member_expire_notified',  $$false$$),
        ('member_type',             $$'trial'$$),
        ('member_start',            $$0$$),
        ('member_expire',           $$0$$),
        ('role',                    $$'user'$$),
        ('status',                  $$'normal'$$),
        ('total_recharge',          $$0$$),
        ('created_at',              $$0$$),
        ('updated_at',              $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE "user" SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE "user" ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE "user" ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',           $$0$$),
        ('source',            $$''$$),
        ('published_at',      $$TIMESTAMP '0001-01-01 00:00:00'$$),
        ('publisher',         $$''$$),
        ('grade',             $$''$$),
        ('order_no',          $$''$$),
        ('address',           $$''$$),
        ('subject',           $$''$$),
        ('content',           $$''$$),
        ('student_name',      $$''$$),
        ('start_date',        $$0$$),
        ('end_date',          $$0$$),
        ('weekly_slots',      $$''$$),
        ('hourly_rate',       $$0$$),
        ('bill_mode',         $$'hourly'$$),
        ('lesson_price',      $$0$$),
        ('subjects',          $$''$$),
        ('student_situation', $$''$$),
        ('student_gender',    $$''$$),
        ('contact_phone',     $$''$$),
        ('remark',            $$''$$),
        ('remark_flag',       $$false$$),
        ('status',            $$'running'$$),
        ('created_at',        $$0$$),
        ('updated_at',        $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE "order" SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE "order" ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE "order" ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',          $$0$$),
        ('order_id',         $$0$$),
        ('source_date',      $$0$$),
        ('slot_index',       $$0$$),
        ('type',             $$''$$),
        ('new_date',         $$0$$),
        ('new_start',        $$''$$),
        ('new_end',          $$''$$),
        ('duration_minutes', $$0$$),
        ('note',             $$''$$),
        ('created_at',       $$0$$),
        ('updated_at',       $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE lesson_exception SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE lesson_exception ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE lesson_exception ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',          $$0$$),
        ('order_id',         $$0$$),
        ('order_no',         $$''$$),
        ('date',             $$''$$),
        ('weekday',          $$0$$),
        ('slot_index',       $$0$$),
        ('start_time',       $$''$$),
        ('end_time',         $$''$$),
        ('duration_minutes', $$0$$),
        ('grade',            $$''$$),
        ('student_name',     $$''$$),
        ('subject',          $$''$$),
        ('address',          $$''$$),
        ('content',          $$''$$),
        ('remark',           $$''$$),
        ('remark_flag',      $$false$$),
        ('hourly_rate',      $$0$$),
        ('status',           $$''$$),
        ('voided',           $$false$$),
        ('income',           $$0$$),
        ('exception_id',     $$0$$),
        ('origin_date',      $$''$$),
        ('origin_weekday',   $$0$$),
        ('moved_to_date',    $$''$$),
        ('moved_to_start',   $$''$$),
        ('adjust_note',      $$''$$),
        ('created_at',       $$now()$$),
        ('updated_at',       $$now()$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE lesson SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE lesson ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE lesson ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',    $$0$$),
        ('username',   $$''$$),
        ('content',    $$''$$),
        ('contact',    $$''$$),
        ('reply',      $$''$$),
        ('status',     $$'pending'$$),
        ('created_at', $$0$$),
        ('updated_at', $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE feedback SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE feedback ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE feedback ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',       $$0$$),
        ('username',      $$''$$),
        ('amount',        $$0$$),
        ('period',        $$''$$),
        ('period_name',   $$''$$),
        ('days',          $$0$$),
        ('member_type',   $$''$$),
        ('before_expire', $$0$$),
        ('after_expire',  $$0$$),
        ('operator_id',   $$0$$),
        ('operator_name', $$''$$),
        ('source',        $$''$$),
        ('remark',        $$''$$),
        ('created_at',    $$0$$),
        ('updated_at',    $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE recharge SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE recharge ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE recharge ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('pro_monthly_amount',   $$69$$),
        ('pro_quarterly_amount', $$175$$),
        ('pro_yearly_amount',    $$529$$),
        ('student_discount',     $$0.85$$),
        ('renewal_discount',     $$0.9$$),
        ('updated_at',           $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE price_config SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE price_config ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE price_config ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('code',       $$''$$),
        ('phone',      $$''$$),
        ('used',       $$0$$),
        ('used_by_id', $$0$$),
        ('used_at',    $$0$$),
        ('created_at', $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE invite_code SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE invite_code ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE invite_code ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',     $$0$$),
        ('sender_id',   $$0$$),
        ('sender_name', $$''$$),
        ('title',       $$''$$),
        ('content',     $$''$$),
        ('type',        $$'system'$$),
        ('read_at',     $$0$$),
        ('created_at',  $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE message SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE message ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE message ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('user_id',         $$0$$),
        ('username',        $$''$$),
        ('phone',           $$''$$),
        ('invite_code',     $$''$$),
        ('student_card_url',$$''$$),
        ('id_card_url',     $$''$$),
        ('status',          $$'pending'$$),
        ('expire_at',       $$0$$),
        ('reviewer_id',     $$0$$),
        ('reviewer_name',   $$''$$),
        ('remark',          $$''$$),
        ('created_at',      $$0$$),
        ('reviewed_at',     $$0$$)
    ) AS t(col, def)
    LOOP
        EXECUTE format('UPDATE student_application SET %I = %s WHERE %I IS NULL', r.col, r.def, r.col);
        EXECUTE format('ALTER TABLE student_application ALTER COLUMN %I SET DEFAULT %s', r.col, r.def);
        EXECUTE format('ALTER TABLE student_application ALTER COLUMN %I SET NOT NULL', r.col);
    END LOOP;
END $$;

-- ===========================================================================
-- 3. 索引优化
--    原则：用户维度查询一律以 user_id 打头；单列索引能被组合索引左前缀覆盖的一律清理
-- ===========================================================================

-- 清理重复 / 失效索引
DROP INDEX IF EXISTS idx_order_user;
DROP INDEX IF EXISTS idx_order_user_id;
DROP INDEX IF EXISTS idx_order_status;
DROP INDEX IF EXISTS idx_order_no;
DROP INDEX IF EXISTS idx_order_order_no;
DROP INDEX IF EXISTS idx_order_start_date;
DROP INDEX IF EXISTS idx_order_end_date;

DROP INDEX IF EXISTS idx_lesson_order;

DROP INDEX IF EXISTS idx_feedback_user;
DROP INDEX IF EXISTS idx_feedback_user_id;
DROP INDEX IF EXISTS idx_feedback_status;

DROP INDEX IF EXISTS idx_recharge_user;
DROP INDEX IF EXISTS idx_recharge_user_id;

DROP INDEX IF EXISTS idx_message_user_id;
DROP INDEX IF EXISTS idx_message_read_at;

DROP INDEX IF EXISTS idx_student_application_user_id;
DROP INDEX IF EXISTS idx_student_application_status;

DROP INDEX IF EXISTS idx_lesson_exception_type;

DROP INDEX IF EXISTS idx_user_student_verified_at;
DROP INDEX IF EXISTS idx_user_student_expire_at;

-- 新建索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_phone ON "user" (phone) WHERE phone <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_email ON "user" (email) WHERE email <> '';
CREATE INDEX IF NOT EXISTS idx_user_created_at ON "user" (created_at);
CREATE INDEX IF NOT EXISTS idx_user_member_expire ON "user" (member_expire);
CREATE INDEX IF NOT EXISTS idx_user_role_status ON "user" (role, status);
CREATE INDEX IF NOT EXISTS idx_user_teacher_verified ON "user" (teacher_type, verified);
CREATE INDEX IF NOT EXISTS idx_user_student_expire ON "user" (teacher_type, student_expire_at);

CREATE INDEX IF NOT EXISTS idx_order_user_status ON "order" (user_id, status);
CREATE INDEX IF NOT EXISTS idx_order_user_range ON "order" (user_id, start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_order_user_no ON "order" (user_id, order_no);

CREATE INDEX IF NOT EXISTS idx_exc_order_date ON lesson_exception (order_id, source_date);
CREATE INDEX IF NOT EXISTS idx_exc_user_date ON lesson_exception (user_id, new_date);
CREATE INDEX IF NOT EXISTS idx_exc_user_source ON lesson_exception (user_id, source_date);
CREATE UNIQUE INDEX IF NOT EXISTS uk_exc_slot
    ON lesson_exception (order_id, source_date, slot_index)
    WHERE type <> 'extra';

CREATE INDEX IF NOT EXISTS idx_lesson_user_date ON lesson (user_id, date);
CREATE INDEX IF NOT EXISTS idx_lesson_order_date ON lesson (order_id, date);
-- 物化去重依据「订单+日期+时段+例外」，唯一索引兜底杜绝重复落库
CREATE UNIQUE INDEX IF NOT EXISTS uk_lesson_slot ON lesson (order_id, date, slot_index, exception_id);

CREATE INDEX IF NOT EXISTS idx_feedback_user_created ON feedback (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_status_created ON feedback (status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_recharge_user_created ON recharge (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recharge_created ON recharge (created_at);

CREATE INDEX IF NOT EXISTS idx_message_user_id ON message (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_message_unread ON message (user_id) WHERE read_at = 0;

CREATE INDEX IF NOT EXISTS idx_app_user_status ON student_application (user_id, status, id DESC);
CREATE INDEX IF NOT EXISTS idx_app_status_id ON student_application (status, id DESC);

CREATE INDEX IF NOT EXISTS idx_invite_phone ON invite_code (phone);
CREATE INDEX IF NOT EXISTS idx_invite_used_by ON invite_code (used_by_id);

-- ===========================================================================
-- 4. 补全注释
--    user / order / feedback / recharge 的注释见 003_comments.sql；
--    下列为新增表与后加字段，全量注释由服务启动时的 EnsureComments 统一写入
--    （见 internal/model/comments.go）。
-- ===========================================================================

-- user 后加字段
COMMENT ON COLUMN "user".teacher_type IS '师资身份：student=大学生家教 professional=专职老师';
COMMENT ON COLUMN "user".reg_teacher_type IS '注册时选择的师资身份';
COMMENT ON COLUMN "user".verified IS '师资身份认证：0=未认证 1=已认证';
COMMENT ON COLUMN "user".student_card_url IS '学生证照片地址';
COMMENT ON COLUMN "user".id_card_url IS '身份证照片地址';
COMMENT ON COLUMN "user".student_verified_at IS '学生身份审核通过时间（Unix 秒）';
COMMENT ON COLUMN "user".student_expire_at IS '学生身份有效期（毕业时间，Unix 秒）';
COMMENT ON COLUMN "user".student_expire_notified IS '学生身份到期提醒是否已发送';
COMMENT ON COLUMN "user".member_expire_notified IS '会员到期提醒是否已发送';

-- order 后加字段
COMMENT ON COLUMN "order".subjects IS '辅导科目多选，JSON 数组字符串';
COMMENT ON COLUMN "order".bill_mode IS '计费方式：hourly=按小时 lesson=按单次课时';
COMMENT ON COLUMN "order".lesson_price IS '单次课时价格（元）';
COMMENT ON COLUMN "order".contact_phone IS '对方联系电话（收入=学员家长，开支=授课老师）';
COMMENT ON COLUMN "order".published_at IS '订单发布时间，零值表示未填写';
COMMENT ON COLUMN "order".status IS '订单状态：pending=未开始 running=进行中 finished=已结束';

COMMENT ON TABLE lesson_exception IS '课程例外表：单次课程的改期 / 停课 / 改时间 / 加课';
COMMENT ON COLUMN lesson_exception.id IS '主键 ID';
COMMENT ON COLUMN lesson_exception.user_id IS '所属用户 ID，关联 user.id';
COMMENT ON COLUMN lesson_exception.order_id IS '所属订单 ID，关联 order.id';
COMMENT ON COLUMN lesson_exception.source_date IS '原定上课日期（Unix 秒）';
COMMENT ON COLUMN lesson_exception.slot_index IS '对应 weekly_slots 的时段下标（从 0 开始），加课为 -1';
COMMENT ON COLUMN lesson_exception.type IS '类型：cancel=停课 move=改期 time=仅改时间 extra=临时加课';
COMMENT ON COLUMN lesson_exception.new_date IS '调整后上课日期（Unix 秒），cancel 时为 0';
COMMENT ON COLUMN lesson_exception.new_start IS '调整后开始时间 HH:MM，为空沿用订单时段';
COMMENT ON COLUMN lesson_exception.new_end IS '调整后结束时间 HH:MM';
COMMENT ON COLUMN lesson_exception.duration_minutes IS '本次课时长度（分钟），0 表示沿用订单时长';
COMMENT ON COLUMN lesson_exception.note IS '调整原因备注';
COMMENT ON COLUMN lesson_exception.created_at IS '创建时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN lesson_exception.updated_at IS '更新时间，秒级时间戳（Unix 秒）';

COMMENT ON TABLE lesson IS '历史课次表：已物化的单次课程快照';
COMMENT ON COLUMN lesson.id IS '主键 ID';
COMMENT ON COLUMN lesson.user_id IS '所属用户 ID，关联 user.id';
COMMENT ON COLUMN lesson.order_id IS '所属订单 ID，关联 order.id';
COMMENT ON COLUMN lesson.order_no IS '订单编号快照';
COMMENT ON COLUMN lesson.date IS '上课日期 yyyy-mm-dd';
COMMENT ON COLUMN lesson.weekday IS '星期（1=周一 ... 7=周日）';
COMMENT ON COLUMN lesson.slot_index IS '时段下标（从 0 开始），临时加课为 -1';
COMMENT ON COLUMN lesson.start_time IS '开始时间 HH:MM';
COMMENT ON COLUMN lesson.end_time IS '结束时间 HH:MM';
COMMENT ON COLUMN lesson.duration_minutes IS '本次课时时长（分钟）';
COMMENT ON COLUMN lesson.grade IS '年级快照';
COMMENT ON COLUMN lesson.student_name IS '学生姓名快照';
COMMENT ON COLUMN lesson.subject IS '辅导科目快照';
COMMENT ON COLUMN lesson.address IS '上课地址快照';
COMMENT ON COLUMN lesson.content IS '辅导内容快照';
COMMENT ON COLUMN lesson.remark IS '备注快照';
COMMENT ON COLUMN lesson.remark_flag IS '特殊备注标记（课表红点）快照';
COMMENT ON COLUMN lesson.hourly_rate IS '时薪快照（元/小时）';
COMMENT ON COLUMN lesson.status IS 'regular=常规 canceled=停课 movedOut=已调出 movedIn=已调入 time=仅改时间 extra=临时加课';
COMMENT ON COLUMN lesson.voided IS '是否已删除：true 时课表与报表隐藏';
COMMENT ON COLUMN lesson.income IS '本次课时收入（元）';
COMMENT ON COLUMN lesson.exception_id IS '关联的课程例外 ID，0 表示无';
COMMENT ON COLUMN lesson.origin_date IS 'movedIn 的原定上课日期 yyyy-mm-dd';
COMMENT ON COLUMN lesson.origin_weekday IS 'movedIn 的原定星期（1=周一 ... 7=周日）';
COMMENT ON COLUMN lesson.moved_to_date IS 'movedOut 的调整后日期 yyyy-mm-dd';
COMMENT ON COLUMN lesson.moved_to_start IS 'movedOut 的调整后开始时间 HH:MM';
COMMENT ON COLUMN lesson.adjust_note IS '调整原因备注快照';
COMMENT ON COLUMN lesson.created_at IS '物化写入时间（Unix 秒）';
COMMENT ON COLUMN lesson.updated_at IS '更新时间（Unix 秒）';

COMMENT ON TABLE price_config IS '会员价格配置表：单行配置，固定 id=1';
COMMENT ON COLUMN price_config.id IS '主键 ID，固定为 1';
COMMENT ON COLUMN price_config.pro_monthly_amount IS '专职老师包月价（元）';
COMMENT ON COLUMN price_config.pro_quarterly_amount IS '专职老师包季价（元）';
COMMENT ON COLUMN price_config.pro_yearly_amount IS '专职老师包年价（元）';
COMMENT ON COLUMN price_config.student_discount IS '大学生家教折扣（0-1）';
COMMENT ON COLUMN price_config.renewal_discount IS '平季续费折扣（0-1）';
COMMENT ON COLUMN price_config.updated_at IS '更新时间，秒级时间戳（Unix 秒）';

COMMENT ON TABLE invite_code IS '邀请码表：管理员生成，注册时与手机号绑定校验';
COMMENT ON COLUMN invite_code.id IS '主键 ID';
COMMENT ON COLUMN invite_code.code IS '8 位邀请码，唯一';
COMMENT ON COLUMN invite_code.phone IS '申请人手机号，一个手机号仅可申请一个邀请码';
COMMENT ON COLUMN invite_code.used IS '是否已使用：0=未使用 1=已使用';
COMMENT ON COLUMN invite_code.used_by_id IS '使用该邀请码注册的用户 ID，0 表示未使用';
COMMENT ON COLUMN invite_code.used_at IS '使用时间，秒级时间戳（Unix 秒）；0 表示未使用';
COMMENT ON COLUMN invite_code.created_at IS '生成时间，秒级时间戳（Unix 秒）';

COMMENT ON TABLE message IS '站内信表：广播按接收人各存一条';
COMMENT ON COLUMN message.id IS '主键 ID';
COMMENT ON COLUMN message.user_id IS '接收人 ID，关联 user.id';
COMMENT ON COLUMN message.sender_id IS '发送人 ID，0 表示系统';
COMMENT ON COLUMN message.sender_name IS '发送人名称快照';
COMMENT ON COLUMN message.title IS '标题';
COMMENT ON COLUMN message.content IS '正文';
COMMENT ON COLUMN message.type IS '类型：system=系统通知 promo=优惠 / 活动';
COMMENT ON COLUMN message.read_at IS '已读时间，秒级时间戳（Unix 秒）；0 表示未读';
COMMENT ON COLUMN message.created_at IS '发送时间，秒级时间戳（Unix 秒）';

COMMENT ON TABLE student_application IS '大学生家教身份申请表：每次申请一条';
COMMENT ON COLUMN student_application.id IS '主键 ID';
COMMENT ON COLUMN student_application.user_id IS '申请人 ID，关联 user.id';
COMMENT ON COLUMN student_application.username IS '申请人用户名快照';
COMMENT ON COLUMN student_application.phone IS '申请人手机号快照';
COMMENT ON COLUMN student_application.invite_code IS '注册时使用的邀请码快照';
COMMENT ON COLUMN student_application.student_card_url IS '学生证照片地址';
COMMENT ON COLUMN student_application.id_card_url IS '身份证照片地址';
COMMENT ON COLUMN student_application.status IS '审核状态：pending=待审核 approved=已通过 rejected=已驳回';
COMMENT ON COLUMN student_application.expire_at IS '毕业时间（Unix 秒），0 表示未设置';
COMMENT ON COLUMN student_application.reviewer_id IS '审核人 ID，0 表示未审核';
COMMENT ON COLUMN student_application.reviewer_name IS '审核人名称快照';
COMMENT ON COLUMN student_application.remark IS '审核备注';
COMMENT ON COLUMN student_application.created_at IS '申请时间，秒级时间戳（Unix 秒）';
COMMENT ON COLUMN student_application.reviewed_at IS '审核时间，秒级时间戳（Unix 秒）；0 表示未审核';
