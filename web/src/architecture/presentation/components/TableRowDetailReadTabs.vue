<template>
  <el-tabs v-model="activeTabModel" class="detail-tabs">
    <el-tab-pane label="详情" name="detail">
      <div class="tab-content">
        <div v-if="linkFields.length > 0" class="detail-links-section">
          <div class="links-section-title">相关链接</div>
          <div class="links-section-content">
            <LinkWidget
              v-for="linkField in linkFields"
              :key="linkField.code"
              :field="linkField"
              :value="getFieldValue(linkField.code)"
              :field-path="linkField.code"
              mode="detail"
              class="detail-link-item"
            />
          </div>
        </div>

        <div v-if="useGroupedDetailLayout" class="grouped-detail-layout">
          <div v-if="groupedFields.statusFields.length > 0" class="status-section">
            <div
              v-for="field in groupedFields.statusFields"
              :key="field.code"
              class="status-field-card"
            >
              <span class="status-label">{{ field.name }}</span>
              <div class="status-value">
                <WidgetComponent
                  :field="field"
                  :value="getFieldValue(field.code)"
                  mode="detail"
                />
              </div>
            </div>
          </div>

          <div class="main-layout">
            <div class="main-content">
              <div
                v-for="field in groupedFields.mainContentFields"
                :key="field.code"
                :class="field.widget?.type === WidgetType.RICH_TEXT ? 'main-content-rich-text' : 'field-row'"
              >
                <template v-if="field.widget?.type === WidgetType.RICH_TEXT">
                  <div class="rich-text-field-card">
                    <div class="rich-text-field-header">
                      <span class="rich-text-field-name">{{ field.name }}</span>
                    </div>
                    <div class="rich-text-field-content">
                      <div
                        class="rich-text-preview-shell"
                        :class="{
                          'is-collapsed': !isRichTextExpanded(field.code),
                          'is-expandable': isRichTextOverflow(field.code)
                        }"
                        :style="{ '--rich-text-preview-height': `${richTextPreviewHeight}px` }"
                      >
                        <div
                          :ref="(el) => setRichTextContentRef(field.code, el as Element | null)"
                          class="rich-text-preview-inner"
                        >
                          <WidgetComponent
                            :field="field"
                            :value="getFieldValue(field.code)"
                            mode="detail"
                          />
                        </div>
                      </div>
                      <div v-if="isRichTextOverflow(field.code)" class="rich-text-preview-actions">
                        <el-button
                          type="primary"
                          link
                          @click="toggleRichTextExpanded(field.code)"
                        >
                          {{ isRichTextExpanded(field.code) ? '收起' : '展开全文' }}
                        </el-button>
                      </div>
                    </div>
                  </div>
                </template>
                <template v-else>
                  <div class="field-label">
                    {{ field.name }}
                  </div>
                  <div class="field-value">
                    <WidgetComponent
                      :field="field"
                      :value="getFieldValue(field.code)"
                      mode="detail"
                    />
                  </div>
                </template>
              </div>
            </div>

            <div class="sidebar-content">
              <div v-if="groupedFields.idField" class="metadata-section">
                <div class="metadata-section-title">基本信息</div>
                <div class="field-row metadata-field">
                  <div class="field-label">ID</div>
                  <div class="field-value">
                    <WidgetComponent
                      :field="groupedFields.idField"
                      :value="getFieldValue(groupedFields.idField.code)"
                      mode="detail"
                    />
                  </div>
                </div>
              </div>

              <div v-if="groupedFields.userFields.length > 0" class="metadata-section">
                <div class="metadata-section-title">人员信息</div>
                <div
                  v-for="field in groupedFields.userFields"
                  :key="field.code"
                  class="field-row metadata-field"
                >
                  <div class="field-label">
                    {{ field.name }}
                  </div>
                  <div class="field-value">
                    <WidgetComponent
                      :field="field"
                      :value="getFieldValue(field.code)"
                      mode="detail"
                    />
                  </div>
                </div>
              </div>

              <div v-if="groupedFields.dateTimeFields.length > 0" class="metadata-section">
                <div class="metadata-section-title">时间信息</div>
                <div
                  v-for="field in groupedFields.dateTimeFields"
                  :key="field.code"
                  class="field-row metadata-field"
                >
                  <div class="field-label">
                    {{ field.name }}
                  </div>
                  <div class="field-value">
                    <WidgetComponent
                      :field="field"
                      :value="getFieldValue(field.code)"
                      mode="detail"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="groupedFields.complexFields.length > 0" class="complex-section">
            <div
              v-for="field in groupedFields.complexFields"
              :key="field.code"
              class="complex-field-card"
            >
              <el-collapse>
                <el-collapse-item :name="field.code">
                  <template #title>
                    <div class="complex-field-title">
                      <span class="complex-field-name">{{ field.name }}</span>
                    </div>
                  </template>
                  <div class="complex-field-content">
                    <WidgetComponent
                      :field="field"
                      :value="getFieldValue(field.code)"
                      mode="detail"
                    />
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </div>

        <div v-else class="detail-fields-grid">
          <div
            v-for="field in fields.filter((f) => f.widget?.type !== WidgetType.LINK)"
            :key="field.code"
            class="detail-field-row"
          >
            <div class="detail-field-label">
              {{ field.name }}
            </div>
            <div class="detail-field-value">
              <WidgetComponent
                :field="field"
                :value="getFieldValue(field.code)"
                mode="detail"
              />
            </div>
          </div>
        </div>
      </div>
    </el-tab-pane>

    <el-tab-pane v-if="featureFlags.operateLogs" label="操作日志" name="operateLog">
      <div class="tab-content">
        <OperateLogSection
          ref="operateLogSectionRef"
          :full-code-path="fullCodePath"
          :row-id="rowId"
          :function-detail="functionDetail"
          :auto-load="false"
        />
      </div>
    </el-tab-pane>

  </el-tabs>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { FieldConfig, FunctionDetail } from '../../domain/types'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import OperateLogSection from './OperateLogSection.vue'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { featureFlags } from '@/architecture/shared/config/features'

