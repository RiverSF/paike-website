<template>
  <div class="page">
    <section class="card">
      <div class="card-title">使用提示<span class="sub">快速上手，3 分钟学会</span></div>
      <div class="tips-grid">
        <div v-for="(t, i) in tips" :key="i" class="tip-card">
          <div class="tip-icon" :style="{ background: t.bg, color: t.color }">{{ i + 1 }}</div>
          <div>
            <div class="tip-title">{{ t.title }}</div>
            <div class="tip-desc muted">{{ t.desc }}</div>
          </div>
        </div>
      </div>
    </section>

    <section class="card">
      <div class="card-title">常见问题<span class="sub">共 {{ faqTotal }} 个问题，按主题分组，点击展开查看</span></div>
      <div v-for="g in faqGroups" :key="g.key" class="faq-group">
        <div class="faq-group-title">
          <span class="faq-group-icon" :style="{ background: g.bg, color: g.color }">
            <el-icon><component :is="g.icon" /></el-icon>
          </span>
          {{ g.title }}
        </div>
        <el-collapse v-model="activeFaq[g.key]">
          <el-collapse-item v-for="(f, i) in g.items" :key="i" :name="`${g.key}-${i}`">
            <template #title>
              <span class="faq-q">Q：{{ f.q }}</span>
            </template>
            <div class="faq-a">{{ f.a }}</div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </section>

    <section class="card">
      <div class="card-title">
        问题反馈
        <span class="sub">使用中遇到问题？提交后我们会尽快处理并回复</span>
      </div>

      <el-input
        v-model="fb.content"
        type="textarea"
        :rows="4"
        maxlength="200"
        show-word-limit
        placeholder="请描述你遇到的问题或建议（最多 200 字）"
      />
      <div class="limit-tip muted">
        每个账号每小时只能提交一次，内容最多 200 字，请勿发布违规内容。
        <span v-if="waitText" class="wait">（{{ waitText }}）</span>
      </div>
      <div class="fb-row">
        <el-input v-model="fb.contact" placeholder="联系方式（微信 / 手机号，选填）" style="width: 100%; max-width: 260px" />
        <div class="flex-spacer" />
        <el-button v-if="!userStore.isLogin" type="primary" round @click="needLogin">登录后提交</el-button>
        <el-button
          v-else
          type="primary"
          round
          :loading="submitting"
          :disabled="!fb.content.trim() || !!waitText"
          @click="submit"
        >
          提交反馈
        </el-button>
      </div>

      <div class="fb-list">
        <div class="list-title">我的反馈（{{ total }}）</div>
        <div v-if="!userStore.isLogin" class="muted empty">登录后可查看你提交的反馈</div>
        <div v-else-if="!list.length" class="muted empty">暂无反馈内容</div>
        <div v-for="item in list" :key="item.id" class="fb-item">
          <div class="fb-head">
            <el-avatar :size="28">{{ (item.username || '匿').slice(0, 1) }}</el-avatar>
            <span class="fb-name">{{ item.username || '匿名用户' }}</span>
            <el-tag size="small" :type="item.status === 'resolved' ? 'success' : 'warning'" effect="light" round>
              {{ item.status === 'resolved' ? '已回复' : '待处理' }}
            </el-tag>
            <span class="muted fb-time">{{ formatTime(item.createdAt) }}</span>
          </div>
          <div class="fb-content">{{ item.content }}</div>
          <div v-if="item.reply" class="fb-reply">官方回复：{{ item.reply }}</div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Calendar, User, Wallet } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { feedbackApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { sanitizeText, validateTexts } from '@/utils/text'

const userStore = useUserStore()

const tips = [
  {
    title: '先录课程，再生成课表',
    desc: '在「课程安排」页添加课程后，「课表」会自动按补课周期与每周时段展开，无需手工排课。',
    color: '#ff7a45',
    bg: '#fff1e9'
  },
  {
    title: '周时间段支持多个',
    desc: '如周一至周四 18:00-20:00、周六 09:00-11:00，可添加多行时段；如需分阶段改期，为该行设置「生效日期」即可。',
    color: '#2f6fed',
    bg: '#eaf1ff'
  },
  {
    title: '特殊备注用红点标记',
    desc: '打开「特殊备注」开关，课表卡片右上角会显示红点，鼠标悬浮即可查看完整备注。',
    color: '#e34d59',
    bg: '#fdeced'
  },
  {
    title: '临时调课去课表里改',
    desc: '改时间、停课、加课在「课表」定位到具体日期的课次单独调整，只影响当次，不动每周固定安排。',
    color: '#7a5bd6',
    bg: '#f2ecff'
  },
  {
    title: '课时进度自动推算',
    desc: '填写总课时后，系统按每周课次推算预计上到哪天，可一键把结果填入结束日期。',
    color: '#e6a23c',
    bg: '#fff6e6'
  },
  {
    title: '状态决定课表配色',
    desc: '未开始为蓝色、进行中为橙色、已结束为灰色；停课显示删除线，调课 / 加课为紫色。',
    color: '#22a06b',
    bg: '#e8f7f0'
  }
]

// 常见问题：只保留用户真正会用到的疑问，剔除无感与基础内容
const faqGroups = [
  {
    key: 'account',
    title: '账号与注册',
    icon: User,
    color: '#2f6fed',
    bg: '#eaf1ff',
    items: [
      {
        q: '怎么注册？需要准备什么？',
        a: '填写用户名、密码与 11 位手机号即可自助注册（无短信验证码），注册成功即赠送 30 天免费会员。手机号是唯一的登录账号，注册后每月仅可修改一次。'
      },
      {
        q: '忘记密码或登录不了怎么办？',
        a: '登录用手机号 + 密码（用户名不能登录）。忘记密码请联系管理员重置；连续输错 5 次会锁定 30 分钟，被管理员冻结时需联系解冻。'
      }
    ]
  },
  {
    key: 'member',
    title: '会员与付费',
    icon: Wallet,
    color: '#e6a23c',
    bg: '#fff6e6',
    items: [
      {
        q: '免费试用多久？到期会丢数据吗？',
        a: '无需注册即可试用，最多添加 3 门课程（数据保存在本机浏览器，注册后可导入永久保存）；注册再赠 30 天免费会员。试用或会员到期后，已录入的订单和课表仍然可以查看，但无法新增课程与调课，续费即恢复，到期前会提前提醒。'
      },
      {
        q: '会员价格怎么算？',
        a: '全站统一按标准价计费，开学平季续费另有优惠，寒暑假高峰按标准价；试用期间最多添加 3 门课程，开通会员后课程数量不限，并解锁课表导出、费用统计等功能。具体以首页「会员与价格」为准。'
      },
      {
        q: '如何续费？为什么点「续费」没直接开通？',
        a: '系统不支持自助开通。点「开通 / 续费」弹出微信 / 支付宝收款码，扫码转账（备注用户名）后联系管理员人工开通；开通后可在会员中心「支付记录」查看金额、时长与到期变化。'
      }
    ]
  },
  {
    key: 'course',
    title: '课程与课表',
    icon: Calendar,
    color: '#22a06b',
    bg: '#e8f7f0',
    items: [
      {
        q: '一门课每周多次 / 分阶段上课怎么排？',
        a: '在「周时间段」添加多行，每行勾选星期并填起止时间；需分阶段改变安排（如 5 月起改到周日），为该行设置「生效日期」区间即可，同一星期重叠周期内不能重复。'
      },
      {
        q: '临时调课、停课、加课怎么操作？',
        a: '不用改动每周固定安排：到「课表」定位到具体日期的课次单独调整（改时间 / 停课 / 加课），只影响当次。'
      },
      {
        q: '怎么知道还剩多少课时、上到哪天？',
        a: '课程里填写「总课时」并选好费用方式，系统按每周课次推算预计结束日期，可一键填入；课表顶部汇总区间课时、总课时与收入 / 开支。'
      }
    ]
  }
]

const faqTotal = faqGroups.reduce((n, g) => n + g.items.length, 0)
const activeFaq = reactive({ account: 'account-0' })
const list = ref([])
const total = ref(0)
const submitting = ref(false)
const fb = reactive({ content: '', contact: '' })

// 提交频率限制：每小时一次
const nextAllowedAt = ref(0)
const waitText = computed(() => {
  if (!nextAllowedAt.value) return ''
  const left = nextAllowedAt.value - Math.floor(Date.now() / 1000)
  if (left <= 0) return ''
  return `${Math.ceil(left / 60)} 分钟后可再次提交`
})

async function loadFeedback() {
  // 未登录不展示任何反馈；登录后仅展示本人提交的反馈（不可查看他人内容）
  if (!userStore.isLogin) {
    list.value = []
    total.value = 0
    nextAllowedAt.value = 0
    return
  }
  try {
    const res = await feedbackApi.mine()
    list.value = res.data.list || []
    total.value = list.value.length
    nextAllowedAt.value = res.data.nextAllowedAt || 0
  } catch (e) {
    /* 接口异常时静默 */
  }
}

async function submit() {
  if (!fb.content.trim()) {
    ElMessage.warning('请填写反馈内容')
    return
  }
  const textCheck = validateTexts({ [fb.content]: '反馈内容', [fb.contact]: '联系方式' })
  if (textCheck) {
    ElMessage.warning(textCheck)
    return
  }
  submitting.value = true
  try {
    await feedbackApi.create({ content: sanitizeText(fb.content), contact: sanitizeText(fb.contact) })
    ElMessage.success('提交成功，我们会尽快处理')
    fb.content = ''
    fb.contact = ''
    loadFeedback()
  } finally {
    submitting.value = false
  }
}

function needLogin() {
  window.dispatchEvent(new CustomEvent('open-auth'))
}

// 后端返回秒级时间戳
function formatTime(t) {
  if (!t) return ''
  if (typeof t === 'number') return dayjs(t * 1000).format('YYYY-MM-DD HH:mm')
  return String(t).slice(0, 16).replace('T', ' ')
}

watch(() => userStore.isLogin, () => loadFeedback())

onMounted(loadFeedback)
</script>

<style scoped>
.tips-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
}

