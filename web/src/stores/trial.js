import { defineStore } from 'pinia'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'
import { orderApi } from '@/api'

// 免注册试用模式：
// - 未登录用户可直接使用排课功能，数据仅保存在本机浏览器（localStorage），临时有效；
// - 试用期间最多可添加 3 门课程；
// - 进阶功能（导出课表、费用统计、调课提醒等）提示注册解锁；
// - 注册成功后可把本地试用数据导入账号，永久保存。
const STORAGE_KEY = 'tutoring_trial_data'
export const TRIAL_COURSE_LIMIT = 3

function loadRaw() {
  try {
    const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    return raw && typeof raw === 'object' ? raw : {}
  } catch {
    return {}
  }
}

// 计费金额：按小时计费用时薪，整单计费按课次均摊
function incomeOf(order, durationMin) {
  if (!order) return 0
  const dur = Math.max(0, Number(durationMin) || 0)
  if (Number(order.hourlyRate) > 0) return Math.round(((Number(order.hourlyRate) * dur) / 60) * 100) / 100
  if (Number(order.totalAmount) > 0) {
    const planned = Number(order.totalLessons) || 0
    if (planned > 0) return Math.round((Number(order.totalAmount) / planned) * 100) / 100
  }
  return 0
}

function slotMinutes(slot) {
  const [sh, sm] = String(slot.startTime || '00:00').split(':').map(Number)
  const [eh, em] = String(slot.endTime || '00:00').split(':').map(Number)
  const start = (sh || 0) * 60 + (sm || 0)
  const end = (eh || 0) * 60 + (em || 0)
  return { start, end, duration: Math.max(0, end - start) }
}

