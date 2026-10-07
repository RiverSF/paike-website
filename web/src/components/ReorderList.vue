<template>
  <ul class="reorder-list">
    <li
      v-for="(item, idx) in items"
      :key="item.key"
      class="reorder-item"
      :class="{
        dragging: dragIndex === idx,
        over: overIndex === idx && dragIndex !== idx,
        locked: item.locked
      }"
      :draggable="!item.locked"
      @dragstart="onDragStart(idx)"
      @dragover.prevent="onDragOver(idx)"
      @drop.prevent="onDrop(idx)"
      @dragend="onDragEnd"
    >
      <span class="handle" :class="{ disabled: item.locked }" :title="item.locked ? '已固定' : '拖拽排序'">
        <el-icon><Rank /></el-icon>
      </span>
      <span class="label">{{ item.label }}</span>
      <span v-if="item.locked" class="lock-tip">置顶</span>
      <span v-else class="arrows">
        <el-button
          text
          size="small"
          :disabled="idx === 0"
          @click="move(idx, -1)"
        >
          <el-icon><CaretTop /></el-icon>
        </el-button>
        <el-button
          text
          size="small"
          :disabled="idx === items.length - 1"
          @click="move(idx, 1)"
        >
          <el-icon><CaretBottom /></el-icon>
        </el-button>
      </span>
    </li>
  </ul>
</template>

<script setup>
import { ref, watch } from 'vue'
import { CaretBottom, CaretTop, Rank } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: { type: Array, required: true } // [{ key, label, locked? }]
})
const emit = defineEmits(['update:modelValue'])

const items = ref(props.modelValue.map((x) => ({ ...x })))

watch(
  () => props.modelValue,
  (v) => {
    items.value = v.map((x) => ({ ...x }))
  },
  { deep: true }
)

const dragIndex = ref(-1)
const overIndex = ref(-1)

function emitOrder() {
  emit('update:modelValue', items.value.map((x) => ({ key: x.key, label: x.label, locked: x.locked })))
}

function onDragStart(idx) {
  dragIndex.value = idx
}

function onDragOver(idx) {
  overIndex.value = idx
}

function onDrop(idx) {
  const from = dragIndex.value
  if (from < 0 || from === idx) return
  const list = [...items.value]
  const [moved] = list.splice(from, 1)
  list.splice(idx, 0, moved)
  items.value = list
  emitOrder()
}

function onDragEnd() {
  dragIndex.value = -1
  overIndex.value = -1
}

function move(idx, dir) {
  const target = idx + dir
  if (target < 0 || target >= items.value.length) return
  const list = [...items.value]
  const [moved] = list.splice(idx, 1)
  list.splice(target, 0, moved)
  items.value = list
  emitOrder()
}
</script>

<style scoped>
.reorder-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 360px;
  overflow-y: auto;
}

.reorder-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-bottom: 6px;
  border: 1px solid var(--brand-line);
  border-radius: 8px;
  background: #fff;
  cursor: grab;
  transition: box-shadow 0.15s, border-color 0.15s, background 0.15s;
  user-select: none;
}

.reorder-item:hover {
  border-color: var(--el-color-primary);
  box-shadow: 0 2px 8px rgba(31, 35, 41, 0.08);
}

.reorder-item.dragging {
  opacity: 0.45;
  cursor: grabbing;
}

.reorder-item.over {
  border-color: var(--el-color-primary);
  border-style: dashed;
  background: var(--el-color-primary-light-9);
}

.reorder-item.locked {
  cursor: default;
  background: #fafbfc;
}

.handle {
  display: inline-flex;
  color: var(--brand-muted);
  font-size: 16px;
}

.handle.disabled {
  opacity: 0.35;
}

.label {
  flex: 1;
  font-size: 13px;
}

.lock-tip {
  font-size: 11px;
  color: var(--brand-muted);
  padding: 1px 6px;
  border-radius: 4px;
  background: #f0f1f3;
}

.arrows {
  display: inline-flex;
  flex-direction: column;
  gap: 0;
  line-height: 0;
}

.arrows :deep(.el-button) {
  margin: 0;
  padding: 0 4px;
  height: 16px;
}
</style>
