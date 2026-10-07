<template>
  <el-drawer v-model="visible" :title="drawerTitle" size="420px">
    <template v-if="profile">
      <section class="profile-head">
        <div class="avatar-box" @click="pickAvatar">
          <el-avatar :size="72" :src="avatarUrl" />
          <div class="avatar-mask">更换</div>
        </div>
        <div class="head-info">
          <div class="name">{{ profile.username }}</div>
          <div class="head-tags">
            <el-tag v-if="profile.role && profile.role !== 'user'" :type="roleTagType" size="small" round effect="dark">
              {{ profile.roleName }}
            </el-tag>
            <el-tag :type="profile.memberActive ? 'success' : 'danger'" size="small" round effect="light">
              {{ profile.memberTypeName }}
            </el-tag>
          </div>
          <!-- 注册天数 / 会员剩余等提示仅普通注册会员展示，站长与管理员已有身份标签 -->
          <div v-if="!isStaffUser" class="muted small">
            已注册 {{ profile.registeredDays }} 天
            <template v-if="isMemberMode">· 会员剩余 {{ memberLeftText }}</template>
          </div>
        </div>
      </section>

      <!-- 个人信息：注册信息 + 身份信息 -->
      <section v-if="isProfileMode" class="card-box">
        <div class="box-title">注册信息</div>
        <el-descriptions :column="1" border size="small" label-width="80">
          <el-descriptions-item label="用户名">
            <template v-if="editing">
              <el-input v-model="form.username" size="small" maxlength="32" placeholder="2-32 个字符，不含空格" />
            </template>
            <template v-else>{{ profile.username }}</template>
          </el-descriptions-item>
          <el-descriptions-item label="手机号">
            <template v-if="editing">
              <el-input
                v-model="form.phone"
                size="small"
                maxlength="11"
                placeholder="请输入 11 位手机号"
                :disabled="phoneLocked"
              />
              <div v-if="phoneLocked" class="muted small">
                本月已于 {{ phoneChangedText }} 修改过，手机号每月仅可修改一次（下月可再次修改）
              </div>
            </template>
            <template v-else>
              {{ profile.phone || '未填写' }}
              <span v-if="phoneLocked" class="muted small">（本月已修改过，下月可再次修改）</span>
            </template>
          </el-descriptions-item>
          <el-descriptions-item label="邮箱">
            <template v-if="editing">
              <el-input v-model="form.email" size="small" placeholder="请输入邮箱" />
            </template>
            <template v-else>{{ profile.email || '未填写' }}</template>
          </el-descriptions-item>
          <el-descriptions-item label="登录密码">
            <div class="pwd-row">
              <span class="pwd-mask">********</span>
              <el-button link type="primary" size="small" @click="openPwd">修改密码</el-button>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <div class="box-actions">
          <template v-if="editing">
            <el-button size="small" type="primary" :loading="saving" @click="save">保存</el-button>
            <el-button size="small" @click="editing = false">取消</el-button>
          </template>
          <el-button v-else size="small" @click="startEdit">编辑资料</el-button>
        </div>
      </section>

      <!-- 会员中心：会员信息 + 支付记录 -->
      <section v-if="isMemberMode" class="card-box">
        <div class="box-title">会员信息</div>
        <el-descriptions :column="1" border size="small" label-width="80">
          <el-descriptions-item label="会员类型">{{ memberTypeText }}</el-descriptions-item>
          <el-descriptions-item label="计费方式">{{ billingText }}</el-descriptions-item>
          <el-descriptions-item label="有效时段">{{ memberPeriodText }}</el-descriptions-item>
          <el-descriptions-item label="剩余天数">
            <span :class="memberLeftOk ? 'ok' : 'expired'">{{ memberLeftText }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="累计充值">
            <span class="money-cell">¥{{ money(totalPaid) }}</span>
            <span v-if="isStaffUser" class="muted small staff-note">（{{ roleText }}无需付费）</span>
          </el-descriptions-item>
        </el-descriptions>

        <div v-if="memberExpiringSoon" class="identity-warn">
          会员将于 {{ fmtDate(profile.memberExpire) }} 到期（剩余 {{ memberLeftText }}）。到期后已录入的课程与课表仍可查看，但新增课程、调课与费用统计将受限，请尽快续费。
        </div>

        <div class="renew">
          <div class="renew-head">
            <div class="renew-title">开通 / 续费</div>
            <el-tag v-if="renewOffTag" size="small" effect="plain" :type="renewOffTag.type">{{ renewOffTag.text }}</el-tag>
          </div>
          <template v-if="isStaffUser">
            <div class="permanent-tip">{{ staffRenewTip }}</div>
          </template>
          <template v-else-if="isPermanent">
            <div class="permanent-tip">您已是永久会员，无需续费。</div>
          </template>
          <template v-else>
            <div class="plan-list">
              <div v-for="p in renewPlans" :key="p.key" class="plan-row">
                <span class="plan-name">{{ p.name }}</span>
                <span class="plan-price">¥{{ p.price }}</span>
                <span class="plan-unit">/ {{ p.unit }}</span>
                <span v-if="p.origin" class="plan-origin">¥{{ p.origin }}</span>
                <el-button class="plan-btn" size="small" type="primary" plain @click="goHomePay(p.key)">
                  {{ renewBtnText }}
                </el-button>
              </div>
            </div>
            <div class="renew-tip">扫码支付后联系管理员人工开通，开通记录可在「支付记录」查看。</div>
          </template>
        </div>
      </section>

      <section v-if="isProfileMode" class="card-box">
        <div class="box-title">身份信息</div>
        <el-descriptions :column="1" border size="small" label-width="80">
          <el-descriptions-item label="身份类型">{{ identityText }}</el-descriptions-item>
          <el-descriptions-item label="账号状态">
            <el-tag :type="profile.status === 'frozen' ? 'danger' : 'success'" size="small" effect="light">
              {{ profile.statusName || '正常' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="权限说明">
            <span class="muted">{{ permissionText }}</span>
          </el-descriptions-item>
        </el-descriptions>
      </section>

      <section v-if="isMemberMode" class="card-box">
        <div class="box-title">支付记录</div>
        <div v-if="!payments.length" class="muted small empty">暂无支付记录</div>
        <div v-for="item in payments" :key="item.id" class="rc-item">
          <div class="rc-top">
            <span class="rc-amount">¥{{ money(item.amount) }}</span>
            <el-tag size="small" effect="light" round>{{ item.periodName || item.period }}</el-tag>
            <span class="muted rc-time">{{ fmtTs(item.createdAt) }}</span>
          </div>
          <div class="muted small">
            到期时间：{{ fmtTs(item.beforeExpire) }} → <b>{{ fmtTs(item.afterExpire) }}</b>
            <span v-if="item.source === 'admin' && item.operatorName">（由 {{ item.operatorName }} 代开）</span>
          </div>
        </div>
      </section>
    </template>



    <input ref="fileInput" type="file" accept="image/*" hidden @change="onFileChange" />

    <el-dialog v-model="pwdVisible" title="修改密码" width="420px" append-to-body @keyup.enter="submitPwd">
      <el-form label-width="96px" @submit.prevent>
        <el-form-item label="原密码">
          <el-input
            v-model="pwdForm.oldPassword"
            type="password"
            show-password
            autocomplete="current-password"
            placeholder="请输入当前登录密码"
          />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input
            v-model="pwdForm.newPassword"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="6-32 位新密码"
          />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input
            v-model="pwdForm.confirm"
            type="password"
            show-password
            autocomplete="new-password"
            placeholder="请再次输入新密码"
          />
        </el-form-item>
      </el-form>
      <div class="muted small">密码长度 6-32 位，修改成功后下次登录请使用新密码。</div>
      <template #footer>
        <el-button @click="pwdVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdSaving" @click="submitPwd">确认修改</el-button>
      </template>
    </el-dialog>
  </el-drawer>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { userApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { usePriceStore } from '@/stores/price'
import { sanitizeText } from '@/utils/text'
import { userRoleLabel } from '@/utils/identity'
import { topDiscountTag } from '@/utils/priceTags'

// 后端返回秒级时间戳
const fmtTs = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD HH:mm') : '-')
const fmtDate = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD') : '-')

// mode：profile=个人信息（注册信息 + 身份信息）；member=会员中心（会员信息 + 支付记录）
const props = defineProps({
  modelValue: Boolean,
  mode: { type: String, default: 'profile' }
})
const emit = defineEmits(['update:modelValue'])
const userStore = useUserStore()
const priceStore = usePriceStore()
const router = useRouter()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const isProfileMode = computed(() => props.mode !== 'member')
const isMemberMode = computed(() => props.mode === 'member')
const drawerTitle = computed(() => (isMemberMode.value ? '会员中心' : '个人信息'))

const profile = computed(() => userStore.profile)
const avatarUrl = computed(() => profile.value?.avatar || '/avatar-default.png')
// ---- 角色：站长 / 管理员 / 普通用户 ----
const isOwner = computed(() => profile.value?.role === 'owner')
const isAdmin = computed(() => profile.value?.role === 'admin')
const isStaffUser = computed(() => isOwner.value || isAdmin.value)
const roleText = computed(() => profile.value?.roleName || '普通用户')
const roleTagType = computed(() => {
  if (isOwner.value) return 'danger'
  if (isAdmin.value) return 'warning'
  return 'info'
})
const permissionText = computed(() => {
  if (isOwner.value) return '站点站长：拥有站点全部权限，可管理管理员、全部会员与站点配置。'
  if (isAdmin.value) return '站点管理员：可管理普通会员账号、发送站内信。'
  return '普通用户：会员有效期内可使用课程与课表功能，过期后仍可查看已录入数据。'
})
// 身份类型：统一展示使用身份（老师 / 学员 / 机构）；师资身份设计已下线
const identityText = computed(() => {
  if (isOwner.value) return '站点站长'
  if (isAdmin.value) return '站点管理员'
  return userRoleLabel(profile.value)
})

// 续费套餐：包月 / 包季 / 包年，价格与首页同源（renewal 为平季折后价，base 为标准价）；
// 师资身份计费档位已下线，统一按专职标准价
const fmtPrice = (v) => Number(v || 0).toString()
const renewPlans = computed(() => {
  const base = priceStore.price.professional || {}
  const r = priceStore.price.renewal.professional || {}
  const showOrigin = priceStore.price.season !== 'peak' // 平季才有折后对比
  const mk = (key, name, unit, baseAmt, renewAmt) => ({
    key,
    name,
    unit,
    price: fmtPrice(renewAmt),
    origin: showOrigin && renewAmt !== baseAmt ? fmtPrice(baseAmt) : ''
  })
  return [
    mk('monthly', '包月', '月', base.monthlyAmount, r.monthlyAmount),
    mk('quarterly', '包季', '季', base.quarterlyAmount, r.quarterlyAmount),
    mk('yearly', '包年', '年', base.yearlyAmount, r.yearlyAmount)
  ]
})
// 折扣标识：与首页 / 价格设置共用同一套口径（见 utils/priceTags）；无生效折扣时返回 null 不展示
const renewOffTag = computed(() => topDiscountTag(priceStore.price, { season: priceStore.price.season }))
// 未开通显示「开通」，已开通显示「续费」
const renewBtnText = computed(() => {
  const t = profile.value?.memberType
  return !t || t === 'trial' ? '开通' : '续费'
})

// ---- 会员信息（按角色展示更友好的文案） ----
const isPermanent = computed(() => profile.value?.memberType === 'permanent')
const memberTypeText = computed(() => {
  const p = profile.value
  if (!p) return '-'
  if (isOwner.value) return '永久会员（站长专属）'
  if (isAdmin.value) return '永久会员（管理员权益）'
  return p.memberTypeName || '免费试用'
})
const billingText = computed(() => {
  if (isPermanent.value) return '永久有效，无需续费'
  const t = profile.value?.memberType
  if (t === 'daily') return '按天购买'
  if (t === 'monthly') return '包月'
  if (t === 'quarterly') return '包季'
  if (t === 'yearly') return '包年'
  return '免费试用（注册赠送 30 天）'
})
const memberPeriodText = computed(() => {
  const p = profile.value
  if (!p) return '-'
  if (isPermanent.value) return '长期有效'
  if (!p.memberStart && !p.memberExpire) return '未开通'
  return `${fmtTs(p.memberStart)} ~ ${fmtTs(p.memberExpire)}`
})
const memberLeftOk = computed(() => isPermanent.value || !!profile.value?.memberActive)
const memberLeftText = computed(() => {
  if (isPermanent.value) return '永久有效'
  const p = profile.value
  if (!p) return '-'
  if (!p.memberActive) return '已过期'
  return p.memberLeftDays > 0 ? `${p.memberLeftDays} 天` : '不足 1 天'
})
const staffRenewTip = computed(() =>
  isOwner.value
    ? '您是站点站长，享有站点永久权益，无需开通或续费。'
    : '您是站点管理员，站点功能不受会员有效期限制，无需开通或续费。'
)

// 会员有效期剩余 7 天内，提示尽快续费（站长 / 管理员与永久会员无需提醒）
const memberExpiringSoon = computed(() => {
  const p = profile.value
  if (isStaffUser.value || isPermanent.value) return false
  return !!(p && p.memberActive && p.memberLeftDays <= 7)
})

const editing = ref(false)
const saving = ref(false)
const renewing = ref(false)
// 可自助修改：用户名 + 手机号（每自然月一次） + 邮箱
const form = ref({ username: '', phone: '', email: '' })

// 手机号本月是否已修改过（同一自然月内不可再次修改）
const phoneLocked = computed(() => {
  const ts = profile.value?.phoneChangedAt || 0
  if (!ts) return false
  return dayjs(ts * 1000).isSame(dayjs(), 'month')
})
const phoneChangedText = computed(() =>
  profile.value?.phoneChangedAt ? dayjs(profile.value.phoneChangedAt * 1000).format('YYYY-MM-DD') : ''
)
const fileInput = ref(null)

// ---- 修改密码 ----
const pwdVisible = ref(false)
const pwdSaving = ref(false)
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirm: '' })

function openPwd() {
  pwdForm.oldPassword = ''
  pwdForm.newPassword = ''
  pwdForm.confirm = ''
  pwdVisible.value = true
}

async function submitPwd() {
  if (pwdSaving.value) return
  if (!pwdForm.oldPassword) {
    ElMessage.warning('请输入原密码')
    return
  }
  if (pwdForm.newPassword.length < 6 || pwdForm.newPassword.length > 32) {
    ElMessage.warning('新密码长度需为 6-32 位')
    return
  }
  if (pwdForm.newPassword !== pwdForm.confirm) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  if (pwdForm.newPassword === pwdForm.oldPassword) {
    ElMessage.warning('新密码不能与原密码相同')
    return
  }
  pwdSaving.value = true
  try {
    await userApi.changePassword({
      oldPassword: pwdForm.oldPassword,
      newPassword: pwdForm.newPassword
    })
    ElMessage.success('密码已修改，下次登录请使用新密码')
    pwdVisible.value = false
  } finally {
    pwdSaving.value = false
  }
}

// 支付记录
const payments = ref([])
const totalPaid = computed(() => profile.value?.totalPaid || 0)
const money = (v) => Number(v || 0).toFixed(2)

async function loadPayments() {
  if (!userStore.isLogin) return
  try {
    const res = await userApi.payments()
    payments.value = res.data?.list || []
  } catch (e) {
    payments.value = []
  }
}

// 跳转首页唤起支付弹窗，并定位到用户点击的套餐（续费需由管理员人工开通）
function goHomePay(plan = 'monthly') {
  visible.value = false
  router.push('/')
  setTimeout(() => {
    window.dispatchEvent(new CustomEvent('open-payment', { detail: { plan } }))
  }, 200)
}

// 打开会员中心时刷新支付记录与价格配置（续费价格与首页/支付弹窗同源，避免硬编码不一致）
watch(visible, (v) => {
  if (v && isMemberMode.value) {
    loadPayments()
    priceStore.fetch()
  }
})

function startEdit() {
  form.value = {
    username: profile.value.username || '',
    phone: profile.value.phone || '',
    email: profile.value.email || ''
  }
  editing.value = true
}

async function save() {
  const username = String(form.value.username || '').trim()
  if (!username) {
    ElMessage.warning('用户名不能为空')
    return
  }
  if (/\s/.test(username)) {
    ElMessage.warning('用户名不能包含空格')
    return
  }
  const ulen = [...username].length
  if (ulen < 2 || ulen > 32) {
    ElMessage.warning('用户名长度需为 2-32 个字符')
    return
  }
  const phone = String(form.value.phone || '').trim()
  if (!/^1[3-9]\d{9}$/.test(phone)) {
    ElMessage.warning('请输入正确的 11 位手机号')
    return
  }
  if (phoneLocked.value && phone !== (profile.value?.phone || '')) {
    ElMessage.warning('手机号每月仅可修改一次，请下月再试')
    return
  }
  saving.value = true
  try {
    await userStore.updateProfile({ username, phone, email: sanitizeText(form.value.email) })
    ElMessage.success('资料已更新')
    editing.value = false
  } finally {
    saving.value = false
  }
}

function pickAvatar() {
  fileInput.value?.click()
}

async function onFileChange(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const formData = new FormData()
  formData.append('file', file)
  try {
    const res = await userApi.upload(formData)
    await userStore.updateProfile({ avatar: res.data.url })
    ElMessage.success('头像已更新')
  } catch (err) {
    ElMessage.error('头像上传失败')
  }
  e.target.value = ''
}
</script>

<style scoped>
.profile-head {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--brand-line);
}

.avatar-box {
  position: relative;
  cursor: pointer;
  border-radius: 50%;
  overflow: hidden;
}

.avatar-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}

