<template>
  <el-dialog
    v-model="visible"
    class="fuzzy-search-dialog-shell"
    width="680px"
    :close-on-click-modal="false"
    :close-on-press-escape="true"
    :append-to-body="appendToBody"
    @close="handleClose"
  >
    <template #header>
      <div class="dialog-header">
        <div class="dialog-title-row">
          <span class="dialog-title">{{ dialogTitle }}</span>
          <span class="dialog-mode-chip">{{ isMultiselect ? '多选' : '单选' }}</span>
        </div>
        <div class="dialog-subtitle">
          <span>{{ resultSummaryText }}</span>
          <span class="dialog-shortcut-hint">Enter 选中 · ↑↓ 切换 · Esc 关闭</span>
        </div>
      </div>
    </template>

    <div class="fuzzy-search-dialog">
      <!-- 搜索输入框 -->
      <div class="search-input-section">
        <el-input
          class="dialog-search-input"
          v-model="searchKeyword"
          :placeholder="placeholder"
          clearable
          @input="handleSearch"
          @keyup.enter="handleEnter"
          @keyup.esc="handleClose"
          ref="searchInputRef"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <div class="search-toolbar-meta">
          <span class="toolbar-chip">
            {{ loading ? '搜索中...' : `${suggestions.length} 个候选项` }}
          </span>
          <span
            v-if="isMultiselect && selectedItems.length > 0"
            class="toolbar-chip toolbar-chip-active"
          >
            已选 {{ selectedItems.length }}{{ maxSelectionsText }}项
          </span>
        </div>
      </div>

      <!-- 搜索结果列表 -->
      <div class="search-results-section">
        <div v-if="loading" class="loading-section">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>搜索中...</span>
        </div>
        
        <div v-else-if="!searchKeyword && suggestions.length === 0" class="empty-section">
          <el-icon><InfoFilled /></el-icon>
          <span>请输入关键词开始搜索</span>
        </div>
        
        <div v-else-if="suggestions.length === 0" class="empty-section">
          <el-icon><Search /></el-icon>
          <span>未找到相关结果</span>
        </div>
        
        <div v-else class="suggestions-list">
          <div
            v-for="(item, index) in suggestions"
            :key="index"
            class="suggestion-item"
            :class="{ 'active': selectedIndex === index, 'selected': isItemSelected(item) }"
            @click="handleItemClick(item)"
            @mouseenter="selectedIndex = index"
          >
            <!-- 多选模式下的复选框 -->
            <div v-if="isMultiselect" class="item-checkbox">
              <el-checkbox
                :model-value="isItemSelected(item)"
                @change="handleItemCheckboxChange(item, $event)"
                @click.stop
              />
            </div>
            
            <!-- 图标 -->
            <div v-if="item.icon" class="item-icon">
              <!-- 如果是 emoji 或普通文本，直接显示；否则作为组件 -->
              <el-icon v-if="isValidComponentName(item.icon)">
                <component :is="item.icon" />
              </el-icon>
              <span v-else class="item-icon-emoji">{{ item.icon }}</span>
            </div>
            
            <!-- 颜色指示器 -->
            <span
              v-if="getItemColor(item.value)"
              class="option-color-indicator"
              :style="getItemColorStyle(item.value)"
            />
            
            <!-- 主要内容 -->
            <div class="item-content">
              <div class="item-header">
                <div class="item-label">{{ item.label || item.value }}</div>
                <span v-if="isItemSelected(item)" class="item-state-chip">已选</span>
              </div>
              
              <!-- 显示信息 -->
              <div v-if="(item.display_info && Object.keys(item.display_info).length > 0) || (item.displayInfo && Object.keys(item.displayInfo).length > 0)" class="item-display-info">
                <div
                  v-for="([infoKey, infoValue], infoIndex) in getDisplayInfoEntries(item)"
                  :key="`${String(item.value)}-${infoKey}-${infoIndex}`"
                  class="info-item"
                >
                  <span class="info-key">{{ infoKey }}</span>
                  <span class="info-value">{{ infoValue }}</span>
                </div>
              </div>
            </div>
            
            <!-- 选择指示器 -->
            <div class="item-indicator">
              <el-icon v-if="selectedIndex === index"><ArrowRight /></el-icon>
              <el-icon v-else-if="isItemSelected(item)" class="selected-icon"><Check /></el-icon>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <div class="footer-left">
          <!-- 多选框的全选按钮（有限制时不显示） -->
          <el-button 
            v-if="isMultiselect && suggestions.length > 0 && (!maxSelections || maxSelections === 0)"
            size="small"
            type="info"
            @click="handleSelectAll"
          >
            全选 ({{ suggestions.length }})
          </el-button>
          <!-- 多选模式下显示已选数量 -->
          <span v-if="isMultiselect && selectedItems.length > 0" class="selected-count">
            已选择 {{ selectedItems.length }}{{ maxSelectionsText }}项
          </span>
          <!-- 多选模式下显示限制提示 -->
          <span v-if="isMultiselect && props.maxSelections && props.maxSelections > 0" class="max-selections-hint">
            （最多可选{{ props.maxSelections }}个）
          </span>
        </div>
        <div class="footer-right">
          <el-button @click="handleClose">取消</el-button>
          <el-button 
            v-if="!isMultiselect && selectedItem"
            type="primary" 
            @click="handleConfirm"
          >
            确定
          </el-button>
          <el-button 
            v-else-if="isMultiselect && selectedItems.length > 0"
            type="primary" 
            @click="handleConfirmMultiple"
          >
            确定 ({{ selectedItems.length }})
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { Search, Loading, InfoFilled, ArrowRight, Check } from '@element-plus/icons-vue'

