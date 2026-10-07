<template>
  <div class="price-eff">
    <el-radio-group :model-value="mode" size="small" @update:model-value="(v) => $emit('update:mode', v)">
      <el-radio value="now">立即生效</el-radio>
      <el-radio value="later">指定日期</el-radio>
    </el-radio-group>
    <div v-if="mode === 'later'" class="eff-picker">
      <el-date-picker
        :model-value="time"
        type="date"
        placeholder="选择生效日期"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        :disabled-date="disableNotAfterToday"
        size="small"
        style="width: 100%"
        @update:model-value="(v) => $emit('update:time', v)"
      />
      <div class="eff-hint">生效日期当天 00:00 起生效</div>
    </div>
  </div>
</template>

<script setup>
import dayjs from 'dayjs'

defineProps({
  mode: { type: String, default: 'now' },
  time: { type: String, default: '' }
})
defineEmits(['update:mode', 'update:time'])

// 生效时间精确到「日」：仅允许选择明天及以后（当天视为立即生效，已由「立即生效」覆盖）
const disableNotAfterToday = (date) => date && !dayjs(date).isAfter(dayjs().endOf('day'))
</script>

<style scoped>
.price-eff {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  width: 100%;
}

.eff-picker {
  width: 100%;
}

.eff-hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: #a0a6b1;
}
</style>
