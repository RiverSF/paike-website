<template>
  <div class="invite-panel">
    <div class="pane-desc">为申请人（手机号）生成邀请码，一个手机号（微信）仅可申请一个；注册时需与手机号匹配。</div>

    <div class="filters">
      <el-input
        v-model="keyword"
        placeholder="搜索邀请码 / 手机号"
        clearable
        style="width: 240px"
        @keyup.enter="load"
        @clear="load"
      />
      <el-button @click="load">查询</el-button>
      <el-button type="primary" class="push-right" @click="openCreate">生成邀请码</el-button>
    </div>

    <el-table v-loading="loading" :data="list" border stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column label="邀请码" width="200">
        <template #default="{ row }">
          <b class="code">{{ row.code }}</b>
          <el-button link type="primary" class="copy-btn" @click="copy(row.code)">复制</el-button>
        </template>
      </el-table-column>
      <el-table-column prop="phone" label="手机号" width="140" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.invalid ? 'info' : (row.used ? 'info' : 'success')" size="small" effect="light" round>
            {{ row.invalid ? '已作废' : (row.used ? '已使用' : '未使用') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="使用用户 ID" width="110">
        <template #default="{ row }">{{ row.usedByUserId || '-' }}</template>
      </el-table-column>
      <el-table-column label="创建时间" width="150">
        <template #default="{ row }">{{ fmtDate(row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <template v-if="!row.invalid">
            <el-button link type="warning" @click="invalidate(row)">作废</el-button>
            <el-button v-if="!row.used" link type="danger" @click="remove(row)">删除</el-button>
          </template>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
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

    <!-- 生成邀请码 -->
    <el-dialog v-model="createVisible" title="生成邀请码" width="420px" align-center @keyup.enter="create">
      <div class="muted small" style="margin-bottom: 12px">
        一个手机号（微信）仅可申请一个邀请码；注册时填写的手机号需与该邀请码关联的手机号一致。
      </div>
      <el-form label-width="112px" @submit.prevent>
        <el-form-item label="申请人手机号">
          <el-input
            v-model="createPhone"
            maxlength="11"
            placeholder="请输入 11 位手机号"
            @keyup.enter="create"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="create">生成</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resultVisible" title="邀请码已生成" width="420px" align-center>
      <div class="result-code">{{ resultCode }}</div>
      <div class="result-tip">请将邀请码发送给申请人，注册时填写的手机号需与该邀请码关联的手机号一致。</div>
      <template #footer>
        <el-button @click="copyResult">复制邀请码</el-button>
        <el-button type="primary" @click="resultVisible = false">知道了</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import dayjs from 'dayjs'
import { adminApi } from '@/api'
import { copyText } from '@/utils/clipboard'
import { isValidPhone } from '@/utils/text'

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const keyword = ref('')
const createPhone = ref('')
const creating = ref(false)
const createVisible = ref(false)
const resultVisible = ref(false)
const resultCode = ref('')

function openCreate() {
  createPhone.value = ''
  createVisible.value = true
}

// 复制邀请码到剪贴板
async function copy(code) {
  const ok = await copyText(code)
  ElMessage[ok ? 'success' : 'error'](ok ? '邀请码已复制' : '复制失败，请手动选择复制')
}

async function copyResult() {
  await copy(resultCode.value)
}

const fmtDate = (ts) => (ts ? dayjs(ts * 1000).format('YYYY-MM-DD HH:mm') : '-')

async function load() {
  loading.value = true
  try {
    const res = await adminApi.inviteCodes({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    list.value = res.data.list || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

async function create() {
  const phone = createPhone.value.trim()
  if (!isValidPhone(phone)) {
    ElMessage.warning('请填写正确的 11 位手机号')
    return
  }
  creating.value = true
  try {
    const res = await adminApi.createInviteCode({ phone })
    resultCode.value = res.data.code
    createVisible.value = false
    resultVisible.value = true
    createPhone.value = ''
    page.value = 1
    load()
  } finally {
    creating.value = false
  }
}

async function remove(row) {
  await ElMessageBox.confirm(`确认删除邀请码「${row.code}」？删除后记录将彻底移除。`, '提示', { type: 'warning' })
  await adminApi.deleteInviteCode(row.id)
  ElMessage.success('已删除')
  load()
}

// 作废：保留记录但标记为不可用，注册时该码将被拒绝（区别于删除：删除会移除记录）
async function invalidate(row) {
  await ElMessageBox.confirm(
    `确认作废邀请码「${row.code}」？作废后该码无法用于注册，但记录会保留以便审计。`,
    '提示',
    { type: 'warning' }
  )
  await adminApi.invalidateInviteCode(row.id)
  ElMessage.success('已作废')
  load()
}

onMounted(load)
</script>

<style scoped>
.pane-desc {
  font-size: 13px;
  color: var(--brand-muted);
  margin-bottom: 12px;
}

.filters {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

/* 生成按钮靠列表右上角 */
.push-right {
  margin-left: auto;
}

.copy-btn {
  margin-left: 8px;
}

.code {
  font-family: monospace;
  letter-spacing: 1px;
  color: var(--el-color-primary);
}

.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.result-code {
  font-family: monospace;
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 2px;
  text-align: center;
  color: var(--el-color-primary);
  padding: 12px 0;
}

.result-tip {
  text-align: center;
  font-size: 13px;
  color: var(--brand-muted);
}
</style>
