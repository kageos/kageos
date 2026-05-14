<!--
  TableView - 表格视图
  新架构的展示层组件
  
  ============================================
  📋 需求说明
  ============================================
  
  1. **表格数据展示**：
     - 从后端获取表格数据并渲染
     - 支持搜索、排序、分页
     - 支持批量操作（批量删除）
  
  2. **权限控制**：
     - 根据节点权限控制按钮显示/隐藏
     - 新增按钮：需要 `function:write` 权限
     - 编辑按钮：需要 `function:update` 权限
     - 删除按钮：需要 `function:delete` 权限
     - 提交时再次检查权限，防止绕过 UI 检查
  
  3. **URL 参数同步**：
     - 搜索条件同步到 URL（`field=value`）
     - 排序条件同步到 URL（`sorts=[{"field":"created_at","order":"desc"}]`）
     - 分页信息同步到 URL（`page=1&page_size=10`）
     - 新增弹窗状态同步到 URL（`_tab=OnTableAddRow`）
  
  ============================================
  🎯 设计思路
  ============================================
  
  1. **分层架构**：
     - Presentation Layer：纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
  
  2. **URL 同步**：
     - 使用 `useTableInitialization` 从 URL 初始化表格状态
     - 使用 `useTableParamURLSync` 同步表格状态到 URL
     - 使用事件驱动，避免直接操作路由
  
  ============================================
  📝 关键功能
  ============================================
  
  1. **表格操作**：
     - 新增：打开 FormDialog，提交时检查权限
     - 编辑：打开详情抽屉，在 useWorkspaceDetail 中检查权限
     - 删除：批量删除，提交前检查权限
  
  2. **数据加载**：
     - 从 TableApplicationService 加载数据
     - 支持搜索、排序、分页参数
     - 支持从 URL 恢复表格状态
  
  ============================================
  ⚠️ 注意事项
  ============================================
  
  1. **URL 同步**：
     - 新增弹窗状态使用 `_tab=OnTableAddRow` 标识
     - 详情抽屉状态使用 `_tab=detail` 标识（编辑模式不设置 `_tab`）
     - 表单字段参数只在新增模式下同步到 URL
  
  3. **数据流**：
     - TableView → TableApplicationService → TableDomainService → TableStateManager
     - 状态变化通过事件总线通知其他组件
-->

