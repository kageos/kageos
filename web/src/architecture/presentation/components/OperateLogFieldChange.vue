<template>
  <div class="operate-log-field-change" :class="{ 'is-file-change': isFilesField }">
    <div class="operate-log-field-change__name">{{ fieldName || fieldCode }}</div>
    <div class="operate-log-field-change__values">
      <div class="operate-log-field-change__panel is-old">
        <div class="operate-log-field-change__label">{{ oldLabel }}</div>
        <OperateLogFieldValue
          v-if="hasOldValue"
          :field="field"
          :raw-value="oldValue"
          :field-path="fieldCode"
          :empty-text="emptyText"
          compact
        />
        <span v-else class="operate-log-field-change__empty">{{ noOldValueText }}</span>
      </div>

      <div class="operate-log-field-change__panel is-new">
        <div class="operate-log-field-change__label">{{ newLabel }}</div>
        <OperateLogFieldValue
          :field="field"
          :raw-value="newValue"
          :field-path="fieldCode"
          :empty-text="emptyText"
          compact
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { WidgetType } from '@/architecture/domain/constants/widget'
import type { FieldConfig } from '@/architecture/domain/types'
import OperateLogFieldValue from './OperateLogFieldValue.vue'

const props = withDefaults(defineProps<{
  fieldCode: string
  fieldName: string
  field?: FieldConfig | null
  oldValue: unknown
  newValue: unknown
  hasOldValue: boolean
  oldLabel?: string
  newLabel?: string
  emptyText?: string
  noOldValueText?: string
}>(), {
  field: null,
  oldLabel: '更新前',
  newLabel: '更新后',
  emptyText: '-',
  noOldValueText: '未记录旧值',
})

const isFilesField = computed(() => props.field?.widget?.type === WidgetType.FILES)
</script>

<style scoped>
.operate-log-field-change {
  display: grid;
  grid-template-columns: minmax(96px, 160px) minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-width: 0;
  padding: 10px;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}

.operate-log-field-change__name {
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 700;
  line-height: 24px;
  word-break: break-word;
}

.operate-log-field-change__values {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 8px;
  min-width: 0;
}

.operate-log-field-change__panel {
  min-width: 0;
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid var(--el-border-color-extra-light);
}

.operate-log-field-change__panel.is-old {
  background: rgba(245, 108, 108, 0.06);
  border-color: rgba(245, 108, 108, 0.18);
}

.operate-log-field-change__panel.is-new {
  background: rgba(103, 194, 58, 0.07);
  border-color: rgba(103, 194, 58, 0.2);
}

.operate-log-field-change__label {
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}

.operate-log-field-change__empty {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
  line-height: 1.5;
}

.operate-log-field-change.is-file-change {
  grid-template-columns: 1fr;
  gap: 8px;
}

.operate-log-field-change.is-file-change .operate-log-field-change__name {
  line-height: 1.35;
}

.operate-log-field-change.is-file-change .operate-log-field-change__panel {
  padding: 10px;
}

@media (max-width: 920px) {
  .operate-log-field-change__values {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .operate-log-field-change {
    grid-template-columns: 1fr;
  }
}
</style>
