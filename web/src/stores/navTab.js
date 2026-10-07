import { ref, watch } from 'vue'

// 二级导航页签记忆：同一视图离开前的页签在返回时恢复（持久化到 localStorage）
const KEY = 'tutoring_nav_tabs'

function loadAll() {
  try {
    return JSON.parse(localStorage.getItem(KEY) || '{}') || {}
  } catch (e) {
    return {}
  }
}

const all = loadAll()

function persist() {
  try {
    localStorage.setItem(KEY, JSON.stringify(all))
  } catch (e) {
    /* 忽略存储异常 */
  }
}

/**
 * @param {string} scope 视图唯一标识，如 'orders'
 * @param {string} defaultTab 默认页签
 * @param {string} [prefer] 优先级最高的初始页签（如路由 ?tab= 指定）
 */
export function useNavTab(scope, defaultTab, prefer) {
  const saved = typeof all[scope] === 'string' ? all[scope] : ''
  const tab = ref(prefer || saved || defaultTab)
  watch(tab, (v) => {
    all[scope] = v
    persist()
  })
  return tab
}
