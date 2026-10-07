import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 开发态由 vite 代理 /api 到后端；生产态（单机部署）由 Gin 托管 dist 并转发 /api。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    // 开发态前端端口（与 Makefile 的 DEV_WEB_PORT 一致，无需 root；绑定 80 需 sudo）
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://127.0.0.1:9090',
        changeOrigin: true
      },
      '/uploads': {
        target: process.env.VITE_PROXY_TARGET || 'http://127.0.0.1:9090',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    // dist 为纯构建产物（已在 .gitignore 忽略）；每次构建清空旧产物，保证与 public/src 一致，避免残留旧 logo/文件
    emptyOutDir: true,
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        // echarts 体积较大，单独拆包便于缓存
        manualChunks: {
          echarts: ['echarts']
        }
      }
    }
  }
})
