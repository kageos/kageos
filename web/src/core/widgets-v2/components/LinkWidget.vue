<!--
  LinkWidget - 链接组件
  功能：
  - 支持函数跳转（内部链接）
  - 支持外链跳转
  - 支持新窗口打开
  - 支持图标和样式自定义
-->

<template>
  <div class="link-widget">
    <!-- 编辑模式：不显示（链接是只读的） -->
    <div v-if="mode === 'edit'" class="link-disabled">
      <el-icon><Link /></el-icon>
      <span>{{ field.name }}</span>
    </div>
    
    <!-- 表格/详情模式：作为按钮显示（在操作区域） -->
    <el-button
      v-else-if="resolvedUrl && (mode === 'table-cell' || mode === 'detail')"
      :type="linkConfig.type === 'link' ? 'primary' : (linkConfig.type || 'primary')"
      size="small"
      :link="mode === 'table-cell' || linkConfig.type === 'link'"
      :plain="mode === 'detail'"
      class="link-button"
      @click.prevent="handleClick"
    >
      <el-icon v-if="linkConfig.icon" class="link-icon"><component :is="linkConfig.icon" /></el-icon>
      <el-icon v-else-if="isExternalLink" class="link-icon external-icon"><TopRight /></el-icon>
      <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
      <span class="link-text">{{ linkText }}</span>
    </el-button>
    
    <!-- 响应模式：作为链接显示 -->
    <el-link
      v-else-if="resolvedUrl"
      :href="linkConfig.target === '_blank' ? resolvedUrl : undefined"
      :target="linkConfig.target || '_self'"
      :type="linkConfig.type || 'primary'"
      :underline="true"
      class="link-response"
      @click.prevent="handleClick"
    >
      <el-icon v-if="linkConfig.icon" class="link-icon"><component :is="linkConfig.icon" /></el-icon>
      <el-icon v-else-if="isExternalLink" class="link-icon external-icon"><TopRight /></el-icon>
      <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
      <span class="link-text">{{ linkText }}</span>
    </el-link>
    
    <!-- 空值显示 -->
    <span v-else class="empty-text">-</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Link, Right, TopRight } from '@element-plus/icons-vue'
import { useAppEnvironment } from '@/composables/useAppEnvironment'
import type { WidgetComponentProps } from '../types'

const props = defineProps<WidgetComponentProps>()
const router = useRouter()
const { shouldOpenInCurrentWindow, isStandalone } = useAppEnvironment()

// 解析 URL 和文本（后端可能返回 "[text]url" 格式）
const parsedLink = computed(() => {
  const url = props.value?.raw || ''
  if (!url) return { text: '', url: '' }
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  if (match) {
    return {
      text: match[1],
      url: match[2]
    }
  }
  
  // 没有文本信息，使用原始 URL
  return {
    text: '',
    url: url
  }
})

// 解析后的 URL（处理站内跳转，添加 /workspace-v2 前缀）
const resolvedUrl = computed(() => {
  const url = parsedLink.value.url
  if (!url) return ''
  
  // 如果是外链（包含 http:// 或 https://），直接使用
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  
  // 🔥 如果已经是完整路径（包含 /workspace 或 /workspace-v2），转换为 /workspace-v2
  if (url.startsWith('/workspace/')) {
    // 将 /workspace 替换为 /workspace-v2
    return url.replace('/workspace/', '/workspace-v2/')
  }
  if (url.startsWith('/workspace-v2/')) {
    return url
  }
  
  // 如果是绝对路径（以 / 开头），添加 /workspace-v2 前缀
  if (url.startsWith('/')) {
    const pathWithoutSlash = url.substring(1)
    return `/workspace-v2/${pathWithoutSlash}`
  }
  
  // 相对路径，需要转换为完整路径
  return buildFullPath(url)
})

// 链接文本
const linkText = computed(() => {
  // 优先使用解析出的文本，其次使用 widget 配置的 text，最后使用字段名称
  if (parsedLink.value.text) {
    return parsedLink.value.text
  }
  return props.field.widget?.text || props.value?.display || props.field.name || '链接'
})

// 链接配置
const linkConfig = computed(() => {
  const widget = props.field.widget
  if (!widget || widget.type !== 'link') {
    return {}
  }
  
  // 后端返回的 JSON 字段名是 type（因为 json:"type,omitempty"）
  // 但结构体字段名是 LinkType，所以这里直接读取 config.type
  return {
    type: (widget.config as any)?.type || 'primary',
    target: (widget.config as any)?.target || '_self',
    icon: (widget.config as any)?.icon,
  }
})