/**
 * OnSelectFuzzy / 模糊搜索回调的标准候选项结构。
 *
 * 约定：
 * - `label` 是用户在候选列表里第一眼看到的主文案
 * - `value` 是真正提交给后端的值
 * - `display_info` / `displayInfo` 是候选项的次级结构化信息
 *   例如职位的部门/地点/薪资，供应商的联系人/电话
 *   适合展示成 key-value chip，不适合直接把整个对象或 statistics 原样抛给用户
 */
export interface InputFuzzyItem {
  value: unknown
  label?: string
  icon?: string
  display_info?: Record<string, unknown>
  displayInfo?: Record<string, unknown>
}

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  suggestions: InputFuzzyItem[]
  loading?: boolean
  isMultiselect?: boolean
  maxSelections?: number // 最大选择数量，0 表示不限制
  selectedValues?: unknown[] // 已选中的值（多选模式）
  getItemColor?: (value: unknown) => string | null // 获取选项颜色的函数
  appendToBody?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', item: InputFuzzyItem): void
  (e: 'selectMultiple', items: InputFuzzyItem[]): void
  (e: 'search', keyword: string): void
  (e: 'selectAll', items: InputFuzzyItem[]): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '搜索选择',
  placeholder: '请输入搜索关键词',
  loading: false,
  isMultiselect: false,
  maxSelections: 0,
  selectedValues: () => [],
  getItemColor: () => null,
  appendToBody: true
})

const emit = defineEmits<Emits>()

// 响应式数据
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const searchKeyword = ref('')
const selectedIndex = ref(-1)
const selectedItem = ref<InputFuzzyItem | null>(null)
const selectedItems = ref<InputFuzzyItem[]>([]) // 多选模式下的已选项目
const searchInputRef = ref<HTMLInputElement>()

// 对话框标题（如果有限制，显示最多可选数量）
const dialogTitle = computed(() => {
  if (props.isMultiselect && props.maxSelections && props.maxSelections > 0) {
    return `${props.title}（最多可选${props.maxSelections}个）`
  }
  return props.title
})

