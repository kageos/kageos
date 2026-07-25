<template>
  <el-dialog
    :model-value="modelValue"
    title="导入表格数据"
    width="min(1440px, 96vw)"
    top="4vh"
    class="table-spreadsheet-import-dialog"
    destroy-on-close
    :close-on-click-modal="!importing"
    :close-on-press-escape="!importing"
    @update:model-value="emit('update:modelValue', $event)"
    @closed="reset"
  >
    <div class="import-dialog-body">
      <input
        ref="fileInput"
        class="file-input"
        type="file"
        accept=".xlsx,.csv"
        @change="handleFileChange"
      >

      <div v-if="!preview" class="upload-placeholder" @click="selectFile">
        <el-icon class="upload-icon"><UploadFilled /></el-icon>
        <strong>选择 Excel 或 CSV 文件</strong>
        <span>先解析并预览，确认后才会写入；单次最多 500 行、文件不超过 8 MB。</span>
        <el-button type="primary" plain>选择文件</el-button>
      </div>

      <template v-else>
        <div class="preview-toolbar">
          <div class="file-summary">
            <el-icon class="file-icon"><Document /></el-icon>
            <div>
              <strong>{{ fileName }}</strong>
              <span>已完成解析，请确认检查结果</span>
            </div>
          </div>
          <div class="preview-actions">
            <el-checkbox
              v-if="invalidRows.length"
              v-model="showOnlyErrors"
              label="只看错误行"
              :disabled="importing"
            />
            <el-button :disabled="importing" @click="selectFile">重新选择文件</el-button>
          </div>
        </div>

        <div class="validation-summary" aria-label="导入校验概览">
          <div class="summary-card">
            <span>数据行</span>
            <strong>{{ preview.rows.length }}</strong>
          </div>
          <div class="summary-card is-success">
            <span>校验通过</span>
            <strong>{{ validRows.length }}</strong>
          </div>
          <div class="summary-card" :class="invalidRows.length ? 'is-danger' : 'is-muted'">
            <span>需要修正</span>
            <strong>{{ invalidRows.length }}</strong>
          </div>
          <div class="summary-card" :class="validationIssueCount ? 'is-danger' : 'is-muted'">
            <span>问题数量</span>
            <strong>{{ validationIssueCount }}</strong>
          </div>
        </div>

        <el-alert
          v-if="preview.fatalErrors.length"
          type="error"
          :closable="false"
          show-icon
          :title="preview.fatalErrors.join('；')"
        />
        <el-alert
          v-else-if="invalidRows.length"
          type="error"
          :closable="false"
          show-icon
          :title="`发现 ${invalidRows.length} 行数据未通过校验，暂不能导入`"
          description="请按表格右侧的检查结果修改原文件，再重新选择文件。全部数据通过后才会开放确认导入。"
        />
        <el-alert
          v-if="preview.fatalErrors.length === 0 && preview.ignoredHeaders.length"
          type="warning"
          :closable="false"
          show-icon
          :title="`以下列不属于当前新增表单，导入时会忽略：${preview.ignoredHeaders.join('、')}`"
        />
        <el-alert
          v-if="importResult"
          :type="importResult.failedCount > 0 ? 'warning' : 'success'"
          :closable="false"
          show-icon
          :title="`导入完成：成功 ${importResult.createdCount} 行，失败 ${importResult.failedCount} 行`"
        />

        <el-table
          :data="displayRows"
          max-height="56vh"
          border
          stripe
          class="preview-table"
          :row-class-name="previewRowClassName"
        >
          <el-table-column prop="rowNumber" label="原文件行" width="88" fixed="left" />
          <el-table-column
            v-for="field in preview.recognizedFields"
            :key="field.code"
            :label="field.name"
            min-width="150"
          >
            <template #default="{ row }">
              <span class="preview-cell">{{ displayCell(row.data[field.code]) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="检查结果" min-width="300" fixed="right">
            <template #default="{ row }">
              <span v-if="rowErrors(row).length" class="row-error">
                {{ rowErrors(row).join('；') }}
              </span>
              <span v-else class="row-valid">{{ importResult ? '已导入' : '可以导入' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <div v-if="preview && !importResult" class="footer-validation" :class="canImport ? 'is-success' : 'is-danger'">
          <el-icon><CircleCheckFilled v-if="canImport" /><WarningFilled v-else /></el-icon>
          <span>{{ importStatusText }}</span>
        </div>
        <div class="footer-actions">
          <el-button :disabled="importing" @click="emit('update:modelValue', false)">关闭</el-button>
          <el-button
            v-if="preview && !importResult"
            type="primary"
            :loading="importing"
            :disabled="!canImport"
            @click="confirmImport"
          >
            {{ canImport ? `确认导入 ${preview.rows.length} 行` : '校验通过后才能导入' }}
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { CircleCheckFilled, Document, UploadFilled, WarningFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { FieldConfig } from '@/architecture/domain/types'
import {
  parseTableSpreadsheetFile
} from '@/architecture/presentation/views/utils/tableSpreadsheetFile'
import type {
  TableImportPreview,
  TableImportRow
} from '@/architecture/presentation/views/utils/tableSpreadsheetRuntime'
import { isTableImportPreviewSubmittable } from '@/architecture/presentation/views/utils/tableSpreadsheetRuntime'

interface BatchImportResult {
  createdCount: number
  failedCount: number
  errors: Array<{ rowNumber: number, message: string }>
}

const props = defineProps<{
  modelValue: boolean
  fields: FieldConfig[]
  importRows: (rows: Array<{ rowNumber: number, data: Record<string, unknown> }>) => Promise<BatchImportResult>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  imported: [createdCount: number]
}>()

const fileInput = ref<HTMLInputElement>()
const fileName = ref('')
const preview = ref<TableImportPreview>()
const importing = ref(false)
const importResult = ref<BatchImportResult>()
const parseError = ref('')
const showOnlyErrors = ref(false)

const validRows = computed(() => preview.value?.rows.filter((row) => row.errors.length === 0) || [])
const invalidRows = computed(() => preview.value?.rows.filter((row) => row.errors.length > 0) || [])
const validationIssueCount = computed(() => invalidRows.value.reduce((total, row) => total + row.errors.length, 0))
const displayRows = computed(() => showOnlyErrors.value ? invalidRows.value : (preview.value?.rows || []))
const canImport = computed(() => (
  !importing.value
  && Boolean(preview.value)
  && !importResult.value
  && isTableImportPreviewSubmittable(preview.value!)
))
const importStatusText = computed(() => {
  if (!preview.value) return ''
  if (preview.value.fatalErrors.length) return '文件检查未通过，请处理上方问题后重新选择文件'
  if (invalidRows.value.length) {
    return `还有 ${invalidRows.value.length} 行、${validationIssueCount.value} 个问题需要修正`
  }
  if (preview.value.rows.length === 0) return '文件中没有可以导入的数据'
  return `全部 ${preview.value.rows.length} 行已通过文件校验，可以提交导入`
})

const backendErrorMap = computed(() => {
  return new Map((importResult.value?.errors || []).map((error) => [error.rowNumber, error.message]))
})

const selectFile = () => fileInput.value?.click()

const handleFileChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = ''
  if (!file) return

  fileName.value = file.name
  preview.value = undefined
  importResult.value = undefined
  parseError.value = ''
  try {
    preview.value = await parseTableSpreadsheetFile(file, props.fields)
    showOnlyErrors.value = preview.value.rows.some((row) => row.errors.length > 0)
  } catch (error) {
    parseError.value = error instanceof Error ? error.message : String(error)
    preview.value = {
      rows: [],
      recognizedFields: [],
      ignoredHeaders: [],
      fatalErrors: [parseError.value]
    }
    showOnlyErrors.value = false
  }
}

const rowErrors = (row: TableImportRow): string[] => {
  const backendError = backendErrorMap.value.get(row.rowNumber)
  return backendError ? [...row.errors, `写入失败：${backendError}`] : row.errors
}

const displayCell = (value: unknown): string => {
  if (value === null || value === undefined || value === '') return '—'
  if (Array.isArray(value)) return value.join('、')
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

const previewRowClassName = ({ row }: { row: TableImportRow }): string => {
  return rowErrors(row).length ? 'import-row-invalid' : 'import-row-valid'
}

const confirmImport = async () => {
  if (!canImport.value) return
  importing.value = true
  try {
    importResult.value = await props.importRows(preview.value!.rows.map((row) => ({
      rowNumber: row.rowNumber,
      data: row.data
    })))
    if (importResult.value.createdCount > 0) emit('imported', importResult.value.createdCount)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '批量导入失败')
  } finally {
    importing.value = false
  }
}

const reset = () => {
  fileName.value = ''
  preview.value = undefined
  importResult.value = undefined
  parseError.value = ''
  importing.value = false
  showOnlyErrors.value = false
}
</script>

<style scoped>
:global(.table-spreadsheet-import-dialog) {
  display: flex;
  flex-direction: column;
  max-height: 92vh;
}

:global(.table-spreadsheet-import-dialog .el-dialog__body) {
  min-height: 0;
  overflow: auto;
}

.import-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 360px;
}

.file-input {
  display: none;
}

.upload-placeholder {
  min-height: 420px;
  border: 1px dashed var(--el-border-color);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  cursor: pointer;
  color: var(--el-text-color-secondary);
}

.upload-placeholder strong {
  color: var(--el-text-color-primary);
  font-size: 16px;
}

.upload-icon {
  font-size: 42px;
  color: var(--el-color-primary);
}

.preview-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.file-summary {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 12px;
}

.file-summary > div {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 4px;
}

.file-summary strong {
  overflow: hidden;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-summary span {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.file-icon {
  flex: 0 0 auto;
  padding: 9px;
  border-radius: 10px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 22px;
}

.preview-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 16px;
}

.validation-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(130px, 1fr));
  gap: 10px;
}

.summary-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 58px;
  padding: 10px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-regular);
}

