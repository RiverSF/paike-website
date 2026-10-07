-- 009: 调课记录完整路径
-- lesson_exception 新增 history 列：JSON 数组，记录一节课的完整调课路径（每次调整一步），
-- 生效字段（type/new_date/new_start/new_end/duration_minutes/note）始终等于路径最后一步。
-- 撤销仅回退最后一步；路径只剩一步时撤销即删除例外、恢复原课次。
ALTER TABLE lesson_exception ADD COLUMN IF NOT EXISTS history TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN lesson_exception.history IS '完整调课路径（JSON 数组），每步含 from/to 日期时间、时长、类型、备注与操作时间';

-- 课表「仅保留调整后的课时」：原课次行在调整后软删除（voided），
-- 撤销回原位时恢复；此处无需结构变更，仅补充注释说明。
COMMENT ON COLUMN "lesson".voided IS '是否已隐藏：true=已删除(老师未上课)或已被调整替代的原课次，课表与报表不展示，撤销调整后恢复';
