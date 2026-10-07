<template>
  <div class="home">
    <!-- Hero -->
    <section class="hero">
      <div class="hero-inner">
        <div class="hero-left">
          <div class="hero-tagline">{{ brand.tagline }}</div>
          <h1>课程与课表，<span>一键排好</span></h1>
          <p class="hero-desc">
            老师排课对账，家长看课表：录一次课自动生成每周课表，课时进度与费用一目了然。
          </p>
          <div class="hero-actions">
            <el-button type="primary" size="large" round @click="onStart">
              {{ userStore.isLogin ? '进入排课管理' : '免费试用 · 无需注册' }}
            </el-button>
            <el-button v-if="userStore.isLogin && !userStore.isStaff" size="large" round @click="openPay('monthly')">
              会员续费
            </el-button>
            <el-link type="primary" :underline="false" class="hero-guide-link" @click="scrollToDemo">
              <el-icon class="guide-link-ico"><VideoCamera /></el-icon>观看功能演示
            </el-link>
          </div>
          <ul class="hero-trust">
            <li>打开网页即用</li>
            <li>无需下载安装</li>
            <li>数据仅自己可见</li>
          </ul>
        </div>
        <div class="hero-right">
          <div class="mini-week">
            <div class="mini-head">
              <span>本周课表预览</span>
              <el-tag size="small" :type="hasRealWeek ? 'success' : 'info'" effect="light" round>
                {{ miniStateTag }}
              </el-tag>
              <el-button link type="primary" size="small" class="mini-link" @click="goSchedule">
                {{ miniStateLink }}
              </el-button>
            </div>
            <div class="mini-grid">
              <div v-for="d in weekDays" :key="d" class="mini-col">
                <div class="mini-day">{{ d }}</div>
                <div class="mini-cell" :class="{ on: miniData[d].length }">
                  <div v-for="(e, i) in miniData[d]" :key="i" class="mini-block">
                    {{ e }}
                  </div>
                  <div v-if="!miniData[d].length" class="mini-free">—</div>
                </div>
              </div>
            </div>
            <div class="mini-tip muted small">{{ miniStateTip }}</div>
          </div>
        </div>
      </div>
    </section>

    <div class="page">
      <!-- 功能 -->
      <section class="card">
        <div class="card-title">核心功能<span class="sub">录一次课，省一学期重复排</span></div>
        <div class="feature-grid">
          <div v-for="f in features" :key="f.title" class="feature-item">
            <div class="f-icon" :style="{ background: f.bg }">
              <el-icon :color="f.color"><component :is="f.icon" /></el-icon>
            </div>
            <div class="f-title">{{ f.title }}</div>
            <div class="f-desc muted">{{ f.desc }}</div>
          </div>
        </div>
      </section>

      <!-- 流程 -->
      <section class="card">
        <div class="card-title">三步上手</div>
        <div class="steps-flow">
          <template v-for="(s, i) in steps" :key="i">
            <div class="flow-node">
              <div class="step-no">{{ i + 1 }}</div>
              <div class="step-title">{{ s.title }}</div>
              <div class="muted flow-desc">{{ s.desc }}</div>
            </div>
            <div v-if="i < steps.length - 1" class="flow-arrow" aria-hidden="true">→</div>
          </template>
        </div>
      </section>

      <!-- 功能演示：独立版块，复用指南页的视频框组件，不挤占首屏与主 CTA -->
      <div ref="demoRef" class="demo-anchor">
        <DemoSection />
      </div>

      <!-- 会员 -->
      <section class="card">
        <div class="card-title">
          <div class="membership-title">
            会员权益
            <el-tooltip content="查看会员权益对照表" placement="top">
              <span class="benefit-tip-icon" @click="benefitVisible = true">
                <el-icon><InfoFilled /></el-icon>
              </span>
            </el-tooltip>
          </div>
          <span class="sub" v-if="userStore.isLogin">
            当前身份：<b>{{ userRoleLabel(userStore.profile) }}</b>
          </span>
          <span v-else class="sub">注册后查看价格，支持免注册试用</span>
        </div>

        <!-- 价格调整预告：一行式提示，简单直白 -->
        <div v-if="pendingVisible && pendingSchedules.length" class="price-notice">
          <span class="pn-tag">价格调整预告</span>
          <div class="pn-body">
            <div v-for="it in pendingSchedules" :key="it.id" class="pn-line">
              <b>{{ it.cardName }}</b> {{ fmtPendingDate(it.effectiveAt) }} 调整：{{ pendingSummary(it) }}
            </div>
          </div>
          <el-icon class="pn-close" title="关闭" @click="pendingVisible = false"><Close /></el-icon>
        </div>

        <div class="plan-grid">
          <div v-for="p in plans" :key="p.name" class="plan" :class="{ hot: p.hot }">
            <el-tag v-if="p.hot" type="danger" size="small" effect="dark" class="plan-tag">推荐</el-tag>
            <div class="plan-name">
              {{ p.name }}
              <el-tag v-if="p.showSeason && seasonTag" size="small" :type="seasonTag.type" effect="light" class="season-tag">{{ seasonTag.text }}</el-tag>
            </div>
            <!-- 价格数字：免费试用卡始终展示，付费卡未登录时隐藏 -->
            <div v-if="userStore.isLogin || p.key === 'trial'" class="plan-price">
              <span class="price-now">{{ p.price }}</span>
              <span v-if="p.origin" class="price-origin">{{ p.origin }}</span>
            </div>
            <div v-else class="plan-price is-locked">
              <span class="price-locked">登录后查看价格</span>
            </div>
            <ul class="plan-list">
              <li v-for="l in p.list" :key="l">{{ l }}</li>
            </ul>
            <el-button :type="p.hot ? 'primary' : 'default'" round class="plan-btn" @click="onJoin(p)">
              {{ p.btn }}
            </el-button>
          </div>
        </div>
      </section>

      <!-- 联系与支付：未登录只展示咨询码，登录后再展示收款码 -->
      <section class="card">
        <div class="card-title">
          联系与支付
          <span class="sub">
            {{ userStore.isLogin ? '扫码添加好友咨询，支持微信 / 支付宝转账' : '扫码添加好友咨询功能与开通方式' }}
          </span>
        </div>
        <div class="qr-wrap">
          <QrCard
            title="添加微信好友"
            desc="咨询功能、反馈问题、申请开通"
            :src="qrcodes.wechatFriend"
          />
          <template v-if="userStore.isLogin">
            <QrCard title="微信收款码" desc="会员续费转账（备注用户名）" :src="qrcodes.wechatPay" />
            <QrCard title="支付宝收款码" desc="会员续费转账（备注用户名）" :src="qrcodes.alipay" />
          </template>
        </div>
      </section>
    </div>

    <PaymentDialog v-model="payVisible" :plan="payPlan" />

    <el-dialog v-model="benefitVisible" title="会员权益对照表" width="660px" align-center>
      <div class="benefit-tip">
        所有账号统一按<b>标准价</b>计费（师资身份与折扣档位已下线）。
        平季续费{{ seasonDiscountText }}；免费试用期间最多可添加 3 门课程，课程数量不限等完整权益请开通会员。
      </div>
      <el-table :data="benefits" border size="small">
        <el-table-column prop="name" label="权益项" min-width="150" />
        <el-table-column label="免费试用" align="center">
          <template #default="{ row }"><span :class="row.trial ? 'ok' : 'muted'">{{ row.trial ? '✓' : '—' }}</span></template>
        </el-table-column>
        <el-table-column label="包月" align="center">
          <template #default="{ row }"><span :class="row.monthly ? 'ok' : 'muted'">{{ row.monthly ? '✓' : '—' }}</span></template>
        </el-table-column>
        <el-table-column label="包季" align="center">
          <template #default="{ row }"><span :class="row.quarterly ? 'ok' : 'muted'">{{ row.quarterly ? '✓' : '—' }}</span></template>
        </el-table-column>
        <el-table-column label="包年" align="center">
          <template #default="{ row }"><span :class="row.yearly ? 'ok' : 'muted'">{{ row.yearly ? '✓' : '—' }}</span></template>
        </el-table-column>
      </el-table>
      <div class="benefit-season">
        <b>淡旺季说明：</b>开学期间为常规补课期，续费{{ seasonDiscountText }}；寒暑假（约 7–8 月、1–2 月）为补课高峰，排课密集，建议提前续费。
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { Calendar, Clock, Close, Document, Edit, InfoFilled, TrendCharts, VideoCamera } from '@element-plus/icons-vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { useRouter } from 'vue-router'
import QrCard from '@/components/QrCard.vue'
import PaymentDialog from '@/components/PaymentDialog.vue'
import DemoSection from '@/components/DemoSection.vue'
import { qrcodes } from '@/config/site'
import { brand } from '@/config/brand'
import { scheduleApi, priceApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { usePriceStore } from '@/stores/price'
import { useTrialStore } from '@/stores/trial'
import { userRoleLabel } from '@/utils/identity'
import { zhe, topDiscountTag } from '@/utils/priceTags'

const router = useRouter()
const userStore = useUserStore()
const priceStore = usePriceStore()
const trialStore = useTrialStore()

const demoRef = ref(null)

function scrollToDemo() {
  demoRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const features = [
  {
    title: '课程集中管理',
    desc: '学生、年级、科目、费用一次录入，随时检索与修改。',
    icon: Document,
    color: '#2f6fed',
    bg: '#eaf1ff'
  },
  {
    title: '周课表自动生成',
    desc: '按补课周期与每周时段，自动铺满整周课表。',
    icon: Calendar,
    color: '#ff7a45',
    bg: '#fff1e9'
  },
  {
    title: '多时段灵活排课',
    desc: '一门课支持多个时间段，按天按小时分开显示。',
    icon: Clock,
    color: '#22a06b',
    bg: '#e8f7f0'
  },
  {
    title: '调课改期留痕',
    desc: '停课、改期、加课随手调整，课时与费用照常统计。',
    icon: Edit,
    color: '#e6a23c',
    bg: '#fff6e6'
  },
  {
    title: '课时进度看得见',
    desc: '填写总课时自动推算上到哪天，已上节数一目了然。',
    icon: TrendCharts,
    color: '#7a5bd6',
    bg: '#f2ecff'
  }
]

const steps = [
  { title: '录入课程', desc: '填写学生情况、补课周期与每周上课时段。' },
  { title: '生成课表', desc: '按周自动展开课表，多时段、多学员不冲突。' },
  { title: '调课改期', desc: '停课、改期、加课随手操作，课时与费用自动汇总。' }
]

const fmtPrice = (v) => Number(v || 0).toString()

// 师资身份与计费档位已下线：全站统一按专职标准价展示
// 折扣标签：首页与后台价格设置共用同一套口径（活动 / 寒暑假 / 平季），见 utils/priceTags
const seasonTag = computed(() => topDiscountTag(priceStore.price, { season: priceStore.price.season }))

const plans = computed(() => {
  const cur = 'professional'
  const r = priceStore.price.renewal[cur] || priceStore.price.renewal.professional
  // 原价统一为「专职老师标准价」，学生折后价在此原价基础上叠加学生折扣与平季折扣
  const b = priceStore.price.professional
  const discount = priceStore.price.season !== 'peak' // 平季才有折后对比
  // 头条展示折后价（续费价）；原价（老师设定标准价）划线对比
  const mk = (key, name, period, baseAmt, renewAmt, list, btn, hot) => ({
    key,
    name,
    price: `¥${fmtPrice(renewAmt)} / ${period}`,
    origin: discount && renewAmt !== baseAmt ? `¥${fmtPrice(baseAmt)}` : '',
    showSeason: key !== 'trial',
    list,
    btn,
    hot
  })
  return [
    mk('trial', '免费试用', '30 天', 0, 0, ['无需注册直接用', '最多添加 3 门课程', '周课表自动生成'], '立即试用', false),
    mk('monthly', '包月会员', '月', b.monthlyAmount, r.monthlyAmount, ['全部免费权益', '优先问题响应', '课程数量不限'], '立即开通', false),
    mk('quarterly', '包季会员', '季', b.quarterlyAmount, r.quarterlyAmount, ['包月全部权益', '按季付费更划算', '课表批量导出'], '立即开通', false),
    mk('yearly', '包年会员', '年', b.yearlyAmount, r.yearlyAmount, ['包月全部权益', '年度更优惠', '新功能优先体验'], '立即开通', true)
  ]
})

// 淡旺季说明文案：活动优先，其次平季折扣；都不可用时按标准价续费
// （避免此前直接拼接「续费享原价优惠」这类自相矛盾的表述）
const seasonDiscountText = computed(() => {
  const p = priceStore.price
  if (p.activityEnabled) return `享 ${zhe(p.activityDiscount)} 折优惠`
  if (p.season !== 'peak' && p.renewalEnabled) return `享 ${zhe(p.renewalDiscount)} 折优惠`
  return '按标准价（无折扣）'
})

const benefitVisible = ref(false)
const benefits = [
  { name: '课程集中管理', trial: true, monthly: true, quarterly: true, yearly: true },
  { name: '周课表自动生成', trial: true, monthly: true, quarterly: true, yearly: true },
  { name: '课程数量不限', trial: false, monthly: true, quarterly: true, yearly: true },
  { name: '优先问题响应', trial: false, monthly: true, quarterly: true, yearly: true },
  { name: '课表批量导出', trial: false, monthly: false, quarterly: true, yearly: true },
  { name: '新功能优先体验', trial: false, monthly: false, quarterly: false, yearly: true }
]

const weekDays = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
// 默认演示数据：未登录或尚未添加订单时使用
const defaultMiniData = {
  周一: ['18:00 初二 数学'],
  周二: ['18:00 初二 数学'],
  周三: ['19:00 高一 英语'],
  周四: ['18:00 初二 数学'],
  周五: [],
  周六: ['09:00 六年级 奥数', '14:00 初三 物理'],
  周日: ['10:00 五年级 语文']
}

// 真实课表预览：登录后按本周订单数据生成；无订单时回退到演示数据
const realWeekData = ref(null)
const hasRealWeek = computed(() => !!realWeekData.value)
const miniData = computed(() => realWeekData.value || defaultMiniData)

// 未登录态把「示例数据」明确为示意图，并把入口文案改成注册承诺，避免用假数据误导
const miniStateTag = computed(() => {
  if (hasRealWeek.value) return '我的课表'
  return userStore.isLogin ? '暂无课程' : '示意图'
})
const miniStateLink = computed(() => {
  if (hasRealWeek.value) return '查看完整课表'
  return userStore.isLogin ? '去录入课程' : '注册后查看我的课表'
})
const miniStateTip = computed(() => {
  if (hasRealWeek.value) return '以上为根据您添加的课程自动生成的本周课表'
  return userStore.isLogin ? '录入课程后，这里会按课程数据自动生成课表' : '注册后，这里会用你的课程数据生成真实课表'
})

async function loadWeekPreview() {
  // 免注册试用：直接用本地试用数据生成真实课表预览
  if (!userStore.isLogin) {
    if (trialStore.hasData) {
      const res = trialStore.weekSchedule(dayjs().startOf('week').add(1, 'day'), dayjs().endOf('week').add(1, 'day'))
      const map = {}
      weekDays.forEach((d) => (map[d] = []))
      res.events.forEach((ev) => {
        const label = weekDays[(ev.weekday || 1) - 1]
        if (!label) return
        const txt = [ev.startTime, ev.grade, ev.subject].filter(Boolean).join(' ')
        if (txt) map[label].push(txt)
      })
      realWeekData.value = map
    } else {
      realWeekData.value = null
    }
    return
  }
  try {
    const res = await scheduleApi.week()
    const events = res.data?.events || []
    if (!events.length) {
      realWeekData.value = null // 未添加订单
      return
    }
    const map = {}
    weekDays.forEach((d) => {
      map[d] = []
    })
    events.forEach((ev) => {
      const label = weekDays[(ev.weekday || 1) - 1]
      if (!label) return
      const txt = [ev.startTime, ev.grade, ev.subject].filter(Boolean).join(' ')
      if (txt) map[label].push(txt)
    })
    realWeekData.value = map
  } catch (e) {
    realWeekData.value = null
  }
}

// 预览卡片右侧入口：按当前状态跳转到匹配的页面
// - 已有课表数据 → 课表页查看完整课表
// - 已登录但暂无课程 → 课程安排页并直接打开「添加课程」，避免跳到空课表页无事可做
// - 未登录 → 打开注册弹窗（按钮文案承诺的就是注册）
function goSchedule() {
  if (hasRealWeek.value) {
    router.push('/schedule')
    return
  }
  if (userStore.isLogin) {
    router.push({ path: '/orders', query: { new: '1' } })
    return
  }
  window.dispatchEvent(
    new CustomEvent('open-auth', {
      detail: { mode: 'register', tip: '注册后，这里会用你的课程数据生成真实课表。' }
    })
  )
}

// 二维码图片见 src/config/site.js

const payVisible = ref(false)
const payPlan = ref('monthly')
// 未登录点击开通时先记住套餐，登录成功后自动弹出支付二维码
let pendingPlan = ''

function openPay(plan) {
  payPlan.value = plan
  payVisible.value = true
}

// 免注册试用：未登录点击「立即试用」直接进入排课页
function onStart() {
  router.push('/orders')
}

function onJoin(plan) {
  // 免费试用卡：登录 / 未登录都直接进入排课页（未登录进入免注册试用模式）
  if (plan.key === 'trial') {
    router.push('/orders')
    return
  }
  if (userStore.profile?.memberType === 'permanent') {
    ElMessage.success('您已是永久会员，无需再开通')
    return
  }
  if (!userStore.isLogin) {
    pendingPlan = plan.key
    window.dispatchEvent(new CustomEvent('open-auth'))
    return
  }
  openPay(plan.key)
}

watch(
  () => userStore.isLogin,
  (login) => {
    loadWeekPreview() // 登录/退出后刷新本周课表预览
    if (login && pendingPlan) {
      openPay(pendingPlan)
      pendingPlan = ''
    }
  }
)

// 免注册试用：本地数据变化时同步刷新课表预览
watch(
  () => trialStore.orderCount,
  () => {
    if (!userStore.isLogin) loadWeekPreview()
  }
)

// 个人中心「查看收款码」等入口通过事件唤起支付弹窗
function onOpenPayment(e) {
  openPay(e?.detail?.plan || 'monthly')
}

// 价格调整预告：拉取尚未生效（生效日期在未来）的价格预约，首页提前告知用户。
// 仅展示 future 的预约，已到点 / 已应用的不再预告（由站内信通知）。
const pendingSchedules = ref([])
const pendingVisible = ref(true)
async function loadPending() {
  try {
    const res = await priceApi.schedulesPending()
    pendingSchedules.value = res?.data || []
  } catch (e) {
    pendingSchedules.value = []
  }
}
const fmtPendingDate = (ts) => dayjs.unix(ts).format('YYYY-MM-DD')
function pendingSummary(it) {
  const b = it.before || {}
  const t = it.target || {}
  const yuan = (v) => `¥${Number(v || 0).toString()}`
  const zheTxt = (v) => `${zhe(v)} 折`
  switch (it.card) {
    case 'pro':
      return `标准价将由 ${yuan(b.proMonthlyAmount)}/月、${yuan(b.proQuarterlyAmount)}/季、${yuan(b.proYearlyAmount)}/年 调整为 ${yuan(t.proMonthlyAmount)}/月、${yuan(t.proQuarterlyAmount)}/季、${yuan(t.proYearlyAmount)}/年`
    case 'student':
      return `学生折扣将由 ${zheTxt(b.studentDiscount)} 调整为 ${zheTxt(t.studentDiscount)}`
    case 'renewal':
      return t.renewalEnabled ? `平季续费优惠将开启，享 ${zheTxt(t.renewalDiscount)}` : '平季续费优惠将关闭，恢复按标准价'
    case 'activity':
      if (t.activityEnabled) return `限时活动「${t.activityName || '未命名'}」将于 ${fmtPendingDate(it.effectiveAt)} 上线，享 ${zheTxt(t.activityDiscount)}`
      return `限时活动「${b.activityName || '未命名'}」将于 ${fmtPendingDate(it.effectiveAt)} 结束`
    default:
      return '价格即将调整'
  }
}

onMounted(() => {
  priceStore.fetch()
  loadPending()
  loadWeekPreview()
  window.addEventListener('open-payment', onOpenPayment)
})

onBeforeUnmount(() => {
  window.removeEventListener('open-payment', onOpenPayment)
})
</script>

<style scoped>
.hero {
  background: linear-gradient(135deg, #fff6f0 0%, #eef3ff 100%);
  border-bottom: 1px solid var(--brand-line);
}

.hero-inner {
  max-width: 1240px;
  margin: 0 auto;
  padding: 48px 16px 40px;
  display: flex;
  align-items: center;
  gap: 40px;
}

.hero-left {
  flex: 1;
}

.hero-left h1 {
  margin: 14px 0 12px;
  font-size: 34px;
  line-height: 1.3;
}

.hero-left h1 span {
  color: var(--el-color-primary);
}

/* 品牌副标题：从 Header 移到首页首屏，强化品牌心智 */
.hero-tagline {
  font-size: 15px;
  color: var(--brand-sub);
  margin: 12px 0 6px;
}

.hero-desc {
  color: var(--brand-sub);
  line-height: 1.8;
  /* 560px 可让整句在一行内排完，避免末行只剩一两个字的"孤字" */
  max-width: 560px;
  margin: 0 0 22px;
}

.hero-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 26px;
}

/* 次要入口：功能演示降级为文字链，避免与主 CTA 抢注意力 */
.hero-guide-link {
  font-size: 14px;
  font-weight: 400;
}

.hero-guide-link .guide-link-ico {
  margin-right: 3px;
  font-size: 15px;
  vertical-align: -2px;
}

/* 信任条：替代原数字条，用极简文案建立可信度 */
.hero-trust {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 13px;
  color: var(--brand-muted);
}

.hero-trust li {
  display: flex;
  align-items: center;
  gap: 6px;
}

.hero-trust li::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--el-color-primary);
  opacity: 0.55;
}

