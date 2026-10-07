<template>
  <el-drawer v-model="visible" title="站内信" size="420px" @open="load">
    <div v-if="loading" class="muted small">加载中…</div>
    <div v-else-if="!list.length" class="muted small empty">暂无站内信</div>
    <div
      v-for="m in list"
      :key="m.id"
      class="msg"
      :class="{ unread: !m.read }"
      @click="markRead(m)"
    >
      <div class="msg-top">
        <el-tag size="small" effect="light" round :type="m.type === 'promo' ? 'danger' : 'info'">
          {{ m.type === 'promo' ? '优惠' : '通知' }}
        </el-tag>
        <span class="msg-title">{{ m.title }}</span>
        <span v-if="!m.read" class="dot" />
        <span class="msg-time">{{ fmtTs(m.createdAt) }}</span>
      </div>
      <div class="msg-content">
        <template v-for="(ln, i) in contentParts(m.content)" :key="i">
          <div v-if="ln.reason" class="msg-reason">{{ ln.text }}</div>
          <div v-else-if="ln.text" class="msg-line">{{ ln.text }}</div>
        </template>
      </div>
    </div>
  </el-drawer>
</template>

<script setup>
import { computed, ref } from 'vue'
import dayjs from 'dayjs'
import { messageApi } from '@/api'
import { useUserStore } from '@/stores/user'

const props = defineProps({ modelValue: Boolean })
const emit = defineEmits(['update:modelValue'])
const userStore = useUserStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const list = ref([])
const loading = ref(false)
const fmtTs = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD HH:mm') : '-')

// 站内信按行分段：空行过滤，以「驳回原因」开头的行高亮展示
const contentParts = (content) =>
  String(content || '')
    .split('\n')
    .filter((t) => t.trim())
    .map((t) => ({ text: t.trim(), reason: t.trim().startsWith('驳回原因') }))

async function load() {
  loading.value = true
  try {
    const res = await messageApi.list({ page: 1, pageSize: 30 })
    list.value = res.data?.list || []
  } catch (e) {
    list.value = []
  } finally {
    loading.value = false
  }
}

async function markRead(m) {
  if (m.read) return
  try {
    await messageApi.read(m.id)
    m.read = true
    userStore.loadUnreadMessages()
  } catch (e) {
    // ignore
  }
}
</script>

<style scoped>
.empty {
  padding: 12px 0;
}
.msg {
  padding: 12px;
  margin-bottom: 10px;
  border-radius: 10px;
  background: #fafbfc;
  border: 1px solid var(--brand-line);
  cursor: pointer;
  transition: background 0.2s;
}
.msg:hover {
  background: #f3f6fc;
}
.msg.unread {
  background: #eef4ff;
  border-color: #cfe0ff;
}
.msg-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.msg-title {
  font-weight: 600;
  font-size: 14px;
}
.msg-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--brand-muted);
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-color-danger);
}
.msg-content {
  font-size: 13px;
  color: var(--brand-sub);
  line-height: 1.7;
}

.msg-line {
  margin-bottom: 2px;
}

/* 驳回原因：浅橙底色 + 橙色文字，简洁醒目 */
.msg-reason {
  margin: 6px 0;
  padding: 6px 10px;
  border-radius: 8px;
  background: #fff4e6;
  color: #c2570a;
  font-weight: 500;
}
.muted {
  color: var(--brand-muted);
}
.small {
  font-size: 12px;
}
</style>
