<template>
  <el-popover
    placement="bottom-start"
    trigger="hover"
    :show-after="250"
    :width="680"
    popper-class="table-spreadsheet-guide-popper"
  >
    <template #reference>
      <el-button
        text
        circle
        class="spreadsheet-guide-trigger"
        aria-label="预览表格填写说明"
        data-testid="table-spreadsheet-guide"
      >
        <el-icon><QuestionFilled /></el-icon>
      </el-button>
    </template>

    <div class="guide-content">
      <strong class="guide-title">Excel / CSV 怎么填写</strong>
      <p class="guide-summary">
        下面的规则来自当前表格新增函数。附件和子表不进入批量模板，请在页面新增或详情中上传和维护。
      </p>
      <el-table :data="guideRows" size="small" max-height="420" border>
        <el-table-column prop="name" label="字段" width="132">
          <template #default="{ row }">
            <span>{{ row.name }}</span>
            <el-tag v-if="row.required" type="danger" size="small" effect="plain">必填</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="填写类型" width="108" />
        <el-table-column prop="example" label="示例" min-width="148" show-overflow-tooltip />
        <el-table-column prop="guide" label="填写规则" min-width="260">
          <template #default="{ row }">
            <span class="guide-rule">{{ row.guide }}</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { QuestionFilled } from '@element-plus/icons-vue'
import type { FieldConfig } from '@/architecture/domain/types'
import {
  describeTableSpreadsheetField,
  getTableSpreadsheetFieldExample,
  getTableSpreadsheetFieldTypeLabel,
  isTableSpreadsheetFieldRequired,
  isTableSpreadsheetFieldSupported
} from '@/architecture/presentation/views/utils/tableSpreadsheetRuntime'

const props = defineProps<{
  fields: FieldConfig[]
}>()

const guideRows = computed(() => props.fields
  .filter(isTableSpreadsheetFieldSupported)
  .map((field) => ({
    code: field.code,
    name: field.name,
    required: isTableSpreadsheetFieldRequired(field),
    type: getTableSpreadsheetFieldTypeLabel(field),
    example: getTableSpreadsheetFieldExample(field),
    guide: describeTableSpreadsheetField(field)
  })))
</script>

<style scoped>
.spreadsheet-guide-trigger {
  width: 30px;
  height: 30px;
  color: var(--el-text-color-secondary);
}

.spreadsheet-guide-trigger:hover {
  color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.08);
}

.guide-title {
  display: block;
  margin-bottom: 6px;
  font-size: 15px;
  color: var(--el-text-color-primary);
}

.guide-summary {
  margin: 0 0 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.guide-content :deep(.el-tag) {
  margin-left: 6px;
}

.guide-rule {
  white-space: normal;
  line-height: 1.5;
}
</style>
