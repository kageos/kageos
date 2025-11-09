<!--
  FormWidget - 表单容器组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持 mode="edit" - 可编辑表单
  - 支持 mode="response" - 只读表单
  - 支持 mode="table-cell" - 表格单元格（简化显示 + 详情抽屉）
  - 递归渲染子组件
  - 支持条件渲染
-->

<template>
  <div class="form-widget">
    <!-- 编辑模式 -->
    <el-form
      v-if="mode === 'edit'"
      :model="formData"
      label-width="100px"
    >
      <el-form-item
        v-for="subField in visibleSubFields"
        :key="subField.code"
        :label="subField.name"
        :required="isFieldRequired(subField)"
      >
        <!-- 🔥 递归渲染子组件 -->
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :value="getSubFieldValue(subField.code)"
            :model-value="getSubFieldValue(subField.code)"
            @update:model-value="(v) => updateSubFieldValue(subField.code, v)"
            :field-path="`${fieldPath}.${subField.code}`"
            :form-manager="formManager"
            :form-renderer="formRenderer"
            :mode="mode"
            :depth="(depth || 0) + 1"
          />
      </el-form-item>
    </el-form>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-form">
      <div
        v-for="subField in visibleSubFields"
        :key="subField.code"
        class="response-field"
      >
        <div class="field-label">{{ subField.name }}</div>
        <div class="field-value">
          <!-- 🔥 递归渲染子组件 -->
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :value="getSubFieldValue(subField.code)"
            :model-value="getSubFieldValue(subField.code)"
            :field-path="`${fieldPath}.${subField.code}`"
            mode="response"
            :depth="(depth || 0) + 1"
          />
        </div>
      </div>
    </div>
    
    <!-- 表格单元格模式（简化显示 + 详情抽屉） -->
    <template v-else-if="mode === 'table-cell'">
      <el-button
        link
        type="primary"
        size="small"
        @click="showDetailDrawer = true"
        class="form-field-button"
      >
        <span>共 {{ fieldCount }} 个字段</span>
        <el-icon style="margin-left: 4px">
          <View />
        </el-icon>
      </el-button>
      
      <!-- 详情抽屉（支持编辑和查看） -->
      <el-drawer
        v-model="showDetailDrawer"
        :title="field.name"
        size="60%"
        destroy-on-close
        :z-index="3000"
        append-to-body
      >
        <template #default>
          <div class="form-detail-content">
            <!-- 🔥 抽屉中使用与正常编辑模式完全一致的渲染逻辑 -->
            <!-- 直接使用 edit 模式的渲染方式，确保逻辑一致 -->
            <el-form
              :model="formData"
              label-width="120px"
            >
              <el-form-item
                v-for="subField in visibleSubFields"
                :key="subField.code"
                :label="subField.name"
                :required="isFieldRequired(subField)"
              >
                <!-- 🔥 递归渲染子组件，使用与正常编辑模式完全相同的逻辑 -->
                <component
                  :is="getWidgetComponent(subField.widget?.type || 'input')"
                  :field="subField"
                  :value="getSubFieldValue(subField.code)"
                  :model-value="getSubFieldValue(subField.code)"
                  @update:model-value="(v) => updateSubFieldValue(subField.code, v)"
                  :field-path="`${fieldPath}.${subField.code}`"
                  :form-manager="formManager"
                  :form-renderer="formRenderer"
                  mode="edit"
                  :depth="(depth || 0) + 1"
                />
              </el-form-item>
            </el-form>
          </div>
        </template>
      </el-drawer>
    </template>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-form">
      <div
        v-for="subField in visibleSubFields"
        :key="subField.code"
        class="detail-field"
      >
        <div class="field-label">{{ subField.name }}</div>
        <div class="field-value">
          <component
            :is="getWidgetComponent(subField.widget?.type || 'input')"
            :field="subField"
            :value="getSubFieldValue(subField.code)"
            :model-value="getSubFieldValue(subField.code)"
            :field-path="`${fieldPath}.${subField.code}`"
            mode="detail"
            :depth="(depth || 0) + 1"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElForm, ElFormItem, ElButton, ElDrawer, ElIcon } from 'element-plus'
import { View } from '@element-plus/icons-vue'
import type { WidgetComponentProps } from '../types'
import { useFormWidget } from '../composables/useFormWidget'
import { widgetComponentFactory } from '../../factories-v2'
import type { FieldConfig } from '../../types/field'

const props = defineProps<WidgetComponentProps>()

// 使用组合式函数
const { visibleSubFields, getSubFieldValue, updateSubFieldValue } = useFormWidget(props)

// 详情抽屉状态（用于 table-cell 模式）
const showDetailDrawer = ref(false)

// 字段数量（用于 table-cell 模式显示）
const fieldCount = computed(() => {
  const raw = props.value?.raw
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
    return Object.keys(raw).length
  }
  return visibleSubFields.value.length
})

// 表单数据（用于 el-form 绑定）
const formData = computed(() => {
  const data: Record<string, any> = {}
  visibleSubFields.value.forEach(subField => {
    const value = getSubFieldValue(subField.code)
    data[subField.code] = value?.raw
  })
  return data
})

// 获取组件
function getWidgetComponent(type: string) {
  return widgetComponentFactory.getRequestComponent(type)
}

// 检查字段是否必填
function isFieldRequired(field: FieldConfig): boolean {
  const validation = field.validation || ''
  return validation.includes('required') && !validation.includes('omitempty')
}
</script>

<style scoped>
.form-widget {
  width: 100%;
}

.response-form {
  width: 100%;
}

.response-field {
  margin-bottom: 16px;
}

.field-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.field-value {
  color: var(--el-text-color-regular);
}

/* 表格单元格模式 */
.form-field-button {
  padding: 0;
  height: auto;
  font-size: 14px;
}

/* 详情抽屉内容 */
.form-detail-content {
  padding: 16px 0;
  /* 确保下拉菜单可以正常显示 */
  overflow: visible;
}

.detail-field {
  margin-bottom: 24px;
}

.detail-form {
  width: 100%;
}

/* 确保抽屉内的下拉菜单可以正常显示 */
:deep(.el-select-dropdown) {
  z-index: 3001 !important;
}

:deep(.el-popper) {
  z-index: 3001 !important;
}
</style>