<template>
  <div class="table-view" data-testid="table-view">
    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <!-- 新增按钮：不支持时展示禁用态，避免误以为缺按钮 -->
        <el-button 
          :type="hasAddCallback ? 'primary' : 'default'"
          :plain="!hasAddCallback"
          :disabled="!hasAddCallback"
          @click="handleAdd"
          :icon="hasAddCallback ? Plus : InfoFilled"
          class="action-btn"
          data-testid="table-add"
        >
          {{ hasAddCallback ? '新增' : '新增（当前表格不支持）' }}
        </el-button>
        <!-- 批量删除按钮：不支持时保留禁用提示 -->
        <el-button 
          v-if="!isBatchDeleteMode" 
          :type="hasDeleteCallback ? 'danger' : 'default'"
          :plain="!hasDeleteCallback"
          :disabled="!hasDeleteCallback"
          @click="hasDeleteCallback ? enterBatchDeleteMode() : undefined"
          :icon="hasDeleteCallback ? Delete : InfoFilled"
          class="action-btn"
        >
          {{ hasDeleteCallback ? '批量删除' : '批量删除（当前表格不支持）' }}
        </el-button>
        <template v-if="hasDeleteCallback && isBatchDeleteMode">
          <el-button 
            type="danger" 
            @click="handleBatchDelete"
            :icon="Delete"
            :disabled="selectedRows.length === 0"
            class="action-btn action-btn-danger"
          >
            删除选中 ({{ selectedRows.length }})
          </el-button>
          <el-button 
            @click="exitBatchDeleteMode"
            class="toolbar-secondary-btn"
          >
            取消
          </el-button>
        </template>
      </div>

      <div class="toolbar-right">
        <el-radio-group
          v-model="viewMode"
          size="small"
          class="view-mode-toggle"
          aria-label="展示方式"
        >
          <el-radio-button value="table">
            <span class="view-mode-option">
              <el-icon><List /></el-icon>
              表格
            </span>
          </el-radio-button>
          <el-radio-button value="card">
            <span class="view-mode-option">
              <el-icon><Grid /></el-icon>
              卡片
            </span>
          </el-radio-button>
        </el-radio-group>

        <template v-if="searchableFields.length > 0">
          <el-button
            :type="searchBarExpanded ? 'primary' : 'default'"
            class="toolbar-search-btn"
            @click="searchBarExpanded ? handleSearch() : (searchBarExpanded = true)"
          >
            <el-icon><Search /></el-icon>
            <span v-if="searchBarExpanded">搜索</span>
            <span v-else>
              {{ activeSearchCount > 0 ? `筛选 (${activeSearchCount})` : '筛选' }}
            </span>
          </el-button>
          <el-button
            v-if="searchBarExpanded"
            class="toolbar-secondary-btn"
            @click="handleReset"
          >
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
          <el-button
            v-if="searchBarExpanded"
            text
            class="toolbar-collapse-btn"
            @click="searchBarExpanded = false"
          >
            <el-icon><ArrowUp /></el-icon>
            收起
          </el-button>
        </template>

        <div v-if="viewMode === 'card' && sortableFields.length > 0" class="card-sort-controls">
          <span class="card-sort-label">排序</span>
          <el-select
            :model-value="cardSortField"
            size="small"
            class="card-sort-select"
            popper-class="table-action-dropdown"
            @update:model-value="handleCardSortFieldChange"
          >
            <el-option
              v-for="field in sortableFields"
              :key="field.code"
              :label="field.name"
              :value="field.code"
            />
          </el-select>
          <el-radio-group
            :model-value="cardSortOrder"
            size="small"
            class="card-sort-order"
            @update:model-value="handleCardSortOrderChange"
          >
            <el-radio-button value="asc">
              <el-icon><ArrowUp /></el-icon>
            </el-radio-button>
            <el-radio-button value="desc">
              <el-icon><ArrowDown /></el-icon>
            </el-radio-button>
          </el-radio-group>
        </div>
      </div>
    </div>

    <!-- 搜索栏：字段区单独占位，动作并到工具栏 -->
    <div v-if="searchableFields.length > 0 && searchBarExpanded" class="search-bar-wrapper" data-testid="table-search">
      <div class="search-bar">
        <div class="search-bar-inner">
          <el-form
            :inline="false"
            label-position="top"
            :model="searchForm"
            class="search-form"
          >
            <template v-for="field in searchableFields" :key="field.code">
              <el-form-item :label="field.name" :class="getSearchFieldLayoutClass(field)">
                <SearchInput
                  :field="field"
                  search-type=""
                  :model-value="getSearchValue(field)"
                  :function-method="props.functionDetail.method || 'GET'"
                  :function-router="props.functionDetail.router"
                  @update:model-value="(value: any) => {
                    updateSearchValue(field, value, true)
                  }"
                />
              </el-form-item>
            </template>
          </el-form>
        </div>
      </div>
    </div>

    <!-- 🔥 排序信息条：显示当前排序状态 -->
    <div v-if="displaySorts.length > 0" class="sort-info-bar">
      <div class="sort-info-content">
        <span class="sort-label">排序：</span>
        <div class="sort-items">
          <template v-for="(sort, index) in displaySorts" :key="sort.field">
            <el-tag
              :type="index === 0 ? 'primary' : 'info'"
              size="small"
              closable
              @close="handleRemoveSort(sort.field)"
              class="sort-tag"
            >
              <span class="sort-field-name">{{ getFieldName(sort.field) }}</span>
              <el-icon class="sort-icon">
                <ArrowUp v-if="sort.order === 'asc'" />
                <ArrowDown v-else />
              </el-icon>
            </el-tag>
            <span v-if="index < displaySorts.length - 1" class="sort-separator">></span>
          </template>
        </div>
        <el-button
          v-if="sorts.length > 0"
          size="small"
          @click="handleClearAllSorts"
          class="table-clear-sorts-btn"
        >
          <el-icon><Refresh /></el-icon>
          清除排序
        </el-button>
      </div>
    </div>

    <!-- 加载中：表格骨架屏 -->
    <div v-if="loading" class="table-skeleton-wrap">
      <el-skeleton :rows="20" animated class="table-skeleton" />
    </div>
    <!-- 表格 -->
    <el-table
      v-else-if="viewMode === 'table'"
      ref="tableRef"
      :data="tableData"
      :stripe="false"
      style="width: 100%"
      class="table-with-fixed-column table-row-clickable"
      data-testid="table-grid"
      @sort-change="handleSortChange"
      @selection-change="handleSelectionChange"
      @row-click="handleRowClick"
    >
      <!-- 复选框列（用于批量操作，仅在批量删除模式下显示） -->
      <el-table-column
        v-if="hasDeleteCallback && isBatchDeleteMode"
        type="selection"
        width="55"
        fixed="left"
        :selectable="checkSelectable"
      />

      <!-- 🔥 控制中心列（ID列） -->
      <el-table-column
        v-if="idField"
        :prop="idField.code"
        label=""
        fixed="left"
        width="120"
        class-name="control-column"
        :label-class-name="getSortHeaderClass(idField.code)"
        :sortable="getSortableConfig(idField)"
        :sort-order="sortOrderMap[idField.code] || null"
      >
        <template #default="{ row }">
          <button
            type="button"
            class="detail-icon-button"
            :title="`查看详情 ${row[idField.code]}`"
            @click.stop="handleDetail(row)"
          >
            <el-icon><View /></el-icon>
            <span class="detail-id-text">{{ row[idField.code] }}</span>
          </button>
        </template>
      </el-table-column>

      <!-- 数据列（排除ID列） -->
      <el-table-column
        v-for="field in dataFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        :sortable="getSortableConfig(field)"
        :sort-order="sortOrderMap[field.code] || null"
        :label-class-name="getSortHeaderClass(field.code)"
        :min-width="getColumnWidth(field)"
        show-overflow-tooltip
      >
        <template #default="{ row }">
          <WidgetComponent
            :field="field"
            :value="getRowFieldValue(row, field)"
            mode="table-cell"
            :row-data="row"
          />
        </template>
      </el-table-column>

      <!-- 操作列：统一为「更多」下拉，所有操作（链接 / 更新 / 删除）均放入下拉 -->
      <el-table-column 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <el-dropdown
            trigger="click"
            placement="bottom-end"
            popper-class="table-action-dropdown"
            @command="(cmd: string) => handleActionCommand(cmd, row)"
          >
            <el-button size="small" class="action-more-btn">
              <el-icon><More /></el-icon>
              更多
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <!-- 链接项：1 个或多个都放下拉 -->
                <el-dropdown-item
                  v-for="linkField in linkFields"
                  :key="linkField.code"
                  :command="'link:' + linkField.code"
                >
                  <div class="dropdown-link-content">
                    <el-icon v-if="linkField.widget?.config?.icon" class="link-icon">
                      <component :is="linkField.widget.config.icon" />
                    </el-icon>
                    <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
                    <span>{{ getLinkText(linkField, row[linkField.code]) }}</span>
                  </div>
                </el-dropdown-item>
                <el-dropdown-item
                  :command="hasUpdateCallback ? 'update' : undefined"
                  :disabled="!hasUpdateCallback"
                  :divided="linkFields.length > 0"
                >
                  <span class="dropdown-action-item">
                    <el-icon><component :is="hasUpdateCallback ? Edit : InfoFilled" /></el-icon>
                    {{ hasUpdateCallback ? '更新' : '更新（当前表格不支持）' }}
                  </span>
                </el-dropdown-item>
                <el-dropdown-item
                  :command="hasDeleteCallback ? 'delete' : undefined"
                  :disabled="!hasDeleteCallback"
                  :divided="true"
                >
                  <span class="dropdown-action-item" :class="{ 'delete-action-text': hasDeleteCallback }">
                    <el-icon><component :is="hasDeleteCallback ? Delete : InfoFilled" /></el-icon>
                    {{ hasDeleteCallback ? '删除' : '删除（当前表格不支持）' }}
                  </span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
    </el-table>

    <!-- 卡片 -->
    <div v-else class="table-card-grid" data-testid="table-card-grid">
      <el-empty
        v-if="tableData.length === 0"
        description="暂无数据"
        class="table-card-empty"
      />
      <template v-else>
        <article
          v-for="row in tableData"
          :key="getCardRowKey(row)"
          class="table-card-item"
          :class="{ 'is-selected': isCardRowSelected(row), 'is-selectable': isBatchDeleteMode }"
          tabindex="0"
          @click="handleCardClick(row)"
          @keydown.enter="handleCardClick(row)"
          @keydown.space.prevent="handleCardClick(row)"
        >
          <div class="table-card-header">
            <el-checkbox
              v-if="hasDeleteCallback && isBatchDeleteMode"
              :model-value="isCardRowSelected(row)"
              class="table-card-checkbox"
              @click.stop
              @change="handleCardSelectionChange(row, $event)"
            />
            <div
              v-if="cardMediaField"
              class="table-card-media"
              :class="{ 'has-image': Boolean(getCardMediaPreviewUrl(row)) }"
              @click.stop="handleDetail(row)"
            >
              <img
                v-if="getCardMediaPreviewUrl(row)"
                :src="getCardMediaPreviewUrl(row)"
                :alt="getCardMediaLabel(row)"
                class="table-card-media-image"
                loading="lazy"
              />
              <el-icon v-else class="table-card-media-icon">
                <Picture v-if="isCardImageMedia(row)" />
                <Document v-else />
              </el-icon>
              <span v-if="getCardMediaCount(row) > 1" class="table-card-media-count">
                {{ getCardMediaCount(row) }}
              </span>
            </div>
            <div class="table-card-title-wrap">
              <div class="table-card-kicker">
                <span>{{ cardTitleField?.name || idField?.name || '记录' }}</span>
                <span v-if="idField && row[idField.code] !== undefined" class="table-card-id-chip">
                  #{{ row[idField.code] }}
                </span>
              </div>
              <div class="table-card-title">
                <WidgetComponent
                  v-if="cardTitleField"
                  :field="cardTitleField"
                  :value="getRowFieldValue(row, cardTitleField)"
                  mode="table-cell"
                  :row-data="row"
                />
                <span v-else-if="idField">{{ row[idField.code] }}</span>
                <span v-else>未命名记录</span>
              </div>
              <div
                v-if="cardSubtitleField && hasRowFieldValue(row, cardSubtitleField)"
                class="table-card-subtitle"
              >
                <WidgetComponent
                  :field="cardSubtitleField"
                  :value="getRowFieldValue(row, cardSubtitleField)"
                  mode="table-cell"
                  :row-data="row"
                />
              </div>
              <div
                v-if="hasCardRowStatusMeta(row)"
                class="table-card-status-badges"
                @click.stop
              >
                <div
                  v-for="field in getCardFilledStatusFields(row)"
                  :key="field.code"
                  class="table-card-status-item"
                >
                  <span class="table-card-status-label">{{ field.name }}</span>
                  <span class="table-card-status-tags">
                    <el-tag
                      v-for="tag in getCardStatusTags(row, field)"
                      :key="tag.key"
                      size="small"
                      effect="light"
                      class="table-card-status-tag"
                      :style="tag.style"
                    >
                      {{ tag.label }}
                    </el-tag>
                  </span>
                </div>
              </div>
            </div>
            <button
              type="button"
              class="table-card-view-btn"
              :title="idField ? `查看详情 ${row[idField.code]}` : '查看详情'"
              @click.stop="handleDetail(row)"
            >
              <el-icon><View /></el-icon>
              <span>查看</span>
            </button>
          </div>

          <div class="table-card-fields">
            <div
              v-for="field in cardBodyFields"
              :key="field.code"
              class="table-card-field"
            >
              <div class="table-card-field-label">{{ field.name }}</div>
              <div class="table-card-field-value">
                <WidgetComponent
                  :field="field"
                  :value="getRowFieldValue(row, field)"
                  mode="table-cell"
                  :row-data="row"
                />
              </div>
            </div>
          </div>

          <details
            v-if="cardExtraFields.length > 0"
            class="table-card-more-section"
            @click.stop
          >
            <summary class="table-card-more-summary">
              <span>更多信息</span>
              <span class="table-card-more-count">{{ cardExtraFields.length }}</span>
            </summary>
            <div class="table-card-more-fields-panel">
              <div
                v-for="field in cardExtraFields"
                :key="field.code"
                class="table-card-more-field"
              >
                <div class="table-card-field-label">{{ field.name }}</div>
                <div class="table-card-field-value">
                  <WidgetComponent
                    :field="field"
                    :value="getRowFieldValue(row, field)"
                    mode="table-cell"
                    :row-data="row"
                  />
                </div>
              </div>
            </div>
          </details>

          <div v-if="hasCardRowTimeMeta(row)" class="table-card-time-meta">
            <div
              v-if="cardCreatedAtField && hasRowFieldValue(row, cardCreatedAtField)"
              class="table-card-time-item"
            >
              <el-icon><Clock /></el-icon>
              <span class="table-card-time-label">创建</span>
              <span class="table-card-time-value">
                <WidgetComponent
                  :field="cardCreatedAtField"
                  :value="getRowFieldValue(row, cardCreatedAtField)"
                  mode="table-cell"
                  :row-data="row"
                />
              </span>
            </div>
            <div
              v-if="cardUpdatedAtField && hasRowFieldValue(row, cardUpdatedAtField)"
              class="table-card-time-item"
            >
              <el-icon><Clock /></el-icon>
              <span class="table-card-time-label">更新</span>
              <span class="table-card-time-value">
                <WidgetComponent
                  :field="cardUpdatedAtField"
                  :value="getRowFieldValue(row, cardUpdatedAtField)"
                  mode="table-cell"
                  :row-data="row"
                />
              </span>
            </div>
          </div>

          <div class="table-card-footer">
            <div class="table-card-creator">
              <template v-if="cardCreatorField && hasRowFieldValue(row, cardCreatorField)">
                <span class="table-card-creator-copy">
                  <span class="table-card-creator-label">创建人</span>
                  <span class="table-card-creator-value">
                    <WidgetComponent
                      :field="cardCreatorField"
                      :value="getRowFieldValue(row, cardCreatorField)"
                      mode="table-cell"
                      :row-data="row"
                    />
                  </span>
                </span>
              </template>
            </div>
            <div class="table-card-footer-actions">
              <span v-if="cardHiddenFieldCount > 0" class="table-card-more-fields">
                {{ cardHiddenFieldCount }} 个字段可展开
              </span>
              <el-dropdown
                trigger="click"
                placement="bottom-end"
                popper-class="table-action-dropdown"
                @command="(cmd: string) => handleActionCommand(cmd, row)"
              >
                <el-button size="small" class="action-more-btn" @click.stop>
                  <el-icon><More /></el-icon>
                  更多
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="linkField in linkFields"
                      :key="linkField.code"
                      :command="'link:' + linkField.code"
                    >
                      <div class="dropdown-link-content">
                        <el-icon v-if="linkField.widget?.config?.icon" class="link-icon">
                          <component :is="linkField.widget.config.icon" />
                        </el-icon>
                        <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
                        <span>{{ getLinkText(linkField, row[linkField.code]) }}</span>
                      </div>
                    </el-dropdown-item>
                    <el-dropdown-item
                      :command="hasUpdateCallback ? 'update' : undefined"
                      :disabled="!hasUpdateCallback"
                      :divided="linkFields.length > 0"
                    >
                      <span class="dropdown-action-item">
                        <el-icon><component :is="hasUpdateCallback ? Edit : InfoFilled" /></el-icon>
                        {{ hasUpdateCallback ? '更新' : '更新（当前表格不支持）' }}
                      </span>
                    </el-dropdown-item>
                    <el-dropdown-item
                      :command="hasDeleteCallback ? 'delete' : undefined"
                      :disabled="!hasDeleteCallback"
                      :divided="true"
                    >
                      <span class="dropdown-action-item" :class="{ 'delete-action-text': hasDeleteCallback }">
                        <el-icon><component :is="hasDeleteCallback ? Delete : InfoFilled" /></el-icon>
                        {{ hasDeleteCallback ? '删除' : '删除（当前表格不支持）' }}
                      </span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </article>
      </template>
    </div>

    <!-- 分页 -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <FormDialog
      v-if="hasAddCallback"
      v-model="createDialogVisible"
      title="新增"
      :fields="getTableCreateFields(props.functionDetail)"
      mode="create"
      :router="props.functionDetail.router ?? ''"
      :method="props.functionDetail.method || 'POST'"
      @submit="handleCreateSubmit"
      @close="handleCreateDialogClose"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElIcon, ElTable, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElText, ElCard, ElSkeleton } from 'element-plus'