// 已选择数量的显示文本（如果有限制，显示限制）
const maxSelectionsText = computed(() => {
  if (props.isMultiselect && props.maxSelections && props.maxSelections > 0) {
    return `/${props.maxSelections} `
  }
  return ' '
})

const resultSummaryText = computed(() => {
  if (props.loading) {
    return '正在加载候选项'
  }

  if (!searchKeyword.value && props.suggestions.length === 0) {
    return '输入关键词后立即开始搜索'
  }

  if (props.suggestions.length === 0) {
    return '没有匹配结果'
  }

  return `当前共 ${props.suggestions.length} 个结果`
})

// 监听对话框显示状态
watch(visible, (newVisible) => {
  if (newVisible) {
    // 对话框打开时，聚焦到搜索框
    nextTick(() => {
      searchInputRef.value?.focus()
    })
    // 初始化已选项目（多选模式）
    if (props.isMultiselect && props.selectedValues) {
      selectedItems.value = props.suggestions.filter(item => 
        props.selectedValues.some(val => String(val) === String(item.value))
      )
    }
  } else {
    // 对话框关闭时，重置状态
    searchKeyword.value = ''
    selectedIndex.value = -1
    selectedItem.value = null
    selectedItems.value = []
  }
})

// 监听 suggestions 变化，更新已选项目（多选模式）
watch(() => props.suggestions, (newSuggestions) => {
  if (props.isMultiselect && props.selectedValues && visible.value) {
    selectedItems.value = newSuggestions.filter(item => 
      props.selectedValues.some(val => String(val) === String(item.value))
    )
  }
}, { immediate: true })

// 判断是否是有效的组件名（用于区分 emoji 和真正的组件）
function isValidComponentName(icon: string): boolean {
  if (!icon || typeof icon !== 'string') return false
  // 组件名通常是字母、数字、连字符、下划线的组合，且以字母开头
  // emoji 和其他特殊字符不符合这个规则
  return /^[A-Za-z][A-Za-z0-9_-]*$/.test(icon)
}

// 判断项目是否已选中
function isItemSelected(item: InputFuzzyItem): boolean {
  if (props.isMultiselect) {
    return selectedItems.value.some(selected => String(selected.value) === String(item.value))
  }
  return false
}

// 获取选项颜色
function getItemColor(value: unknown): string | null {
  if (props.getItemColor) {
    return props.getItemColor(value)
  }
  return null
}

// 获取选项颜色样式
function getItemColorStyle(value: unknown): Record<string, string> {
  const color = getItemColor(value)
  if (!color) return {}
  
  return {
    marginRight: '8px',
    display: 'inline-block',
    width: '12px',
    height: '12px',
    minWidth: '12px',
    minHeight: '12px',
    borderRadius: '2px',
    flexShrink: '0',
    border: 'none',
    verticalAlign: 'middle',
    backgroundColor: color,
    filter: 'brightness(0.95) saturate(0.9)',
    opacity: '0.9'
  }
}

function getDisplayInfoEntries(item: InputFuzzyItem): Array<[string, unknown]> {
  const displayInfo = item.display_info || item.displayInfo || {}

  return Object.entries(displayInfo)
    .filter(([, value]) => value !== null && value !== undefined && value !== '')
    .slice(0, 6)
}

// 处理搜索
const handleSearch = (value: string) => {
  searchKeyword.value = value
  selectedIndex.value = -1
  selectedItem.value = null
  emit('search', value)
}

// 处理回车键
const handleEnter = () => {
  if (props.isMultiselect) {
    // 多选模式：确认当前选择
    if (selectedItems.value.length > 0) {
      handleConfirmMultiple()
    }
  } else {
    // 单选模式
    if (selectedItem.value) {
      handleSelectItem(selectedItem.value)
    } else if (props.suggestions.length > 0) {
      const firstSuggestion = props.suggestions[0]
      if (firstSuggestion) {
        handleSelectItem(firstSuggestion)
      }
    }
  }
}

