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
      :type="resolvedLinkType"
      size="small"
      :link="mode === 'table-cell' || isLinkStyle"
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
      :type="resolvedLinkType"
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
import { resolveWorkspaceUrl } from '@/utils/route'
import { parseLinkValue, addLinkTypeToUrl } from '@/utils/linkNavigation'
import { eventBus, RouteEvent } from '@/architecture/infrastructure/eventBus'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import type { LinkWidgetConfig } from '@/core/types/widget-configs'

const props = defineProps<WidgetComponentProps>()
const router = useRouter()
const { shouldOpenInCurrentWindow } = useAppEnvironment()

// 解析 Link 值（JSON 格式）
const parsedLink = computed(() => {
  const raw = props.value?.raw || ''
  return parseLinkValue(raw)
})

// 解析后的 URL（处理站内跳转，添加 /workspace 前缀）
const resolvedUrl = computed(() => {
  const url = parsedLink.value.url
  if (!url) return ''
  
  return resolveWorkspaceUrl(url, router.currentRoute.value)
})

// 链接文本
const linkText = computed(() => {
  // 优先使用解析出的文本，其次使用 widget 配置的 text，最后使用字段名称
  if (parsedLink.value.name) {
    return parsedLink.value.name
  }
  return props.field.widget?.text || props.value?.display || props.field.name || '链接'
})

// 链接配置（带类型）
const linkConfig = computed(() => {
  const widget = props.field.widget
  if (!widget || widget.type !== 'link') {
    return {} as LinkWidgetConfig
  }
  
  return (widget.config || {}) as LinkWidgetConfig
})

const isLinkStyle = computed(() => linkConfig.value.type === 'link')

const resolvedLinkType = computed<Exclude<LinkWidgetConfig['type'], 'link' | undefined>>(() => {
  const type = linkConfig.value.type
  return type && type !== 'link' ? type : 'primary'
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
  
  // ⚠️ 关键：在 PWA/桌面环境中，即使配置了 _blank，内部链接也应该在当前窗口打开
  // 因为新窗口打开会跳转到浏览器，破坏用户体验
  // 外链仍然使用新窗口打开（因为无法使用路由导航）
  if (isExternalLink.value) {
    // 外链：始终使用新窗口打开（无论是浏览器还是 PWA 环境）
    window.open(url, '_blank')
  } else {
    // 内部链接
    if (shouldOpenInCurrentWindow(target)) {
      // 在当前窗口打开（使用路由导航）
      // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
      // 如果 link 值中有 type 信息，通过 query 参数传递
      const finalUrl = addLinkTypeToUrl(url, parsedLink.value.type)
      
      // 解析 URL，提取 path 和 query
      // 注意：finalUrl 可能是相对路径（如 /workspace/xxx?param=value）
      let path = finalUrl
      const query: Record<string, string> = {}
      
      // 检查是否有查询参数
      const queryIndex = finalUrl.indexOf('?')
      if (queryIndex >= 0) {
        path = finalUrl.substring(0, queryIndex)
        const queryString = finalUrl.substring(queryIndex + 1)
        const params = new URLSearchParams(queryString)
        params.forEach((value, key) => {
          // 🔥 URLSearchParams 会自动解码 URL 编码的参数值（如 id%3A2 -> id:2）
          query[key] = value
        })
      }
      
      // 🔥 发出路由更新请求事件
      // 注意：query 中的参数来自 link URL，应该优先使用，不会被当前路由的参数覆盖
      eventBus.emit(RouteEvent.updateRequested, {
        path,
        query,
        replace: false,  // link 跳转使用 push，保留历史记录
        preserveParams: {
          linkNavigation: true  // link 跳转：保留所有参数
        },
        source: 'link-widget'
      })
    } else {
      // 新窗口打开（仅在浏览器环境中，PWA 环境会被 shouldOpenInCurrentWindow 拦截）
      window.open(url, '_blank')
    }
  }
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