.tip-card {
  display: flex;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--brand-line);
  border-radius: 12px;
}

.tip-icon {
  width: 26px;
  height: 26px;
  flex: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
}

.tip-title {
  font-weight: 600;
  margin-bottom: 4px;
}

.tip-desc {
  font-size: 13px;
  line-height: 1.7;
}

.faq-group + .faq-group {
  margin-top: 18px;
}

.faq-group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--brand-title, #1f2329);
}

.faq-group-icon {
  width: 22px;
  height: 22px;
  flex: none;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
}

.faq-group :deep(.el-collapse-item__header) {
  height: auto;
  min-height: 44px;
  line-height: 1.6;
  padding: 8px 0;
}

.faq-group :deep(.el-collapse-item__content) {
  padding-bottom: 14px;
}

.faq-q {
  font-weight: 600;
}

.faq-a {
  font-size: 13px;
  line-height: 1.9;
  color: var(--brand-sub);
}

.fb-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.limit-tip {
  margin-top: 8px;
  font-size: 12px;
}

.limit-tip .wait {
  color: var(--brand-danger);
}

.flex-spacer {
  flex: 1;
}

.fb-list {
  margin-top: 24px;
  border-top: 1px solid var(--brand-line);
  padding-top: 16px;
}

.list-title {
  font-weight: 600;
  margin-bottom: 12px;
}

.empty {
  font-size: 13px;
  padding: 8px 0;
}

.fb-item {
  padding: 12px 14px;
  margin-bottom: 10px;
  border-radius: 10px;
  background: #fafbfc;
}

.fb-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.fb-name {
  font-weight: 600;
  font-size: 13px;
}

.fb-time {
  margin-left: auto;
  font-size: 12px;
}

.fb-content {
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
}

.fb-reply {
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  font-size: 13px;
  line-height: 1.7;
}
</style>
