<template>
  <header class="app-header">
    <div class="header-inner">
      <div class="brand" @click="go('/')">
        <img class="brand-logo" :src="currentLogo.favicon" :alt="brand.brandName" />
        <div class="brand-text">
          <strong>{{ brand.brandName }}</strong>
        </div>
      </div>

      <nav class="nav">
        <button
          v-for="item in menus"
          :key="item.path"
          class="nav-item"
          :class="{ active: current === item.path, locked: item.memberOnly && !canUseMember }"
          @click="onMenu(item)"
        >
          {{ item.title }}
          <el-icon v-if="item.memberOnly && !canUseMember" class="lock"><Lock /></el-icon>
        </button>
      </nav>

      <div class="actions">
        <template v-if="userStore.isLogin">
          <!-- 登录问候：右上角按时段展示，移动端隐藏 -->
          <span class="greet">{{ greetText }}</span>
          <!-- 会员标签：仅普通用户展示（站长 / 管理员为永久权益，身份已在名称下方标注） -->
          <el-tag
            v-if="!userStore.isStaff"
            class="member-tag"
            :type="userStore.isMember ? 'success' : 'danger'"
            effect="light"
            round
            @click="emit('profile', 'member')"
          >
            {{ memberTagText }}
          </el-tag>

          <el-badge :value="userStore.unreadMessages" :hidden="!userStore.unreadMessages" :max="99">
            <el-button circle text @click="messageVisible = true">
              <el-icon :size="18"><Bell /></el-icon>
            </el-button>
          </el-badge>

          <el-dropdown trigger="click" @command="onCommand">
            <span class="avatar-wrap">
              <el-avatar :size="34" :src="avatarUrl" />
              <span class="user-meta">
                <span class="username">{{ userStore.profile.username }}</span>
                <!-- 身份标签：站长 / 管理员显示后台角色；普通用户按身份类型（老师 / 学员 / 机构）渲染 -->
                <span
                  class="user-role"
                  :class="userStore.isStaff ? (userStore.profile.role === 'owner' ? 'is-owner' : 'is-admin') : 'is-user'"
                >
                  {{ userStore.isStaff ? userStore.profile.roleName : identityLabel(userStore.profile) }}
                </span>
              </span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>个人信息
                </el-dropdown-item>
                <el-dropdown-item command="member">
                  <el-icon><Medal /></el-icon>会员中心
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>

        <template v-else>
          <el-button round @click="emit('login')">登录</el-button>
          <!-- 免注册试用：直接进入排课，无需注册 -->
          <el-button type="primary" round @click="go('/orders')">免费试用</el-button>
        </template>
      </div>
    </div>
    <MessagePanel v-model="messageVisible" />
  </header>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowDown, Bell, Lock, Medal, SwitchButton, User } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import MessagePanel from '@/components/MessagePanel.vue'
import { brand, logos, activeLogo } from '@/config/brand'
import { timeGreeting } from '@/utils/greeting'
import { identityLabel } from '@/utils/identity'

const emit = defineEmits(['login', 'register', 'profile'])
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const messageVisible = ref(false)

const currentLogo = computed(() => logos[activeLogo])

const menus = computed(() => {
  // 顺序即主流程：首页 → 录入课程（课程安排）→ 排课（课表）→ 管理（管理员可见）→ 帮助（使用指南置后）
  const list = [
    { title: '首页', path: '/' },
    { title: '课程安排', path: '/orders', memberOnly: true },
    { title: '课表', path: '/schedule', memberOnly: true }
  ]
  if (userStore.isStaff) {
    list.push({ title: '管理中心', path: '/admin/members' })
  }
  list.push({ title: '使用指南', path: '/guide' })
  return list
})

const current = computed(() => route.path)
// 课程安排 / 课表：登录会员可用；未登录用户进入免注册试用模式；过期用户仍可只读查看
const canUseMember = computed(() => (userStore.isLogin && userStore.isMember) || !userStore.isLogin)
const avatarUrl = computed(() => userStore.profile?.avatar || '/avatar-default.png')

// 右上角登录问候：只显示时段问候语，不带用户名；每分钟校正一次
const now = ref(new Date())
let greetTimer = null
const greetText = computed(() => timeGreeting(now.value))

