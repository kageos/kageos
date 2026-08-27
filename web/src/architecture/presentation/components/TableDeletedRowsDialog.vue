<template>
  <el-dialog
    :model-value="modelValue"
    title="回收站"
    width="min(1680px, 98vw)"
    top="2vh"
    class="recycle-bin-dialog"
    destroy-on-close
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="recycle-toolbar">
      <div class="recycle-toolbar-left">
        <el-input
          v-model="keyword"
          clearable
          placeholder="搜索当前页记录"
          class="recycle-search"
        />
        <span class="recycle-hint">共 {{ total }} 条已删除记录，自动清理由平台保留策略决定。</span>
      </div>
      <div class="recycle-toolbar-actions">
        <el-button :loading="loading || policyLoading" @click="refreshRecycleBin">刷新</el-button>
        <el-button
          type="danger"
          plain
          :disabled="selectedRows.length === 0"
          :loading="purging"
          @click="purge(selectedRows)"
        >
          彻底删除选中（{{ selectedRows.length }}）
        </el-button>
        <el-button
          type="primary"
          :disabled="selectedRows.length === 0"
          :loading="restoring"
          @click="restore(selectedRows)"
        >
          恢复选中（{{ selectedRows.length }}）
        </el-button>
      </div>
    </div>

    <section class="recycle-policy-card" v-loading="policyLoading">
      <div class="recycle-policy-summary">
        <div>
          <div class="recycle-policy-title">
            自动清理策略
            <el-tag size="small" :type="recyclePolicy?.source === 'table' ? 'primary' : 'info'">
              {{ recyclePolicy?.source === 'table' ? '当前表策略' : '平台默认策略' }}
            </el-tag>
          </div>
          <div class="recycle-policy-description">
            <template v-if="recyclePolicy?.enabled">
              {{ recyclePolicy.mode === 'purge' ? '自动清理已启用' : '仅预演，不会实际删除' }}，保留最近
              {{ recyclePolicy.retention_days }} 天，每 {{ recyclePolicy.interval_minutes }} 分钟检查一次。
            </template>
            <template v-else>自动清理未启用，已删除记录会一直保留，直到管理员手动彻底删除。</template>
          </div>
          <div v-if="recyclePolicy?.updated_by" class="recycle-policy-updater">
            最近修改人：{{ recyclePolicy.updated_by }}
          </div>
        </div>
        <el-button v-if="!policyEditing" type="primary" plain @click="startPolicyEditing">
          编辑策略
        </el-button>
      </div>

      <el-form v-if="policyEditing" class="recycle-policy-form" inline label-position="top">
        <el-form-item label="自动检查">
          <el-switch v-model="policyForm.enabled" active-text="启用" inactive-text="停用" />
        </el-form-item>
        <el-form-item label="执行方式">
          <el-select v-model="policyForm.mode" style="width: 150px">
            <el-option label="仅预演（不删除）" value="dry_run" />
            <el-option label="自动彻底删除" value="purge" />
          </el-select>
        </el-form-item>
        <el-form-item label="保留天数">
          <el-input-number v-model="policyForm.retention_days" :min="1" :max="3650" />
        </el-form-item>
        <el-form-item label="权限说明" class="recycle-policy-permission">
          <span>策略修改、恢复和彻底删除均需要当前函数的 Admin 或 Owner 权限。</span>
        </el-form-item>
        <el-form-item class="recycle-policy-form-actions">
          <el-button @click="policyEditing = false">取消</el-button>
          <el-button type="primary" :loading="policySaving" @click="saveRecyclePolicy">保存策略</el-button>
        </el-form-item>
      </el-form>
    </section>

    <el-table
      v-loading="loading"
      :data="filteredRows"
      :stripe="false"
      row-key="id"
      style="width: 100%"
      max-height="62vh"
      empty-text="回收站为空"
      @selection-change="selectedRows = $event"
      @row-click="handleRowClick"
      class="recycle-bin-table table-with-fixed-column table-row-clickable"
    >
      <el-table-column type="selection" width="55" fixed="left" />
      <el-table-column
        prop="id"
        label=""
        width="132"
        fixed="left"
        class-name="control-column"
        label-class-name="control-column"
      >
        <template #default="scope">
          <button
            type="button"
            class="detail-icon-button"
            :title="`查看详情 ${formatValue(scope.row.id)}`"
            @click.stop="showDetails(scope.row)"
          >
            <el-icon><View /></el-icon>
            <span class="detail-id-text">{{ formatValue(scope.row.id) }}</span>
          </button>
        </template>
      </el-table-column>
      <el-table-column
        v-for="field in displayFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name || field.code"
        class-name="table-data-column"
        :min-width="getColumnWidth(field)"
        show-overflow-tooltip
      >
        <template #default="scope">
          <WidgetComponent
            :field="field"
            :value="getRowFieldValue(scope.row, field)"
            mode="table-cell"
            :row-data="scope.row"
          />
        </template>
      </el-table-column>
      <el-table-column label="删除时间" min-width="180" class-name="table-data-column">
        <template #default="scope">{{ formatDeletedAt(scope.row.deleted_at) }}</template>
      </el-table-column>
      <el-table-column label="删除人" min-width="140" class-name="table-data-column" show-overflow-tooltip>
        <template #default="scope">{{ formatDeletedBy(scope.row.deleted_by) }}</template>
      </el-table-column>
      <el-table-column
        label="操作"
        width="90"
        fixed="right"
        class-name="action-column"
        label-class-name="action-column"
      >
        <template #default="scope">
          <el-dropdown
            trigger="click"
            placement="bottom-end"
            popper-class="table-action-dropdown"
            @click.stop
            @command="(command: string) => handleActionCommand(command, scope.row)"
          >
            <el-button size="small" class="action-more-btn">
              <el-icon><More /></el-icon>
              更多
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="detail">查看详情</el-dropdown-item>
                <el-dropdown-item command="restore" :disabled="restoring">恢复记录</el-dropdown-item>
                <el-dropdown-item command="purge" divided :disabled="purging">
                  <span class="danger-action-text">彻底删除</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <div class="recycle-pagination">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next"
        @current-change="loadRows"
        @size-change="handleSizeChange"
      />
    </div>

    <el-drawer v-model="detailsVisible" title="删除记录详情" size="min(1080px, 96vw)" append-to-body>
      <template v-if="detailRow">
        <section class="detail-section">
          <h3>业务信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item
              v-for="entry in businessDetailEntries"
              :key="entry.code"
              :label="entry.label"
            >
              <OperateLogFieldValue
                :field="entry.field"
                :raw-value="entry.value"
                compact
              />
            </el-descriptions-item>
          </el-descriptions>
          <el-empty v-if="businessDetailEntries.length === 0" description="暂无业务字段" :image-size="72" />
        </section>

        <section class="detail-section">
          <h3>删除信息</h3>
          <el-descriptions :column="1" border>
            <el-descriptions-item
              v-for="entry in deletionDetailEntries"
              :key="entry.code"
              :label="entry.label"
            >
              <div class="detail-value">{{ entry.value }}</div>
            </el-descriptions-item>
          </el-descriptions>
        </section>

        <el-collapse v-if="systemDetailEntries.length > 0" class="detail-system-collapse">
          <el-collapse-item title="系统信息（技术字段）" name="system">
            <el-descriptions :column="1" border>
              <el-descriptions-item
                v-for="entry in systemDetailEntries"
                :key="entry.code"
                :label="entry.label"
              >
                <div class="detail-value">{{ formatValue(entry.value) }}</div>
              </el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ElIcon, ElMessage, ElMessageBox } from 'element-plus'