import { Search, Refresh, Delete, Plus, ArrowUp, ArrowDown, More, Right, Edit, View, InfoFilled, Grid, List, Clock, Picture, Document } from '@element-plus/icons-vue'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../../presentation/widgets/WidgetComponent.vue'
import SearchInput from '@/architecture/presentation/components/SearchInput.vue'
import FormDialog from '@/architecture/presentation/components/FormDialog.vue'
import { getSortableConfig } from '@/utils/fieldSort'
import { useTableInitialization } from '../composables/useTableInitialization'
import { useTableBatchDelete } from '../composables/useTableBatchDelete'
import { useTableAddDialogUrlSync } from '../composables/useTableAddDialogUrlSync'
import { useTableCreateAndPermissions } from '../composables/useTableCreateAndPermissions'
import { useTableLoadAndPagination } from '../composables/useTableLoadAndPagination'
import { useTableReferencePreload } from '../composables/useTableReferencePreload'
import { useTableRowActions } from '../composables/useTableRowActions'
import { useTableSearchAndSort } from '../composables/useTableSearchAndSort'
import { useTableUrlSync } from '../composables/useTableUrlSync'
import { useTableViewLifecycle } from '../composables/useTableViewLifecycle'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import { getOptionLightPalette } from '@/architecture/runtime/constants/select'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import type { TableRow } from '../../domain/services/TableDomainService'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/architecture/runtime/utils/createFieldValue'
import { getFunctionCallbacks, getTableCreateFields } from '@/utils/functionSchemaSelectors'
import { normalizeStorageFileDisplayUrl } from '@/architecture/presentation/utils/storageFileUrl'
import { getFieldWidgetOptionColors } from '@/utils/widgetOptionColors'
import { isWidgetConfigFlagEnabled } from '@/utils/widgetConfigFlag'
import { deriveThumbnailPreviewUrl } from '@/utils/storagePreviewUrl'
import { writeTablePageSizePreference } from './utils/tablePageSizePreference'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

type TableViewMode = 'table' | 'card'

const CARD_CREATOR_FIELD_CODES = ['created_by']
const CARD_CREATED_AT_FIELD_CODES = ['created_at']
const CARD_UPDATED_AT_FIELD_CODES = ['updated_at']
const CARD_SYSTEM_FIELD_CODES = new Set([
  'id',
  '_id',
  ...CARD_CREATOR_FIELD_CODES,
  ...CARD_CREATED_AT_FIELD_CODES,
  ...CARD_UPDATED_AT_FIELD_CODES
])
const CARD_STATUS_WIDGET_TYPES = new Set<string>([
  WidgetType.SELECT,
  WidgetType.MULTI_SELECT,
  WidgetType.RADIO,
  WidgetType.CHECKBOX,
  WidgetType.SWITCH
])
const CARD_SUBTITLE_FIELD_HINTS = [
  'subtitle',
  'summary',
  'description',
  'desc',
  'remark',
  'note',
  '摘要',
  '描述',
  '说明',
  '备注',
  '简介'
]
const CARD_MEDIA_FIELD_HINTS = [
  'cover',
  'image',
  'avatar',
  'photo',
  'picture',
  'thumbnail',
  'logo',
  '封面',
  '图片',
  '图像',
  '头像',
  '照片',
  '缩略图'
]
const CARD_IMAGE_EXTENSIONS = new Set(['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'avif'])

interface CardStatusTag {
  key: string
  label: string
  style: Record<string, string>
}

const route = useRoute()
const router = useRouter()
// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const stateManager = serviceProvider.getTableStateManager()
const domainService = serviceProvider.getTableDomainService()
const applicationService = serviceProvider.getTableApplicationService()
const workspaceStateManager = serviceProvider.getWorkspaceStateManager()  // ⭐ 用于获取当前函数节点的权限信息

// 🔥 从状态管理器获取状态（统一状态管理）
const tableData = computed(() => stateManager.getState().data)
const loading = computed(() => stateManager.getState().loading)
const pagination = computed(() => stateManager.getState().pagination)
const searchForm = computed({
  get: () => stateManager.getState().searchForm,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, searchForm: value })
  }
})
const sorts = computed({
  get: () => stateManager.getState().sorts,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, sorts: value })
  }
})
const hasManualSort = computed({
  get: () => stateManager.getState().hasManualSort,
  set: (value) => {
    const state = stateManager.getState()
    stateManager.setState({ ...state, hasManualSort: value })
  }
})