export const useTrialStore = defineStore('trial', {
  state: () => {
    const raw = loadRaw()
    return {
      orders: Array.isArray(raw.orders) ? raw.orders : [],
      // 首次试用时间，用于后续试用期判断与提示文案
      startedAt: raw.startedAt || ''
    }
  },

  getters: {
    isGuest() {
      const userStore = useUserStore()
      return !userStore.isLogin
    },
    hasData: (state) => state.orders.length > 0,
    orderCount: (state) => state.orders.length,
    remaining: (state) => Math.max(0, TRIAL_COURSE_LIMIT - state.orders.length)
  },

  actions: {
    persist() {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ orders: this.orders, startedAt: this.startedAt || dayjs().format('YYYY-MM-DD') })
      )
    },
    load() {
      const raw = loadRaw()
      this.orders = Array.isArray(raw.orders) ? raw.orders : []
      this.startedAt = raw.startedAt || ''
    },

    // 进阶功能拦截：未注册一律提示「注册后解锁此功能」并弹出注册引导
    requireAuth(feature = '此功能') {
      ElMessage.warning(`${feature}为注册用户专享，注册后解锁此功能`)
      window.dispatchEvent(new CustomEvent('open-auth', { detail: { mode: 'register' } }))
    },

    // 试用引导：可带自定义说明文案
    openRegisterGuide() {
      window.dispatchEvent(
        new CustomEvent('open-auth', {
          detail: { mode: 'register', tip: '您的排课数据尚未保存，注册即可永久保存并随时查看。' }
        })
      )
    },

    ensureTrialStart() {
      if (!this.startedAt) {
        this.startedAt = dayjs().format('YYYY-MM-DD')
        this.persist()
      }
    },

    addOrder(form) {
      this.ensureTrialStart()
      const order = {
        ...JSON.parse(JSON.stringify(form)),
        id: Date.now(),
        orderNo: `TRIAL-${dayjs().format('YYYYMMDDHHmmss')}`,
        status: form.status || 'running',
        createdAt: dayjs().unix()
      }
      this.orders.unshift(order)
      this.persist()
      return order
    },
    updateOrder(id, form) {
      const idx = this.orders.findIndex((o) => o.id === id)
      if (idx === -1) return
      // 保留本地生成的元信息，仅覆盖表单字段
      const { id: _i, orderNo, createdAt, ...rest } = this.orders[idx]
      this.orders[idx] = { ...rest, ...JSON.parse(JSON.stringify(form)), id, orderNo, createdAt }
      this.persist()
    },
    removeOrder(id) {
      this.orders = this.orders.filter((o) => o.id !== id)
      this.persist()
    },
    changeStatus(id, status) {
      const o = this.orders.find((x) => x.id === id)
      if (!o) return
      o.status = status
      // 与后端口径一致：结课补记结束日期； future 转 running 从今天起算
      if (status === 'finished' && !o.endDate) o.endDate = dayjs().format('YYYY-MM-DD')
      if (status === 'running' && o.startDate && dayjs(o.startDate).isAfter(dayjs(), 'day')) {
        o.startDate = dayjs().format('YYYY-MM-DD')
      }
      this.persist()
    },

    // 课程进度：planned=理论总课次，done=已上课次（起止区间内到今天为止的课次）
    progressOf(order) {
      const planned = Number(order.totalLessons) || 0
      let done = 0
      const today = dayjs().format('YYYY-MM-DD')
      if (order.startDate && order.weeklySlots?.length) {
        let cursor = dayjs(order.startDate)
        const end = order.endDate && dayjs(order.endDate).isBefore(today) ? dayjs(order.endDate) : dayjs(today)
        while (cursor.isBefore(end) || cursor.isSame(end, 'day')) {
          const wd = cursor.day() === 0 ? 7 : cursor.day()
          if (order.weeklySlots.some((s) => Number(s.day) === wd)) done++
          cursor = cursor.add(1, 'day')
        }
      }
      return { done: Math.min(done, planned), planned }
    },

    // 周课表生成：与后端 /schedule 接口返回结构对齐（days / hours / events / summary）
    weekSchedule(start, end) {
      const s = dayjs(start)
      const e = dayjs(end)
      const today = dayjs().format('YYYY-MM-DD')
      const nowStr = dayjs().format('HH:mm')
      const weekdays = ['一', '二', '三', '四', '五', '六', '日']
      const days = []
      let cursor = s
      while (cursor.isBefore(e) || cursor.isSame(e, 'day')) {
        const date = cursor.format('YYYY-MM-DD')
        const wd = cursor.day() === 0 ? 7 : cursor.day()
        days.push({
          date,
          weekday: wd,
          label: `周${weekdays[wd - 1]}`,
          isToday: date === today
        })
        cursor = cursor.add(1, 'day')
      }

      const events = []
      const summary = {
        count: 0,
        hours: 0,
        income: 0,
        incomeAmount: 0,
        expenseAmount: 0,
        adjusted: 0
      }
      for (const order of this.orders) {
        if (order.status === 'voided') continue
        for (const slot of order.weeklySlots || []) {
          for (const day of days) {
            if (Number(slot.day) !== day.weekday) continue
            if (day.date < order.startDate) continue
            if (order.endDate && day.date > order.endDate) continue
            const { start: startMinute, end: endMinute, duration } = slotMinutes(slot)
            const income = incomeOf(order, duration)
            let timeState = 'future'
            if (day.date < today) timeState = 'past'
            else if (day.date === today) {
              const endTime = String(slot.endTime || '')
              const startTime = String(slot.startTime || '')
              if (endTime && endTime < nowStr) timeState = 'past'
              else if (startTime <= nowStr && (!endTime || endTime >= nowStr)) timeState = 'ongoing'
            }
            events.push({
              key: `${order.id}-${day.date}-${slot.startTime}`,
              orderId: order.id,
              orderNo: order.orderNo,
              date: day.date,
              weekday: day.weekday,
              startTime: slot.startTime,
              endTime: slot.endTime,
              startMinute,
              endMinute,
              grade: order.grade,
              studentName: order.studentName,
              subject: order.subject,
              address: order.address,
              content: slot.content || '',
              remark: slot.remark || '',
              remarkFlag: slot.remarkFlag || false,
              hourlyRate: order.hourlyRate,
              income,
              status: order.status,
              timeState,
              slotIndex: 0,
              lessonId: null,
              adjustType: ''
            })
            if (timeState !== 'past') {
              summary.count++
              summary.hours += duration / 60
              if ((order.direction || 'income') === 'income') summary.incomeAmount += income
              else summary.expenseAmount += income
            }
            if (timeState === 'future' && Number(order.hourlyRate) > 0) summary.income += income
          }
        }
      }
      summary.hours = Math.round(summary.hours * 10) / 10
      summary.income = Math.round(summary.income)
      return { days, hours: Array.from({ length: 18 }, (_, i) => i + 6), events, summary }
    },

    // 注册成功后：把本地试用数据导入账号（逐门调用后端新增接口），导入完成后清空本地
    async importToServer() {
      const list = [...this.orders]
      let ok = 0
      for (const o of list) {
        const { id: _id, orderNo: _no, createdAt: _c, ...form } = o
        try {
          await orderApi.create(form)
          ok++
        } catch {
          // 试用上限等失败场景：保留本地数据，用户可手动处理
        }
      }
      if (ok > 0) this.clear()
      return ok
    },

    clear() {
      this.orders = []
      this.startedAt = ''
      localStorage.removeItem(STORAGE_KEY)
    }
  }
})
