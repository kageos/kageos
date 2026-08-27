<template>
  <el-dialog
    :model-value="modelValue"
    title="导出全部数据"
    width="min(980px, calc(100vw - 40px))"
    top="5vh"
    class="table-spreadsheet-export-dialog"
    destroy-on-close
    :close-on-click-modal="!exporting"
    :close-on-press-escape="!exporting"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="export-dialog-body">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        :title="stableExport ? '已建立稳定导出快照' : '按当前筛选条件、固定 ID 升序导出'"
        :description="stableExport
          ? '本次导出使用工作空间回调固定数据边界；导出期间新增或删除记录不会扰乱所选分块。'
          : '导出不继承列表排序，并从旧记录向新记录读取，可避免新增记录扰乱分页；删除记录仍可能影响尚未读取的分块。'"
      />

      <div class="export-summary" aria-label="导出数据分析">
        <div class="summary-card">
          <span>当前筛选数据</span>
          <strong>{{ formatNumber(total) }}</strong>
          <small>条记录</small>
        </div>
        <div class="summary-card">
          <span>预计生成</span>
          <strong>{{ chunks.length }}</strong>
          <small>个 Excel</small>
        </div>
        <div class="summary-card is-primary">
          <span>已选择</span>
          <strong>{{ selectedChunks.length }}</strong>
          <small>个 Excel / {{ formatNumber(selectedRowCount) }} 条</small>
        </div>
      </div>

      <el-table :data="chunks" border stripe max-height="44vh" class="export-chunk-table">
        <el-table-column width="56" fixed="left" align="center">
          <template #header>
            <el-checkbox
              :model-value="allSelected"
              :indeterminate="partiallySelected"
              :disabled="exporting || chunks.length === 0"
              aria-label="选择全部导出分块"
              @change="toggleAll"
            />
          </template>
          <template #default="{ row }">
            <el-checkbox
              :model-value="selectedIndexes.includes(row.index)"
              :disabled="exporting"
              :aria-label="`选择第 ${row.index} 个导出分块`"
              @change="toggleChunk(row.index, $event)"
            />
          </template>
        </el-table-column>
        <el-table-column label="Excel 分块" width="120">
          <template #default="{ row }">第 {{ row.index }} 个</template>
        </el-table-column>
        <el-table-column label="数据范围" min-width="210">
          <template #default="{ row }">
            第 {{ formatNumber(row.startRow) }} - {{ formatNumber(row.endRow) }} 条
          </template>
        </el-table-column>
        <el-table-column label="记录数" width="130" align="right">
          <template #default="{ row }">{{ formatNumber(row.rowCount) }} 条</template>
        </el-table-column>
        <el-table-column label="文件名" min-width="360" show-overflow-tooltip>
          <template #default="{ row }">
            {{ exportFileName(row) }}
          </template>
        </el-table-column>
        <el-table-column v-if="exporting || completedIndexes.length" label="状态" width="110" fixed="right">
          <template #default="{ row }">
            <el-tag v-if="completedIndexes.includes(row.index)" type="success" effect="plain">已生成</el-tag>
            <el-tag v-else-if="currentIndex === row.index" type="primary" effect="plain">生成中</el-tag>
            <span v-else class="pending-status">等待</span>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="chunks.length === 0" description="当前筛选条件下没有可导出的数据" />
    </div>

    <template #footer>
      <div class="dialog-footer">
        <span class="footer-hint">
          <template v-if="exporting">
            <template v-if="completedIndexes.length === selectedChunks.length">
              Excel 已生成，正在打包下载，请勿关闭
            </template>
            <template v-else>
              正在生成第 {{ completedIndexes.length + 1 }} / {{ selectedChunks.length }} 个 Excel，请勿关闭
            </template>
          </template>
          <template v-else>选择多个分块时，将打包为一个 ZIP 下载</template>
        </span>
        <div class="footer-actions">
          <el-button :disabled="exporting" @click="emit('update:modelValue', false)">取消</el-button>
          <el-button
            type="primary"
            :loading="exporting"
            :disabled="selectedChunks.length === 0"
            @click="startExport"
          >
            导出所选 {{ selectedChunks.length }} 个 Excel
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  buildTableExportChunks,
  type TableExportChunk
} from '@/architecture/presentation/views/utils/tableSpreadsheetRuntime'
import { buildTableExportFileName } from '@/architecture/presentation/views/utils/tableSpreadsheetFile'