// 分页相关（从 StateManager 获取）
const currentPage = computed({
  get: () => pagination.value.currentPage,
  set: (val) => {
    const state = stateManager.getState()
    stateManager.setState({
      ...state,
      pagination: { ...state.pagination, currentPage: val }
    })
  }
})
const pageSize = computed({
  get: () => pagination.value.pageSize,
  set: (val) => {
    const state = stateManager.getState()
    stateManager.setState({
      ...state,
      pagination: { ...state.pagination, pageSize: val }
    })
  }
})
const total = computed(() => pagination.value.total)

// ==================== 展示方式切换 ====================

const getViewModeStorageKey = (): string => {
  const functionKey = props.functionDetail.id ?? props.functionDetail.router ?? props.functionDetail.code ?? 'default'
  return `aos:table-view-mode:${functionKey}`
}

const readViewModePreference = (): TableViewMode => {
  if (typeof window === 'undefined') {
    return 'table'
  }

  return window.localStorage.getItem(getViewModeStorageKey()) === 'card' ? 'card' : 'table'
}

const viewMode = ref<TableViewMode>(readViewModePreference())

watch(
  () => props.functionDetail.id ?? props.functionDetail.router ?? props.functionDetail.code ?? 'default',
  () => {
    viewMode.value = readViewModePreference()
  }
)

watch(viewMode, (mode) => {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(getViewModeStorageKey(), mode)
  }
})

const sortableFields = computed(() => {
  return visibleFields.value.filter(field => getSortableConfig(field) !== false)
})

const cardSortField = computed(() => {
  return displaySorts.value[0]?.field || idField.value?.code || sortableFields.value[0]?.code || ''
})

const cardSortOrder = computed<'asc' | 'desc'>(() => {
  return displaySorts.value[0]?.order || 'desc'
})

function applyCardSort(fieldCode: string, order: 'asc' | 'desc'): void {
  if (!fieldCode) return

  const currentState = stateManager.getState()
  stateManager.setState({
    ...currentState,
    sorts: [{ field: fieldCode, order }],
    hasManualSort: true,
    pagination: {
      ...currentState.pagination,
      currentPage: 1
    }
  })
  syncToURL()
  void loadTableDataRef()
}

function handleCardSortFieldChange(value: string | number | boolean | Record<string, any> | undefined): void {
  if (typeof value !== 'string') return
  applyCardSort(value, cardSortOrder.value)
}

function handleCardSortOrderChange(value: string | number | boolean | undefined): void {
  if (value !== 'asc' && value !== 'desc') return
  applyCardSort(cardSortField.value, value)
}

const getSortHeaderClass = (fieldCode: string): string => {
  const order = sortOrderMap.value[fieldCode]
  if (!order) return ''
  return `sort-header-active sort-header-${order === 'ascending' ? 'asc' : 'desc'}`
}

// ==================== 对话框相关 ====================

// 创建对话框
const createDialogVisible = ref(false)

const {
  preloadUserInfoFromSearchForm,
  preloadDepartmentInfoFromSearchForm
} = useTableReferencePreload()

let loadTableDataRef: () => Promise<void> = async () => {}
let buildDefaultSortsRef: () => { field: string; order: 'asc' | 'desc' }[] = () => []

const { syncToURL } = useTableUrlSync({
  functionDetail: () => props.functionDetail,
  routeQuery: () => route.query as Record<string, any>,
  stateManager,
  buildDefaultSorts: () => buildDefaultSortsRef()
})

const {
  searchBarExpanded,
  idField,
  searchableFields,
  activeSearchCount,
  visibleFields,
  getSearchFieldLayoutClass,
  linkFields,
  dataFields,
  buildDefaultSorts,
  sortOrderMap,
  displaySorts,
  getFieldName,
  handleRemoveSort,
  handleClearAllSorts,
  handleSortChange,
  getSearchValue,
  updateSearchValue,
  handleSearch,
  handleReset
} = useTableSearchAndSort({
  functionDetail: () => props.functionDetail,
  domainService,
  stateManager,
  searchForm,
  sorts,
  hasManualSort,
  syncToURL,
  loadTableData: () => loadTableDataRef()
})

buildDefaultSortsRef = buildDefaultSorts

// 🔥 restoreFromURL 已移至 useTableInitialization composable

const {
  isMounted,
  skipNextTableLoad,
  loadTableData: loadTableData,
  handleSizeChange,
  handleCurrentChange
} = useTableLoadAndPagination({
  functionDetail: () => props.functionDetail,
  stateManager,
  domainService,
  applicationService,
  buildDefaultSorts: () => buildDefaultSortsRef(),
  syncToURL,
  onPageSizeChange: (size) => writeTablePageSizePreference(props.functionDetail, size)
})

loadTableDataRef = loadTableData

// ==================== 批量选择相关 ====================

const {
  isBatchDeleteMode,
  selectedRows,
  tableRef,
  enterBatchDeleteMode,
  exitBatchDeleteMode,
  handleSelectionChange,
  checkSelectable,
  handleBatchDelete
} = useTableBatchDelete({
  functionDetail: () => props.functionDetail,
  idField: () => idField.value || undefined,
  loadTableData
})

watch(viewMode, () => {
  if (isBatchDeleteMode.value) {
    exitBatchDeleteMode()
  }
})

// ==================== 其他方法 ====================

const getRowFieldValue = (row: TableRow, field: FieldConfig): FieldValue => {
  const value = row[field.code]
  if (value === null || value === undefined || value === '') {
    return createEmptyRawFieldValue()
  }
  return createAutoFieldValue(value, field)
}

const normalizeFieldCode = (code: string): string => code.trim().toLowerCase()

const findVisibleFieldByCode = (codes: string[]): FieldConfig | null => {
  const codeSet = new Set(codes.map(normalizeFieldCode))
  return visibleFields.value.find(field => codeSet.has(normalizeFieldCode(field.code))) || null
}

const isCardSystemField = (field: FieldConfig): boolean => {
  return CARD_SYSTEM_FIELD_CODES.has(normalizeFieldCode(field.code))
}

const isCardStatusField = (field: FieldConfig): boolean => {
  return CARD_STATUS_WIDGET_TYPES.has(field.widget?.type || '')
}

const hasRowFieldValue = (row: TableRow, field: FieldConfig): boolean => {
  const value = row[field.code]
  return value !== null && value !== undefined && value !== ''
}

const getCardFieldIdentity = (field: FieldConfig): string => {
  return `${field.code || ''} ${field.name || ''}`.trim().toLowerCase()
}

const fieldMatchesHints = (field: FieldConfig, hints: string[]): boolean => {
  const identity = getCardFieldIdentity(field)
  return hints.some(hint => identity.includes(hint))
}

const isCardMediaField = (field: FieldConfig): boolean => {
  return field.widget?.type === WidgetType.FILES || fieldMatchesHints(field, CARD_MEDIA_FIELD_HINTS)
}

