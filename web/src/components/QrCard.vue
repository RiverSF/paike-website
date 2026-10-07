<template>
  <div class="qr-card">
    <div class="qr-img">
      <img v-if="src" :src="src" :alt="title" />
      <div v-else class="qr-empty">
        <el-icon><Picture /></el-icon>
        <span>二维码待上传</span>
      </div>
    </div>
    <div class="qr-title">{{ title }}</div>
    <div class="qr-desc muted">{{ desc }}</div>
  </div>
</template>

<script setup>
import { Picture } from '@element-plus/icons-vue'

defineProps({
  title: { type: String, default: '' },
  desc: { type: String, default: '' },
  src: { type: String, default: '' }
})
</script>

<style scoped>
.qr-card {
  flex: 1 1 200px;
  min-width: 200px;
  padding: 18px;
  border-radius: var(--radius-card);
  background: #fff;
  border: 1px solid var(--brand-line);
  text-align: center;
}

.qr-img {
  width: 160px;
  height: 160px;
  margin: 0 auto 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.qr-img img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.qr-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed #c9ced8;
  border-radius: 10px;
  color: var(--brand-muted);
  font-size: 12px;
  background: #fafbfc;
}

.qr-empty .el-icon {
  font-size: 26px;
}

.qr-title {
  font-size: 14px;
  font-weight: 600;
}

.qr-desc {
  margin-top: 4px;
  font-size: 12px;
}
</style>
