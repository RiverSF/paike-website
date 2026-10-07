<template>
  <el-dialog v-model="visible" width="420px" :close-on-click-modal="false" align-center>
    <template #header>
      <div class="dialog-title">{{ active === 'login' ? '登录' : '注册' }}</div>
    </template>

    <el-tabs v-model="active" class="auth-tabs">
      <el-tab-pane label="登录" name="login" />
      <el-tab-pane label="注册" name="register" />
    </el-tabs>

    <el-form v-if="active === 'login'" :model="loginForm" label-position="top" @submit.prevent>
      <el-form-item label="手机号 / 邮箱">
        <el-input
          v-model="loginForm.username"
          placeholder="请输入手机号或邮箱"
          autocomplete="username"
        />
      </el-form-item>
      <el-form-item label="密码">
        <el-input
          v-model="loginForm.password"
          type="password"
          show-password
          placeholder="请输入密码"
          autocomplete="current-password"
          @keyup.enter="submit"
        />
      </el-form-item>
    </el-form>

    <el-form v-else :model="regForm" label-position="top">
      <el-form-item label="用户名" required>
        <el-input v-model="regForm.username" placeholder="2-32 位，不可包含空格" />
      </el-form-item>
      <el-form-item label="密码" required>
        <el-input v-model="regForm.password" type="password" show-password placeholder="至少 6 位" />
      </el-form-item>
      <el-form-item label="手机号" required>
        <el-input
          v-model="regForm.phone"
          maxlength="11"
          placeholder="请输入 11 位手机号（必填）"
          @keyup.enter="submit"
        />
      </el-form-item>
      <el-form-item label="邀请码" required>
        <el-input v-model="regForm.inviteCode" maxlength="16" placeholder="请填写邀请码（必填）" />
      </el-form-item>
      <div class="tip invite-tip">还没有邀请码？请扫描首页下方<b>添加微信好友</b>二维码进行申请，一个微信账号仅可申请一个邀请码。</div>
      <el-form-item label="邮箱（选填）">
        <el-input v-model="regForm.email" placeholder="便于找回账号，可稍后补充" />
      </el-form-item>
      <el-form-item label="你是谁">
        <el-radio-group v-model="regForm.userRole">
          <el-radio value="teacher">老师</el-radio>
          <el-radio value="parent">学员</el-radio>
          <el-radio value="org">机构</el-radio>
        </el-radio-group>
        <div class="tip role-tip">{{ roleTip }}</div>
      </el-form-item>
      <div v-if="props.tip && active === 'register'" class="tip guide-tip">{{ props.tip }}</div>
      <div class="tip">注册即赠送 <b>30 天</b>免费会员，可直接使用课程与课表功能。</div>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="submit">
        {{ active === 'login' ? '登录' : '注册并登录' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { sanitizeText, validateText } from '@/utils/text'

const props = defineProps({
  modelValue: Boolean,
  // 打开弹窗时默认展示的表单：login=登录、register=注册
  mode: { type: String, default: 'login' },
  // 注册引导说明（如试用数据未保存提示）
  tip: { type: String, default: '' }
})
const emit = defineEmits(['update:modelValue'])
const userStore = useUserStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const active = ref('login')

// 每次打开弹窗时，根据 mode 切到正确的表单（登录 / 注册）
watch(visible, (open) => {
  if (open && (props.mode === 'login' || props.mode === 'register')) {
    active.value = props.mode
  }
})
const loading = ref(false)
const loginForm = reactive({ username: '', password: '' })
const regForm = reactive({ username: '', password: '', phone: '', email: '', inviteCode: '', userRole: 'teacher' })

// 使用身份说明：只影响课程表单与文案，不参与计费（统一按专职标准价计费）
const roleTip = computed(
  () =>
    ({
      teacher: '接单上课、管理学员课程。',
      parent: '安排孩子或自己的课程，一周不冲突。',
      org: '多学员、多科目统一排课，一个账号管全部课表。'
    })[regForm.userRole] || ''
)

async function submit() {
  if (active.value === 'login') {
    if (!loginForm.username || !loginForm.password) {
      ElMessage.warning('请输入手机号/邮箱和密码')
      return
    }
    loading.value = true
    try {
      await userStore.login({ username: loginForm.username, password: loginForm.password })
      ElMessage.success('登录成功')
      visible.value = false
    } finally {
      loading.value = false
    }
    return
  }

  if (!regForm.username.trim()) {
    ElMessage.warning('请输入用户名')
    return
  }
  if (regForm.username.trim().length < 2 || regForm.username.trim().length > 32) {
    ElMessage.warning('用户名需为 2-32 个字符，可使用中文、字母、数字，不可含空格')
    return
  }
  if (!regForm.password) {
    ElMessage.warning('请输入密码')
    return
  }
  if (regForm.password.length < 6) {
    ElMessage.warning('密码至少 6 位')
    return
  }
  const nameCheck = validateText(regForm.username, '用户名', { max: 32 })
  if (nameCheck) {
    ElMessage.warning(nameCheck)
    return
  }
  const phone = regForm.phone.trim()
  if (!/^1[3-9]\d{9}$/.test(phone)) {
    ElMessage.warning('请填写正确的 11 位手机号')
    return
  }
  const inviteCode = regForm.inviteCode.trim()
  if (!inviteCode) {
    ElMessage.warning('请填写邀请码')
    return
  }
  loading.value = true
  try {
    await userStore.register({
      username: sanitizeText(regForm.username),
      password: regForm.password,
      phone,
      email: regForm.email,
      inviteCode,
      userRole: regForm.userRole
    })
    ElMessage.success('注册成功，已赠送 30 天会员')
    visible.value = false
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.dialog-title {
  font-size: 17px;
  font-weight: 600;
}

.auth-tabs {
  margin-top: -12px;
}

.tip {
  font-size: 12px;
  color: var(--brand-muted);
}

.tip b {
  color: var(--el-color-primary);
}

.invite-tip {
  margin-bottom: 14px;
}

.role-tip {
  margin-top: 6px;
  line-height: 1.5;
}

/* 试用数据未保存等注册引导提示 */
.guide-tip {
  margin-bottom: 14px;
  padding: 8px 10px;
  border-radius: 6px;
  background: #fff9e6;
  color: #8a6d3b;
  border-left: 3px solid var(--brand-warning);
}
</style>