const normalizeCardMediaEntry = (entry: unknown): string => {
  if (entry === null || entry === undefined) {
    return ''
  }

  if (typeof entry === 'string') {
    return entry.trim()
  }

  if (typeof entry === 'object') {
    const file = entry as Record<string, any>
    return String(
      file.download_url ||
      file.server_download_url ||
      file.url ||
      file.href ||
      file.ref ||
      file.key ||
      ''
    ).trim()
  }

  return String(entry).trim()
}

const parseCardMediaEntries = (raw: unknown): string[] => {
  if (raw === null || raw === undefined || raw === '') {
    return []
  }

  if (Array.isArray(raw)) {
    return raw.map(normalizeCardMediaEntry).filter(Boolean)
  }

  if (typeof raw === 'object') {
    return [normalizeCardMediaEntry(raw)].filter(Boolean)
  }

  const text = String(raw).trim()
  if (!text) {
    return []
  }

  if (text.startsWith('[') || text.startsWith('{')) {
    try {
      const parsed = JSON.parse(text)
      return parseCardMediaEntries(parsed)
    } catch {
      // Fall through to comma-separated refs.
    }
  }

  return text.split(',').map(item => item.trim()).filter(Boolean)
}

const getCardMediaEntries = (row: TableRow): string[] => {
  const field = cardMediaField.value
  return field ? parseCardMediaEntries(row[field.code]) : []
}

const getCardMediaExtension = (value: string): string => {
  const clean = value.split('?')[0]?.split('#')[0] || value
  const filename = clean.split('/').filter(Boolean).pop() || clean
  const ext = filename.includes('.') ? filename.split('.').pop() || '' : ''
  return ext.toLowerCase()
}

const isImageMediaValue = (value: string): boolean => {
  if (value.startsWith('data:image/')) {
    return true
  }
  return CARD_IMAGE_EXTENSIONS.has(getCardMediaExtension(value))
}

const shouldUseCardMediaThumbnail = (): boolean => {
  const field = cardMediaField.value
  if (field?.widget?.type !== WidgetType.FILES) {
    return false
  }
  const config = field.widget?.config || {}
  return isWidgetConfigFlagEnabled(config.thumbnail) || isWidgetConfigFlagEnabled(config.list_preview)
}

const getCardMediaPreviewUrl = (row: TableRow): string => {
  const entries = getCardMediaEntries(row)
  if (shouldUseCardMediaThumbnail()) {
    const thumbnailEntry = entries
      .map(deriveThumbnailPreviewUrl)
      .find(isImageMediaValue)
    if (thumbnailEntry) {
      return normalizeStorageFileDisplayUrl(thumbnailEntry)
    }
  }

  const imageEntry = entries.find(isImageMediaValue)
  return imageEntry ? normalizeStorageFileDisplayUrl(imageEntry) : ''
}

const getCardMediaCount = (row: TableRow): number => {
  return getCardMediaEntries(row).length
}

const isCardImageMedia = (row: TableRow): boolean => {
  return getCardMediaEntries(row).some(isImageMediaValue)
}

const getCardMediaLabel = (row: TableRow): string => {
  const count = getCardMediaCount(row)
  const fieldName = cardMediaField.value?.name || '附件'
  return count > 0 ? `${fieldName} ${count} 项` : fieldName
}

const cardCreatorField = computed(() => findVisibleFieldByCode(CARD_CREATOR_FIELD_CODES))
const cardCreatedAtField = computed(() => findVisibleFieldByCode(CARD_CREATED_AT_FIELD_CODES))
const cardUpdatedAtField = computed(() => findVisibleFieldByCode(CARD_UPDATED_AT_FIELD_CODES))

const cardBusinessFields = computed(() => {
  return dataFields.value.filter(field => !isCardSystemField(field))
})

const cardStatusFields = computed(() => {
  return cardBusinessFields.value.filter(isCardStatusField)
})

const cardContentFields = computed(() => {
  return cardBusinessFields.value.filter(field => !isCardStatusField(field))
})

const cardTitleField = computed(() => {
  const nameLikeField = cardContentFields.value.find((field) => {
    const fieldIdentity = getCardFieldIdentity(field)
    return fieldIdentity.includes('name') || fieldIdentity.includes('title') || field.name.includes('名称') || field.name.includes('标题')
  })

  return nameLikeField || cardContentFields.value[0] || idField.value || null
})

const cardMediaField = computed(() => {
  const hintedMediaField = cardBusinessFields.value.find(field => fieldMatchesHints(field, CARD_MEDIA_FIELD_HINTS))
  return hintedMediaField || cardBusinessFields.value.find(isCardMediaField) || null
})

const cardSubtitleField = computed(() => {
  const excludedCodes = new Set([
    cardTitleField.value?.code,
    cardMediaField.value?.code
  ].filter(Boolean))

  return cardContentFields.value.find((field) => {
    if (excludedCodes.has(field.code)) {
      return false
    }
    return fieldMatchesHints(field, CARD_SUBTITLE_FIELD_HINTS) || field.widget?.type === WidgetType.TEXT_AREA
  }) || null
})

const cardBodyFields = computed(() => {
  const excludedCodes = new Set([
    cardTitleField.value?.code,
    cardSubtitleField.value?.code,
    cardMediaField.value?.code
  ].filter(Boolean))

  return cardContentFields.value.filter(field => !excludedCodes.has(field.code)).slice(0, 4)
})

const cardExtraFields = computed(() => {
  const pinnedCodes = new Set([
    cardTitleField.value?.code,
    cardSubtitleField.value?.code,
    cardMediaField.value?.code,
    ...cardBodyFields.value.map(field => field.code)
  ].filter(Boolean))

  return cardContentFields.value.filter(field => !pinnedCodes.has(field.code))
})

const cardHiddenFieldCount = computed(() => {
  return cardExtraFields.value.length
})

const getCardRowKey = (row: TableRow): string | number => {
  const idCode = idField.value?.code
  if (idCode && row[idCode] !== undefined && row[idCode] !== null) {
    return row[idCode]
  }
  return tableData.value.indexOf(row)
}

const getCardSelectionKey = (row: TableRow): string => {
  const idCode = idField.value?.code
  if (idCode && row[idCode] !== undefined && row[idCode] !== null) {
    return `${idCode}:${String(row[idCode])}`
  }

  const rowIndex = tableData.value.indexOf(row)
  return rowIndex >= 0 ? `index:${rowIndex}` : `row:${String(getCardRowKey(row))}`
}

const isSameCardRow = (left: TableRow, right: TableRow): boolean => {
  return getCardSelectionKey(left) === getCardSelectionKey(right)
}

const isCardRowSelected = (row: TableRow): boolean => {
  return selectedRows.value.some(item => isSameCardRow(item, row))
}

const toggleCardSelection = (row: TableRow, checked?: boolean): void => {
  const selected = isCardRowSelected(row)
  const shouldSelect = checked ?? !selected

  if (shouldSelect && !selected) {
    selectedRows.value = [...selectedRows.value, row]
    return
  }

  if (!shouldSelect && selected) {
    selectedRows.value = selectedRows.value.filter(item => !isSameCardRow(item, row))
  }
}

const handleCardSelectionChange = (row: TableRow, checked: string | number | boolean): void => {
  toggleCardSelection(row, Boolean(checked))
}

const hasCardRowTimeMeta = (row: TableRow): boolean => {
  const createdAtField = cardCreatedAtField.value
  const updatedAtField = cardUpdatedAtField.value

  return !!(
    (createdAtField && hasRowFieldValue(row, createdAtField)) ||
    (updatedAtField && hasRowFieldValue(row, updatedAtField))
  )
}

const hasCardRowStatusMeta = (row: TableRow): boolean => {
  return cardStatusFields.value.some(field => hasRowFieldValue(row, field))
}

const getCardFilledStatusFields = (row: TableRow): FieldConfig[] => {
  return cardStatusFields.value.filter(field => hasRowFieldValue(row, field))
}

const normalizeCardOption = (option: any): { label: string; value: any } => {
  if (option && typeof option === 'object') {
    return {
      label: String(option.label ?? option.value ?? ''),
      value: option.value ?? option.label
    }
  }

  return {
    label: String(option ?? ''),
    value: option
  }
}

