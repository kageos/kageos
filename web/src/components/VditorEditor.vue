<template>
  <div ref="vditorRef" class="vditor-wrapper"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { useThemeStore } from '@/stores/theme'

interface Props {
  modelValue: string
  height?: number | string
  placeholder?: string
  disabled?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: string): void
}

const props = withDefaults(defineProps<Props>(), {
  height: 500,
  placeholder: '请输入内容（支持 Markdown）',
  disabled: false
})

const emit = defineEmits<Emits>()

const vditorRef = ref<HTMLElement>()
let vditor: Vditor | null = null

const themeStore = useThemeStore()

// 初始化编辑器
onMounted(() => {
  if (!vditorRef.value) return

  const isDark = themeStore.currentTheme.mode === 'dark'

  vditor = new Vditor(vditorRef.value, {
    height: props.height,
    mode: 'ir', // 即时渲染模式 - 最佳所见即所得体验
    placeholder: props.placeholder,
    theme: isDark ? 'dark' : 'classic',
    
    // 工具栏配置
    toolbar: [
      'headings',
      'bold',
      'italic',
      'strike',
      '|',
      'line',
      'quote',
      'list',
      'ordered-list',
      'check',
      '|',
      'code',
      'inline-code',
      'link',
      'table',
      '|',
      'undo',
      'redo',
      '|',
      'edit-mode', // 模式切换按钮
      'preview',
      'fullscreen'
    ],
    
    // 预览配置
    preview: {
      theme: {
        current: isDark ? 'dark' : 'light'
      },
      hljs: {
        style: isDark ? 'monokai' : 'github',
        enable: true,
        lineNumber: true
      }
    },
    
    // 内容变化回调
    input: (value: string) => {
      emit('update:modelValue', value)
    },
    
    // 初始化完成回调
    after: () => {
      if (vditor && props.modelValue) {
        vditor.setValue(props.modelValue)
      }
    },
    
    // 缓存配置
    cache: {
      enable: false // 禁用本地缓存，避免冲突
    },
    
    // 计数器
    counter: {
      enable: true,
      type: 'markdown'
    },
    
    // 上传配置（暂时禁用，后续可以集成）
    upload: {
      handler: () => {
        return null
      }
    }
  })
})

// 监听主题变化，自动切换编辑器主题
watch(() => themeStore.currentTheme.mode, (mode) => {
  if (vditor) {
    const isDark = mode === 'dark'
    vditor.setTheme(
      isDark ? 'dark' : 'classic',
      isDark ? 'dark' : 'light',
      isDark ? 'monokai' : 'github'
    )
  }
})

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (vditor && vditor.getValue() !== newValue) {
    vditor.setValue(newValue || '')
  }
})

// 监听禁用状态
watch(() => props.disabled, (disabled) => {
  if (vditor) {
    vditor.disabled = disabled
  }
})

// 清理
onBeforeUnmount(() => {
  vditor?.destroy()
  vditor = null
})

// 暴露方法供父组件调用
defineExpose({
  getValue: () => vditor?.getValue() || '',
  setValue: (value: string) => vditor?.setValue(value),
  focus: () => vditor?.focus(),
  blur: () => vditor?.blur(),
  setDisabled: (disabled: boolean) => {
    if (vditor) vditor.disabled = disabled
  }
})
</script>

<style scoped>
.vditor-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* ========== 主题适配 ========== */

/* 主容器 */
:deep(.vditor) {
  border-radius: var(--border-radius-base);
  border-color: var(--border-base);
  background-color: var(--bg-primary);
  height: 100%; /* 确保占据容器高度 */
  display: flex;
  flex-direction: column;
}

/* 工具栏 */
:deep(.vditor-toolbar) {
  background-color: var(--bg-secondary);
  border-bottom: 1px solid var(--border-base);
  padding: 8px;
}

:deep(.vditor-toolbar button) {
  color: var(--text-primary);
  transition: all 0.2s ease;
}

