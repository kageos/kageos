<template>
  <div ref="vditorRef" class="vditor-wrapper"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'
import { useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import { ElMessage } from 'element-plus'
import { uploadFile, notifyUploadComplete } from '@/architecture/presentation/context/uploadContext'

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
  placeholder: '开始写文档...',
  disabled: false
})

const emit = defineEmits<Emits>()

const vditorRef = ref<HTMLElement>()
let vditor: Vditor | null = null

const themeStore = useThemeStore()

// 文件上传路由（与富文本一致：按用户区分路径）
const fileUploadRouter = computed(() => {
  const savedUserStr = localStorage.getItem('user')
  if (savedUserStr) {
    try {
      const savedUser = JSON.parse(savedUserStr)
      return `${savedUser.username || 'default'}/docs/files`
    } catch {
      return 'default/docs/files'
    }
  }
  return 'default/docs/files'
})

// 判断是否为视频文件（与富文本一致）
function isVideoFile(file: File): boolean {
  if (file.type.startsWith('video/')) return true
  const ext = (file.name.split('.').pop() || '').toLowerCase()
  return ['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv', 'webm', 'm4v', '3gp'].includes(ext)
}

// 上传单个文件并插入 Markdown / HTML
async function uploadOneAndInsert(file: File): Promise<string | null> {
  try {
    ElMessage.info(`正在上传 ${file.name}...`)
    const uploadResult = await uploadFile(fileUploadRouter.value, file, () => {})
    if (!uploadResult.fileInfo) return null
    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
    })
    const url = completeResult?.download_url
    if (!url) return null
    const isImage = file.type.startsWith('image/')
    const isVideo = isVideoFile(file)
    let insertContent: string
    if (isImage) {
      insertContent = `![${file.name}](${url})\n`
    } else if (isVideo) {
      // 插入 HTML 视频标签，与富文本一样直接渲染播放器（Markdown 预览用 marked 会保留 HTML）
      insertContent = `<video src="${url}" controls width="100%"></video>\n\n`
    } else {
      insertContent = `[${file.name}](${url})\n`
    }
    if (vditor) {
      vditor.insertValue(insertContent)
      emit('update:modelValue', vditor.getValue())
    }
    ElMessage.success(`${file.name} 上传成功`)
    return url
  } catch (err: any) {
    ElMessage.error(`上传 ${file.name} 失败: ${err?.message || '未知错误'}`)
    return null
  }
}

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
    
    // 上传配置：拖拽/粘贴文件时走项目统一上传并插入 Markdown
    upload: {
      accept: 'image/*,*/*',
      handler: async (files: File[]) => {
        for (const file of files) {
          await uploadOneAndInsert(file)
        }
        return ''
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
    if (disabled) {
      vditor.disabled()
    } else {
      vditor.enable()
    }
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
    if (vditor) {
      if (disabled) {
        vditor.disabled()
      } else {
        vditor.enable()
      }
    }
  }
})
</script>

<style scoped>
.vditor-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

