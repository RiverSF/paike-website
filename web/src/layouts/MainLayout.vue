<template>
  <div class="layout">
    <AppHeader @login="openAuth('login')" @register="openAuth('register')" @profile="openProfile" />

    <main class="layout-main">
      <slot />
    </main>

    <!-- 底部信息区（站点 / 合规 / 推广位）按 docs/底部信息区规划.md 重建，
         备案号或推广方案明确后再恢复，当前不展示任何内容 -->

    <AuthDialog v-model="authVisible" :mode="authMode" :tip="authTip" />
    <ProfilePanel v-model="profileVisible" :mode="profileMode" />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AppHeader from '@/components/AppHeader.vue'
import AuthDialog from '@/components/AuthDialog.vue'
import ProfilePanel from '@/components/ProfilePanel.vue'
import { useUserStore } from '@/stores/user'
import { useTrialStore } from '@/stores/trial'

const authVisible = ref(false)
const authMode = ref('login')
// 注册引导说明（如「试用数据尚未保存」提示），随 open-auth 事件下发
const authTip = ref('')
const profileVisible = ref(false)
// profile=个人信息（注册信息）；member=会员中心（会员信息 + 支付记录）
const profileMode = ref('profile')

const userStore = useUserStore()
const trialStore = useTrialStore()

// 页面内（订单/课表）通过自定义事件唤起登录或会员面板
function openAuth(mode = 'login', tip = '') {
  authMode.value = mode === 'register' ? 'register' : 'login'
  authTip.value = tip || ''
  authVisible.value = true
}
function openProfile(mode = 'profile') {
  profileMode.value = mode === 'member' ? 'member' : 'profile'
  profileVisible.value = true
}
// 会员过期等场景统一唤起「会员中心」
function openMember() {
  openProfile('member')
}
function onOpenAuth(e) {
  openAuth(e?.detail?.mode || 'login', e?.detail?.tip || '')
}

// ===== 免注册试用守护 =====
const VISITED_KEY = 'tutoring_trial_visited'

// 未注册且已有排课数据时：关闭 / 刷新页面前给出保存提示
function beforeUnloadGuard(e) {
  if (trialStore.isGuest && trialStore.hasData) {
    e.preventDefault()
    e.returnValue = '您的排课数据尚未保存，注册即可永久保存并随时查看。'
    return e.returnValue
  }
}

// 登录成功后：把本地试用数据导入账号，永久保存
let importing = false
watch(
  () => userStore.isLogin,
  async (login) => {
    if (!login || importing || !trialStore.hasData) return
    importing = true
    try {
      await ElMessageBox.confirm(
        `检测到本机有 ${trialStore.orderCount} 门试用课程尚未保存到账号，是否立即导入？导入后可永久保存并随时查看。`,
        '导入试用数据',
        { confirmButtonText: '立即导入', cancelButtonText: '暂不导入', type: 'warning' }
      )
      const ok = await trialStore.importToServer()
      ElMessage.success(ok > 0 ? `已导入 ${ok} 门课程到您的账号` : '本地数据已保存过或导入失败，可手动重新录入')
    } catch {
      // 用户选择暂不导入：保留本地数据，之后可从订单页再次触发
    } finally {
      importing = false
    }
  }
)

onMounted(() => {
  window.addEventListener('open-auth', onOpenAuth)
  window.addEventListener('open-profile', openMember)
  window.addEventListener('beforeunload', beforeUnloadGuard)

  // 免注册试用生命周期：
  // 1) 首次到访 → 记录标记；
  // 2) 同一设备第二次打开且上次有未保存的课表 → 直接弹出注册引导
  if (trialStore.isGuest) {
    trialStore.load()
    if (!localStorage.getItem(VISITED_KEY)) {
      localStorage.setItem(VISITED_KEY, String(Date.now()))
    } else if (trialStore.hasData) {
      window.setTimeout(() => {
        openAuth('register', '您的排课数据尚未保存，注册即可永久保存并随时查看。')
      }, 600)
    }
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('open-auth', onOpenAuth)
  window.removeEventListener('open-profile', openMember)
  window.removeEventListener('beforeunload', beforeUnloadGuard)
})
</script>

<style scoped>
.layout {
  min-height: 100%;
  display: flex;
  flex-direction: column;
}

.layout-main {
  flex: 1;
  padding-top: 64px;
}
</style>
