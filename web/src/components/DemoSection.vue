<template>
  <section class="card">
    <div class="card-title">
      功能演示<span class="sub">50 秒看懂核心操作：录入课程 → 自动生成周课表</span>
    </div>
    <div class="demo-layout">
      <div class="video-wrap">
        <video
          ref="videoRef"
          class="demo-video"
          src="/videos/guide-demo.mp4"
          controls
          preload="metadata"
          playsinline
          @play="coverHidden = true"
          @ended="coverHidden = false"
        />
        <!-- 0 秒 logo 封面：未开始播放时展示品牌封面与播放按钮，点击即播放 -->
        <transition name="cover-fade">
          <div v-if="!coverHidden" class="video-cover" @click="playVideo">
            <img :src="logo.src" class="cover-logo" :alt="brand.brandName" />
            <div class="cover-name">{{ brand.brandName }} · 功能演示</div>
            <div class="cover-sub">50 秒看懂核心操作</div>
            <div class="cover-play"><el-icon><VideoPlay /></el-icon></div>
          </div>
        </transition>
      </div>
      <div class="demo-aside">
        <div class="demo-aside-title">看完你会知道</div>
        <ul class="demo-points">
          <li>录一次课程，整周课表自动铺开</li>
          <li>多时段、分阶段改期不用重录</li>
          <li>课时与费用随课表自动汇总</li>
        </ul>
        <el-button type="primary" round class="demo-cta" @click="goStart">
          {{ userStore.isLogin ? '进入排课管理' : '免费试用 · 无需注册' }}
        </el-button>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { VideoPlay } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { brand, logos, activeLogo } from '@/config/brand'

const userStore = useUserStore()
const router = useRouter()
const logo = logos[activeLogo]

// 封面显隐：开始播放后隐藏，播完恢复；点击封面直接播放
const videoRef = ref(null)
const coverHidden = ref(false)

function playVideo() {
  videoRef.value?.play?.()
}

function goStart() {
  if (userStore.isLogin) {
    router.push('/orders')
  } else {
    window.dispatchEvent(new CustomEvent('open-auth'))
  }
}
</script>

<style scoped>
.demo-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 20px;
  align-items: start;
}

.video-wrap {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: 12px;
  overflow: hidden;
  background: #0b1220;
  box-shadow: 0 6px 20px rgba(15, 23, 42, 0.12);
}

.demo-video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display: block;
  object-fit: contain;
}

/* ===== 0 秒 logo 封面 ===== */
.video-cover {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: radial-gradient(120% 120% at 50% 0%, #2a3550 0%, #0b1220 62%);
  cursor: pointer;
}

.cover-logo {
  width: 76px;
  height: 76px;
  padding: 14px;
  background: #fff;
  border-radius: 20px;
  object-fit: contain;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.35);
}

.cover-name {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.cover-sub {
  color: rgba(255, 255, 255, 0.62);
  font-size: 13px;
}

.cover-play {
  margin-top: 10px;
  width: 58px;
  height: 58px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
  background: var(--el-color-primary);
  box-shadow: 0 10px 26px color-mix(in srgb, var(--el-color-primary) 45%, transparent);
  transition: transform 0.2s;
}

.video-cover:hover .cover-play {
  transform: scale(1.08);
}

/* 封面淡出 */
.cover-fade-leave-active {
  transition: opacity 0.25s ease;
}

.cover-fade-leave-to {
  opacity: 0;
}

.demo-aside {
  padding-top: 2px;
}

.demo-aside-title {
  font-weight: 600;
  margin-bottom: 12px;
  font-size: 15px;
}

.demo-points {
  list-style: none;
  margin: 0 0 18px;
  padding: 0;
}

.demo-points li {
  position: relative;
  padding-left: 18px;
  margin-bottom: 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--brand-sub);
}

.demo-points li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 9px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--brand-primary, #2f6fed);
}

.demo-cta {
  width: 100%;
}

@media (max-width: 860px) {
  .demo-layout {
    grid-template-columns: 1fr;
  }

  .demo-aside {
    order: 2;
  }
}
</style>