.avatar-box:hover .avatar-mask {
  opacity: 1;
}

.head-info .name {
  font-size: 17px;
  font-weight: 600;
  margin-bottom: 6px;
}

.head-tags {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
}

.permanent-tip {
  font-size: 13px;
  color: var(--brand-success);
}

.money-cell {
  color: var(--brand-success);
  font-weight: 600;
}

.staff-note {
  margin-left: 4px;
}

.pwd-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.pwd-mask {
  letter-spacing: 2px;
  color: var(--brand-muted);
}

.empty {
  padding: 6px 0;
}

.rc-item {
  padding: 10px 12px;
  margin-bottom: 8px;
  border-radius: 10px;
  background: #fafbfc;
}

.rc-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.rc-amount {
  font-size: 15px;
  font-weight: 600;
  color: var(--brand-success);
}

.rc-time {
  margin-left: auto;
  font-size: 12px;
}

.small {
  font-size: 12px;
}

.card-box {
  margin-top: 20px;
}

.box-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
  padding-left: 8px;
  border-left: 3px solid var(--el-color-primary);
}

.box-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}

.renew {
  margin-top: 16px;
  padding: 14px;
  border-radius: 10px;
  background: var(--el-color-primary-light-9);
}

.renew-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.renew-title {
  font-size: 13px;
  font-weight: 600;
}

.plan-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 套餐行：金额与划线原价靠左，按钮紧随其后靠右，视线不必来回跳 */
.plan-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  padding: 8px 10px;
  border-radius: 8px;
  background: #fff;
}

.plan-name {
  font-size: 13px;
  min-width: 34px;
}

.plan-price {
  font-size: 16px;
  font-weight: 700;
  color: var(--el-color-primary);
  line-height: 1;
}

.plan-unit {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.plan-origin {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-decoration: line-through;
}

.plan-btn {
  margin-left: auto;
}

.renew-tip {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.ok {
  color: var(--brand-success);
}

.expired {
  color: var(--brand-danger);
}

.identity-warn {
  margin-top: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  background: #fff9e6;
  color: #8a6d3b;
  border-left: 3px solid var(--brand-warning);
  font-size: 13px;
}
</style>