// 处理项目点击（单选模式）
const handleItemClick = (item: InputFuzzyItem) => {
  if (props.isMultiselect) {
    // 多选模式：切换选中状态
    toggleItemSelection(item)
  } else {
    // 单选模式：直接选择并关闭对话框
    handleSelectItem(item)
  }
}

// 切换项目选中状态（多选模式）
function toggleItemSelection(item: InputFuzzyItem) {
  const index = selectedItems.value.findIndex(selected => String(selected.value) === String(item.value))
  if (index >= 0) {
    // 已选中，取消选择
    selectedItems.value.splice(index, 1)
  } else {
    // 未选中，检查是否超过最大选择数量
    if (props.maxSelections > 0 && selectedItems.value.length >= props.maxSelections) {
      return
    }
    selectedItems.value.push(item)
  }
}

// 处理复选框变化（多选模式）
function handleItemCheckboxChange(item: InputFuzzyItem, checked: boolean) {
  if (checked) {
    if (props.maxSelections > 0 && selectedItems.value.length >= props.maxSelections) {
      return
    }
    if (!selectedItems.value.some(selected => String(selected.value) === String(item.value))) {
      selectedItems.value.push(item)
    }
  } else {
    const index = selectedItems.value.findIndex(selected => String(selected.value) === String(item.value))
    if (index >= 0) {
      selectedItems.value.splice(index, 1)
    }
  }
}

// 处理选择项目（单选模式）
const handleSelectItem = (item: InputFuzzyItem) => {
  selectedItem.value = item
  emit('select', item)
  visible.value = false
}

// 处理确认（单选模式）
const handleConfirm = () => {
  if (selectedItem.value) {
    handleSelectItem(selectedItem.value)
  }
}

// 处理确认（多选模式）
const handleConfirmMultiple = () => {
  if (selectedItems.value.length > 0) {
    emit('selectMultiple', selectedItems.value)
    visible.value = false
  }
}

// 处理全选
const handleSelectAll = () => {
  if (props.maxSelections > 0 && props.maxSelections < props.suggestions.length) {
    // 如果有限制，只选择前 maxSelections 个
    selectedItems.value = props.suggestions.slice(0, props.maxSelections)
  } else {
    selectedItems.value = [...props.suggestions]
  }
  emit('selectAll', selectedItems.value)
}

// 处理关闭
const handleClose = () => {
  visible.value = false
}

// 键盘导航
const handleKeydown = (event: KeyboardEvent) => {
  if (!visible.value) return
  
  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      if (selectedIndex.value < props.suggestions.length - 1) {
        selectedIndex.value++
      }
      break
    case 'ArrowUp':
      event.preventDefault()
      if (selectedIndex.value > 0) {
        selectedIndex.value--
      }
      break
    case 'Enter':
      event.preventDefault()
      handleEnter()
      break
    case 'Escape':
      event.preventDefault()
      handleClose()
      break
  }
}

// 监听键盘事件
watch(visible, (newVisible) => {
  if (newVisible) {
    document.addEventListener('keydown', handleKeydown)
  } else {
    document.removeEventListener('keydown', handleKeydown)
  }
})
</script>

<style scoped>
:deep(.fuzzy-search-dialog-shell) {
  border-radius: 18px;
  overflow: hidden;
}

:deep(.fuzzy-search-dialog-shell .el-dialog__header) {
  padding: 18px 22px 0;
}

:deep(.fuzzy-search-dialog-shell .el-dialog__body) {
  padding: 18px 22px 12px;
}

:deep(.fuzzy-search-dialog-shell .el-dialog__footer) {
  padding: 0 22px 20px;
}

.dialog-header {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.dialog-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  letter-spacing: 0.01em;
}