:deep(.vditor) {
  --doc-editor-surface: var(--app-shell-panel-bg, #ffffff);
  --doc-editor-muted: var(--el-text-color-secondary, #64748b);
  --doc-editor-subtle: color-mix(in srgb, var(--bg-secondary, #f8fafc) 58%, var(--doc-editor-surface));
  --doc-editor-line: color-mix(in srgb, var(--border-base, #d8dee8) 76%, transparent);
  --doc-editor-ink: var(--el-text-color-primary, #111827);

  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--doc-editor-line);
  border-radius: 8px;
  background: var(--doc-editor-surface);
  box-shadow: 0 18px 55px -46px rgba(15, 23, 42, 0.72);
}

:deep(.vditor-toolbar) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  min-height: 46px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--doc-editor-line);
  background: color-mix(in srgb, var(--doc-editor-surface) 88%, var(--doc-editor-subtle));
}

:deep(.vditor-toolbar button) {
  width: 30px;
  height: 30px;
  margin: 0;
  border-radius: 6px;
  color: var(--doc-editor-muted);
  transition: color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease;
}

:deep(.vditor-toolbar button svg) {
  width: 15px;
  height: 15px;
}

:deep(.vditor-toolbar button:hover) {
  background: color-mix(in srgb, var(--color-primary, #1677ff) 9%, transparent);
  color: var(--color-primary, #1677ff);
}

:deep(.vditor-toolbar button.vditor-toolbar__item--current) {
  background: color-mix(in srgb, var(--color-primary, #1677ff) 13%, transparent);
  color: var(--color-primary, #1677ff);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary, #1677ff) 28%, transparent);
}

:deep(.vditor-toolbar .vditor-tooltipped::after) {
  background: var(--doc-editor-ink);
  color: var(--bg-primary, #ffffff);
  border: none;
  border-radius: 6px;
}

:deep(.vditor-toolbar .vditor-toolbar__divider) {
  width: 1px;
  height: 18px;
  margin: 0 5px;
  background: var(--doc-editor-line);
}

:deep(.vditor-ir),
:deep(.vditor-wysiwyg),
:deep(.vditor-sv),
:deep(.vditor-preview) {
  flex: 1;
  min-height: 0;
  background: var(--doc-editor-surface);
}

:deep(.vditor-ir),
:deep(.vditor-wysiwyg),
:deep(.vditor-preview) {
  overflow-y: auto;
}

:deep(.vditor-ir pre.vditor-reset),
:deep(.vditor-wysiwyg pre.vditor-reset),
:deep(.vditor-preview .vditor-reset) {
  max-width: 820px;
  min-height: 100%;
  margin: 0 auto;
  padding: 42px clamp(24px, 5vw, 64px) 72px;
  color: var(--doc-editor-ink);
  font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 16px;
  line-height: 1.75;
}

:deep(.vditor-sv .vditor-input) {
  padding: 30px clamp(22px, 4vw, 44px);
  background: var(--doc-editor-surface);
  color: var(--doc-editor-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 14px;
  line-height: 1.75;
}

:deep(.vditor-ir .vditor-reset .vditor-ir__marker) {
  color: color-mix(in srgb, var(--doc-editor-muted) 82%, transparent);
}

:deep(.vditor-reset h1),
:deep(.vditor-reset h2),
:deep(.vditor-reset h3),
:deep(.vditor-reset h4),
:deep(.vditor-reset h5),
:deep(.vditor-reset h6) {
  color: var(--doc-editor-ink);
  font-weight: 650;
  line-height: 1.28;
  letter-spacing: 0;
}

:deep(.vditor-reset h1) {
  margin: 0 0 0.85em;
  padding-bottom: 0.32em;
  border-bottom: 1px solid var(--doc-editor-line);
  font-size: 2em;
}

:deep(.vditor-reset h2) {
  margin-top: 1.9em;
  padding-bottom: 0.28em;
  border-bottom: 1px solid color-mix(in srgb, var(--doc-editor-line) 78%, transparent);
  font-size: 1.45em;
}

:deep(.vditor-reset h3) {
  margin-top: 1.65em;
  font-size: 1.18em;
}

:deep(.vditor-reset p),
:deep(.vditor-reset li) {
  color: var(--el-text-color-regular, #374151);
}

:deep(.vditor-reset a) {
  color: var(--color-primary, #1677ff);
  text-decoration-color: transparent;
  transition: text-decoration-color 0.18s ease;
}

:deep(.vditor-reset a:hover) {
  text-decoration-color: currentColor;
}

:deep(.vditor-reset code:not(.hljs)) {
  border-radius: 5px;
  background: color-mix(in srgb, var(--color-primary, #1677ff) 10%, transparent);
  color: color-mix(in srgb, var(--color-primary, #1677ff) 76%, var(--doc-editor-ink));
  font-weight: 500;
}

:deep(.vditor-reset pre) {
  border: 1px solid var(--doc-editor-line);
  border-radius: 8px;
  background: color-mix(in srgb, #111827 92%, var(--doc-editor-surface));
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.04);
}

:deep(.vditor-reset pre > code) {
  border: none;
  background: transparent;
}

:deep(.vditor-reset blockquote) {
  border-left: 3px solid color-mix(in srgb, var(--color-primary, #1677ff) 46%, var(--doc-editor-line));
  background: transparent;
  color: var(--doc-editor-muted);
}

:deep(.vditor-reset table th),
:deep(.vditor-reset table td) {
  border-color: var(--doc-editor-line);
}

:deep(.vditor-reset table th) {
  background: var(--doc-editor-subtle);
  color: var(--doc-editor-ink);
}

:deep(.vditor-reset hr) {
  background: var(--doc-editor-line);
}

:deep(.vditor-resize) {
  min-height: 30px;
  border-top: 1px solid var(--doc-editor-line);
  background: color-mix(in srgb, var(--doc-editor-surface) 86%, var(--doc-editor-subtle));
}

:deep(.vditor-resize span),
:deep(.vditor-counter) {
  color: var(--doc-editor-muted);
}

:deep(.vditor--fullscreen) {
  z-index: 2000;
  border-radius: 0;
  background: var(--doc-editor-surface);
}

:deep(.vditor-tip),
:deep(.vditor-hint),
:deep(.vditor-upload),
:deep(.vditor-panel) {
  border: 1px solid var(--doc-editor-line);
  border-radius: 8px;
  background: var(--doc-editor-surface);
  color: var(--doc-editor-ink);
  box-shadow: 0 18px 46px -34px rgba(15, 23, 42, 0.65);
}

:deep(.vditor-hint button) {
  color: var(--doc-editor-ink);
}

:deep(.vditor-hint button:hover),
:deep(.vditor-hint button.vditor-hint--current) {
  background: color-mix(in srgb, var(--color-primary, #1677ff) 9%, transparent);
  color: var(--color-primary, #1677ff);
}

@media (max-width: 768px) {
  :deep(.vditor-toolbar) {
    padding: 7px;
  }

  :deep(.vditor-ir pre.vditor-reset),
  :deep(.vditor-wysiwyg pre.vditor-reset),
  :deep(.vditor-preview .vditor-reset) {
    padding: 30px 20px 60px;
    font-size: 15px;
  }
}
</style>
