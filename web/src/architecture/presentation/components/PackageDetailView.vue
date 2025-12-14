<!--
  PackageDetailView - 服务目录详情页面
  
  职责：
  - 显示服务目录信息
  - 提供"生成系统"按钮，点击后打开智能体选择对话框
-->
<template>
  <div class="package-detail-view">
    <div class="detail-header">
      <div class="header-left">
        <el-button @click="handleBack" :icon="ArrowLeft">返回</el-button>
        <h2 class="detail-title">{{ packageNode?.name || '服务目录' }}</h2>
      </div>
      <div class="header-right">
        <el-button type="primary" :icon="MagicStick" @click="handleGenerateSystem">
          生成系统
        </el-button>
      </div>
    </div>
    
    <div class="detail-content">
      <el-card v-if="packageNode" class="info-card">
        <template #header>
          <div class="card-header">
            <span>目录信息</span>
          </div>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="目录名称">{{ packageNode.name }}</el-descriptions-item>
          <el-descriptions-item label="目录代码">{{ packageNode.code }}</el-descriptions-item>
          <el-descriptions-item label="完整路径" :span="2">{{ packageNode.full_code_path }}</el-descriptions-item>
        </el-descriptions>
      </el-card>
      
      <el-card class="children-card" v-if="packageNode?.children && packageNode.children.length > 0">
        <template #header>
          <div class="card-header">
            <span>子目录和函数</span>
            <span class="count-badge">{{ packageNode.children.length }}</span>
          </div>
        </template>
        <div class="children-list">
          <div
            v-for="child in packageNode.children"
            :key="child.id"
            class="child-item"
          >
            <el-icon>
              <Folder v-if="child.type === 'package'" />
              <Document v-else />
            </el-icon>
            <span class="child-name">{{ child.name }}</span>
            <el-tag v-if="child.type === 'function'" size="small" type="info">
              {{ child.template_type === 'table' ? '表格' : child.template_type === 'form' ? '表单' : '函数' }}
            </el-tag>
          </div>
        </div>
      </el-card>
      
      <el-empty v-else description="该目录下暂无子目录或函数" :image-size="100" />
    </div>
    
    <!-- 智能体选择对话框 -->
    <AgentSelectDialog
      v-model="agentSelectDialogVisible"
      :tree-id="packageNode?.id || null"
      :package="packageNode?.code || ''"
      :current-node-name="packageNode?.name || ''"
      @confirm="handleAgentSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, MagicStick, Folder, Document } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import type { AgentInfo } from '@/api/agent'
import { extractWorkspacePath } from '@/utils/route'

interface Props {
  packageNode?: ServiceTree | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'generate-system': [agent: AgentInfo]
}>()

const router = useRouter()
const route = useRoute()

const agentSelectDialogVisible = ref(false)
const selectedAgent = ref<AgentInfo | null>(null)

// 返回上一级
function handleBack() {
  // 获取当前路径，去掉最后一段
  const currentPath = extractWorkspacePath(route.path)
  if (currentPath) {
    const pathSegments = currentPath.split('/').filter(Boolean)
    if (pathSegments.length > 2) {
      // 至少是 user/app/package，去掉最后一段
      pathSegments.pop()
      const parentPath = `/workspace/${pathSegments.join('/')}`
      router.push(parentPath)
    } else {
      // 回到根目录
      router.push('/workspace')
    }
  } else {
    router.push('/workspace')
  }
}

// 打开生成系统对话框
function handleGenerateSystem() {
  agentSelectDialogVisible.value = true
}

// 选择智能体后的处理
function handleAgentSelect(agent: AgentInfo) {
  selectedAgent.value = agent
  // 触发生成系统事件，让父组件处理
  emit('generate-system', agent)
  agentSelectDialogVisible.value = false
}
</script>

<style scoped lang="scss">
.package-detail-view {
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
    .children-card {
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
    
    .children-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      
      .child-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        border-radius: 4px;
        background: var(--el-fill-color-lighter);
        
        .child-name {
          flex: 1;
        }
      }
    }
  }
}
</style>

