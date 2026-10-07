<template>
  <div class="slot-editor">
    <div v-for="(slot, idx) in local" :key="idx" class="slot-row">
      <div class="slot-days">
        <el-checkbox-group v-model="slot.days">
          <el-checkbox v-for="d in 7" :key="d" :value="d">{{ dayLabel(d) }}</el-checkbox>
        </el-checkbox-group>
      </div>
      <div class="slot-time">
        <span class="row-label">上课时间</span>
        <el-time-picker
          v-model="slot.start"
          format="HH:mm"
          value-format="HH:mm"
          placeholder="开始"
          style="width: 110px"
        />
        <span class="sep">~</span>
        <el-time-picker
          v-model="slot.end"
          format="HH:mm"
          value-format="HH:mm"
          placeholder="结束"
          style="width: 110px"
        />
        <el-button link type="danger" :icon="Delete" @click="removeSlot(idx)">删除</el-button>
      </div>
      <div class="slot-effective">
        <span class="row-label">生效日期</span>
        <el-date-picker
          v-model="slot.effectiveFrom"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="起始（可选）"
          clearable
          style="width: 150px"
        />
        <span class="sep">~</span>
        <el-date-picker
          v-model="slot.effectiveTo"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="截止（可选）"
          clearable
          style="width: 150px"
        />
      </div>
    </div>

    <div class="slot-foot">
      <el-button size="small" :icon="Plus" @click="addSlot">添加时间段</el-button>
      <span class="tip muted">
        <span class="tip-text">例：周一至周四 18:00 ~ 20:00</span>
        <el-tooltip placement="top" :width="330" effect="light">
          <template #content>
            生效日期：同一星期在不同阶段采用不同安排（如 5 月 1 日起改为周日），起止均含当天，留空表示不限，需与补课周期有交集。<br />
            临时调整某一次课程：请到「课表」定位对应日期的课次单独调整，不影响每周固定安排。
          </template>
          <span class="tip-help">填写说明</span>
        </el-tooltip>
      </span>
    </div>
    <div v-if="conflict" class="slot-error">{{ conflict }}</div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { Delete, Plus } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] }
})
const emit = defineEmits(['update:modelValue'])

function defaultSlot() {
  return { days: [], start: '18:00', end: '20:00', effectiveFrom: '', effectiveTo: '' }
}

function clone(list) {
  return (list || []).map((s) => ({
    days: [...(s.days || [])],
    start: s.start || '',
    end: s.end || '',
    effectiveFrom: s.effectiveFrom || '',
    effectiveTo: s.effectiveTo || ''
  }))
}

const local = ref(props.modelValue?.length ? clone(props.modelValue) : [defaultSlot()])

function addSlot() {
  local.value.push(defaultSlot())
  sync()
}

function removeSlot(idx) {
  local.value.splice(idx, 1)
  if (!local.value.length) local.value.push(defaultSlot())
  sync()
}

function sync() {
  const valid = local.value.filter((s) => s.days?.length && s.start && s.end)
  if (JSON.stringify(valid) === JSON.stringify(props.modelValue)) return
  emit('update:modelValue', valid)
}

function dayLabel(d) {
  return ['周一', '周二', '周三', '周四', '周五', '周六', '周日'][d - 1]
}

// 生效周期是否重叠（留空表示不设限：起始留空视为从最早生效，截止留空视为一直生效）
function rangesOverlap(a, b) {
  const af = a.effectiveFrom || ''
  const at = a.effectiveTo || ''
  const bf = b.effectiveFrom || ''
  const bt = b.effectiveTo || ''
  return (af === '' || bt === '' || af <= bt) && (bf === '' || at === '' || bf <= at)
}

// 同一订单多个时间段：同一星期在相同生效周期内不可重复安排（避免同一周出现两种上课安排）
const conflict = computed(() => {
  for (let i = 0; i < local.value.length; i++) {
    for (let j = i + 1; j < local.value.length; j++) {
      const a = local.value[i]
      const b = local.value[j]
      const shared = (a.days || []).find((d) => (b.days || []).includes(d))
      if (shared == null) continue
      if (rangesOverlap(a, b)) {
        return `时间段 ${i + 1} 与时间段 ${j + 1} 在「${dayLabel(shared)}」的生效周期重叠，同一周内同一星期只能有一种安排，请调整生效日期`
      }
    }
  }
  return ''
})

watch(local, () => sync(), { deep: true })

watch(
  () => props.modelValue,
  (val) => {
    const next = val?.length ? clone(val) : [defaultSlot()]
    if (JSON.stringify(next) === JSON.stringify(local.value)) return
    local.value = next
  }
)
</script>

<style scoped>
.slot-editor {
  width: 100%;
}

.slot-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 8px;
  border-radius: 10px;
  /* 外层区块已有描边，这里不再加边框，用略深的底色做层次，避免线条嵌套 */
  background: #f1f4fa;
}

/* 周一至周日排成一行：压缩勾选项间距与字号 */
.slot-editor :deep(.el-checkbox) {
  margin-right: 12px;
  height: 26px;
}

.slot-editor :deep(.el-checkbox__label) {
  padding-left: 6px;
  font-size: 13px;
}

.slot-time {
  display: flex;
  align-items: center;
  gap: 8px;
}

.slot-effective {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 行内标签：固定宽度保证「上课时间」与「生效日期」的控件左边缘对齐 */
.row-label {
  flex: none;
  min-width: 48px;
  font-size: 12px;
  color: var(--brand-muted);
}

.sep {
  color: var(--brand-muted);
}

/* 添加按钮与示例说明同行，减少一行高度、左右更整齐 */
.slot-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.tip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  line-height: 1.6;
}

.tip-help {
  flex: none;
  color: var(--el-color-primary);
  cursor: help;
  white-space: nowrap;
  border-bottom: 1px dashed currentColor;
}

.slot-error {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-color-danger);
  line-height: 1.5;
}
</style>
