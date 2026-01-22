<!--
  DocsPathSelector - 文档路径选择器组件
  功能：
  - 支持多选服务树中的文档路径（package 或 docs 类型节点）
  - 显示已选中的路径
  - 点击后弹出对话框，显示服务树供选择
-->
<template>
  <div class="docs-path-selector">
    <!-- 已选中的路径显示 -->
    <div class="selected-paths">
      <div class="input-wrapper">
        <el-input
          v-model="pathsInput"
          type="textarea"
          :rows="2"
          placeholder="请输入文档路径，多个路径用逗号分隔，如：/system/official/sdk,/system/official/plugins"
          @blur="handleInputBlur"
        />
        <el-button
          :icon="Document"
          @click="handleOpenTreeDialog"
          style="margin-left: 8px;"
        >
          搜索文档
        </el-button>
      </div>
      <div v-if="selectedPaths.length > 0" class="path-tags" style="margin-top: 8px;">
        <el-tag
          v-for="(path, index) in selectedPaths"
          :key="index"
          closable
          @close="handleRemovePath(index)"
          style="margin-right: 8px; margin-bottom: 4px;"
        >
          {{ path }}
        </el-tag>
      </div>
    </div>
    
    <!-- 文档路径选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="选择文档路径"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="selector-content">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索文档路径或名称..."
          clearable
          style="margin-bottom: 12px;"
          @input="handleSearchInput"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        
        <div v-loading="searchLoading" style="min-height: 200px;">
          <div v-if="searchResults.length === 0 && !searchLoading" class="empty-state">
            <el-empty description="请输入关键词搜索文档" />
          </div>
          
          <el-checkbox-group v-model="tempSelectedPaths" v-else>
            <div
              v-for="doc in searchResults"
              :key="doc.full_code_path"
              class="doc-item"
            >
              <el-checkbox :label="doc.full_code_path">
                <div class="doc-item-content">
                  <el-icon class="doc-icon"><Document /></el-icon>
                  <span class="doc-name">{{ doc.name }}</span>
                  <span class="doc-path">({{ doc.full_code_path }})</span>
                </div>
              </el-checkbox>
            </div>
          </el-checkbox-group>
          
          <!-- 分页 -->
          <el-pagination
            v-if="searchTotal > 0"
            v-model:current-page="searchPage"
            v-model:page-size="searchPageSize"
            :total="searchTotal"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next"
            style="margin-top: 16px; justify-content: center;"
            @size-change="handleSearch"
            @current-change="handleSearch"
          />
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            @click="handleConfirm"
          >
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElButton, ElDialog, ElTag, ElInput, ElIcon, ElMessage, ElCheckbox, ElCheckboxGroup, ElPagination, ElEmpty } from 'element-plus'
import { Document, Search } from '@element-plus/icons-vue'
import { queryDocs, type DocSearchResult } from '@/api/doc'

interface Props {
  modelValue: string // 逗号分隔的路径字符串，如："/system/official/sdk,/user/myapp/docs"
  user?: string // 用户（可选，如果不提供则只显示标准库路径）
  app?: string // 应用（可选，如果不提供则只显示标准库路径）
}

const props = withDefaults(defineProps<Props>(), {
  user: '',
  app: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const searchLoading = ref(false)
const searchResults = ref<DocSearchResult[]>([])
const searchTotal = ref(0)
const searchPage = ref(1)
const searchPageSize = ref(20)
const tempSelectedPaths = ref<string[]>([])
const pathsInput = ref('')


// 当前选中的路径数组
const selectedPaths = computed({
  get: () => {
    if (!props.modelValue || typeof props.modelValue !== 'string') return []
    return props.modelValue.split(',').map(p => p.trim()).filter(p => p)
  },
  set: (value: string[]) => {
    emit('update:modelValue', value.length > 0 ? value.join(',') : '')
  }
})

// 同步输入框和选中路径
watch(() => props.modelValue, (newVal) => {
  pathsInput.value = newVal || ''
}, { immediate: true })

// 处理输入框失焦
const handleInputBlur = () => {
  // 从输入框更新 modelValue
  const paths = pathsInput.value.split(',').map(p => p.trim()).filter(p => p)
  selectedPaths.value = paths
}

// 搜索文档（防抖）
let searchTimer: ReturnType<typeof setTimeout> | null = null
const handleSearchInput = () => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    if (searchKeyword.value.trim()) {
      searchPage.value = 1
      handleSearch()
    } else {
      searchResults.value = []
      searchTotal.value = 0
    }
  }, 300) // 300ms 防抖
}

// 执行搜索
const handleSearch = async () => {
  if (!searchKeyword.value.trim()) {
    searchResults.value = []
    searchTotal.value = 0
    return
  }
  
  searchLoading.value = true
  try {
    const resp = await queryDocs({
      keyword: searchKeyword.value.trim(),
      page: searchPage.value,
      page_size: searchPageSize.value,
      include_content: false // 列表展示不需要内容，减少数据传输
    })
    searchResults.value = resp.docs || []
    searchTotal.value = resp.total || 0
  } catch (error: any) {
    console.error('搜索文档失败:', error)
    ElMessage.error(error.message || '搜索文档失败')
    searchResults.value = []
    searchTotal.value = 0
  } finally {
    searchLoading.value = false
  }
}

// 打开对话框
const handleOpenTreeDialog = () => {
  dialogVisible.value = true
  searchKeyword.value = ''
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  tempSelectedPaths.value = [...selectedPaths.value]
}

// 确认选择
const handleConfirm = () => {
  // 合并已选中的路径和从搜索选择的路径（去重）
  const existingPaths = selectedPaths.value
  const newPaths = tempSelectedPaths.value
  const mergedPaths = Array.from(new Set([...existingPaths, ...newPaths]))
  selectedPaths.value = mergedPaths
  pathsInput.value = mergedPaths.join(',')
  dialogVisible.value = false
}

// 移除路径
const handleRemovePath = (index: number) => {
  const newPaths = [...selectedPaths.value]
  newPaths.splice(index, 1)
  selectedPaths.value = newPaths
  pathsInput.value = newPaths.join(',')
}
</script>

<style scoped lang="scss">
.docs-path-selector {
  .selected-paths {
    width: 100%;
    
    .input-wrapper {
      display: flex;
      align-items: flex-start;
      
      .el-textarea {
        flex: 1;
      }
    }
    
    .path-tags {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
    }
  }
  
  .selector-content {
    .empty-state {
      padding: 40px 0;
      text-align: center;
    }
    
    .doc-item {
      padding: 8px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);
      
      &:last-child {
        border-bottom: none;
      }
      
      .doc-item-content {
        display: flex;
        align-items: center;
        
        .doc-icon {
          margin-right: 8px;
          color: var(--el-color-primary);
        }
        
        .doc-name {
          font-weight: 500;
          margin-right: 8px;
        }
        
        .doc-path {
          font-size: 12px;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }
}
</style>