.summary-card strong {
  color: var(--el-text-color-primary);
  font-size: 22px;
  font-variant-numeric: tabular-nums;
}

.summary-card.is-success {
  border-color: var(--el-color-success-light-7);
  background: var(--el-color-success-light-9);
}

.summary-card.is-success strong {
  color: var(--el-color-success);
}

.summary-card.is-danger {
  border-color: var(--el-color-danger-light-7);
  background: var(--el-color-danger-light-9);
}

.summary-card.is-danger strong {
  color: var(--el-color-danger);
}

.summary-card.is-muted strong {
  color: var(--el-text-color-placeholder);
}

.preview-table {
  width: 100%;
}

.preview-cell {
  white-space: pre-wrap;
  word-break: break-word;
}

.row-error {
  color: var(--el-color-danger);
  white-space: normal;
}

.row-valid {
  color: var(--el-color-success);
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  width: 100%;
}

.footer-validation {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
  text-align: left;
}

.footer-validation.is-success {
  color: var(--el-color-success);
}

.footer-validation.is-danger {
  color: var(--el-color-danger);
}

.footer-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 10px;
}

:deep(.import-row-invalid td.el-table__cell) {
  background: var(--el-color-danger-light-9) !important;
}

@media (max-width: 760px) {
  .preview-toolbar,
  .dialog-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .preview-actions,
  .footer-actions {
    justify-content: flex-end;
  }

  .validation-summary {
    grid-template-columns: repeat(2, minmax(120px, 1fr));
  }
}
</style>