// 会员标签：付费 / 试用显示「付费会员 / 免费试用」并带剩余天数，与会员类型口径一致
const memberTagText = computed(() => {
  const p = userStore.profile
  if (!p) return ''
  if (p.memberType === 'permanent') return '永久会员'
  if (!p.memberActive) return '已过期'
  const label = p.memberType === 'trial' ? '免费试用' : '付费会员'
  return `${label} · 剩 ${p.memberLeftDays} 天`
})

function go(path) {
  router.push(path)
}

function onMenu(item) {
  if (item.memberOnly && !canUseMember.value) {
    // 会员已过期：课程与课表仍可查看（只读），新增 / 调课等写操作在续费后恢复
    ElMessage.warning('会员已过期：课程与课表仍可查看，续费后可继续新增与调课')
  }
  go(item.path)
}

function onCommand(cmd) {
  if (cmd === 'logout') {
    userStore.logout()
    userStore.unreadMessages = 0
    ElMessage.success('已退出登录')
    if (route.path !== '/') router.push('/')
    return
  }
  // 个人信息 → 注册信息与身份信息；会员中心 → 会员信息与支付记录
  emit('profile', cmd === 'member' ? 'member' : 'profile')
}

// 未读站内信角标：定时 + 回到页面时刷新，保证「上课提醒」等系统通知无需手动刷新即可看到
let unreadTimer = null
function refreshUnread() {
  if (userStore.isLogin) userStore.loadUnreadMessages()
}
onMounted(() => {
  refreshUnread()
  window.addEventListener('focus', refreshUnread)
  unreadTimer = window.setInterval(refreshUnread, 120000)
  // 问候语跨时段自动切换
  greetTimer = window.setInterval(() => {
    now.value = new Date()
  }, 60000)
})
onBeforeUnmount(() => {
  window.removeEventListener('focus', refreshUnread)
  if (unreadTimer) window.clearInterval(unreadTimer)
  if (greetTimer) window.clearInterval(greetTimer)
})
</script>

<style scoped>
.app-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  height: 64px;
  background: #fff;
  border-bottom: 1px solid var(--brand-line);
}

.header-inner {
  max-width: 1240px;
  height: 100%;
  margin: 0 auto;
  padding: 0 16px;
  display: flex;
  align-items: center;
  gap: 24px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.brand-logo {
  height: 36px;
  width: 36px;
  object-fit: contain;
  border-radius: 8px;
}

.brand-text {
  display: flex;
  align-items: center;
  line-height: 1.25;
}

.brand-text strong {
  font-size: 16px;
}

.nav {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
}

.nav-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--brand-sub);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.nav-item:hover {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.nav-item.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 600;
}

.nav-item.locked {
  color: var(--brand-muted);
}

.lock {
  font-size: 12px;
}

.actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 12px;
}

.member-tag {
  cursor: pointer;
}

.avatar-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  outline: none;
}

/* 用户名 + 身份标签：上下两行 */
.user-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 1px;
  line-height: 1.1;
  min-width: 0;
}

.username {
  font-size: 13px;
  color: var(--brand-ink);
  max-width: 90px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* 身份标签（站长 / 管理员）：名称下方的小胶囊 */
.user-role {
  padding: 0 6px;
  border-radius: 999px;
  font-size: 10px;
  line-height: 15px;
  white-space: nowrap;
}

.user-role.is-owner {
  background: #fff3e6;
  color: #e08214;
}

.user-role.is-admin {
  background: #f6f0ff;
  color: #7a5bd6;
}

.user-role.is-user {
  background: #eef3ff;
  color: #2f6fed;
}

/* 右上角登录问候：弱化显示，不与会员标签抢注意力 */
.greet {
  font-size: 13px;
  color: var(--brand-sub);
  max-width: 160px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

@media (max-width: 768px) {
  .header-inner {
    gap: 8px;
    padding: 0 12px;
  }

  /* 窄屏隐藏仅起装饰/次要作用的文字，避免挤压导航与操作区 */
  .brand-text,
  .greet,
  .user-meta {
    display: none;
  }

  /* 导航项较多时允许横向滚动，避免一行被挤爆溢出页面 */
  .nav {
    flex: 1 1 auto;
    min-width: 0;
    gap: 0;
    overflow-x: auto;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }

  .nav::-webkit-scrollbar {
    display: none;
  }

  .nav-item {
    flex: none;
    padding: 0 6px;
    font-size: 13px;
  }

  .actions {
    gap: 8px;
    flex: none;
  }

  /* 操作区按钮在极窄屏下收敛文字，优先保证可点 */
  .actions .el-button {
    padding: 8px 12px;
  }
}
</style>
