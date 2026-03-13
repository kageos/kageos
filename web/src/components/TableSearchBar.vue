<template>
  <div class="ui-filter-bar">
    <el-form :model="searchForm" label-position="left" label-width="96px" class="ui-filter-form">
      <template v-for="field in searchableFields" :key="field.code">
        <!-- 🔥 通过 Widget 渲染搜索输入（组件自治） -->
        <el-form-item :label="field.name" class="ui-filter-item">
          <SearchInput
            :field="field"
            :search-type="field.search"
            :model-value="getSearchValue(field)"
            :function-method="functionData.method"
            :function-router="functionData.router"
            @update:model-value="(value: any) => {
              // 🔥 判断是否清空：值为 null 或空字符串，且之前有值
              const isClearing = (value === null || value === '') && 
                                 searchForm && 
                                 searchForm[field.code] !== undefined
              updateSearchValue(field, value, isClearing)
            }"
          />
        </el-form-item>
      </template>

      <el-form-item class="ui-filter-item ui-filter-actions">
        <el-button type="primary" @click="handleSearch">
          <el-icon><Search /></el-icon>
          搜索
        </el-button>
        <el-button @click="handleReset">
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { Search, Refresh } from '@element-plus/icons-vue'
import { ElIcon, ElButton } from 'element-plus'
import SearchInput from './SearchInput.vue'
import type { Function as FunctionType } from '@/types'
import type { FieldConfig } from '@/core/types/field'

interface Props {
  /** 可搜索字段列表 */
  searchableFields: FieldConfig[]
  /** 搜索表单数据 */
  searchForm: Record<string, any>
  /** 函数配置数据 */
  functionData: FunctionType
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'search'): void
  (e: 'reset'): void
  (e: 'update:searchForm', value: Record<string, any>): void
}>()

/**
 * 获取搜索值
 * @param field 字段配置
 * @returns 搜索值
 */
const getSearchValue = (field: FieldConfig): any => {
  const value = props.searchForm[field.code]
  // 🔥 如果值是 undefined，返回 null；否则返回原值（包括空对象、空数组等）
  return value === undefined ? null : value
}

/**
 * 更新搜索值
 * @param field 字段配置
 * @param value 新的搜索值
 * @param shouldSearch 是否自动搜索（默认 false，清空时设为 true）
 */
const updateSearchValue = (field: FieldConfig, value: any, shouldSearch: boolean = false): void => {
  // 🔥 如果值为空（空数组、空字符串、null、undefined），删除该字段
  const newSearchForm = { ...props.searchForm }
  if (value === null || value === undefined || 
      (Array.isArray(value) && value.length === 0) || 
      (typeof value === 'string' && value.trim() === '')) {
    delete newSearchForm[field.code]
  } else {
    newSearchForm[field.code] = value
  }
  
  // 触发更新事件
  emit('update:searchForm', newSearchForm)
  
  // 🔥 如果需要自动搜索（清空时），触发搜索
  if (shouldSearch) {
    emit('search')
  }
}

/**
 * 处理搜索
 */
const handleSearch = (): void => {
  emit('search')
}

/**
 * 处理重置
 */
const handleReset = (): void => {
  emit('reset')
}
</script>