.dialog-mode-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.12), rgba(22, 119, 255, 0.06));
  border: 1px solid rgba(24, 144, 255, 0.18);
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.dialog-subtitle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.dialog-shortcut-hint {
  white-space: nowrap;
}

.fuzzy-search-dialog {
  display: flex;
  flex-direction: column;
  height: 480px;
  gap: 14px;
}

.search-input-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-search-input :deep(.el-input__wrapper) {
  min-height: 42px;
  border-radius: 12px;
  box-shadow: none;
}

.search-toolbar-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.toolbar-chip-active {
  background: rgba(24, 144, 255, 0.12);
  color: var(--el-color-primary);
}

.search-results-section {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.loading-section,
.empty-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}

.loading-section .el-icon,
.empty-section .el-icon {
  font-size: 24px;
  margin-bottom: 8px;
}

.suggestions-list {
  flex: 1;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background:
    linear-gradient(180deg, rgba(24, 144, 255, 0.03), rgba(24, 144, 255, 0)),
    var(--el-bg-color);
  padding: 8px;
}

.suggestion-item {
  display: flex;
  align-items: flex-start;
  padding: 14px 16px;
  cursor: pointer;
  border: 1px solid transparent;
  border-radius: 12px;
  transition: background-color 0.2s, border-color 0.2s, transform 0.2s;
  margin-bottom: 8px;
}

.suggestion-item:hover,
.suggestion-item.active {
  background-color: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  transform: translateY(-1px);
}

.suggestion-item.selected {
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.12), rgba(24, 144, 255, 0.04)) !important;
  border-color: rgba(24, 144, 255, 0.35) !important;
  box-shadow: 0 8px 20px rgba(24, 144, 255, 0.08);
}

.suggestion-item.selected:hover {
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.16), rgba(24, 144, 255, 0.06)) !important;
}

.item-checkbox {
  margin-right: 12px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.item-icon {
  margin-right: 12px;
  color: var(--el-text-color-regular);
  font-size: 16px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.item-icon-emoji {
  font-size: 18px;
  line-height: 1;
}

.option-color-indicator {
  margin-right: 8px;
  flex-shrink: 0;
}

.item-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.item-label {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  line-height: 1.45;
}

.item-state-chip {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 999px;
  background: rgba(24, 144, 255, 0.12);
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

.suggestion-item.selected .item-label {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.suggestion-item.selected .item-content {
  color: var(--el-text-color-primary);
}

.suggestion-item.selected .info-item {
  color: var(--el-text-color-regular);
}

.item-display-info {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 9px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.info-key {
  color: var(--el-text-color-placeholder);
}

.info-value {
  color: var(--el-text-color-regular);
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-indicator {
  margin-left: 12px;
  color: var(--el-color-primary);
  flex-shrink: 0;
}

.selected-icon {
  color: var(--el-color-success);
}

.dialog-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.footer-right {
  display: flex;
  gap: 8px;
}

.selected-count {
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.max-selections-hint {
  color: var(--el-color-info);
  font-size: 13px;
  margin-left: 8px;
}

/* 滚动条样式 */
.suggestions-list::-webkit-scrollbar {
  width: 6px;
}

.suggestions-list::-webkit-scrollbar-track {
  background: var(--el-fill-color-lighter);
  border-radius: 3px;
}

.suggestions-list::-webkit-scrollbar-thumb {
  background: var(--el-border-color);
  border-radius: 3px;
}

.suggestions-list::-webkit-scrollbar-thumb:hover {
  background: var(--el-border-color-dark);
}

@media (max-width: 768px) {
  .dialog-subtitle {
    flex-direction: column;
    align-items: flex-start;
  }

  .dialog-shortcut-hint {
    white-space: normal;
  }

  .fuzzy-search-dialog {
    height: 70vh;
  }

  .item-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .info-value {
    max-width: none;
  }

  .dialog-footer {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .footer-left,
  .footer-right {
    justify-content: space-between;
    width: 100%;
  }
}
</style>
