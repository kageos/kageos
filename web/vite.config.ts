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
      // Hub API 也通过网关代理
      '/hub': {
        target: 'http://localhost:9090',  // 网关地址（网关会转发到 Hub 服务）
        changeOrigin: true,
      },
    },
  },
})
