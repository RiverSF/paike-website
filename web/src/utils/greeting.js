// 按时段返回问候语，供页头展示（凌晨 / 早上 / 中午 / 下午 / 晚上）
export function timeGreeting(date = new Date()) {
  const h = date.getHours()
  if (h < 5) return '夜深了'
  if (h < 11) return '早上好'
  if (h < 13) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
}