import { More, View } from '@element-plus/icons-vue'
import type { FieldConfig, FieldValue, FunctionDetail, TableRow } from '@/architecture/domain/types'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { getTableAllFields, getTableListFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import {
  tableGetDeletedRows,
  tableGetRecyclePolicy,
  tablePurgeRows,
  tableRestoreRows,
  tableUpdateRecyclePolicy,
  type TableRecyclePolicy
} from '@/architecture/presentation/context/api/function'
import { formatDateTimeValue } from '@/architecture/shared/date'
import OperateLogFieldValue from '@/architecture/presentation/components/OperateLogFieldValue.vue'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { resolveTableColumnWidth } from '@/architecture/presentation/views/utils/tableColumnWidth'

const props = defineProps<{
  modelValue: boolean
  functionDetail: FunctionDetail
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  restored: []
}>()

const rows = ref<Array<Record<string, unknown>>>([])
const selectedRows = ref<Array<Record<string, unknown>>>([])
const loading = ref(false)
const restoring = ref(false)
const purging = ref(false)
const policyLoading = ref(false)
const policySaving = ref(false)
const policyEditing = ref(false)
const recyclePolicy = ref<TableRecyclePolicy | null>(null)
const policyForm = ref<Pick<TableRecyclePolicy, 'enabled' | 'mode' | 'retention_days'>>({
  enabled: false,
  mode: 'dry_run',
  retention_days: 30
})
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const detailsVisible = ref(false)
const detailRow = ref<Record<string, unknown> | null>(null)

const displayFields = computed<FieldConfig[]>(() => getTableListFields(props.functionDetail)
  .filter((field) => !['id', 'deleted_at', 'deleted_by'].includes(field.code)))

const detailFields = computed<FieldConfig[]>(() => getTableAllFields(props.functionDetail)
  .filter((field) => !systemFieldCodes.has(field.code)))

const filteredRows = computed(() => {
  const target = keyword.value.trim().toLowerCase()
  if (!target) return rows.value
  return rows.value.filter((row) => Object.values(row).some((value) => formatValue(value).toLowerCase().includes(target)))
})

const businessDetailEntries = computed(() => {
  if (!detailRow.value) return []
  return detailFields.value
    .filter((field) => Object.prototype.hasOwnProperty.call(detailRow.value, field.code))
    .map((field) => ({
      code: field.code,
      label: field.name && field.name !== field.code ? field.name : humanizeFieldCode(field.code),
      field,
      value: detailRow.value?.[field.code]
    }))
})

const deletionDetailEntries = computed(() => [
  { code: 'id', label: '记录 ID', value: formatValue(detailRow.value?.id) },
  { code: 'deleted_at', label: '删除时间', value: formatDeletedAt(detailRow.value?.deleted_at) },
  { code: 'deleted_by', label: '删除人', value: formatDeletedBy(detailRow.value?.deleted_by) }
])

const systemDetailEntries = computed(() => {
  if (!detailRow.value) return []
  const businessCodes = new Set(detailFields.value.map((field) => field.code))
  return Object.entries(detailRow.value)
    .filter(([code]) => !businessCodes.has(code) && !deletionFieldCodes.has(code))
    .map(([code, value]) => ({
      code,
      label: systemFieldLabels[code] || humanizeFieldCode(code),
      value: dateTimeFieldCodes.has(code) ? formatDateTime(value) : value
    }))
})

const routerPath = computed(() => props.functionDetail.full_code_path || props.functionDetail.router || '')

const loadRows = async () => {
  if (!routerPath.value) return
  loading.value = true
  try {
    const result = await tableGetDeletedRows(routerPath.value, page.value, pageSize.value)
    rows.value = Array.isArray(result.rows) ? result.rows : []
    total.value = Number(result.total || 0)
    selectedRows.value = []
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载已删除记录失败')
  } finally {
    loading.value = false
  }
}

const loadRecyclePolicy = async () => {
  if (!routerPath.value) return
  policyLoading.value = true
  try {
    recyclePolicy.value = await tableGetRecyclePolicy(routerPath.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '加载回收站策略失败')
  } finally {
    policyLoading.value = false
  }
}

const refreshRecycleBin = async () => {
  await Promise.all([loadRows(), loadRecyclePolicy()])
}

const startPolicyEditing = () => {
  if (recyclePolicy.value) {
    policyForm.value = {
      enabled: recyclePolicy.value.enabled,
      mode: recyclePolicy.value.mode,
      retention_days: recyclePolicy.value.retention_days
    }
  }
  policyEditing.value = true
}

const saveRecyclePolicy = async () => {
  if (!routerPath.value) return
  if (policyForm.value.enabled && policyForm.value.mode === 'purge') {
    try {
      await ElMessageBox.confirm(
        `启用后，超过 ${policyForm.value.retention_days} 天的回收站记录会被自动彻底删除且无法恢复。确认保存？`,
        '确认自动清理策略',
        { type: 'warning', confirmButtonText: '确认保存', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
  }
  policySaving.value = true
  try {
    recyclePolicy.value = await tableUpdateRecyclePolicy(routerPath.value, policyForm.value)
    policyEditing.value = false
    ElMessage.success('回收站自动清理策略已保存')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存回收站策略失败')
  } finally {
    policySaving.value = false
  }
}

// 地址栏冷启动时组件可能以 modelValue=true 直接挂载，不能只依赖
// el-dialog 的 open 事件；显式监听可确保直达链接一定发起回收站请求。
watch(
  () => props.modelValue,
  async (visible) => {
    if (!visible) return
    await nextTick()
    await refreshRecycleBin()
  },
  { immediate: true }
)

const handleSizeChange = () => {
  page.value = 1
  void loadRows()
}

const showDetails = (row: Record<string, unknown>) => {
  detailRow.value = row
  detailsVisible.value = true
}

const getRowFieldValue = (row: Record<string, unknown>, field: FieldConfig): FieldValue => {
  const value = row[field.code]
  if (value === null || value === undefined || value === '') {
    return createEmptyRawFieldValue()
  }
  return createAutoFieldValue(value, field)
}

const getColumnWidth = (field: FieldConfig): number => {
  return resolveTableColumnWidth(field, rows.value as TableRow[])
}

const handleRowClick = (
  row: Record<string, unknown>,
  _column: unknown,
  event: MouseEvent
) => {
  const target = event.target as HTMLElement | null
  if (target?.closest('.el-checkbox, .action-column, .el-dropdown')) return
  showDetails(row)
}

const handleActionCommand = (command: string, row: Record<string, unknown>) => {
  if (command === 'detail') {
    showDetails(row)
  } else if (command === 'restore') {
    void restore([row])
  } else if (command === 'purge') {
    void purge([row])
  }
}

const restore = async (targets: Array<Record<string, unknown>>) => {
  const ids = targets.map((row) => Number(row.id)).filter((id) => Number.isInteger(id) && id > 0)
  if (ids.length === 0) return
  try {
    await ElMessageBox.confirm(`确认恢复选中的 ${ids.length} 条记录？`, '恢复记录', {
      type: 'warning',
      confirmButtonText: '恢复',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  restoring.value = true
  try {
    const result = await tableRestoreRows(routerPath.value, ids)
    ElMessage.success(`已恢复 ${result.restored || ids.length} 条记录`)
    if (rows.value.length === ids.length && page.value > 1) page.value -= 1
    await loadRows()
    emit('restored')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '恢复记录失败')
  } finally {
    restoring.value = false
  }
}

const purge = async (targets: Array<Record<string, unknown>>) => {
  const ids = targets.map((row) => Number(row.id)).filter((id) => Number.isInteger(id) && id > 0)
  if (ids.length === 0) return
  try {
    await ElMessageBox.prompt(
      `选中的 ${ids.length} 条记录将立即从数据库中删除，操作日志仅保留删除快照。请输入“彻底删除”确认。`,
      '彻底删除记录',
      {
        type: 'error',
        confirmButtonText: '彻底删除',
        cancelButtonText: '取消',
        inputPattern: /^彻底删除$/,
        inputErrorMessage: '请输入“彻底删除”'
      }
    )
  } catch {
    return
  }
  purging.value = true
  try {
    const result = await tablePurgeRows(routerPath.value, ids)
    ElMessage.success(`已彻底删除 ${result.purged || ids.length} 条记录`)
    if (rows.value.length === ids.length && page.value > 1) page.value -= 1
    detailsVisible.value = false
    await loadRows()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '彻底删除失败')
  } finally {
    purging.value = false
  }
}

const formatValue = (value: unknown): string => {
  if (value === null || value === undefined || value === '') return '-'
  if (typeof value === 'object') {
    try { return JSON.stringify(value) } catch { return String(value) }
  }
  return String(value)
}

const formatDeletedAt = (value: unknown): string => {
  if (value && typeof value === 'object' && 'Time' in value) {
    return formatDateTime((value as { Time?: unknown }).Time)
  }
  return formatDateTime(value)
}

const formatDeletedBy = (value: unknown): string => {
  const text = formatValue(value)
  return text === '-' ? '历史记录未记录' : text
}

function formatDateTime(value: unknown): string {
  if (typeof value === 'string' || typeof value === 'number' || value instanceof Date) {
    return formatDateTimeValue(value)
  }
  return '-'
}

function humanizeFieldCode(code: string): string {
  return code
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (letter) => letter.toUpperCase())
}

const deletionFieldCodes = new Set(['id', 'deleted_at', 'deleted_by'])
const systemFieldCodes = new Set([
  ...deletionFieldCodes,
  'created_at', 'created_by', 'updated_at', 'updated_by'
])
const dateTimeFieldCodes = new Set(['created_at', 'updated_at'])
const systemFieldLabels: Record<string, string> = {
  created_at: '创建时间',
  created_by: '创建人',
  updated_at: '更新时间',
  updated_by: '更新人'
}
</script>

<style scoped>
.recycle-toolbar,
.recycle-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.recycle-policy-card {
  margin-bottom: 14px;
  padding: 14px 16px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 10px;
  background: var(--app-shell-panel-muted-bg);
}

.recycle-policy-summary {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.recycle-policy-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-primary);
  font-weight: 700;
}

.recycle-policy-description,
.recycle-policy-updater {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.recycle-policy-updater {
  font-size: 12px;
}

.recycle-policy-form {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--app-shell-panel-border);
}

.recycle-policy-form :deep(.el-form-item) {
  margin: 0;
}

.recycle-policy-permission {
  flex: 1;
  min-width: 260px;
  color: var(--el-text-color-secondary);
}

.recycle-policy-form-actions {
  margin-left: auto !important;
}

.recycle-toolbar {
  margin-bottom: 14px;
}

.recycle-toolbar-left,
.recycle-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.recycle-toolbar-left {
  min-width: 0;
}

.recycle-search {
  width: 240px;
  flex: 0 0 auto;
}

.recycle-hint {
  color: var(--el-text-color-secondary);
}

.recycle-pagination {
  justify-content: flex-end;
  margin-top: 16px;
}

/* 回收站与 TableView 共用同一套表格视觉密度和单元格行为。 */
:deep(.recycle-bin-table.el-table) {
  background-color: var(--app-shell-panel-bg-strong) !important;
  border: 1px solid var(--app-shell-panel-border) !important;
  border-radius: 8px !important;
  box-shadow: var(--app-shell-panel-shadow-soft);
  overflow: hidden;
}

:deep(.recycle-bin-table .el-table__inner-wrapper) {
  border: none !important;
  border-radius: 8px !important;
}

:deep(.recycle-bin-table .el-table__inner-wrapper::before) {
  display: none !important;
}

:deep(.recycle-bin-table .el-table__header-wrapper),
:deep(.recycle-bin-table .el-table__body-wrapper) {
  border: none !important;
}

:deep(.recycle-bin-table th),
:deep(.recycle-bin-table td) {
  border-right: none !important;
  border-left: none !important;
}

:deep(.recycle-bin-table .el-table__body tr) {
  background-color: transparent !important;
}

:deep(.recycle-bin-table .el-table__body tr > td.el-table__cell) {
  background: var(--app-shell-panel-bg-strong) !important;
}

:deep(.recycle-bin-table .el-table__body tr:hover > td.el-table__cell) {
  background-color: rgba(var(--el-color-primary-rgb), 0.04) !important;
}

:deep(.recycle-bin-table.table-row-clickable .el-table__body tr) {
  cursor: pointer;
}

:deep(.recycle-bin-table .el-table__header th.el-table__cell) {
  background-color: var(--app-shell-panel-muted-bg);
  color: var(--el-text-color-secondary);
  font-weight: 600;
  border-top: none;
}

:deep(.recycle-bin-table td.el-table__cell),
:deep(.recycle-bin-table th.el-table__cell.is-leaf) {
  border-bottom: 1px solid var(--app-shell-panel-border);
}

:deep(.recycle-bin-table .table-data-column .cell) {
  display: block;
  min-width: 0;
  line-height: 1.45;
  white-space: normal;
  overflow: hidden;
}

:deep(.recycle-bin-table .table-data-column .cell > *) {
  min-width: 0;
  max-width: 100%;
}

:deep(.recycle-bin-table .table-data-column .table-cell-value),
:deep(.recycle-bin-table .table-data-column .files-display-text) {
  display: block;
  min-width: 0;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.recycle-bin-table .table-data-column .input-widget .table-cell-value),
:deep(.recycle-bin-table .table-data-column .textarea-widget .table-cell-value),
:deep(.recycle-bin-table .table-data-column .rich-text-widget .table-cell-value),
:deep(.recycle-bin-table .table-data-column .text-widget .table-cell-value),
:deep(.recycle-bin-table .table-data-column .table-cell-text),
:deep(.recycle-bin-table .table-data-column .formatted-content),
:deep(.recycle-bin-table .table-data-column .text-content),
:deep(.recycle-bin-table .table-data-column .code-content),
:deep(.recycle-bin-table .table-data-column .html-table-cell),
:deep(.recycle-bin-table .table-data-column .markdown-table-cell),
:deep(.recycle-bin-table .table-data-column .csv-preview),
:deep(.recycle-bin-table .table-data-column .csv-preview-text),
:deep(.recycle-bin-table .table-data-column .html-content-preview) {
  display: -webkit-box;
  min-width: 0;
  max-width: 100%;
  line-height: 1.45;
  white-space: normal;
  overflow-wrap: anywhere;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

:deep(.recycle-bin-table .table-data-column .table-cell-multiselect),
:deep(.recycle-bin-table .table-data-column .files-table-cell),
:deep(.recycle-bin-table .table-data-column .files-select-display) {
  min-width: 0;
  max-width: 100%;
  flex-wrap: nowrap;
  overflow: hidden;
}

:deep(.recycle-bin-table .table-data-column .formatted-content),
:deep(.recycle-bin-table .table-data-column .text-content),
:deep(.recycle-bin-table .table-data-column .code-content),
:deep(.recycle-bin-table .table-data-column .html-table-cell),
:deep(.recycle-bin-table .table-data-column .markdown-table-cell) {
  padding: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

:deep(.recycle-bin-table .control-column .cell) {
  min-width: 0;
  overflow: hidden;
}

.detail-value {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.detail-section + .detail-section,
.detail-system-collapse {
  margin-top: 22px;
}

.detail-section h3 {
  margin: 0 0 12px;
  font-size: 15px;
  color: var(--el-text-color-primary);
}

.detail-section :deep(.el-descriptions__label) {
  width: 150px;
}

/* Element Plus 固定列使用 sticky；必须显式给单元格实体底色，避免横向滚动时叠字。 */
:deep(.recycle-bin-table td.el-table-fixed-column--left),
:deep(.recycle-bin-table td.control-column.el-table-fixed-column--left),
:deep(.recycle-bin-table td.action-column.el-table-fixed-column--right) {
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color)) !important;
  opacity: 1 !important;
}

:deep(.recycle-bin-table th.el-table-fixed-column--left),
:deep(.recycle-bin-table th.control-column.el-table-fixed-column--left),
:deep(.recycle-bin-table th.action-column.el-table-fixed-column--right) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light)) !important;
  opacity: 1 !important;
}

:deep(.recycle-bin-table .el-table__body tr:hover > td.el-table-fixed-column--left),
:deep(.recycle-bin-table .el-table__body tr:hover > td.control-column.el-table-fixed-column--left),
:deep(.recycle-bin-table .el-table__body tr:hover > td.action-column.el-table-fixed-column--right) {
  background: var(--el-fill-color-light) !important;
}

:deep(.recycle-bin-table td.action-column .cell) {
  opacity: 1 !important;
  visibility: visible !important;
}

.detail-icon-button {
  min-width: 44px;
  width: 100%;
  max-width: 100%;
  height: 32px;
  padding: 0 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.detail-icon-button .el-icon {
  flex: 0 0 auto;
}

.detail-icon-button:hover {
  background-color: var(--el-color-primary-light-9);
}

.detail-icon-button:focus-visible {
  outline: 2px solid var(--el-color-primary-light-5);
  outline-offset: 2px;
}

.detail-id-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
}

.action-more-btn {
  margin: 0;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  background: var(--table-control-bg);
  color: var(--el-text-color-regular);
  font-weight: 600;
  box-shadow: none;
}

.action-more-btn:hover,
.action-more-btn:focus {
  border-color: var(--table-control-border-hover);
  background: var(--table-control-bg-hover);
  color: var(--el-color-primary);
  box-shadow: var(--table-control-shadow-hover);
}

.danger-action-text {
  color: var(--el-color-danger);
}

@media (max-width: 900px) {
  .recycle-toolbar,
  .recycle-toolbar-left {
    align-items: stretch;
    flex-direction: column;
  }

  .recycle-search {
    width: 100%;
  }

  .recycle-policy-summary,
  .recycle-policy-form {
    align-items: stretch;
    flex-direction: column;
  }

  .recycle-policy-form-actions {
    margin-left: 0 !important;
  }
}
</style>