interface GroupedFields {
  statusFields: FieldConfig[]
  mainContentFields: FieldConfig[]
  idField?: FieldConfig | null
  userFields: FieldConfig[]
  dateTimeFields: FieldConfig[]
  complexFields: FieldConfig[]
}

interface LoadableOperateLogSection {
  load: () => void
}

const props = defineProps<{
  modelValue: string
  fields: FieldConfig[]
  linkFields: FieldConfig[]
  groupedFields: GroupedFields
  useGroupedDetailLayout: boolean
  richTextPreviewHeight: number
  fullCodePath: string
  rowId: number
  functionDetail: FunctionDetail | null
  getFieldValue: (fieldCode: string) => any
  setRichTextContentRef: (fieldCode: string, el: Element | null) => void
  isRichTextExpanded: (fieldCode: string) => boolean
  isRichTextOverflow: (fieldCode: string) => boolean
  toggleRichTextExpanded: (fieldCode: string) => void
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const operateLogSectionRef = ref<LoadableOperateLogSection | null>(null)

const activeTabModel = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value),
})

function loadTab(tabName: string) {
  if (tabName === 'operateLog' && featureFlags.operateLogs) {
    operateLogSectionRef.value?.load()
  }
}

watch(
  () => props.modelValue,
  (tabName) => {
    if (tabName === 'operateLog' && !featureFlags.operateLogs) {
      emit('update:modelValue', 'detail')
      return
    }
    if (tabName === 'operateLog') {
      nextTick(() => loadTab(tabName))
    }
  },
  { immediate: true }
)
</script>

<style scoped lang="scss">
.detail-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 20px;
  }

  :deep(.el-tabs__item) {
    font-size: 14px;
    font-weight: 500;
  }

  :deep(.el-tabs__active-bar) {
    background-color: var(--el-color-primary);
  }
}

.tab-content {
  padding: 0;
}

.detail-links-section {
  margin-bottom: 24px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.links-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.links-section-content {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.detail-link-item {
  flex-shrink: 0;
}

.detail-fields-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 4px;
}

.detail-field-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: start;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.detail-field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.detail-field-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-field-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  word-break: break-word;
  line-height: 1.6;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 24px;
  pointer-events: auto;
  position: relative;
}

.grouped-detail-layout {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.status-section {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
}

.status-field-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.2s ease;
}

.status-field-card:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.status-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.status-value {
  flex: 1;
  min-width: 0;
}

.main-layout {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

@media (max-width: 1200px) {
  .main-layout {
    grid-template-columns: 1fr;
  }

  .sidebar-content {
    position: static !important;
    max-height: none !important;
  }
}

.main-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 0;
  position: sticky;
  top: 20px;
  max-height: calc(100vh - 200px);
  overflow-y: auto;
}

.metadata-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metadata-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.metadata-field {
  padding: 8px 0;
  border-bottom: none;
}

.grouped-detail-layout .main-content .field-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 12px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: start;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.grouped-detail-layout .main-content .field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.grouped-detail-layout .sidebar-content .field-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: stretch;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.grouped-detail-layout .sidebar-content .field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.grouped-detail-layout .field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.grouped-detail-layout .sidebar-content .field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.grouped-detail-layout .field-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  word-break: break-word;
  line-height: 1.6;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 24px;
  position: relative;
}

.grouped-detail-layout .sidebar-content .field-value {
  font-size: 13px;
  width: 100%;
}

.main-content-rich-text {
  padding-top: 12px;
}

.rich-text-field-card {
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
}

.rich-text-field-header {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.rich-text-field-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.rich-text-field-content {
  padding: 16px;
}

.rich-text-preview-shell {
  position: relative;
  width: 100%;
}

.rich-text-preview-shell.is-collapsed {
  max-height: var(--rich-text-preview-height);
  overflow: hidden;
}

.rich-text-preview-shell.is-collapsed.is-expandable::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 72px;
  background: linear-gradient(to bottom, rgba(255, 255, 255, 0), var(--el-bg-color));
  pointer-events: none;
}

.rich-text-preview-inner {
  width: 100%;
}

.rich-text-preview-actions {
  display: flex;
  justify-content: center;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.complex-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
}

.complex-field-card {
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
  overflow: hidden;
}

.complex-field-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.complex-field-name {
  flex: 1;
}

.complex-field-content {
  padding: 16px;
}
</style>
