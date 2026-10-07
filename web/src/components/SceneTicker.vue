<template>
  <!--
  使用场景轮播（**当前未在首页启用**）。

  评估结论：内容为示例数据，对注册转化的贡献有限；放在 Hero 之后会与右侧「示意图」重复表达，
  放在功能区与 CTA 收口条之间又语义不归属，且横向动效会削弱 CTA 的收口效果，故先从首页移除。

  后续如要启用，建议：
  1. 优先放到「使用指南」页作为场景示例，或做成静态单行（不滚动）；
  2. 若要作为社会证明，请改用真实数据（注册量 / 活跃量等聚合值），不要伪造注册记录；
  3. 保持「示例数据」标注，避免被误解为真实用户动态。
-->
  <section class="scene-ticker">
    <div class="ticker-head">
      <span class="ticker-title">常见排课场景</span>
      <el-tag size="small" type="info" effect="plain" round>示例数据</el-tag>
    </div>

    <div class="ticker-viewport">
      <div class="ticker-track" :class="{ paused: hovering }" @mouseenter="hovering = true" @mouseleave="hovering = false">
        <div v-for="(item, i) in loopScenes" :key="i" class="scene-item">
          <span class="scene-role" :class="roleClass(item.role)">{{ item.role }}</span>
          <span class="scene-name">{{ item.name }}</span>
          <span class="scene-phone">{{ item.phone }}</span>
          <span class="scene-text">{{ item.text }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'

// 示例场景：手机号统一使用 0000 占位尾号，明确不是真实号码
const scenes = [
  { name: '李老师', role: '老师', phone: '138****0000', text: '每周二、四 18:00-20:00 一对一数学' },
  { name: '王同学', role: '学员', phone: '156****0000', text: '高中英语 1v1，按周次自动生成课表' },
  { name: '陈妈妈', role: '学员', phone: '189****0000', text: '孩子奥数 + 语文，周末时段互不冲突' },
  { name: '周老师', role: '老师', phone: '137****0000', text: '小班课 6 人，一处改期全班同步' },
  { name: '林同学', role: '学员', phone: '158****0000', text: '初中物理，两处校区上课排布' },
  { name: '刘妈妈', role: '学员', phone: '135****0000', text: '钢琴 + 书法，固定周一时段' },
  { name: '赵老师', role: '老师', phone: '186****0000', text: '寒暑假高峰密集排课，提前续费' },
  { name: '孙同学', role: '学员', phone: '150****0000', text: '英语口语 + 作文，课时与费用一目了然' },
  { name: '吴妈妈', role: '学员', phone: '159****0000', text: '期中前临时加课，加课记录可追溯' }
]

// 复制一份实现无缝滚动（两段内容首尾相接）
const loopScenes = computed(() => [...scenes, ...scenes])
const hovering = ref(false)

function roleClass(role) {
  return { 老师: 'is-pro', 学员: 'is-user' }[role] || 'is-pro'
}
</script>

<style scoped>
.scene-ticker {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 16px 8px;
}

.ticker-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.ticker-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--brand-text, #303133);
}

/* 视窗溢出隐藏，形成横向滚动带 */
.ticker-viewport {
  overflow: hidden;
  -webkit-mask-image: linear-gradient(90deg, transparent, #000 6%, #000 94%, transparent);
  mask-image: linear-gradient(90deg, transparent, #000 6%, #000 94%, transparent);
}

.ticker-track {
  display: flex;
  gap: 12px;
  width: max-content;
  animation: scene-scroll 48s linear infinite;
}

.ticker-track.paused {
  animation-play-state: paused;
}

.scene-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 999px;
  background: #fff;
  border: 1px solid var(--brand-line, #e8eef7);
  font-size: 13px;
  color: var(--brand-sub, #8a94a6);
  white-space: nowrap;
}

.scene-role {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
}

.scene-role.is-pro {
  background: #eaf7ee;
  color: #37985a;
}

.scene-role.is-user {
  background: #eaf2ff;
  color: #3b7fd4;
}

.scene-name {
  color: var(--brand-text, #303133);
  font-weight: 500;
}

.scene-phone {
  font-variant-numeric: tabular-nums;
}

@keyframes scene-scroll {
  from {
    transform: translateX(0);
  }
  to {
    /* 滚动一份内容宽度，形成无缝衔接 */
    transform: translateX(calc(-50% - 6px));
  }
}

@media (prefers-reduced-motion: reduce) {
  .ticker-viewport {
    overflow: visible;
  }

  .ticker-track {
    animation: none;
    flex-wrap: wrap;
    width: auto;
  }
}
</style>
