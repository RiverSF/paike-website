<template>
  <div class="feedback-panel">
    <div class="filters">
      <el-select v-model="status" placeholder="全部状态" clearable style="width: 130px" @change="load">
        <el-option label="待处理" value="pending" />
        <el-option label="已回复" value="resolved" />
      </el-select>
      <el-checkbox v-model="allUsers" @change="load">包含管理员/站长提交的内容</el-checkbox>
      <el-button @click="load">刷新</el-button>
    </div>

    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column label="提交用户" width="140">
        <template #default="{ row }">
          <div>{{ row.username || '匿名用户' }}</div>
          <div class="cell-sub">{{ row.contact || '' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="反馈内容" min-width="260" show-overflow-tooltip>
        <template #default="{ row }">{{ row.content }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'resolved' ? 'success' : 'warning'" size="small" effect="light" round>
            {{ row.status === 'resolved' ? '已回复' : '待处理' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="官方回复" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.reply">{{ row.reply }}</span>
          <span v-else class="muted">未回复</span>
        </template>
      </el-table-column>
      <el-table-column label="提交时间" width="140">
        <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openReply(row)">
            {{ row.reply ? '修改回复' : '答复' }}
          </el-button>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty description="暂无反馈内容" />
      </template>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="total, prev, pager, next"
        background
        @current-change="load"
      />
    </div>

    <el-dialog v-model="visible" title="答复反馈" width="480px">
      <div v-if="current" class="fb-detail">
        <div class="fb-meta">
          {{ current.username }} · {{ fmtTime(current.createdAt) }}
          <el-tag size="small" effect="plain" round>
            {{ current.status === 'resolved' ? '已回复' : '待处理' }}
          </el-tag>
        </div>
        <div class="fb-content">{{ current.content }}</div>
      </div>

      <el-form label-position="top">
        <el-form-item label="回复内容">
          <el-input v-model="reply" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="请输入回复内容" />
        </el-form-item>
        <el-form-item label="审核状态">
          <el-radio-group v-model="replyStatus">
            <el-radio value="resolved">已回复（公开显示）</el-radio>
            <el-radio value="pending">待处理（暂不公开）</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import { adminApi } from '@/api'
import { sanitizeText, validateText } from '@/utils/text'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const status = ref('')
const allUsers = ref(false)

const visible = ref(false)
const current = ref(null)
const reply = ref('')
const replyStatus = ref('resolved')
const saving = ref(false)

const fmtTime = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD HH:mm') : '-')

async function load() {
  loading.value = true
  try {
    const res = await adminApi.feedbacks({
      page: page.value,
      pageSize: pageSize.value,
      status: status.value,
      all: allUsers.value ? '1' : ''
    })
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

function openReply(row) {
  current.value = row
  reply.value = row.reply || ''
  replyStatus.value = row.status === 'resolved' ? 'resolved' : 'resolved'
  visible.value = true
}

async function submit() {
  if (!reply.value.trim()) {
    ElMessage.warning('请填写回复内容')
    return
  }
  const textCheck = validateText(reply.value, '回复内容', { max: 500 })
  if (textCheck) {
    ElMessage.warning(textCheck)
    return
  }
  saving.value = true
  try {
    await adminApi.replyFeedback(current.value.id, {
      reply: sanitizeText(reply.value),
      status: replyStatus.value
    })
    ElMessage.success('已保存')
    visible.value = false
    load()
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.filters {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.cell-sub {
  font-size: 12px;
  color: var(--brand-muted);
}

.fb-detail {
  padding: 12px 14px;
  margin-bottom: 12px;
  border-radius: 10px;
  background: #fafbfc;
}

.fb-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--brand-muted);
  margin-bottom: 6px;
}

.fb-content {
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