// 判断是否是外链
const isExternalLink = computed(() => {
  const url = parsedLink.value.url
  return url.startsWith('http://') || url.startsWith('https://')
})

// 处理点击事件
const handleClick = (e: Event) => {
  e.preventDefault()
  e.stopPropagation()
  
  const url = resolvedUrl.value
  if (!url) return
  
  const target = linkConfig.value.target || '_self'
  
  // 判断是否是外链
  const isExternal = url.startsWith('http://') || url.startsWith('https://')
  
  // ⚠️ 关键：在 PWA/桌面环境中，即使配置了 _blank，内部链接也应该在当前窗口打开
  // 因为新窗口打开会跳转到浏览器，破坏用户体验
  // 外链仍然使用新窗口打开（因为无法使用路由导航）
  if (isExternal) {
    // 外链：始终使用新窗口打开（无论是浏览器还是 PWA 环境）
    window.open(url, '_blank')
  } else {
    // 内部链接
    if (shouldOpenInCurrentWindow(target)) {
      // 在当前窗口打开（使用路由导航）
      router.push(url)
    } else {
      // 新窗口打开（仅在浏览器环境中，PWA 环境会被 shouldOpenInCurrentWindow 拦截）
      window.open(url, '_blank')
    }
  }
}

// 构建完整路径
function buildFullPath(relativePath: string): string {
  // 如果已经是绝对路径（以 / 开头），直接添加 /workspace-v2 前缀
  if (relativePath.startsWith('/')) {
    if (relativePath.startsWith('/workspace-v2/')) {
      return relativePath
    }
    // 🔥 如果是 /workspace 路径，转换为 /workspace-v2
    if (relativePath.startsWith('/workspace/')) {
      return relativePath.replace('/workspace/', '/workspace-v2/')
    }
    const pathWithoutSlash = relativePath.substring(1)
    return `/workspace-v2/${pathWithoutSlash}`
  }
  
  // 解析相对路径：function_name?query
  const [functionPath, query] = relativePath.split('?')
  
  // 从当前路由获取 user 和 app
  const currentRoute = router.currentRoute.value
  const pathParts = currentRoute.path.split('/').filter(Boolean)
  
  if (pathParts.length < 3) {
    // 如果路径格式不正确，尝试添加 /workspace-v2 前缀
    return `/workspace-v2/${relativePath}`
  }
  
  // 🔥 支持 /workspace-v2 和 /workspace 两种路径
  const user = pathParts[1]
  const app = pathParts[2]
  
  // 构建完整路径（使用 /workspace-v2）
  const fullPath = `/workspace-v2/${user}/${app}/${functionPath}`
  return query ? `${fullPath}?${query}` : fullPath
}

// 将 URL 转换为路由路径
function convertUrlToRoute(url: string): string {
  // 🔥 如果已经是完整路径（包含 /workspace 或 /workspace-v2），转换为 /workspace-v2
  if (url.startsWith('/workspace/')) {
    return url.replace('/workspace/', '/workspace-v2/')
  }
  if (url.startsWith('/workspace-v2/')) {
    return url
  }
  
  // 如果是外链，直接返回
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  
  // 如果是绝对路径（以 / 开头），添加 /workspace-v2 前缀
  if (url.startsWith('/')) {
    const pathWithoutSlash = url.substring(1)
    return `/workspace-v2/${pathWithoutSlash}`
  }
  
  // 否则使用 buildFullPath（相对路径）
  return buildFullPath(url)
}
</script>

<style scoped>
.link-widget {
  display: inline-flex;
  align-items: center;
}

.link-disabled {
  display: inline-flex;
  align-items: center;
  color: var(--el-text-color-placeholder);
  gap: 4px;
}

/* 表格/详情模式：作为按钮显示 */
.link-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.2s;
}

.link-button:hover {
  transform: translateX(2px);
}

/* 响应模式：作为链接显示 */
.link-response {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
}

.link-response:hover {
  background-color: var(--el-fill-color-light);
  transform: translateX(2px);
}

/* 链接图标 */
.link-icon {
  font-size: 14px;
  transition: transform 0.2s;
}

.link-cell:hover .link-icon,
.link-detail:hover .link-icon {
  transform: translateX(2px);
}

/* 内部链接图标（右箭头） */
.internal-icon {
  color: var(--el-color-primary);
}

/* 外部链接图标（右上角箭头） */
.external-icon {
  color: var(--el-color-info);
}

/* 链接文本 */
.link-text {
  flex: 1;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}
</style>

