<!--
  FunctionBreadcrumb - 函数详情面包屑导航组件
  
  职责：
  - 显示从根目录到当前函数的路径
  - 支持点击面包屑节点导航到对应目录或函数
-->
<template>
  <div v-if="breadcrumbItems.length > 0" class="function-breadcrumb">
    <el-breadcrumb separator="/">
      <el-breadcrumb-item
        v-for="(item, index) in breadcrumbItems"
        :key="index"
        :class="{ 'is-current': index === breadcrumbItems.length - 1 }"
      >
        <span
          v-if="index === breadcrumbItems.length - 1"
          class="breadcrumb-item-text"
        >
          {{ item.name }}
        </span>
        <a
          v-else
          href="javascript:void(0)"
          class="breadcrumb-item-link"
          @click="handleBreadcrumbClick(item)"
        >
          {{ item.name }}
        </a>
      </el-breadcrumb-item>
    </el-breadcrumb>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ServiceTree } from '@/types'

interface Props {
  currentNode: ServiceTree | null
  serviceTree: ServiceTree[]
}

interface Emits {
  (e: 'node-click', node: ServiceTree): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

/**
 * 根据 full_code_path 查找节点（递归查找）
 */
function findNodeByPath(tree: ServiceTree[], path: string): ServiceTree | null {
  for (const node of tree) {
    // 规范化路径比较（移除末尾斜杠）
    const nodePath = (node.full_code_path || '').replace(/\/$/, '')
    const targetPath = path.replace(/\/$/, '')
    
    if (nodePath === targetPath) {
      return node
    }
    if (node.children && node.children.length > 0) {
      const found = findNodeByPath(node.children, path)
      if (found) return found
    }
  }
  return null
}

/**
 * 构建面包屑路径
 */
const breadcrumbItems = computed(() => {
  if (!props.currentNode || !props.currentNode.full_code_path) {
    return []
  }

  const path = props.currentNode.full_code_path
  const parts = path.split('/').filter(Boolean)
  
  if (parts.length === 0) {
    return []
  }

  const items: Array<{ name: string; path: string; node: ServiceTree | null }> = []
  
  // 构建路径数组，从根到当前节点
  for (let i = 0; i < parts.length; i++) {
    const currentPath = '/' + parts.slice(0, i + 1).join('/')
    const node = findNodeByPath(props.serviceTree, currentPath)
    
    // 如果找不到节点，使用路径的最后一部分作为名称
    const name = node?.name || node?.code || parts[i] || ''
    
    items.push({
      name,
      path: currentPath,
      node: node || null
    })
  }

  return items
})

/**
 * 处理面包屑点击
 */
function handleBreadcrumbClick(item: { name: string; path: string; node: ServiceTree | null }) {
  if (item.node) {
    emit('node-click', item.node)
  }
}
</script>

<style scoped lang="scss">
.function-breadcrumb {
  padding: 12px 20px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  
  :deep(.el-breadcrumb) {
    font-size: 14px;
    
    .el-breadcrumb__item {
      .breadcrumb-item-link {
        color: var(--el-text-color-regular);
        text-decoration: none;
        transition: color 0.2s;
        
        &:hover {
          color: #6366f1;
        }
      }
      
      .breadcrumb-item-text {
        color: var(--el-text-color-primary);
        font-weight: 500;
      }
      
      &.is-current {
        .el-breadcrumb__inner {
          color: var(--el-text-color-primary);
          font-weight: 500;
        }
      }
    }
  }
}
</style>

