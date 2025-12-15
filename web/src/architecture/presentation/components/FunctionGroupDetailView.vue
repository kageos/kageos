<!--
  FunctionGroupDetailView - 函数组详情页面
  
  职责：
  - 显示函数组信息
  - 显示函数组下的所有函数
-->
<template>
  <div class="function-group-detail-view">
    <div class="detail-header">
      <div class="header-left">
        <el-button @click="handleBack" :icon="ArrowLeft">返回</el-button>
        <h2 class="detail-title">{{ groupName || '函数组' }}</h2>
      </div>
    </div>
    
    <div class="detail-content">
      <el-card class="info-card">
        <template #header>
          <div class="card-header">
            <span>函数组信息</span>
          </div>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="函数组名称">{{ groupName }}</el-descriptions-item>
          <el-descriptions-item label="函数组代码">{{ fullGroupCode.value }}</el-descriptions-item>
          <el-descriptions-item label="函数数量" :span="2">
            {{ functions.length }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>
      
      <el-card class="functions-card" v-if="functions.length > 0">
        <template #header>
          <div class="card-header">
            <span>函数列表</span>
            <span class="count-badge">{{ functions.length }}</span>
          </div>
        </template>
        <div class="functions-list">
          <div
            v-for="func in functions"
            :key="func.id"
            class="function-item"
            @click="handleFunctionClick(func)"
          >
            <el-icon>
              <Grid v-if="func.template_type === 'table'" />
              <Postcard v-else-if="func.template_type === 'form'" />
              <Document v-else />
            </el-icon>
            <span class="function-name">{{ func.name }}</span>
            <el-tag size="small" :type="func.template_type === 'table' ? 'success' : 'primary'">
              {{ func.template_type === 'table' ? '表格' : func.template_type === 'form' ? '表单' : '函数' }}
            </el-tag>
            <el-icon class="arrow-icon"><ArrowRight /></el-icon>
          </div>
        </div>
      </el-card>
      
      <el-empty v-else description="该函数组下暂无函数" :image-size="100" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, ArrowRight, Grid, Postcard, Document } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { extractFullGroupCodeFromRoute, getParentPathFromFullGroupCode } from '@/utils/route'
import { findFunctionGroup } from '@/utils/serviceTreeUtils'

interface Props {
  serviceTree?: ServiceTree[]
  fullGroupCode?: string
}

const props = defineProps<Props>()

const router = useRouter()
const route = useRoute()

const functions = ref<ServiceTree[]>([])
const groupName = ref<string>('')
const fullGroupCode = computed(() => {
  // 优先使用 props，如果没有则从路由路径中提取
  if (props.fullGroupCode) {
    return props.fullGroupCode
  }
  // 从路由路径中提取 full_group_code
  return extractFullGroupCodeFromRoute(route.path)
})

// 查找函数组和函数
function loadFunctionGroup() {
  if (!props.serviceTree || !fullGroupCode.value) {
    return
  }
  
  // 使用工具函数查找函数组
  const { groupNode, functions: matchedFunctions } = findFunctionGroup(
    props.serviceTree,
    fullGroupCode.value
  )
  
  if (groupNode && (groupNode as any).isGroup) {
    // 如果找到函数组节点，获取其子函数
    functions.value = (groupNode.children || []).filter(child => child.type === 'function')
    groupName.value = groupNode.name || (groupNode as any).group_name || '函数组'
  } else {
    // 如果没有找到函数组节点，使用匹配的函数列表
    functions.value = matchedFunctions
    // 使用第一个函数的 group_name 作为组名
    if (matchedFunctions.length > 0 && (matchedFunctions[0] as any).group_name) {
      groupName.value = (matchedFunctions[0] as any).group_name
    } else {
      // 从 full_group_code 提取组名
      const segments = fullGroupCode.value.split('/').filter(Boolean)
      groupName.value = segments[segments.length - 1] || '函数组'
    }
  }
}

// 返回上一级
function handleBack() {
  // 移除 _node_type 查询参数，并返回到父目录
  const parentPath = getParentPathFromFullGroupCode(fullGroupCode.value)
  if (parentPath) {
    router.push({
      path: `/workspace${parentPath}`,
      query: {}
    })
  } else {
    // 回到根目录
    router.push('/workspace')
  }
}

// 点击函数，跳转到函数详情
function handleFunctionClick(func: ServiceTree) {
  if (func.full_code_path) {
    const targetPath = `/workspace${func.full_code_path}`
    router.push(targetPath)
  }
}

// 监听 props 和路由变化
watch(() => [props.serviceTree, fullGroupCode.value], () => {
  loadFunctionGroup()
}, { immediate: true, deep: true })

onMounted(() => {
  loadFunctionGroup()
})
</script>

<style scoped lang="scss">
.function-group-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
    
    .header-left {
      display: flex;
      align-items: center;
      gap: 16px;
      
      .detail-title {
        margin: 0;
        font-size: 20px;
        font-weight: 500;
      }
    }
  }
  
  .detail-content {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
    
    .info-card,
    .functions-card {
      margin-bottom: 16px;
      
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        
        .count-badge {
          padding: 2px 8px;
          background: var(--el-color-primary-light-9);
          color: var(--el-color-primary);
          border-radius: 12px;
          font-size: 12px;
        }
      }
    }
    
    .functions-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      
      .function-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 12px;
        border-radius: 4px;
        background: var(--el-fill-color-lighter);
        cursor: pointer;
        transition: all 0.2s;
        
        &:hover {
          background: var(--el-fill-color);
          transform: translateX(4px);
        }
        
        .function-name {
          flex: 1;
        }
        
        .arrow-icon {
          color: var(--el-text-color-secondary);
        }
      }
    }
  }
}
</style>

