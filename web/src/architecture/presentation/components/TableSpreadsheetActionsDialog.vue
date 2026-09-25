<template>
  <el-dialog
    :model-value="modelValue"
    title="导入 / 导出"
    width="min(760px, calc(100vw - 32px))"
    top="10vh"
    class="spreadsheet-actions-dialog"
    :close-on-click-modal="!busy"
    :close-on-press-escape="!busy"
    :show-close="!busy"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="transfer-context">
      <el-icon><FolderOpened /></el-icon>
      <span class="transfer-context-name">{{ tableName }}</span>
      <el-tag size="small" effect="plain">{{ total.toLocaleString() }} 条筛选结果</el-tag>
    </div>
    <section v-for="section in sections" :key="section.title" class="transfer-section">
      <h3>{{ section.title }}</h3>
      <div class="transfer-grid">
        <button
          v-for="action in section.actions"
          :key="action.command"
          type="button"
          class="transfer-card"
          :disabled="busy || action.disabled"
          @click="emit('command', action.command)"
        >
          <span class="transfer-icon"><el-icon><component :is="action.icon" /></el-icon></span>
          <strong>{{ action.title }}</strong>
          <span class="transfer-description">{{ action.description }}</span>
          <span class="transfer-meta">{{ action.hint }}<el-icon v-if="!action.disabled"><Right /></el-icon></span>
        </button>
      </div>
    </section>
    <template #footer>
      <div class="transfer-footer">
        <span role="status">{{ busy ? '正在准备文件，请稍候…' : '导入前可预览检查；导出范围以当前筛选为准。' }}</span>
        <el-button :disabled="busy" @click="emit('update:modelValue', false)">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { DocumentAdd, Download, FolderOpened, Right, Upload } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: boolean
  busy: boolean
  canImport: boolean
  tableName: string
  total: number
  currentPage: number
  pageCount: number
}>()
const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'command', value: string): void
}>()
const sections = computed(() => [
  {
    title: '导入数据',
    actions: [
      { command: 'template', icon: DocumentAdd, title: '下载导入模板', description: '按当前表格字段生成模板，填写后即可导入。', hint: props.canImport ? 'Excel 模板' : '当前表格不支持新增', disabled: !props.canImport },
      { command: 'import', icon: Upload, title: '从文件导入', description: '选择文件、预览并校验，确认后写入表格。', hint: props.canImport ? 'Excel / CSV · 最多 500 行 · 8 MB' : '当前表格不支持新增', disabled: !props.canImport }
    ]
  },
  {
    title: '导出数据',
    actions: [
      { command: 'export-current-page', icon: Download, title: '导出当前页', description: '下载当前页显示的数据，保留列表排序。', hint: props.pageCount ? `第 ${props.currentPage} 页 · ${props.pageCount} 条记录` : '当前页暂无数据', disabled: props.pageCount === 0 },
      { command: 'export-all', icon: FolderOpened, title: '导出全部筛选结果', description: '按当前筛选生成导出计划，可选择分块下载。', hint: props.total ? `共 ${props.total.toLocaleString()} 条 · 自动分块` : '当前筛选下暂无数据', disabled: props.total === 0 }
    ]
  }
])
</script>

<style scoped>
.transfer-context { display: flex; align-items: center; gap: 10px; padding: 12px 16px; background: var(--el-fill-color-light); border-radius: 10px; color: var(--el-text-color-regular); }
.transfer-context-name { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.transfer-section { margin-top: 24px; }
.transfer-section h3 { margin: 0 0 12px; font-size: 14px; color: var(--el-text-color-primary); }
.transfer-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.transfer-card { display: flex; flex-direction: column; align-items: flex-start; gap: 10px; padding: 20px; border: 1px solid var(--el-border-color-light); border-radius: 12px; background: var(--el-bg-color); color: var(--el-text-color-primary); text-align: left; font: inherit; cursor: pointer; transition: border-color .18s, background .18s; }
/* Keep an opaque theme surface: Element Plus light tints stay pale in classic dark mode. */
.transfer-card:hover:not(:disabled),
.transfer-card:focus-visible { border-color: var(--el-color-primary); background: var(--bg-secondary, var(--el-bg-color)); color: var(--el-text-color-primary); box-shadow: inset 0 0 0 1px var(--el-color-primary); }
.transfer-card:hover:not(:disabled) .transfer-description,
.transfer-card:focus-visible .transfer-description { color: var(--el-text-color-primary); }
.transfer-card:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 3px; }
.transfer-card:disabled { cursor: not-allowed; opacity: .55; }
.transfer-icon { display: grid; place-items: center; width: 36px; height: 36px; border-radius: 10px; background: var(--bg-secondary, var(--el-bg-color)); color: var(--el-color-primary); font-size: 20px; }
.transfer-card strong { font-size: 15px; }
.transfer-description { color: var(--el-text-color-regular); font-size: 13px; line-height: 1.7; }
.transfer-meta { display: flex; justify-content: space-between; align-items: center; gap: 8px; width: 100%; margin-top: auto; padding-top: 6px; font-size: 12px; color: var(--el-text-color-regular); }
.transfer-footer { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-top: 8px; }
.transfer-footer > span { text-align: left; color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.6; }
@media (max-width: 560px) { .transfer-grid { grid-template-columns: 1fr; } .transfer-card { padding: 16px; } .transfer-context { flex-wrap: wrap; } }
</style>