const props = defineProps<{
  modelValue: boolean
  total: number
  tableName: string
  blocks?: TableExportChunk[]
  stableExport?: boolean
  exportChunks: (
    chunks: TableExportChunk[],
    onProgress: (completedIndex: number, currentIndex: number | null) => void
  ) => Promise<void>
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const chunks = computed(() => props.blocks ?? buildTableExportChunks(props.total))
const selectedIndexes = ref<number[]>([])
const exporting = ref(false)
const completedIndexes = ref<number[]>([])
const currentIndex = ref<number | null>(null)

const selectedChunks = computed(() => chunks.value.filter((chunk) => selectedIndexes.value.includes(chunk.index)))
const selectedRowCount = computed(() => selectedChunks.value.reduce((sum, chunk) => sum + chunk.rowCount, 0))
const allSelected = computed(() => chunks.value.length > 0 && selectedIndexes.value.length === chunks.value.length)
const partiallySelected = computed(() => selectedIndexes.value.length > 0 && !allSelected.value)

watch(
  () => [props.modelValue, props.total] as const,
  ([visible]) => {
    if (!visible || exporting.value) return
    selectedIndexes.value = chunks.value.map((chunk) => chunk.index)
    completedIndexes.value = []
    currentIndex.value = null
  },
  { immediate: true }
)

const formatNumber = (value: number): string => value.toLocaleString('zh-CN')

const exportFileName = (chunk: TableExportChunk): string => buildTableExportFileName(props.tableName, {
  scope: 'all-filtered',
  rangeStart: chunk.startRow,
  rangeEnd: chunk.endRow
})

const toggleAll = (checked: string | number | boolean): void => {
  selectedIndexes.value = Boolean(checked) ? chunks.value.map((chunk) => chunk.index) : []
}

const toggleChunk = (index: number, checked: string | number | boolean): void => {
  if (Boolean(checked)) {
    selectedIndexes.value = [...new Set([...selectedIndexes.value, index])].sort((a, b) => a - b)
    return
  }
  selectedIndexes.value = selectedIndexes.value.filter((candidate) => candidate !== index)
}

const startExport = async (): Promise<void> => {
  if (selectedChunks.value.length === 0 || exporting.value) return
  exporting.value = true
  completedIndexes.value = []
  currentIndex.value = selectedChunks.value[0]?.index ?? null
  try {
    await props.exportChunks(selectedChunks.value, (completedIndex, nextIndex) => {
      completedIndexes.value = [...completedIndexes.value, completedIndex]
      currentIndex.value = nextIndex
    })
    ElMessage.success(`已完成 ${selectedChunks.value.length} 个 Excel，共 ${formatNumber(selectedRowCount.value)} 条数据`)
    emit('update:modelValue', false)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '导出失败，请重试')
  } finally {
    exporting.value = false
    currentIndex.value = null
  }
}
</script>

<style scoped lang="scss">
:global(.table-spreadsheet-export-dialog) {
  max-height: 90vh;
  margin-bottom: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

:global(.table-spreadsheet-export-dialog .el-dialog__header) {
  flex: 0 0 auto;
  padding: 20px 24px 16px;
}

:global(.table-spreadsheet-export-dialog .el-dialog__body) {
  min-height: 0;
  padding: 18px 24px;
  overflow: hidden;
}

:global(.table-spreadsheet-export-dialog .el-dialog__footer) {
  flex: 0 0 auto;
  padding: 16px 24px 20px;
}

.export-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.export-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.summary-card {
  min-height: 82px;
  padding: 12px 14px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-light));
  border-radius: 10px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: end;
  column-gap: 8px;
}

.summary-card span {
  grid-column: 1 / -1;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.summary-card strong {
  font-size: 26px;
  line-height: 1;
  color: var(--el-text-color-primary);
}

.summary-card small {
  color: var(--el-text-color-secondary);
}

.summary-card.is-primary {
  border-color: color-mix(in srgb, var(--el-color-primary) 42%, var(--el-border-color));
  background: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color));
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.summary-card.is-primary span,
.summary-card.is-primary small {
  color: color-mix(in srgb, var(--el-color-primary) 58%, var(--el-text-color-primary));
}

.summary-card.is-primary strong {
  color: var(--el-color-primary);
}

.export-chunk-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.export-chunk-table :deep(.el-table__header th.el-table__cell) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  color: var(--el-text-color-regular);
}

.export-chunk-table :deep(.el-table__body td.el-table__cell) {
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
}

.pending-status,
.footer-hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.footer-actions {
  display: flex;
  gap: 10px;
}

@media (max-width: 720px) {
  :global(.table-spreadsheet-export-dialog .el-dialog__header),
  :global(.table-spreadsheet-export-dialog .el-dialog__body),
  :global(.table-spreadsheet-export-dialog .el-dialog__footer) {
    padding-left: 16px;
    padding-right: 16px;
  }

  .export-summary {
    grid-template-columns: 1fr;
  }

  .dialog-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-actions {
    justify-content: flex-end;
  }
}
</style>
