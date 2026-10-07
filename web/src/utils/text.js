// 输入清洗与文本合法性校验：
// 1) trim 清洗；2) 常见违禁词过滤（赌博 / 色情 / 毒品 / 诈骗黑产 / 暴力违禁品 / 辱骂）。
// 词库可持续补充；匹配不区分大小写。密码、手机号、邮箱、日期等结构化字段不做违禁词校验。

const BANNED_WORDS = [
  // 赌博博彩
  '赌博', '博彩', '赌场', '网赌', '六合彩', '时时彩', '百家乐', '幸运飞艇', '德州扑克', '押注', '返水', '上分', '下分',
  // 色情低俗
  '援交', '卖淫', '嫖娼', '约炮', '一夜情', '裸聊', '色情', '包夜', '嫖',
  // 毒品
  '冰毒', '摇头丸', '大麻', 'k粉', '吸毒', '贩毒', '麻古',
  // 诈骗与黑产
  '刷单', '诈骗', '传销', '洗钱', '跑分', '代开发票', '办证', '代考', '替考', '考试答案', '作弊器', '外挂', '木马', '钓鱼网站', '网贷', '套现',
  // 暴力与违禁品
  '枪支', '弹药', '管制刀具', '炸药', '雷管', '老虎机',
  // 辱骂
  '傻逼', '草泥马', '操你', '去死吧'
]

const bannedPattern = new RegExp(BANNED_WORDS.map(escapeReg).join('|'), 'i')

function escapeReg(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** 去除首尾与多余空白 */
export function sanitizeText(v) {
  return typeof v === 'string' ? v.replace(/\s+/g, ' ').trim() : v
}

/** 返回命中的违禁词，未命中返回空串 */
export function findBannedWord(v) {
  const s = String(v ?? '')
  const m = s.match(bannedPattern)
  return m ? m[0] : ''
}

/**
 * 通用文本校验：非必填字段传空会跳过。
 * 返回错误消息；通过返回 ''。
 */
export function validateText(value, label = '内容', { max = 200, required = false } = {}) {
  const v = sanitizeText(value)
  if (!v) return required ? `请填写${label}` : ''
  if (v.length > max) return `${label}不能超过 ${max} 个字符`
  const w = findBannedWord(v)
  if (w) return `${label}包含违规内容「${w}」，请修改后重试`
  return ''
}

/** 批量校验：values 为 { 字段值: 标签 } 形式，返回第一个错误消息，全部通过返回 '' */
export function validateTexts(values) {
  for (const [value, label] of Object.entries(values)) {
    const msg = validateText(value, label)
    if (msg) return msg
  }
  return ''
}

/** 手机号（中国大陆 11 位）格式校验 */
export function isValidPhone(v) {
  return /^1[3-9]\d{9}$/.test(String(v || '').trim())
}
