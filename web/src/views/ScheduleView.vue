<template>
  <div class="page">
    <template v-if="mainVisible">
      <section class="card" ref="boardRef">
        <div class="card-title">
          课表
          <span class="sub">根据课程的补课周期与每周时段自动生成，默认当前自然周，可自定义日期区间（可跨周）</span>
        </div>

        <!-- 试用 / 过期提示条 -->
        <div v-if="trialBanner" class="trial-banner" :class="trialBanner.type">
          <span>{{ trialBanner.text }}</span>
          <el-button v-if="trialStore.isGuest" type="primary" size="small" round @click="trialStore.openRegisterGuide">
            注册保存数据
          </el-button>
          <el-button v-else type="warning" size="small" round @click="needMember">去续费</el-button>
        </div>

        <div class="toolbar">
          <div class="toolbar-left">
            <el-button-group class="week-nav">
              <el-button @click="shiftWeek(-1)">
                <el-icon class="btn-ico"><ArrowLeft /></el-icon>上周
              </el-button>
              <el-button class="btn-now" @click="thisWeek">本周</el-button>
              <el-button @click="shiftWeek(1)">
                下周<el-icon class="btn-ico-r"><ArrowRight /></el-icon>
              </el-button>
            </el-button-group>

            <div class="date-pickers">
              <el-date-picker
                v-model="rangeStart"
                type="date"
                value-format="YYYY-MM-DD"
                placeholder="开始日期"
                :clearable="false"
                style="width: 140px"
                @change="onPickStart"
              />
              <span class="date-sep">至</span>
              <el-date-picker
                v-model="rangeEnd"
                type="date"
                value-format="YYYY-MM-DD"
                placeholder="结束日期"
                :clearable="false"
                style="width: 140px"
                @change="onPickEnd"
              />
            </div>
          </div>

          <!-- 课表导出：会员可导出带水印图片；免注册试用提示注册解锁 -->
          <el-button class="export-btn" type="primary" plain round :icon="Download" :loading="exporting" @click="onExport">
            导出图片
          </el-button>
        </div>

        <!-- 图例：独立一行，弱化展示 -->
        <div class="legend-row">
          <div class="legend">
            <span v-for="(v, k) in timeStateMap" :key="k" class="legend-item">
              <i class="dot" :style="{ background: timeColor[k] }" />{{ v }}
            </span>
            <span class="legend-item"><i class="dot" style="background: #c9ced8" />已停课</span>
            <span class="legend-item"><i class="dot" style="background: #7a5af8" />调课 / 加课</span>
          </div>
        </div>

        <div class="summary">
          <div class="sum-item"><b>{{ data.summary?.count || 0 }}</b><span>区间课时</span></div>
          <div class="sum-item"><b>{{ data.summary?.hours || 0 }}</b><span>总课时</span></div>
          <div class="sum-item">
            <b :title="moneyDetail">¥{{ data.summary?.income || 0 }}</b>
            <span>{{ feeStatLabel }}</span>
          </div>
          <div v-if="data.summary?.adjusted" class="sum-item"><b>{{ data.summary.adjusted }}</b><span>停课 / 调出</span></div>
        </div>

        <!-- 时段冲突告警：同一天同一时间段存在多节课时给出明显提示 -->
        <div v-if="conflictCount" class="conflict-bar">
          <el-icon class="conflict-icon"><WarningFilled /></el-icon>
          <span>检测到 <b>{{ conflictCount }}</b> 个时段存在课程重叠，请及时调整时间安排</span>
        </div>

        <div v-loading="loading" class="sched-wrap">
          <div class="sched" :style="{ minWidth: Math.max(640, dayCount * 97) + 'px' }">
            <div class="sched-head" :style="{ gridTemplateColumns: headGrid }">
              <div class="corner">时间</div>
              <div v-for="d in data.days" :key="d.date" class="day-head" :class="{ today: d.isToday }">
                <el-tooltip
                  placement="bottom"
                  effect="light"
                  :show-after="120"
                  :disabled="!dayLessons(d.date).length && !dayAdjusts(d.date).length"
                >
                  <template #content>
                    <div class="tip-box">
                      <div class="tip-title">{{ d.date }} {{ weekdayName(d.weekday) }}</div>
                      <div v-if="dayLessons(d.date).length" class="tip-line">
                        共 {{ dayLessons(d.date).length }} 节课
                      </div>
                      <div v-for="ev in dayLessons(d.date)" :key="ev.key" class="tip-line">
                        {{ ev.startTime }}-{{ ev.endTime }} {{ ev.grade || '年级' }}·{{ ev.studentName || '学生' }}
                        <span v-if="ev.subject">（{{ ev.subject }}）</span>
                        <span v-if="ev.address" class="muted"> {{ ev.address }}</span>
                        <span v-if="ev.adjustType" class="tip-adjust"> {{ adjustTag(ev.adjustType) }}</span>
                      </div>
                      <div v-for="ev in dayAdjusts(d.date)" :key="ev.key" class="tip-line muted">
                        {{ ev.startTime }}-{{ ev.endTime }} {{ adjustTip(ev) }}
                      </div>
                    </div>
                  </template>

                  <div class="day-info">
                    <div class="d-label">
                      {{ d.label }}
                      <i v-if="dayLessons(d.date).length" class="day-dot" />
                    </div>
                    <div class="d-date">{{ d.date.slice(5) }}</div>
                  </div>
                </el-tooltip>
              </div>
            </div>

            <div class="sched-body">
              <div class="time-col">
                <div v-for="h in data.hours" :key="h" class="time-cell" :style="{ height: hourHeight + 'px' }">
                  {{ pad(h) }}:00
                </div>
              </div>

              <div class="grid-col" :style="{ height: (data.hours?.length || 0) * hourHeight + 'px' }">
                <div class="hours-bg">
                  <div v-for="h in data.hours" :key="h" class="h-line" :style="{ height: hourHeight + 'px' }" />
                </div>

                <div class="cols" :style="{ gridTemplateColumns: dayGrid }">
                  <div v-for="d in data.days" :key="d.date" class="col">
                    <el-tooltip
                      v-for="ev in eventsByDay[d.date] || []"
                      :key="ev.key"
                      placement="right"
                      effect="light"
                      :show-after="150"
                    >
                      <template #content>
                        <div class="tip-box">
                          <div class="tip-title">{{ ev.grade || '未填年级' }} · {{ ev.studentName || '未填姓名' }}</div>
                          <div class="tip-line">
                            时间：{{ ev.date }} {{ ev.startTime }}-{{ ev.endTime }}
                          </div>
                          <div v-if="adjustTip(ev)" class="tip-line tip-adjust">{{ adjustTip(ev) }}</div>
                          <div class="tip-line">科目：{{ ev.subject || '-' }}</div>
                          <div class="tip-line">地址：{{ ev.address || '-' }}</div>
                          <!-- 费用口径：按时薪计费显示时薪，其余（按次 / 按总费用分摊）显示单节费用 -->
                          <div v-if="ev.hourlyRate > 0" class="tip-line">时薪：¥{{ ev.hourlyRate }}/时</div>
                          <div v-else-if="ev.income" class="tip-line">单节费用：¥{{ ev.income }}</div>
                          <div class="tip-line">课程状态：{{ statusMap[ev.status] || '-' }}</div>
                          <div v-if="ev.content" class="tip-line">内容：{{ ev.content }}</div>
                          <div v-if="ev.remark" class="tip-line tip-remark">备注：{{ ev.remark }}</div>
                        </div>
                      </template>

                      <div
                        class="event"
                        :class="[timeClass(ev.timeState), adjustClass(ev.adjustType), { compact: isCompact(ev), 'is-conflict': ev.conflict }]"
                        :style="eventStyle(ev)"
                        @click="openDetail(ev)"
                      >
                        <div class="ev-time">
                          {{ ev.startTime }}-{{ ev.endTime }}
                          <el-icon v-if="ev.conflict" class="conflict-mini"><WarningFilled /></el-icon>
                        </div>
                        <div class="ev-main">
                          <span v-if="ev.adjustType === 'movedIn'" class="tag tag-move">调</span>
                          <span v-if="ev.adjustType === 'extra'" class="tag tag-extra">加</span>
                          {{ ev.grade || '年级' }}·{{ ev.studentName || '学生' }}
                          <i v-if="ev.remarkFlag" class="red-dot" />
                        </div>
                        <div class="ev-sub">
                          <span v-if="ev.adjustType === 'movedIn' && ev.originDate" class="ev-adjust">原 {{ ev.originDate.slice(5) }}</span>
                          <span v-else-if="ev.adjustType === 'canceled'" class="ev-adjust">本次停课</span>
                          <span v-if="ev.subject">{{ ev.subject }}</span>
                          <span v-if="ev.address" class="ev-addr">{{ ev.address }}</span>
                        </div>
                      </div>
                    </el-tooltip>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="sched-tip muted">
          本周全部课时均会展示：<b class="c-past">灰色</b>为历史课时、<b class="c-ongoing">橙色</b>为进行中、<b class="c-future">蓝色</b>为未开始；
          <b class="c-cancel">置灰删除线</b>为本次停课、<b class="c-adjust">紫色</b>为调课或临时加课。
          鼠标悬浮查看完整信息，带红点表示有特殊备注，点击卡片可调整本次课程。
        </div>
      </section>
    </template>

    <!-- 课程详情 -->
    <el-drawer v-model="detailVisible" title="课程详情" size="460px">
      <el-descriptions v-if="current" :column="1" border>
        <el-descriptions-item label="日期">{{ current.date }}</el-descriptions-item>
        <el-descriptions-item label="时间">{{ current.startTime }} - {{ current.endTime }}</el-descriptions-item>
        <el-descriptions-item label="年级">{{ current.grade || '-' }}</el-descriptions-item>
        <el-descriptions-item label="学生">{{ current.studentName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="科目">{{ current.subject || '-' }}</el-descriptions-item>
        <el-descriptions-item label="地址">{{ current.address || '-' }}</el-descriptions-item>
        <el-descriptions-item label="辅导内容">{{ current.content || '-' }}</el-descriptions-item>
        <el-descriptions-item v-if="current.hourlyRate > 0" label="时薪">¥{{ current.hourlyRate }} / 小时</el-descriptions-item>
        <el-descriptions-item v-if="current.income" label="本次课时费">¥{{ current.income }}</el-descriptions-item>
        <el-descriptions-item label="课程编号">{{ current.orderNo || '-' }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ statusMap[current.status] || '-' }}</el-descriptions-item>
        <el-descriptions-item label="本次调整">
          <span v-if="adjustTip(current)">{{ adjustTip(current) }}</span>
          <span v-else class="muted">无</span>
        </el-descriptions-item>
        <el-descriptions-item v-if="current.remark" label="备注">
          <span class="remark-text">{{ current.remark }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="current" class="drawer-section">
        <div class="sec-title">调课记录</div>
        <div v-if="!exceptions.length" class="muted small">暂无调课记录，可只调整某一次课程而不改动每周频次</div>
        <div v-for="ex in exceptions" :key="ex.id" class="ex-item">
          <div class="ex-chain">
            <div v-for="(s, i) in ex.history || []" :key="i" class="ex-main">
              <el-tag size="small" :type="exTagType(s.type)">{{ i === 0 ? stepTypeName(s.type, ex) : '继续调整' }}</el-tag>
              <span class="ex-text">
                {{ fmtStepTime(s.fromDate, s.fromStart, s.fromEnd) }} →
                {{ s.toDate ? fmtStepTime(s.toDate, s.toStart, s.toEnd) : '本次停课' }}
              </span>
            </div>
          </div>
          <div class="ex-foot">
            <span v-if="ex.note" class="muted small">{{ ex.note }}</span>
            <el-button link type="danger" size="small" @click="revokeException(ex)">
              {{ (ex.history?.length || 1) > 1 ? '回退到上一次课时' : '撤销调整' }}
            </el-button>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <el-button @click="$router.push('/orders')">查看课程安排</el-button>
          <el-button
            v-if="current?.timeState === 'past' && current?.lessonId"
            link
            type="danger"
            @click="deleteLesson"
            >删除课程</el-button
          >
          <el-tooltip :disabled="!adjustDisabledReason" :content="adjustDisabledReason" placement="top">
            <span>
              <el-button type="primary" :disabled="!!adjustDisabledReason" @click="openAdjust">调整课程</el-button>
            </span>
          </el-tooltip>
        </div>
      </template>
    </el-drawer>

    <!-- 单次课程调整 -->
    <el-dialog v-model="adjustVisible" title="调整本次课程" width="500px" class="adjust-dialog">
      <div v-if="current" class="adjust-origin">
        <span class="ao-badge">{{ current.adjustType ? '当前课时' : '原定课时' }}</span>
        <span class="ao-text">
          {{ current.date }}（{{ weekdayName(current.weekday) }}）
          <b>{{ current.startTime }}-{{ current.endTime }}</b>
        </span>
        <span class="ao-fee">¥{{ feeOf(current) }}</span>
      </div>
      <el-form label-width="88px" class="adjust-form">
        <el-form-item label="调整方式">
          <el-radio-group v-model="adjustForm.type">
            <el-radio value="move">调整时间</el-radio>
            <el-radio value="cancel">本次停课</el-radio>
            <el-radio value="extra">临时加课</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="needDate" :label="adjustForm.type === 'extra' ? '加课日期' : '调整后日期'">
          <el-date-picker
            v-model="adjustForm.newDate"
            type="date"
            value-format="YYYY-MM-DD"
            :clearable="false"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item v-if="needTime" label="上课时间">
          <div class="time-pair">
            <el-time-select
              v-model="adjustForm.newStart"
              start="06:00"
              step="00:30"
              end="23:30"
              placeholder="开始"
              class="time-pick"
            />
            <span class="time-sep">至</span>
            <el-time-select
              v-model="adjustForm.newEnd"
              start="06:30"
              step="00:30"
              end="23:59"
              placeholder="结束"
              class="time-pick"
            />
          </div>
        </el-form-item>
        <el-form-item v-if="needTime" label="本次课时费">
          <div class="fee-row">
            <el-input-number v-model="adjustForm.income" :min="0" :max="100000" :step="10" :precision="2" />
            <span class="muted small">元，默认为原课时费用</span>
          </div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="adjustForm.note"
            type="textarea"
            :rows="2"
            maxlength="100"
            show-word-limit
            placeholder="如：家长临时有事，本周六调到周日"
          />
        </el-form-item>
      </el-form>
      <div class="adjust-tip">
        调整时间可改到任意日期与时间（含当天仅改时间），仅影响本次课程，课程的每周固定时段保持不变；若该时段已有其它课程会给出提示。课时费仅对本节生效，其余课程仍按课程标准计费。调整后可继续调整，调课记录保留完整路径，可回退到上一次调整。
      </div>
      <template #footer>
        <el-button @click="adjustVisible = false">取消</el-button>
        <el-button type="primary" :loading="adjusting" @click="submitAdjust">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight, Download, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import { scheduleApi, lessonApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useTrialStore } from '@/stores/trial'
import { exportElementToPng } from '@/utils/exportSchedule'
import { sanitizeText, validateText } from '@/utils/text'

const userStore = useUserStore()
const trialStore = useTrialStore()
const hourHeight = 56

// 金额口径按注册身份决定的收 / 支方向展示（课程不再单独携带方向）：
// 仅收入 → 预计收入；仅开支 → 预计支出；历史数据两种都有 → 中性的「预计金额」（悬停查看明细）。
const incomeAmount = computed(() => Number(data.summary?.incomeAmount || 0))
const expenseAmount = computed(() => Number(data.summary?.expenseAmount || 0))
const hasMixedDirections = computed(() => incomeAmount.value > 0 && expenseAmount.value > 0)
const feeStatLabel = computed(() => {
  if (hasMixedDirections.value) return '预计金额'
  return expenseAmount.value > 0 ? '预计支出' : '预计收入'
})
const moneyDetail = computed(() =>
  hasMixedDirections.value
    ? `收入 ¥${incomeAmount.value} ｜ 开支 ¥${expenseAmount.value}`
    : feeStatLabel.value
)

// 订单状态（与订单管理保持一致：未开始 / 进行中 / 已结束）
const statusMap = { pending: '未开始', running: '进行中', finished: '已结束' }

// 课时按「相对当前时间」着色：历史课时 / 进行中 / 未开始
const timeStateMap = { past: '历史课时', ongoing: '进行中', future: '未开始' }
const timeColor = {
  past: '#8a94a6',
  ongoing: '#ff7a45',
  future: '#2f6fed'
}
const weekdayNames = ['', '周一', '周二', '周三', '周四', '周五', '周六', '周日']

const loading = ref(false)
// 自定义日期区间（可跨周）：默认当前自然周
const rangeStart = ref('')
const rangeEnd = ref('')
const detailVisible = ref(false)
const current = ref(null)
const exceptions = ref([])

// 单次课程调整（改期 / 停课 / 改时间 / 加课）
const adjustVisible = ref(false)
const adjusting = ref(false)
const adjustForm = reactive({
  type: 'move',
  newDate: '',
  newStart: '',
  newEnd: '',
  income: 0,
  note: ''
})

const data = reactive({
  rangeStart: '',
  rangeEnd: '',
  weekStart: '',
  weekEnd: '',
  monthLabel: '',
  days: [],
  hours: [],
  dayStartHour: 6,
  dayEndHour: 23,
  events: [],
  summary: {}
})

/* ---------------- 使用模式：正常会员 / 免注册试用 / 会员过期只读 ---------------- */
// 三种模式共用课表：登录会员正常使用；未登录读取本机试用数据；会员过期只读查看
const mainVisible = computed(() => true)
const readonlyMode = computed(() => userStore.isLogin && !userStore.isMember)

// 顶部提示条文案
const trialBanner = computed(() => {
  if (trialStore.isGuest) {
    return {
      type: 'trial',
      text: '试用模式：您的排课数据暂时保存在本机浏览器，注册即可永久保存并随时查看。'
    }
  }
  if (readonlyMode.value) {
    return {
      type: 'expired',
      text: '会员已过期：课表与已录入的课程仍可查看，调课等功能需续费后使用。'
    }
  }
  return null
})

// 调课按钮禁用原因：已结束、会员过期、免注册试用均不可调课
const adjustDisabledReason = computed(() => {
  if (current.value?.status === 'finished') return '课程已结束，课次不可再调整'
  if (trialStore.isGuest) return '注册后解锁此功能'
  if (readonlyMode.value) return '会员已过期，续费后可继续调课'
  return ''
})

/* ---------------- 课表导出（图片，带 logo 水印） ---------------- */
const boardRef = ref(null)
const exporting = ref(false)
async function onExport() {
  if (trialStore.isGuest) {
    trialStore.requireAuth('导出课表')
    return
  }
  if (readonlyMode.value) {
    ElMessage.warning('会员已过期，续费后可继续导出课表')
    return
  }
  exporting.value = true
  try {
    await exportElementToPng(boardRef.value, `${data.rangeStart || '课表'}_${data.rangeEnd || ''}课表`)
    ElMessage.success('课表已导出为图片')
  } catch {
    ElMessage.error('导出失败，请稍后重试')
  } finally {
    exporting.value = false
  }
}

// x 轴列数随所选日期区间变化（可跨周）
const dayCount = computed(() => data.days?.length || 7)
const headGrid = computed(() => `64px repeat(${dayCount.value}, 1fr)`)
const dayGrid = computed(() => `repeat(${dayCount.value}, 1fr)`)

// 统计存在时间重叠的冲突课程组数：按「日期 + 开始时间」聚合，避免同一组多节课重复计数
const conflictCount = computed(() => {
  const keys = new Set()
  for (const date of Object.keys(eventsByDay.value)) {
    for (const ev of eventsByDay.value[date]) {
      if (ev.conflict) keys.add(`${date}-${ev.startTime}`)
    }
  }
  return keys.size
})

// 改期与改时间已合并为「调整时间」：日期与时间都可改为任意值
const needDate = computed(() => adjustForm.type === 'move' || adjustForm.type === 'extra')
const needTime = computed(() => adjustForm.type === 'move' || adjustForm.type === 'extra')

// 当前课次所属订单是否已结束（已结束订单不可再调整课程，与后端规则一致）
const orderFinished = computed(() => current.value?.status === 'finished')

const eventsByDay = computed(() => {
  const map = {}
  for (const ev of data.events || []) {
    if (!map[ev.date]) map[ev.date] = []
    map[ev.date].push(ev)
  }
  // 同一天的课程按开始时间排序并做分列布局，重叠课程并排显示，避免互相遮挡
  for (const date of Object.keys(map)) {
    map[date] = layoutEvents(map[date])
  }
  return map
})

// layoutEvents 把一天内的课程按重叠情况分配到不同列：
// 先按开始时间排序，再把时间上连通重叠的一组课程横向平分宽度。
function layoutEvents(list) {
  const sorted = [...list].sort((a, b) => a.startMinute - b.startMinute || a.endMinute - b.endMinute)
  const result = []
  let group = []
  let groupEnd = -1

  const flush = () => {
    if (!group.length) return
    const laneEnds = []
    for (const item of group) {
      let idx = laneEnds.findIndex(end => end <= item.ev.startMinute)
      if (idx === -1) {
        laneEnds.push(item.ev.endMinute)
        idx = laneEnds.length - 1
      } else {
        laneEnds[idx] = item.ev.endMinute
      }
      item.lane = idx
    }
    const lanes = laneEnds.length
    const hasConflict = lanes > 1
    for (const item of group) {
      result.push({ ...item.ev, lane: item.lane, lanes, conflict: hasConflict })
    }
    group = []
    groupEnd = -1
  }

  for (const ev of sorted) {
    if (group.length && ev.startMinute >= groupEnd) flush()
    group.push({ ev })
    groupEnd = Math.max(groupEnd, ev.endMinute)
  }
  flush()
  return result
}

function pad(n) {
  return String(n).padStart(2, '0')
}

function eventStyle(ev) {
  const startRange = data.dayStartHour * 60
  const endRange = (data.dayEndHour + 1) * 60
  const top = Math.max(ev.startMinute, startRange)
  const bottom = Math.min(ev.endMinute, endRange)
  const height = Math.max(bottom - top, 30)
  const lanes = ev.lanes || 1
  const lane = ev.lane || 0
  const w = 100 / lanes
  return {
    top: ((top - startRange) / 60) * hourHeight + 'px',
    height: height / 60 * hourHeight - 3 + 'px',
    left: `calc(${lane * w}% + 3px)`,
    width: `calc(${w}% - 6px)`
  }
}

function timeClass(state) {
  return `tm-${state || 'future'}`
}

// isCompact 卡片过矮（短课时）或过窄（同日多节并排）时精简内容，避免文字被裁切。
function isCompact(ev) {
  const startRange = data.dayStartHour * 60
  const endRange = (data.dayEndHour + 1) * 60
  const top = Math.max(ev.startMinute, startRange)
  const bottom = Math.min(ev.endMinute, endRange)
  const px = (Math.max(bottom - top, 30) / 60) * hourHeight - 3
  return px < 46 || (ev.lanes || 1) >= 3
}

function adjustClass(type) {
  return `ad-${type || 'normal'}`
}

function weekdayName(wd) {
  return weekdayNames[wd] || ''
}

// adjustTip 课程的单次调整说明（悬浮提示与详情抽屉复用）。
function adjustTip(ev) {
  if (!ev || !ev.adjustType) return ''
  switch (ev.adjustType) {
    case 'canceled':
      return `本次停课${ev.adjustNote ? '：' + ev.adjustNote : ''}`
    case 'movedOut':
      return `已调至 ${ev.movedToDate || ''}${ev.movedToStart ? ' ' + ev.movedToStart : ''}`
    case 'movedIn':
      return `由 ${fmtStepTime(ev.originDate, ev.originStart, ev.originEnd) || ev.originDate || ''} 调整而来${ev.adjustNote ? '：' + ev.adjustNote : ''}`
    case 'time':
      return `本次时间已调整${ev.adjustNote ? '：' + ev.adjustNote : ''}`
    case 'extra':
      return `临时加课${ev.adjustNote ? '：' + ev.adjustNote : ''}`
    default:
      return ''
  }
}

// 调课记录中的完整课时时间：日期 + 开始-结束
function fmtStepTime(date, start, end) {
  let t = date || ''
  if (start) t += ` ${start}${end ? '-' + end : ''}`
  return t
}

function exTagType(type) {
  return { cancel: 'info', move: 'warning', time: 'primary', extra: 'success' }[type] || 'info'
}

// 调课路径首步的类型名（历史记录的 typeName 为当前生效类型，可能已随继续调整而变化）
function stepTypeName(type, ex) {
  return { cancel: '停课', move: '改期', time: '改时间', extra: '临时加课' }[type] || ex?.typeName || '调整'
}

const adjustTagMap = {
  canceled: '停课',
  movedOut: '已调出',
  movedIn: '调课',
  time: '改时间',
  extra: '加课'
}

function adjustTag(type) {
  return adjustTagMap[type] || ''
}

// dayLessons 某天实际要上的课（停课 / 已调出的占位不算）
function dayLessons(date) {
  return (eventsByDay.value[date] || []).filter(
    (ev) => ev.adjustType !== 'canceled' && ev.adjustType !== 'movedOut'
  )
}

// dayAdjusts 某天的变动提示（停课 / 已调出）
function dayAdjusts(date) {
  return (eventsByDay.value[date] || []).filter(
    (ev) => ev.adjustType === 'canceled' || ev.adjustType === 'movedOut'
  )
}

async function load(range) {
  loading.value = true
  try {
    const params = {}
    if (range && range.start && range.end) {
      params.start = range.start
      params.end = range.end
    }
    // 免注册试用：直接由本机试用数据生成课表（不传区间时默认当前自然周）
    if (trialStore.isGuest) {
      const monday = dayjs().startOf('week').add(1, 'day')
      const s = params.start || monday.format('YYYY-MM-DD')
      const e = params.end || monday.add(6, 'day').format('YYYY-MM-DD')
      const d = trialStore.weekSchedule(s, e)
      data.rangeStart = s
      data.rangeEnd = e
      data.weekStart = s
      data.weekEnd = e
      data.monthLabel = ''
      data.days = d.days || []
      data.hours = d.hours || []
      data.dayStartHour = 6
      data.dayEndHour = 23
      data.events = d.events || []
      data.summary = d.summary || {}
      rangeStart.value = s
      rangeEnd.value = e
      return
    }
    const res = await scheduleApi.week(params)
    const d = res.data || {}
    data.rangeStart = d.rangeStart || d.weekStart || ''
    data.rangeEnd = d.rangeEnd || d.weekEnd || ''
    data.weekStart = data.rangeStart
    data.weekEnd = data.rangeEnd
    data.monthLabel = d.monthLabel
    data.days = d.days || []
    data.hours = d.hours || []
    data.dayStartHour = d.dayStartHour ?? 6
    data.dayEndHour = d.dayEndHour ?? 23
    data.events = d.events || []
    data.summary = d.summary || {}
    // 同步日期控件显示（后端可能按上限截断区间）
    rangeStart.value = data.rangeStart
    rangeEnd.value = data.rangeEnd
  } finally {
    loading.value = false
  }
}

// 上周 / 下周：整个区间按周平移，保持区间长度不变
function shiftWeek(offset) {
  const days = dayCount.value
  const start = data.rangeStart ? dayjs(data.rangeStart).add(offset * 7, 'day') : dayjs()
  const end = start.add(days - 1, 'day')
  load({ start: start.format('YYYY-MM-DD'), end: end.format('YYYY-MM-DD') })
}

function thisWeek() {
  load(null)
}

// 改动起止日期即按新区间加载；开始晚于结束（或反之）时自动对齐
function onPickStart(val) {
  if (!val) return
  if (rangeEnd.value && val > rangeEnd.value) rangeEnd.value = val
  load({ start: rangeStart.value, end: rangeEnd.value })
}

function onPickEnd(val) {
  if (!val) return
  if (rangeStart.value && val < rangeStart.value) rangeStart.value = val
  load({ start: rangeStart.value, end: rangeEnd.value })
}

function openDetail(ev) {
  current.value = ev
  detailVisible.value = true
  loadExceptions(ev.orderId)
}

async function loadExceptions(orderId) {
  // 免注册试用：无服务端调整记录
  if (!orderId || trialStore.isGuest) {
    exceptions.value = []
    return
  }
  try {
    const res = await lessonApi.exceptions(orderId)
    exceptions.value = res.data?.list || []
  } catch {
    exceptions.value = []
  }
}

// 本次课时费：例外指定过金额用之，否则为后端按订单标准计算的金额
function feeOf(ev) {
  return Number(ev?.income || 0).toFixed(2)
}

function openAdjust() {
  if (!current.value) return
  // 免注册试用 / 会员过期：禁止新的调课操作
  if (trialStore.isGuest) {
    trialStore.requireAuth('调课调整')
    return
  }
  if (readonlyMode.value) {
    ElMessage.warning('会员已过期，续费后可继续调课')
    return
  }
  const ev = current.value
  adjustForm.type =
    ev.adjustType === 'canceled' ? 'cancel' : ev.adjustType === 'extra' ? 'extra' : 'move'
  adjustForm.newDate = ev.movedToDate || ev.date
  adjustForm.newStart = ev.startTime || ''
  adjustForm.newEnd = ev.endTime || ''
  adjustForm.income = Number(ev.income || 0)
  adjustForm.note = ev.adjustNote || ''
  adjustVisible.value = true
}

async function submitAdjust() {
  const ev = current.value
  if (!ev) return
  if (needDate.value && !adjustForm.newDate) {
    ElMessage.warning('请选择日期')
    return
  }
  adjusting.value = true
  try {
    const bannedMsg = validateText(adjustForm.note, '备注', { max: 100 })
    if (bannedMsg) {
      ElMessage.warning(bannedMsg)
      return
    }
    adjustForm.note = sanitizeText(adjustForm.note)
    // 已经调入的课：以「原定日期」定位例外（调课路径链头），否则会重复生成一条调整记录
    const sourceDate = ev.adjustType === 'movedIn' && ev.originDate ? ev.originDate : ev.date
    // 临时加课没有原定课时，调整即「删旧建新」
    if (ev.adjustType === 'extra' && ev.exceptionId) {
      await lessonApi.removeException(ev.exceptionId)
    }
    // 合并后的调整时间：起止时间由「结束时间」表达，日期与时间至少调整一项
    const toMin = (t) => {
      const [h, m] = String(t || '').split(':').map(Number)
      return (h || 0) * 60 + (m || 0)
    }
    const startChanged = adjustForm.newStart && adjustForm.newStart !== ev.startTime
    const endChanged = adjustForm.newEnd && adjustForm.newEnd !== ev.endTime
    const dateChanged = adjustForm.newDate && adjustForm.newDate !== ev.date
    if (adjustForm.type === 'move' && !dateChanged && !startChanged && !endChanged) {
      ElMessage.warning('请调整日期或上课时间')
      return
    }
    if (needTime.value && adjustForm.newStart && adjustForm.newEnd && toMin(adjustForm.newEnd) <= toMin(adjustForm.newStart)) {
      ElMessage.warning('结束时间需晚于开始时间')
      return
    }
    const durMin =
      needTime.value && adjustForm.newStart && adjustForm.newEnd && toMin(adjustForm.newEnd) > toMin(adjustForm.newStart)
        ? toMin(adjustForm.newEnd) - toMin(adjustForm.newStart)
        : 0
    const payload = {
      orderId: ev.orderId,
      sourceDate,
      slotIndex: ev.slotIndex ?? 0,
      type: adjustForm.type,
      newDate: needDate.value ? adjustForm.newDate : '',
      newStart: needTime.value && startChanged ? adjustForm.newStart : '',
      newEnd: needTime.value ? adjustForm.newEnd : '',
      durationMinutes: durMin,
      income: needTime.value ? Number(adjustForm.income || 0) : 0,
      note: adjustForm.note
    }
    const res = await lessonApi.adjust(payload)
    const conflicts = res.data?.conflicts || []
    ElMessage.success('已保存调整')
    if (conflicts.length) {
      const names = conflicts.map(c => c.studentName || c.orderNo || '其它课程').join('、')
      ElMessage.warning(`该时段与 ${names} 的课程重叠，请确认时间安排`)
    }
    if (payload.newDate && (payload.newDate < data.rangeStart || payload.newDate > data.rangeEnd)) {
      ElMessage.info(`已调至 ${payload.newDate}，不在当前显示的日期区间内`)
    }
    adjustVisible.value = false
    detailVisible.value = false
    await load({ start: data.rangeStart, end: data.rangeEnd })
  } finally {
    adjusting.value = false
  }
}

async function revokeException(ex) {
  if (trialStore.isGuest) {
    trialStore.requireAuth('调课撤销')
    return
  }
  if (readonlyMode.value) {
    ElMessage.warning('会员已过期，续费后可继续调课')
    return
  }
  const steps = ex.history || []
  const multi = steps.length > 1
  // 撤销 / 回退后恢复到的课时：多步=上一次调整后的课时（末步的调整前位置），单步=原定课时（首步的调整前位置）
  const targetStep = multi ? steps[steps.length - 1] : steps[0]
  const target = targetStep ? fmtStepTime(targetStep.fromDate, targetStep.fromStart, targetStep.fromEnd) : ''
  try {
    await ElMessageBox.confirm(
      multi
        ? `回退后该次课程恢复为上一次调整的课时：${target}，确认回退？`
        : `撤销后该次课程恢复为原定课时：${target}，确认撤销？`,
      multi ? '回退调整' : '撤销调整',
      { type: 'warning', confirmButtonText: multi ? '回退' : '撤销', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  await lessonApi.removeException(ex.id)
  ElMessage.success(multi ? '已回退到上一次调整的课时' : '已撤销，课程恢复为原定课时')
  if (current.value) await loadExceptions(current.value.orderId)
  await load({ start: data.rangeStart, end: data.rangeEnd })
}

// 删除本节课：仅针对已物化的历史课次（老师可能未上课），软删除后从课表与统计中隐藏
async function deleteLesson() {
  const ev = current.value
  if (!ev?.lessonId) return
  if (trialStore.isGuest) {
    trialStore.requireAuth('删除课次')
    return
  }
  if (readonlyMode.value) {
    ElMessage.warning('会员已过期，续费后可继续调课')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确认删除 ${ev.date} ${ev.startTime}-${ev.endTime} 这节课？删除后将从课表与统计中移除。`,
      '删除课程',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
    )
  } catch (e) {
    return
  }
  try {
    await lessonApi.remove(ev.lessonId)
    ElMessage.success('已删除该课次')
    detailVisible.value = false
    load(null)
  } catch (e) {
    // 错误提示由拦截器处理
  }
}

function needMember() {
  window.dispatchEvent(new CustomEvent('open-profile'))
}

// 三种模式统一加载：会员读服务端；免注册试用读本机；过期只读查看
onMounted(() => {
  load(null)
})

// 登录态或会员状态变化后重新加载；免注册试用时随本地数据变化刷新
watch(
  () => [userStore.isLogin, userStore.isMember, trialStore.orderCount],
  () => load(null)
)
</script>

<style scoped>
/* 试用 / 过期提示条 */
.trial-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 13px;
}

.trial-banner.trial {
  background: #f0f7ff;
  color: #2b6cb0;
  border: 1px solid #cfe4ff;
}

.trial-banner.expired {
  background: #fff9e6;
  color: #8a6d3b;
  border: 1px solid #f5e0a3;
}

.trial-banner span {
  flex: 1;
}

/* ===== 工具栏：周切换 + 日期区间（左）｜导出（右），图例独立一行 ===== */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  margin-bottom: 10px;
  padding: 10px 12px;
  border: 1px solid var(--brand-line, #e5e8ef);
  border-radius: 12px;
  background: linear-gradient(180deg, #fbfcfe 0%, #f7f9fc 100%);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

/* 周切换：胶囊分组按钮，圆角外凸、分割线弱化 */
.week-nav {
  border-radius: 999px;
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(31, 41, 61, 0.06);
}

.week-nav .el-button {
  padding: 0 14px;
  border: none;
  background: #fff;
  color: var(--brand-sub, #5b6472);
}

.week-nav .el-button + .el-button {
  border-left: 1px solid var(--brand-line, #eef0f5);
}

.week-nav .el-button:hover {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.week-nav .btn-now {
  font-weight: 600;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.btn-ico {
  margin-right: 4px;
}

.btn-ico-r {
  margin-left: 4px;
}

/* 日期区间：一体式容器，「至」作为内部分隔 */
.date-pickers {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border: 1px solid var(--brand-line, #e5e8ef);
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(31, 41, 61, 0.04);
  transition: border-color 0.2s, box-shadow 0.2s;
}

/* 聚焦 / 悬浮时整体高亮，内部输入框永远无边框 */
.date-pickers:focus-within {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}

.date-pickers :deep(.el-date-editor.el-input),
.date-pickers :deep(.el-date-editor.el-input__wrapper) {
  box-shadow: none;
}

.date-pickers :deep(.el-input__wrapper) {
  background: transparent;
  padding-left: 0;
  box-shadow: none !important;
}

.date-pickers :deep(.el-input__wrapper.is-focus),
.date-pickers :deep(.el-input__wrapper:hover) {
  box-shadow: none !important;
}

.date-pickers :deep(.el-input__inner) {
  cursor: pointer;
}

.date-sep {
  color: var(--brand-muted, #9aa3b2);
  font-size: 12px;
}

/* 导出按钮：工具栏最右侧主操作 */
.export-btn {
  font-weight: 600;
  box-shadow: 0 1px 2px rgba(31, 41, 61, 0.08);
}

/* 图例：独立弱化一行，避免与主操作抢视觉 */
.legend-row {
  display: flex;
  align-items: center;
  margin-bottom: 14px;
}

.legend {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--brand-sub, #5b6472);
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 9px;
  border-radius: 999px;
  background: #fff;
  border: 1px solid var(--brand-line, #eef0f5);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.9) inset;
}

/* ===== 统计条 ===== */
.summary {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 10px 6px;
  margin-bottom: 14px;
  border-radius: 12px;
  background: var(--el-color-primary-light-9);
}

.sum-item {
  flex: 1;
  min-width: 120px;
  text-align: center;
  padding: 2px 12px;
  border-right: 1px solid rgba(47, 111, 237, 0.14);
}

.sum-item:last-child {
  border-right: none;
}

.sum-item b {
  display: block;
  font-size: 20px;
  font-weight: 700;
  color: var(--el-color-primary-dark-2);
  font-variant-numeric: tabular-nums;
}

.sum-item span {
  font-size: 12px;
  color: var(--brand-muted);
}

/* 小屏：统计项两行排布，去掉中间分隔线避免错位 */
@media (max-width: 640px) {
  .summary {
    flex-wrap: wrap;
    gap: 8px 0;
    padding: 12px 6px;
  }

  .sum-item {
    flex: 1 1 50%;
    border-right: none;
  }

  .toolbar {
    padding: 10px;
  }
}

.sched-wrap {
  overflow-x: auto;
  border: 1px solid var(--brand-line);
  border-radius: 12px;
}

.sched {
  width: 100%;
}

.sched-head {
  display: grid;
  grid-template-columns: 64px repeat(7, 1fr);
  border-bottom: 1px solid var(--brand-line);
  background: #fafbfc;
  position: sticky;
  top: 0;
  z-index: 2;
}

.corner {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--brand-muted);
}

.day-head {
  padding: 10px 4px;
  text-align: center;
  border-left: 1px solid var(--brand-line);
}

.day-head.today {
  background: var(--el-color-primary-light-9);
}

.day-info {
  cursor: default;
}

.d-label {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
  font-size: 13px;
}

/* 当天有课：日期标题红点提醒 */
.day-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--brand-danger);
  box-shadow: 0 0 0 2px rgba(227, 77, 89, 0.18);
}

.d-date {
  font-size: 12px;
  color: var(--brand-muted);
}

.sched-body {
  display: flex;
}

.time-col {
  width: 64px;
  flex: none;
}

.time-cell {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 2px;
  font-size: 12px;
  color: var(--brand-muted);
  border-top: 1px dashed var(--brand-line);
}

.grid-col {
  position: relative;
  flex: 1;
}

.hours-bg {
  position: absolute;
  inset: 0;
}

.h-line {
  border-top: 1px dashed var(--brand-line);
}

.cols {
  position: absolute;
  inset: 0;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.col {
  position: relative;
  border-left: 1px solid var(--brand-line);
}

.event {
  position: absolute;
  box-sizing: border-box;
  padding: 4px 6px;
  border-radius: 6px;
  background: #fff;
  border: 1px solid var(--brand-line);
  border-left: 3px solid var(--brand-muted);
  overflow: hidden;
  cursor: pointer;
  transition: box-shadow 0.2s;
}

.event:hover {
  box-shadow: 0 4px 12px rgba(31, 35, 41, 0.12);
  z-index: 3;
}

/* 卡片过矮 / 过窄：精简内容，避免文字被裁切 */
.event.compact {
  padding: 2px 5px;
}

.event.compact .ev-sub {
  display: none;
}

.event.compact .ev-time {
  font-size: 10px;
  line-height: 1.3;
}

.event.compact .ev-main {
  font-size: 11px;
  line-height: 1.3;
}

/* 历史课时：灰色 */
.event.tm-past {
  border-left-color: #8a94a6;
  background: #f5f6f7;
  opacity: 0.85;
}

/* 进行中：品牌橙，高亮 */
.event.tm-ongoing {
  border-left-color: #ff7a45;
  background: #fff8f4;
  box-shadow: 0 0 0 1px rgba(255, 122, 69, 0.35);
}

/* 未开始：蓝色 */
.event.tm-future {
  border-left-color: #2f6fed;
  background: #f2f6ff;
}

/* 本次停课：置灰 + 删除线，不计课时 */
.event.ad-canceled {
  border-left-color: #c9ced8;
  background: #f2f3f5;
  color: #9aa1ad;
  text-decoration: line-through;
  opacity: 0.75;
}

/* 已调至其它日期：原位仅作提示 */
.event.ad-movedOut {
  border-left-color: #c9ced8;
  border-style: dashed;
  background: #f7f8fa;
  color: #9aa1ad;
}

/* 调课 / 临时加课：紫色标识 */
.event.ad-movedIn,
.event.ad-extra {
  border-left-color: #7a5af8;
  background: #f6f3ff;
}

.event.ad-time {
  border-left-color: #7a5af8;
}

.ev-adjust {
  display: inline-block;
  margin-right: 6px;
  padding: 0 4px;
  border-radius: 3px;
  background: rgba(122, 90, 248, 0.12);
  color: #7a5af8;
}

.tag {
  display: inline-block;
  margin-right: 4px;
  padding: 0 4px;
  border-radius: 3px;
  font-size: 10px;
  line-height: 16px;
  color: #fff;
}

.tag-move {
  background: #7a5af8;
}

.tag-extra {
  background: #18a058;
}

.tip-adjust {
  color: #7a5af8;
}

.ev-time {
  font-size: 11px;
  color: var(--brand-muted);
}

.ev-main {
  position: relative;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ev-sub {
  font-size: 11px;
  color: var(--brand-sub);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ev-addr {
  margin-left: 6px;
}

.red-dot {
  position: absolute;
  top: 2px;
  margin-left: 4px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--brand-danger);
  box-shadow: 0 0 0 2px rgba(227, 77, 89, 0.18);
}

.tip-box {
  max-width: 280px;
  font-size: 12px;
  line-height: 1.8;
}

.tip-title {
  font-weight: 600;
  margin-bottom: 4px;
}

.tip-remark {
  color: var(--brand-danger);
}

.sched-tip {
  margin-top: 12px;
  font-size: 12px;
  line-height: 1.8;
}

.c-past {
  color: #8a94a6;
}

.c-ongoing {
  color: #ff7a45;
}

.c-future {
  color: #2f6fed;
}

.remark-text {
  color: var(--brand-danger);
}

.week-range {
  font-size: 12px;
  color: var(--brand-muted);
}

/* 时间刻度与网格严格对齐：避免边框把行高撑开导致错位 */
.time-cell {
  box-sizing: border-box;
}

.h-line {
  box-sizing: border-box;
}

.drawer-section {
  margin-top: 18px;
}

.sec-title {
  margin-bottom: 10px;
  font-size: 14px;
  font-weight: 600;
}

.ex-item {
  padding: 8px 10px;
  margin-bottom: 8px;
  border: 1px solid var(--brand-line);
  border-radius: 8px;
}

.ex-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 完整调课路径：多次调整纵向排列，每步 from → to */
.ex-chain {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ex-text {
  font-size: 13px;
}

.ex-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 6px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

/* 调整弹窗：当前课时信息栏 */
.adjust-origin {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 10px 14px;
  background: linear-gradient(90deg, rgba(255, 122, 69, 0.08), rgba(255, 122, 69, 0.02));
  border: 1px solid rgba(255, 122, 69, 0.25);
  border-radius: 10px;
  font-size: 13px;
  color: var(--brand-sub);
}

.adjust-origin .ao-badge {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  background: #ff7a45;
  color: #fff;
  font-size: 12px;
  line-height: 18px;
}

.adjust-origin .ao-text b {
  color: var(--brand-text, #303133);
}

.adjust-origin .ao-fee {
  margin-left: auto;
  flex-shrink: 0;
  font-weight: 600;
  color: #ff7a45;
}

.adjust-form {
  margin-bottom: 6px;
}

.time-pair {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.time-pair .time-pick {
  flex: 1;
  min-width: 0;
}

.time-pair .time-sep {
  flex-shrink: 0;
  color: var(--brand-muted, #909399);
}

.fee-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.adjust-tip {
  padding: 10px 12px;
  background: #f7f8fa;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.7;
  color: #8a94a6;
}

.c-cancel {
  color: #c9ced8;
}

.c-adjust {
  color: #7a5af8;
}

/* 收支方向小胶囊：课程卡片内一眼区分收入 / 开支 */
.dir-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  margin-right: 4px;
  vertical-align: middle;
}

.dir-pill.dir-income {
  background: #2f6fed;
}

.dir-pill.dir-expense {
  background: #ff7a45;
}

/* 冲突告警：卡片红边框 + 警告图标 */
.event.is-conflict {
  border-color: var(--el-color-danger) !important;
  box-shadow: 0 0 0 1px rgba(227, 77, 89, 0.35);
}

.event.is-conflict:hover {
  box-shadow: 0 4px 12px rgba(227, 77, 89, 0.2), 0 0 0 1px rgba(227, 77, 89, 0.35);
}

.conflict-mini {
  position: absolute;
  right: 4px;
  top: 4px;
  color: var(--el-color-danger);
  font-size: 13px;
}

/* 全局冲突提示条 */
.conflict-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #fff0f0;
  border: 1px solid #ffd1d1;
  color: var(--el-color-danger);
  font-size: 13px;
}

.conflict-bar b {
  font-weight: 600;
}

.conflict-icon {
  font-size: 16px;
}

/* 悬浮提示中的方向标签 */
.tip-dir {
  display: inline-flex;
  align-items: center;
  padding: 0 5px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
  margin-left: 6px;
}

.tip-dir-income {
  background: #eaf1ff;
  color: #2f6fed;
}

.tip-dir-expense {
  background: #fff1e9;
  color: #ff7a45;
}
</style>
