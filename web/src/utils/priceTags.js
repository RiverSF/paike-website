// 折扣标签统一算法：首页权益卡片与后台「价格设置」预览共用，保证口径一致。
// 各折扣类型返回规范文案（含折数 / 活动名）与配色；调用方决定单标签 / 多标签展示。

// 折数：0.9 -> 9（"9 折"）；0.85 -> 8.5（"8.5 折"）
export function zhe(d) {
  if (!d || d >= 1) return 1
  return Math.round((d || 1) * 100) / 10
}

// 取某折扣卡片在当前价格配置中的快照（与后端 cardSnapshot 对应），用于新旧价对比。
export function cardSnapshot(c, card) {
  const base = { card }
  switch (card) {
    case 'pro':
      return { ...base, proMonthlyAmount: c.proMonthlyAmount, proQuarterlyAmount: c.proQuarterlyAmount, proYearlyAmount: c.proYearlyAmount }
    case 'student':
      return { ...base, studentDiscount: c.studentDiscount }
    case 'renewal':
      return { ...base, renewalDiscount: c.renewalDiscount, renewalEnabled: c.renewalEnabled }
    case 'activity':
      return { ...base, activityName: c.activityName, activityDiscount: c.activityDiscount, activityEnabled: c.activityEnabled }
    default:
      return base
  }
}

// 返回当前生效的各折扣标签（按优先级排序：活动 > 寒暑假 > 平季 > 学生）。
// viewType: 'professional' | 'student'；season: 'offpeak' | 'peak'
// 调用方可取 tags[0] 作为单标签（首页），或遍历展示多标签（后台预览）。
export function priceCardTags(c, { viewType = 'professional', season } = {}) {
  const peak = season === 'peak'
  const tags = []
  if (c.activityEnabled && c.activityDiscount > 0 && c.activityDiscount < 1) {
    tags.push({ key: 'activity', label: `${c.activityName || '限时活动'} ${zhe(c.activityDiscount)} 折`, type: 'danger' })
  }
  if (peak) {
    tags.push({ key: 'peak', label: '寒暑假 · 平季优惠暂停', type: 'warning' })
  } else if (c.renewalEnabled && c.renewalDiscount > 0 && c.renewalDiscount < 1) {
    tags.push({ key: 'renewal', label: `平季 ${zhe(c.renewalDiscount)} 折`, type: 'success' })
  }
  if (viewType === 'student') {
    tags.push({ key: 'student', label: `学生 ${zhe(c.studentDiscount)} 折`, type: 'primary' })
  }
  return tags
}

// 取首页单标签（最高优先级折扣）。无生效折扣时返回 null，调用方按 null 隐藏标签即可。
export function topDiscountTag(c, { viewType = 'professional', season } = {}) {
  const tags = priceCardTags(c, { viewType, season })
  if (!tags.length) return null
  return { key: tags[0].key, text: tags[0].label, type: tags[0].type }
}