.hero-right {
  width: 420px;
}

.mini-week {
  background: #fff;
  border-radius: var(--radius-card);
  box-shadow: var(--shadow-card);
  padding: 16px;
}

.mini-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
}

.mini-link {
  margin-left: auto;
  font-weight: 400;
}

.mini-tip {
  margin-top: 10px;
  text-align: center;
}

.mini-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 6px;
}

.mini-day {
  font-size: 11px;
  color: var(--brand-muted);
  text-align: center;
  margin-bottom: 4px;
}

.mini-cell {
  min-height: 120px;
  border-radius: 8px;
  background: #fafbfc;
  padding: 4px;
}

.mini-cell.on {
  background: var(--el-color-primary-light-9);
}

.mini-block {
  font-size: 10px;
  line-height: 1.4;
  color: var(--el-color-primary-dark-2);
  background: #fff;
  border-radius: 5px;
  padding: 4px;
  margin-bottom: 4px;
  border-left: 2px solid var(--el-color-primary);
}

.mini-free {
  text-align: center;
  color: #c9ced8;
  font-size: 12px;
  padding-top: 40px;
}

.feature-grid {
  display: grid;
  /* 5 张卡片在宽屏一行排满，窄屏自动降为两列 / 单列 */
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 16px;
}

.feature-item {
  padding: 18px;
  border: 1px solid var(--brand-line);
  border-radius: 12px;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
}