const getCardFieldOptions = (field: FieldConfig): Array<{ label: string; value: any }> => {
  const options = field.widget?.config?.options
  return Array.isArray(options) ? options.map(normalizeCardOption) : []
}

const getCardOptionColor = (field: FieldConfig, value: any): string | null => {
  const options = getCardFieldOptions(field)
  const optionColors = getFieldWidgetOptionColors(field)
  if (optionColors.length === 0) {
    return null
  }

  const optionIndex = options.findIndex(option => String(option.value) === String(value))
  if (optionIndex < 0 || optionIndex >= optionColors.length) {
    return null
  }

  const color = optionColors[optionIndex]
  return typeof color === 'string' ? color : null
}

const getCardOptionLabel = (field: FieldConfig, value: any): string => {
  const options = getCardFieldOptions(field)
  const option = options.find(item => String(item.value) === String(value))
  return option?.label || String(value)
}

const parseCardStatusValues = (row: TableRow, field: FieldConfig): any[] => {
  const raw = row[field.code]
  if (raw === null || raw === undefined || raw === '') {
    return []
  }

  if (Array.isArray(raw)) {
    return raw
  }

  if (
    (field.widget?.type === WidgetType.MULTI_SELECT || field.widget?.type === WidgetType.CHECKBOX) &&
    typeof raw === 'string' &&
    raw.includes(',')
  ) {
    return raw.split(',').map(value => value.trim()).filter(Boolean)
  }

  return [raw]
}

const getCardStatusTagStyle = (field: FieldConfig, value: any): Record<string, string> => {
  const color = getCardOptionColor(field, value)
  const palette = getOptionLightPalette(color)
  if (!palette) {
    return {}
  }

  return {
    backgroundColor: palette.backgroundColor,
    borderColor: palette.borderColor,
    color: palette.color
  }
}

const getCardStatusTags = (row: TableRow, field: FieldConfig): CardStatusTag[] => {
  const values = parseCardStatusValues(row, field)
  if (values.length === 0) {
    return []
  }

  return values.map((value, index) => ({
    key: `${field.code}-${index}-${String(value)}`,
    label: getCardOptionLabel(field, value),
    style: getCardStatusTagStyle(field, value)
  }))
}

const handleCardClick = (row: TableRow): void => {
  if (isBatchDeleteMode.value) {
    toggleCardSelection(row)
    return
  }

  handleDetail(row)
}

/**
 * 获取操作列宽度（统一「更多」下拉，固定宽度）
 */
const getActionColumnWidth = (): number => {
  return 90
}

// ==================== 回调判断 ====================

const hasAddCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableAddRow')
})

const hasDeleteCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableDeleteRows')
})

const hasUpdateCallback = computed(() => {
  return getFunctionCallbacks(props.functionDetail).includes('OnTableUpdateRow')
})

const {
  currentFunctionNode,
  handleAdd,
  handleCreateSubmit,
  handleCreateDialogClose
} = useTableCreateAndPermissions({
  routeQuery: () => route.query as Record<string, any>,
  functionDetail: () => props.functionDetail,
  workspaceStateManager,
  applicationService,
  createDialogVisible
})

const {
  handleActionCommand,
  getLinkText,
  getColumnWidth,
  handleDetail,
  handleRowClick
} = useTableRowActions({
  functionDetail: () => props.functionDetail,
  router,
  stateManager,
  applicationService,
  linkFields: () => linkFields.value,
  skipNextTableLoad
})

// 🔥 使用 composable 统一管理初始化逻辑
const { initializeTable, setupQueryWatch } = useTableInitialization({
  functionDetail: computed(() => props.functionDetail),
  domainService,
  applicationService,
  stateManager,
  searchForm,
  sorts,
  hasManualSort,
  buildDefaultSorts,
  syncToURL,
  loadTableData,
  isMounted, // 🔥 传递挂载状态，用于防止卸载后继续加载数据
  preloadUserInfoFromSearchForm, // 🔥 时机 1：预加载搜索表单中的用户信息
  preloadDepartmentInfoFromSearchForm // 🔥 时机 1：预加载搜索表单中的部门信息
})

useTableAddDialogUrlSync({
  createDialogVisible,
  hasAddCallback: () => hasAddCallback.value,
  isMounted: () => isMounted.value
})

useTableViewLifecycle({
  functionDetailId: props.functionDetail.id,
  isMounted,
  initializeTable,
  setupQueryWatch,
  stateManager
})
</script>

<style scoped>
.table-view {
  --table-control-bg: var(--app-shell-panel-bg-strong);
  --table-control-bg-hover: rgba(var(--el-color-primary-rgb), 0.06);
  --table-control-border: var(--app-shell-panel-border);
  --table-control-border-hover: rgba(var(--el-color-primary-rgb), 0.28);
  --table-control-shadow-hover: 0 8px 18px rgba(15, 23, 42, 0.08);
  padding: 10px 0 0;
  background: transparent;
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
}

.toolbar {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 0;
  border: none !important;
  border-radius: 0;
  background: transparent !important;
  box-shadow: none !important;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-left: auto;
}

.view-mode-toggle {
  flex: 0 0 auto;
}

.view-mode-toggle :deep(.el-radio-button__inner) {
  height: 36px;
  padding: 0 12px;
  display: inline-flex;
  align-items: center;
  border-color: var(--table-control-border);
  background: var(--table-control-bg);
  font-weight: 600;
}

.view-mode-toggle :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-radius: 8px 0 0 8px;
}

.view-mode-toggle :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: 0 8px 8px 0;
}

.view-mode-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  line-height: 1;
}

.toolbar :deep(.el-button) {
  height: 36px;
  border-radius: 8px;
  font-weight: 600;
  transition: border-color 0.18s ease, background-color 0.18s ease, color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.toolbar .action-btn {
  padding: 0 14px;
  box-shadow: none;
}

.toolbar .action-btn:not(.el-button--primary) {
  border: 1px solid var(--table-control-border);
  background: var(--table-control-bg);
}

.toolbar-search-btn:not(.el-button--primary),
.toolbar-secondary-btn {
  padding: 0 14px;
  border: 1px solid var(--table-control-border);
  background: var(--table-control-bg);
  box-shadow: none;
}

.toolbar .action-btn:not(.el-button--primary):hover,
.toolbar-search-btn:not(.el-button--primary):hover,
.toolbar-secondary-btn:hover {
  border-color: var(--table-control-border-hover);
  color: var(--el-color-primary);
  background: var(--table-control-bg-hover);
  box-shadow: var(--table-control-shadow-hover);
  transform: translateY(-1px);
}

.toolbar .action-btn.el-button--primary {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.toolbar .action-btn.el-button--primary:hover {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
  transform: translateY(-1px);
}

.toolbar .action-btn.el-button--danger,
.toolbar .action-btn-danger {
  border-color: rgba(239, 68, 68, 0.26);
  background: #fff1f2;
  color: #dc2626;
  box-shadow: none;
}

.toolbar .action-btn.el-button--danger:hover,
.toolbar .action-btn-danger:hover {
  border-color: rgba(239, 68, 68, 0.38);
  background: #ffe4e6;
  color: #b91c1c;
  box-shadow: 0 10px 24px rgba(239, 68, 68, 0.14);
  transform: translateY(-1px);
}

.toolbar-search-btn.el-button--primary {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.toolbar-search-btn.el-button--primary:hover {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
  transform: translateY(-1px);
}

.toolbar-collapse-btn {
  padding: 0 2px;
  color: var(--el-text-color-secondary);
}

.toolbar-collapse-btn:hover {
  color: var(--el-color-primary);
}

.card-sort-controls {
  min-height: 36px;
  padding: 3px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  background: var(--table-control-bg);
}

.card-sort-label {
  padding: 0 7px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.card-sort-select {
  width: 138px;
}

.card-sort-select :deep(.el-select__wrapper) {
  min-height: 28px;
  border-radius: 6px;
  background: var(--app-shell-panel-muted-bg);
  box-shadow: none;
}

.card-sort-order :deep(.el-radio-button__inner) {
  min-width: 32px;
  height: 28px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-color: var(--table-control-border);
  background: transparent;
}

.card-sort-order :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-radius: 6px 0 0 6px;
}

.card-sort-order :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: 0 6px 6px 0;
}

.card-sort-order :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: none;
}

.search-bar-wrapper {
  margin-bottom: 16px;
}

.search-bar {
  margin-bottom: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

.search-bar-inner {
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--table-control-border);
  background: var(--app-shell-panel-bg-strong);
  box-shadow: var(--app-auth-card-shadow-soft);
}

.search-bar .search-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px 14px;
  align-items: start;
}

