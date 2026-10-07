<template>
  <el-dialog v-model="visible" width="460px" align-center>
    <template #header>
      <div class="dialog-title">会员开通 / 续费</div>
    </template>

    <div class="plan-line">
      当前套餐：<b>{{ planName }}</b>
      <b class="price">{{ planPrice }}</b>
      <el-tag v-if="seasonTag" size="small" :type="seasonTag.type" effect="light" class="season-tag">{{ seasonTag.text }}</el-tag>
    </div>
    <div class="muted small price-tip">支付后需由管理员人工核实开通，系统不支持自助开通</div>

    <el-tabs v-model="payType" class="pay-tabs" stretch>
      <el-tab-pane label="微信支付" name="wechat">
        <QrCard title="微信收款码" desc="请使用微信扫码付款" :src="qrcodes.wechatPay" />
      </el-tab-pane>
      <el-tab-pane label="支付宝支付" name="alipay">
        <QrCard title="支付宝收款码" desc="请使用支付宝扫码付款" :src="qrcodes.alipay" />
      </el-tab-pane>
    </el-tabs>

    <div class="pay-tip">
      如已完成支付，请扫描下方二维码添加微信好友，联系相关人员为您开通或延长会员有效时间。
      <br />开通后可在「个人中心 - 支付记录」中查看本次支付金额与到期时间。
    </div>

    <div class="contact">
      <QrCard title="添加微信好友" desc="支付后联系开通 / 售后咨询" :src="qrcodes.wechatFriend" />
    </div>
  </el-dialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import QrCard from '@/components/QrCard.vue'
import { qrcodes } from '@/config/site'
import { usePriceStore } from '@/stores/price'

const props = defineProps({
  modelValue: Boolean,
  plan: { type: String, default: 'monthly' }
})
const emit = defineEmits(['update:modelValue'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const payType = ref('wechat')
const priceStore = usePriceStore()
const fmtPrice = (v) => Number(v || 0).toString()

const planName = computed(() => {
  const names = { trial: '免费试用（注册赠送 30 天）', monthly: '包月会员', quarterly: '包季会员', yearly: '包年会员' }
  return names[props.plan] || names.monthly
})
// 师资身份计费档位已下线：统一按专职标准价
const seasonTag = computed(() => {
  const p = priceStore.price
  const zhe = (d) => Math.round((d || 1) * 100) / 10
  if (p.activityEnabled) {
    return { text: `${p.activityName || '限时活动'} · ${zhe(p.activityDiscount)} 折`, type: 'danger' }
  }
  if (p.season === 'peak') {
    return { text: '寒暑假 · 高峰原价', type: 'warning' }
  }
  if (p.renewalEnabled) {
    return { text: `平季 · ${zhe(p.renewalDiscount)} 折`, type: 'success' }
  }
  return { text: '标准价', type: 'info' }
})
const planPrice = computed(() => {
  const r = priceStore.price.renewal.professional || {}
  const prices = {
    trial: '¥0',
    monthly: `¥${fmtPrice(r.monthlyAmount)} / 月`,
    quarterly: `¥${fmtPrice(r.quarterlyAmount)} / 季`,
    yearly: `¥${fmtPrice(r.yearlyAmount)} / 年`
  }
  return prices[props.plan] || ''
})

onMounted(() => {
  priceStore.fetch()
})
</script>

<style scoped>
.dialog-title {
  font-size: 17px;
  font-weight: 600;
}

.plan-line {
  font-size: 13px;
  margin-bottom: 4px;
}

.plan-line b {
  color: var(--el-color-primary);
}

.plan-line .price {
  margin-left: 8px;
  font-size: 16px;
}

.season-tag {
  margin-left: 8px;
  font-weight: 400;
}

.price-tip {
  margin-bottom: 8px;
}

.pay-tabs :deep(.el-tabs__content) {
  display: flex;
  justify-content: center;
}

.pay-tip {
  margin: 14px 0 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  color: var(--brand-sub);
  font-size: 13px;
  line-height: 1.7;
}

.contact {
  display: flex;
  justify-content: center;
}
</style>