.feature-item:hover {
  box-shadow: var(--shadow-card);
  transform: translateY(-2px);
}

.f-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  margin-bottom: 12px;
}

.f-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 6px;
}

.f-desc {
  font-size: 13px;
  line-height: 1.7;
}

/* 三步上手：横向流程线，压缩高度、弱化重复的功能卡信息 */
.steps-flow {
  display: flex;
  align-items: stretch;
  gap: 8px;
}

.flow-node {
  flex: 1;
  min-width: 0;
  padding: 14px 16px;
  border-radius: 12px;
  background: #fafbfc;
  text-align: center;
}

.flow-node .step-no {
  margin: 0 auto 8px;
}

.flow-desc {
  font-size: 13px;
  line-height: 1.6;
}

.flow-arrow {
  align-self: center;
  color: var(--brand-muted);
  font-size: 16px;
}

@media (max-width: 768px) {
  .steps-flow {
    flex-direction: column;
  }

  .flow-arrow {
    transform: rotate(90deg);
    text-align: center;
  }
}

.step-no {
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.step-title {
  font-weight: 600;
  margin-bottom: 4px;
}

.step-body .muted {
  font-size: 13px;
  line-height: 1.6;
}

.plan-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}

.plan {
  position: relative;
  border: 1px solid var(--brand-line);
  border-radius: 12px;
  padding: 20px;
}