.search-bar :deep(.search-form > .el-form-item) {
  display: flex;
  flex-direction: column;
  justify-self: stretch;
  margin-right: 0;
  margin-bottom: 0;
  width: 100%;
  min-width: 0;
  align-items: flex-start;
}

.search-bar :deep(.search-form > .el-form-item.search-field-layout--wide) {
  grid-column: span 2;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__label-wrap) {
  width: 100%;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__label) {
  width: 100%;
  display: flex;
  justify-content: flex-start;
  padding: 0 0 6px;
  line-height: 1.25;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__content) {
  display: flex;
  align-items: stretch;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
  margin-left: 0 !important;
}

.search-bar :deep(.search-form > .el-form-item .el-form-item__content > *) {
  width: 100%;
  min-width: 0;
}

.search-bar :deep(.el-input__wrapper),
.search-bar :deep(.el-select__wrapper),
.search-bar :deep(.el-textarea__inner),
.search-bar :deep(.el-date-editor .el-input__wrapper),
.search-bar :deep(.department-select-display),
.search-bar :deep(.user-search-display) {
  background: var(--app-auth-input-bg);
  border-color: var(--app-auth-input-border);
  border-radius: 8px;
  box-shadow: none;
  transition: all 0.3s ease;
}

.search-bar :deep(.el-input__wrapper:hover),
.search-bar :deep(.el-select__wrapper:hover),
.search-bar :deep(.el-textarea__inner:hover),
.search-bar :deep(.el-date-editor .el-input__wrapper:hover),
.search-bar :deep(.department-select-display:hover),
.search-bar :deep(.user-search-display:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.search-bar :deep(.el-input__wrapper.is-focus),
.search-bar :deep(.el-select__wrapper.is-focused),
.search-bar :deep(.el-textarea__inner:focus),
.search-bar :deep(.el-date-editor .el-input__wrapper.is-focus) {
  border-color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-focus);
}

.search-bar :deep(.el-input__inner::placeholder),
.search-bar :deep(.el-textarea__inner::placeholder) {
  color: #94a3b8;
}

/* 🔥 排序信息条样式 */
.sort-info-bar {
  margin-bottom: 16px;
  padding: 10px 12px;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  box-shadow: var(--app-auth-card-shadow-soft);
  display: flex;
  align-items: center;
}

.sort-info-content {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  flex-wrap: wrap;
}

.sort-label {
  height: 26px;
  display: inline-flex;
  align-items: center;
  padding: 0 9px;
  border-radius: 8px;
  background: var(--app-shell-panel-muted-bg);
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 700;
  white-space: nowrap;
}

.sort-items {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  flex: 1;
}

.sort-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: default;
}

.sort-field-name {
  font-weight: 500;
}

.sort-icon {
  font-size: 12px;
  margin-left: 2px;
}

.sort-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  font-weight: 500;
  margin: 0 4px;
}

.table-clear-sorts-btn {
  margin-left: auto;
  white-space: nowrap;
  height: 30px;
  padding: 0 10px;
  border: 1px solid var(--table-control-border);
  border-radius: 8px;
  background: var(--table-control-bg);
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.table-clear-sorts-btn:hover {
  border-color: var(--table-control-border-hover);
  background: var(--table-control-bg-hover);
  color: var(--el-color-primary);
}

/* 表格骨架屏（加载中） */
.table-skeleton-wrap {
  min-height: 320px;
  padding: 4px 0 16px;
}

.table-skeleton-wrap .table-skeleton {
  width: 100%;
}

@media (max-width: 900px) {
  .search-bar .search-form {
    grid-template-columns: 1fr;
  }

  .search-bar :deep(.search-form > .el-form-item.search-field-layout--wide) {
    grid-column: 1 / -1;
  }
}

.table-skeleton-wrap .el-skeleton__item {
  margin-bottom: 12px;
}

.table-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 340px), 1fr));
  gap: 16px;
  align-items: stretch;
}

.table-card-empty {
  grid-column: 1 / -1;
  min-height: 280px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 8px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: var(--app-shell-panel-shadow-soft);
}

.table-card-item {
  min-width: 0;
  min-height: 292px;
  padding: 18px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background:
    linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.055), transparent 38%),
    radial-gradient(circle at 92% 0%, rgba(14, 165, 233, 0.1), transparent 32%),
    var(--app-shell-panel-bg-strong);
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.075);
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 13px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.table-card-item::before {
  content: '';
  height: 3px;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--el-color-primary), rgba(14, 165, 233, 0.45), transparent);
  opacity: 0.72;
  position: absolute;
  top: 0;
  left: 14px;
  right: 14px;
}

.table-card-item:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  box-shadow: 0 20px 42px rgba(15, 23, 42, 0.11), 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.08);
  transform: translateY(-2px);
}

.table-card-item.is-selected {
  border-color: rgba(var(--el-color-primary-rgb), 0.46);
  background:
    linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.12) 0%, rgba(var(--el-color-primary-rgb), 0.05) 100%),
    var(--app-shell-panel-bg-strong);
  box-shadow: 0 18px 38px rgba(var(--el-color-primary-rgb), 0.16);
}

.table-card-item.is-selectable {
  cursor: pointer;
}

.table-card-item:focus-visible {
  outline: 2px solid var(--el-color-primary-light-5);
  outline-offset: 2px;
}

.table-card-header {
  min-width: 0;
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.table-card-checkbox {
  margin-top: 3px;
  flex: 0 0 auto;
}

.table-card-media {
  width: 56px;
  height: 56px;
  flex: 0 0 auto;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 8px;
  background:
    linear-gradient(135deg, rgba(var(--el-color-primary-rgb), 0.14), rgba(14, 165, 233, 0.08)),
    var(--app-shell-panel-muted-bg);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight), 0 8px 18px rgba(15, 23, 42, 0.08);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.table-card-media.has-image {
  background: var(--app-shell-panel-muted-bg);
}

.table-card-media-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.table-card-media-icon {
  color: var(--el-color-primary);
  font-size: 24px;
}

.table-card-media-count {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.74);
  color: #fff;
  font-size: 10px;
  font-weight: 800;
  line-height: 18px;
  text-align: center;
  position: absolute;
  right: 4px;
  bottom: 4px;
}

.table-card-title-wrap {
  min-width: 0;
  flex: 1 1 auto;
}

.table-card-kicker {
  margin-bottom: 5px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.2;
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.table-card-id-chip {
  max-width: 100px;
  height: 20px;
  padding: 0 6px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 999px;
  background: var(--app-shell-panel-muted-bg);
  color: var(--el-text-color-secondary);
  display: inline-flex;
  align-items: center;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-card-title {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 17px;
  font-weight: 800;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.table-card-title :deep(*) {
  max-width: 100%;
}

.table-card-subtitle {
  min-width: 0;
  max-height: 38px;
  margin-top: 5px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.table-card-subtitle :deep(*) {
  max-width: 100%;
}

.table-card-view-btn {
  flex: 0 0 auto;
  height: 32px;
  padding: 0 10px;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
  border-radius: 8px;
  background: rgba(var(--el-color-primary-rgb), 0.07);
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  cursor: pointer;
  flex: 0 0 auto;
  transition: border-color 0.18s ease, background-color 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
  font-size: 12px;
  font-weight: 700;
}

.table-card-view-btn:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  background: rgba(var(--el-color-primary-rgb), 0.12);
  color: var(--el-color-primary);
  box-shadow: 0 8px 18px rgba(var(--el-color-primary-rgb), 0.12);
}

.table-card-fields {
  min-width: 0;
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
  flex: 1 1 auto;
}

.table-card-field {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
  align-items: start;
  padding: 10px 11px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-shell-panel-bg-strong) 86%, var(--el-color-primary) 4%);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  transition: border-color 0.18s ease, background-color 0.18s ease, transform 0.18s ease;
}

.table-card-field:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.22);
  background: rgba(var(--el-color-primary-rgb), 0.045);
  transform: translateY(-1px);
}

