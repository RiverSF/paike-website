import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { brand, logos, activeLogo } from '@/config/brand'

// 一级菜单路径白名单：用于「上次所在菜单」记忆（刷新 / 重新打开站点时恢复原菜单页面）
const MENU_PATHS = ['/', '/orders', '/schedule', '/admin/members', '/guide']
const LAST_MENU_KEY = 'tutoring_last_menu'

function lastMenu() {
  try {
    const v = localStorage.getItem(LAST_MENU_KEY) || ''
    return MENU_PATHS.includes(v) ? v : ''
  } catch (e) {
    return ''
  }
}

function rememberMenu(path) {
  try {
    localStorage.setItem(LAST_MENU_KEY, path)
  } catch (e) {
    /* 忽略存储异常 */
  }
}

const routes = [
  { path: '/', name: 'home', component: () => import('@/views/HomeView.vue'), meta: { title: '首页' } },
  { path: '/orders', name: 'orders', component: () => import('@/views/OrdersView.vue'), meta: { title: '课程安排' } },
  { path: '/schedule', name: 'schedule', component: () => import('@/views/ScheduleView.vue'), meta: { title: '课表' } },
  { path: '/guide', name: 'guide', component: () => import('@/views/GuideView.vue'), meta: { title: '使用指南' } },
  {
    path: '/admin/members',
    name: 'admin-members',
    component: () => import('@/views/AdminMembersView.vue'),
    meta: { title: '管理中心', requiresStaff: true }
  },
  // 用户反馈已合并进「会员管理」的二级导航，旧入口重定向到对应页签
  { path: '/admin/feedbacks', redirect: '/admin/members?tab=feedback' },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 })
})

// 首屏恢复标记：只在第一次导航时判断一次，避免反复重定向
let menuRestored = false

// 管理端页面仅站点拥有者 / 管理员可访问
router.beforeEach(async (to) => {
  const userStore = useUserStore()
  // 刷新 / 直接打开页面时 profile 尚未加载完成，先补齐再判定身份，
  // 否则「管理中心」等受保护页面会被误判为无权限而跳回首页
  if (userStore.token && !userStore.profile) {
    await userStore.fetchProfile()
  }

  if (!menuRestored) {
    menuRestored = true
    const hash = window.location.hash
    // 地址栏未指定菜单（直接打开站点 / 刷新后停在 #/）时，恢复上次所在菜单
    if (to.path === '/' && (!hash || hash === '#' || hash === '#/')) {
      const saved = lastMenu()
      if (saved && saved !== '/') return saved
    }
  }

  if (to.meta.requiresStaff && !userStore.isStaff) return '/'
  return true
})

router.afterEach((to) => {
  document.title = `${to.meta.title || '首页'} · ${brand.brandName}`
  // 同步 favicon 为当前启用的 Logo 方案
  const link = document.querySelector("link[rel~='icon']")
  if (link) link.href = logos[activeLogo].favicon
  // 记录当前所在菜单，供刷新 / 重新打开站点时恢复
  if (MENU_PATHS.includes(to.path)) rememberMenu(to.path)
})

export default router