.plan.hot {
  border-color: var(--el-color-primary);
  box-shadow: 0 4px 18px rgba(255, 122, 69, 0.16);
}

.plan-tag {
  position: absolute;
  top: -10px;
  right: 12px;
}

.plan-name {
  font-weight: 600;
  margin-bottom: 6px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.season-tag {
  font-weight: 400;
}

.preview-switch {
  font-size: 12px;
  color: var(--brand-muted);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.plan-price {
  font-size: 20px;
  color: var(--el-color-primary);
  margin-bottom: 12px;
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.price-now {
  font-weight: 700;
}

/* 未登录占位：与价格行等高，避免卡片高度跳动 */
.plan-price.is-locked {
  font-size: 15px;
  color: var(--brand-muted);
}

.price-locked {
  font-weight: 400;
}

.price-origin {
  font-size: 13px;
  color: var(--brand-muted);
  text-decoration: line-through;
  font-weight: 400;
}

.plan-list {
  margin: 0 0 16px;
  padding-left: 18px;
  font-size: 13px;
  color: var(--brand-sub);
  line-height: 1.9;
}

.plan-btn {
  width: 100%;
}

.qr-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

@media (max-width: 900px) {
  .hero-inner {
    flex-direction: column;
  }
  .hero-right {
    width: 100%;
  }
}
.membership-title {
  display: flex;
  align-items: center;
}
.benefit-tip-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  margin-left: 6px;
  border-radius: 50%;
  background: rgba(47, 111, 237, 0.12);
  color: var(--brand-blue);
  cursor: pointer;
  vertical-align: middle;
  transition: background 0.2s, color 0.2s;
}
.benefit-tip-icon:hover {
  background: var(--brand-blue);
  color: #fff;
}
.benefit-tip-icon .el-icon {
  font-size: 13px;
}

.benefit-tip {
  font-size: 13px;
  color: var(--brand-sub);
  margin-bottom: 12px;
  line-height: 1.7;
}

.benefit-season {
  margin-top: 12px;
  font-size: 13px;
  color: var(--brand-sub);
  line-height: 1.8;
  background: var(--el-color-primary-light-9);
  padding: 10px 12px;
  border-radius: 8px;
}

.ok {
  color: #22a06b;
  font-weight: 600;
}

/* 价格调整预告：一行式提示，浅色底 + 橙色小标签，不喧宾夺主 */
.price-notice {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 10px 12px;
  border: 1px solid #ffe2cc;
  border-radius: 10px;
  background: #fff8f2;
}

.pn-tag {
  flex: none;
  padding: 2px 8px;
  border-radius: 6px;
  background: #ff7a45;
  color: #fff;
  font-size: 12px;
}

.pn-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 13px;
  color: var(--brand-sub);
}

.pn-line b {
  color: var(--brand-ink);
  font-weight: 600;
}

.pn-close {
  flex: none;
  cursor: pointer;
  color: var(--brand-muted);
  transition: color 0.2s;
}

.pn-close:hover {
  color: var(--brand-ink);
}
</style>