:deep(.vditor-toolbar button:hover) {
  background-color: var(--bg-tertiary);
  color: var(--color-primary);
}

:deep(.vditor-toolbar button.vditor-toolbar__item--current) {
  background-color: var(--color-primary);
  color: white;
}

:deep(.vditor-toolbar .vditor-tooltipped::after) {
  background-color: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-base);
}

/* 分隔线 */
:deep(.vditor-toolbar .vditor-toolbar__divider) {
  background-color: var(--border-base);
}

/* 编辑区（IR 模式） */
:deep(.vditor-ir) {
  background-color: var(--bg-primary);
  flex: 1; /* 占据剩余空间 */
  overflow-y: auto;
}

:deep(.vditor-ir pre.vditor-reset) {
  color: var(--text-primary);
  padding: 16px;
}

/* 光标 */
:deep(.vditor-ir .vditor-reset .vditor-ir__marker) {
  color: var(--text-secondary);
}

/* Markdown 渲染样式适配 */
:deep(.vditor-ir .vditor-reset) {
  /* 标题 */
  h1, h2, h3, h4, h5, h6 {
    color: var(--text-primary);
    border-bottom-color: var(--border-base);
  }
  
  /* 段落和文本 */
  p, li {
    color: var(--text-regular);
  }
  
  /* 链接 */
  a {
    color: var(--color-primary);
    
    &:hover {
      opacity: 0.8;
    }
  }
  
  /* 行内代码 */
  code:not(.hljs) {
    background: var(--bg-tertiary);
    color: var(--color-primary);
    font-weight: 500;
  }
  
  /* 代码块 */
  pre > code {
    background: var(--bg-secondary);
    border: 1px solid var(--border-base);
  }
  
  /* 引用块 */
  blockquote {
    border-left-color: var(--color-primary);
    background: var(--bg-secondary);
    color: var(--text-regular);
  }
  
  /* 表格 */
  table {
    th {
      background: var(--bg-secondary);
      color: var(--text-primary);
      border-color: var(--border-base);
    }
    
    td {
      color: var(--text-regular);
      border-color: var(--border-base);
    }
  }
  
  /* 分割线 */
  hr {
    background-color: var(--border-base);
  }
}

/* 预览区 */
:deep(.vditor-preview) {
  background-color: var(--bg-primary);
}

:deep(.vditor-preview .vditor-reset) {
  color: var(--text-primary);
  
  /* 复用上面的样式 */
  h1, h2, h3, h4, h5, h6 {
    color: var(--text-primary);
    border-bottom-color: var(--border-base);
  }
  
  p, li {
    color: var(--text-regular);
  }
  
  a {
    color: var(--color-primary);
  }
  
  code:not(.hljs) {
    background: var(--bg-tertiary);
    color: var(--color-primary);
    font-weight: 500;
  }
  
  pre {
    background: var(--bg-secondary);
    border: 1px solid var(--border-base);
  }
  
  blockquote {
    border-left-color: var(--color-primary);
    background: var(--bg-secondary);
  }
  
  table th {
    background: var(--bg-secondary);
    border-color: var(--border-base);
  }
  
  table td {
    border-color: var(--border-base);
  }
}

/* 底部状态栏 */
:deep(.vditor-resize) {
  background-color: var(--bg-secondary);
  border-top: 1px solid var(--border-base);
}

:deep(.vditor-resize span) {
  color: var(--text-secondary);
}

/* 全屏模式 */
:deep(.vditor--fullscreen) {
  background-color: var(--bg-primary);
  z-index: 2000;
}

/* 加载提示 */
:deep(.vditor-tip) {
  background-color: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-base);
}

/* 下拉菜单 */
:deep(.vditor-hint) {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
}

:deep(.vditor-hint button) {
  color: var(--text-primary);
}

:deep(.vditor-hint button:hover),
:deep(.vditor-hint button.vditor-hint--current) {
  background-color: var(--bg-tertiary);
  color: var(--color-primary);
}

/* 上传进度 */
:deep(.vditor-upload) {
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-base);
}
</style>