.table-card-field-label {
  min-width: 0;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.table-card-field-value {
  min-width: 0;
  max-height: 42px;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.35;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.table-card-field-value :deep(*) {
  max-width: 100%;
}

.table-card-status-badges,
.table-card-status-strip {
  min-width: 0;
  margin-top: 9px;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex-wrap: wrap;
}

.table-card-status-item {
  min-width: 0;
  max-width: 100%;
  min-height: 24px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 1 auto;
}

.table-card-status-label {
  flex: 0 0 auto;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  overflow-wrap: anywhere;
}

.table-card-status-tags {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
}

.table-card-status-tag {
  margin: 0;
  border-style: solid;
  border-width: 1px;
  border-radius: 999px;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.1);
  max-width: 150px;
}

.table-card-status-tag :deep(.el-tag__content) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-card-more-section {
  min-width: 0;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  padding-top: 8px;
}

.table-card-more-summary {
  width: 100%;
  min-height: 30px;
  padding: 0 10px;
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 8px;
  background: rgba(var(--el-color-primary-rgb), 0.065);
  cursor: pointer;
  user-select: none;
  font-size: 12px;
  font-weight: 700;
  list-style: none;
}

.table-card-more-summary::-webkit-details-marker {
  display: none;
}

.table-card-more-summary::before {
  content: '';
  width: 0;
  height: 0;
  border-top: 4px solid transparent;
  border-bottom: 4px solid transparent;
  border-left: 5px solid currentColor;
  transition: transform 0.18s ease;
}

.table-card-more-section[open] .table-card-more-summary::before {
  transform: rotate(90deg);
}

.table-card-more-count {
  margin-left: auto;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(var(--el-color-primary-rgb), 0.1);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  line-height: 1;
}

.table-card-more-fields-panel {
  margin-top: 8px;
  padding: 4px 0 0;
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}

.table-card-more-field {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
  align-items: start;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid rgba(148, 163, 184, 0.13);
  background: color-mix(in srgb, var(--app-shell-panel-muted-bg) 88%, var(--app-shell-panel-bg-strong) 12%);
}

.table-card-more-field .table-card-field-value {
  max-height: none;
}

.table-card-time-meta {
  min-width: 0;
  padding: 10px 11px;
  border: 1px dashed rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-shell-panel-muted-bg) 72%, transparent);
  display: grid;
  grid-template-columns: 1fr;
  gap: 6px;
}

.table-card-time-item {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.25;
}

.table-card-time-item .el-icon {
  flex: 0 0 auto;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.table-card-time-label {
  flex: 0 0 auto;
  font-weight: 600;
}

.table-card-time-value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-card-footer {
  margin-top: auto;
  padding: 12px 12px 0;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  background: linear-gradient(180deg, transparent, color-mix(in srgb, var(--app-shell-panel-muted-bg) 58%, transparent));
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
}

.table-card-creator {
  min-width: 0;
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.table-card-creator-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.table-card-creator-label {
  min-width: 0;
  color: var(--el-text-color-secondary);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.1;
}

.table-card-creator-value {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-card-footer-actions {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex: 0 0 auto;
}

.table-card-more-fields {
  min-width: 0;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 640px) {
  .table-card-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .table-card-item {
    min-height: 0;
    padding: 15px;
  }

  .table-card-header {
    gap: 9px;
  }

  .table-card-media {
    width: 44px;
    height: 44px;
  }

  .table-card-view-btn {
    width: 32px;
    padding: 0;
  }

  .table-card-view-btn span {
    display: none;
  }

  .table-card-footer {
    padding: 10px 0 0;
    align-items: stretch;
    flex-direction: column;
  }

  .table-card-footer-actions {
    width: 100%;
    justify-content: space-between;
  }

  .table-card-more-fields {
    white-space: normal;
  }
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 🔥 表格基础样式 */
:deep(.el-table) {
  background-color: var(--app-shell-panel-bg-strong) !important;
  border: 1px solid var(--app-shell-panel-border) !important;
  border-radius: 8px !important;
  box-shadow: var(--app-shell-panel-shadow-soft);
  flex: 1;
  overflow: auto;
}

:deep(.el-table__inner-wrapper) {
  border: none !important;
  border-radius: 8px !important;
}

:deep(.el-table__inner-wrapper::before) {
  display: none !important;
}

:deep(.el-table__header-wrapper) {
  border: none !important;
}

:deep(.el-table__body-wrapper) {
  border: none !important;
}

:deep(.el-table th),
:deep(.el-table td) {
  border-right: none !important;
  border-left: none !important;
}

:deep(.el-table th:first-child),
:deep(.el-table td:first-child) {
  border-left: none !important;
}

:deep(.el-table th:last-child),
:deep(.el-table td:last-child) {
  border-right: none !important;
}

:deep(.el-table__body tr) {
  background-color: transparent !important;
}

:deep(.el-table__body tr.el-table__row--striped) {
  background-color: transparent !important;
}

:deep(.el-table__body tr.el-table__row--striped td) {
  background-color: var(--app-shell-panel-bg-strong) !important;
}

:deep(.el-table__body tr:hover > td) {
  background-color: rgba(var(--el-color-primary-rgb), 0.04) !important;
}

/* 整行可点击进入详情 */
:deep(.table-row-clickable .el-table__body tr) {
  cursor: pointer;
}

:deep(.el-table__header th.el-table__cell) {
  background-color: var(--app-shell-panel-muted-bg);
  color: var(--el-text-color-secondary);
  font-weight: 600;
  border-top: none;
}

:deep(.el-table__header th.el-table__cell.sort-header-active) {
  color: var(--el-color-primary);
}

:deep(.el-table__header th.el-table__cell.sort-header-active .cell) {
  font-weight: 700;
}

:deep(.el-table__header th.el-table__cell.sort-header-asc .sort-caret.ascending) {
  border-bottom-color: var(--el-color-primary) !important;
}

:deep(.el-table__header th.el-table__cell.sort-header-desc .sort-caret.descending) {
  border-top-color: var(--el-color-primary) !important;
}

:deep(.el-table__header th.el-table__cell.sort-header-active .caret-wrapper) {
  opacity: 1;
}

:deep(.el-table td.el-table__cell),
:deep(.el-table th.el-table__cell.is-leaf) {
  border-bottom: 1px solid var(--app-shell-panel-border);
}

:deep(.el-table td.el-table__cell) {
  background: var(--app-shell-panel-bg-strong) !important;
}

.detail-icon-button {
  min-width: 44px;
  height: 32px;
  padding: 0 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--el-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  transition: background-color 0.2s ease, color 0.2s ease;
}

.detail-icon-button:hover {
  background-color: var(--el-color-primary-light-9);
}

.detail-icon-button:focus-visible {
  outline: 2px solid var(--el-color-primary-light-5);
  outline-offset: 2px;
}

.detail-id-text {
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

.dropdown-link-content,
.dropdown-action-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.link-icon {
  font-size: 14px;
}

.internal-icon {
  color: var(--el-color-primary);
}

.delete-action-text {
  color: var(--el-color-danger);
}

</style>

<style lang="scss">
.table-action-dropdown.el-popper {
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 8px;
  box-shadow: 0 16px 34px rgba(15, 23, 42, 0.14);
}

.table-action-dropdown .el-dropdown-menu {
  padding: 6px;
  border-radius: 8px;
}

.table-action-dropdown .el-dropdown-menu__item {
  min-height: 34px;
  border-radius: 6px;
  font-weight: 500;
}

.table-action-dropdown .el-dropdown-menu__item:not(.is-disabled):hover {
  background: rgba(var(--el-color-primary-rgb), 0.07);
  color: var(--el-color-primary);
}
</style>
