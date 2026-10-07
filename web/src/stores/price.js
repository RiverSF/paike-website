import { defineStore } from 'pinia'
import { priceApi } from '@/api'

// 会员价格配置：专职存标准价，学生按折扣实时换算（向上取整）；平季续费打折、寒暑假原价。
// 后端 GetPrice 同时返回 base（标准价）与 renew（续费/购买价，平季已打折）两组金额。
export const usePriceStore = defineStore('price', {
  state: () => ({
    price: {
      professional: { monthlyAmount: 79, quarterlyAmount: 225, yearlyAmount: 790 },
      student: { monthlyAmount: 68, quarterlyAmount: 192, yearlyAmount: 672 },
      renewal: {
        professional: { monthlyAmount: 79, quarterlyAmount: 225, yearlyAmount: 790 },
        student: { monthlyAmount: 68, quarterlyAmount: 192, yearlyAmount: 672 }
      },
      studentDiscount: 0.85,
      renewalDiscount: 0.9,
      renewalEnabled: true,
      activityEnabled: false,
      activityName: '',
      activityDiscount: 1,
      season: 'offpeak'
    }
  }),
  actions: {
    async fetch() {
      try {
        const res = await priceApi.get()
        const d = res?.data
        if (d) {
          // 续费价优先；旧后端未返回 renewal 字段时回退到基础价，避免显示 ¥0
          const renew = (r, base) => (r && r > 0 ? r : base)
          this.price = {
            professional: {
              monthlyAmount: d.proMonthlyAmount,
              quarterlyAmount: d.proQuarterlyAmount,
              yearlyAmount: d.proYearlyAmount
            },
            student: {
              monthlyAmount: d.studentMonthlyAmount,
              quarterlyAmount: d.studentQuarterlyAmount,
              yearlyAmount: d.studentYearlyAmount
            },
            renewal: {
              professional: {
                monthlyAmount: renew(d.proMonthlyRenew, d.proMonthlyAmount),
                quarterlyAmount: renew(d.proQuarterlyRenew, d.proQuarterlyAmount),
                yearlyAmount: renew(d.proYearlyRenew, d.proYearlyAmount)
              },
              student: {
                monthlyAmount: renew(d.studentMonthlyRenew, d.studentMonthlyAmount),
                quarterlyAmount: renew(d.studentQuarterlyRenew, d.studentQuarterlyAmount),
                yearlyAmount: renew(d.studentYearlyRenew, d.studentYearlyAmount)
              }
            },
            studentDiscount: d.studentDiscount,
            renewalDiscount: d.renewalDiscount,
            renewalEnabled: d.renewalEnabled !== false,
            activityEnabled: !!d.activityEnabled && d.activityDiscount > 0 && d.activityDiscount < 1,
            activityName: d.activityName || '',
            activityDiscount: d.activityDiscount || 1,
            season: d.season
          }
        }
      } catch (e) {
        // 保持默认价格，不阻断页面
      }
      return this.price
    }
  }
})
