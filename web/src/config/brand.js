// 品牌统一出口：改名 / 换 Logo / 调整备案信息只改此处。
// 文档见 docs/站点命名建议.md（第六节·路线 B）。
export const brand = {
  brandName: '课满满',
  // 网站定位描述，与 brandName 拼成备案网站名；改 brandName 后全站自动跟随
  siteDesc: '课表订单管理工具',
  // 营业执照全称，注册时把（所在城市）替换为实际注册地
  companyName: '课满满（所在城市）信息科技有限公司',
  // 备案网站名称（企业主体，与字号强关联、中性描述，规避“家教/培训/招生/教育”等敏感词）
  siteTitle: '课满满 · 课表订单管理工具',
  metaDesc: '课表与订单管理工具，支持订单录入、周课表自动生成。',
  // 页头副标题：与导航（课程安排 / 课表）和首页主标题避免重复，改用价值主张
  tagline: '自动排课，省时又省心'
}

// 多套可切换 Logo 方案（icon + 中文 wordmark），按 key 切换
export const logos = {
  calendar: {
    key: 'calendar',
    name: '日历满格',
    desc: '填满格子的日历，象征排课排得满满、课时收得满满',
    color: '#2D6BFF',
    src: '/logo/kemanman-calendar.svg',
    favicon: '/logo/favicon-calendar.svg'
  },
  bell: {
    key: 'bell',
    name: '叮课铃铛',
    desc: '铃铛造型，呼应上课提醒 / 调课通知，拟声好记',
    color: '#FF7A1A',
    src: '/logo/kemanman-bell.svg',
    favicon: '/logo/favicon-bell.svg'
  },
  cards: {
    key: 'cards',
    name: '课表卡片',
    desc: '层叠的课表卡片，体现多订单 / 多课表管理',
    color: '#16B364',
    src: '/logo/kemanman-cards.svg',
    favicon: '/logo/favicon-cards.svg'
  },
  // 以下三套复用「课伴」图标概念（圆角徽章 / 对话气泡 / 双圆重叠），字标改为课满满
  'keban-a': {
    key: 'keban-a',
    name: '徽章陪伴点',
    desc: '蓝底课字徽章 + 橙色陪伴点，温情且好认',
    color: '#2D6BFF',
    src: '/logo/kemanman-keban-a.svg',
    favicon: '/logo/favicon-keban-a.svg'
  },
  'keban-b': {
    key: 'keban-b',
    name: '对话气泡',
    desc: '青绿对话气泡承载课字，呼应陪伴 / 交流',
    color: '#16A589',
    src: '/logo/kemanman-keban-b.svg',
    favicon: '/logo/favicon-keban-b.svg'
  },
  'keban-c': {
    key: 'keban-c',
    name: '双圆重叠',
    desc: '两枚交叠圆，伴=两人同行，辨识度高',
    color: '#6C4DF2',
    src: '/logo/kemanman-keban-c.svg',
    favicon: '/logo/favicon-keban-c.svg'
  }
}

// 当前启用方案：'calendar' | 'bell' | 'cards' | 'keban-a' | 'keban-b' | 'keban-c'
export const activeLogo = 'keban-a'
