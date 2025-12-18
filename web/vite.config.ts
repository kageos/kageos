import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    vueDevTools(),
    AutoImport({
      imports: ['vue', 'vue-router'],
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    proxy: {
      // Workspace API 通过网关代理（必须在 Vue Router 之前处理）
      // 注意：只代理 /workspace/api/* 路径，不代理 /workspace 页面路由
      '/workspace/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
        rewrite: (path) => path, // 不重写路径，直接转发
      },
      // Agent API 通过网关代理（只代理 API 请求，不代理页面路由）
      // 注意：使用更精确的匹配，只匹配 /agent/api/* 路径，不匹配 /agent 页面路由
      '/agent/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // Storage API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/storage/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // Hub API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/hub/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // Control API 通过网关代理（只代理 API 请求，不代理页面路由）
      '/control/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // 统一通过网关代理所有 API 请求（兜底，用于兼容旧路径）
      '/api': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
      // Swagger 文档也通过网关
      '/swagger': {
        target: 'http://localhost:9090',  // 网关地址
        changeOrigin: true,
      },
    },
  },
})
