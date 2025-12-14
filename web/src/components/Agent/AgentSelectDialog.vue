<template>
  <el-dialog
    v-model="visible"
    title="选择智能体"
    width="90%"
    :close-on-click-modal="false"
    @close="handleClose"
    class="agent-select-dialog-wrapper"
  >
    <div v-loading="loading" class="agent-select-dialog">
      <!-- 筛选条件 -->
      <div class="filter-section">
        <el-form :inline="true" :model="filterForm" class="filter-form">
          <el-form-item label="智能体类型">
            <el-select
              v-model="filterForm.agent_type"
              placeholder="全部类型"
              clearable
              style="width: 150px"
            >
              <el-option label="纯知识库类型" value="knowledge_only" />
              <el-option label="插件类型" value="plugin" />
            </el-select>
          </el-form-item>
          <el-form-item label="聊天类型">
            <el-select
              v-model="filterForm.chat_type"
              placeholder="全部类型"
              clearable
              style="width: 150px"
            >
              <el-option label="函数生成" value="function_gen" />
              <el-option label="任务对话" value="chat-task" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
            <el-button :icon="Refresh" @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 智能体卡片列表 -->
      <div class="agent-cards">
        <div
          v-for="agent in filteredAgents"
          :key="agent.id"
          :class="['agent-card', { selected: selectedAgentId === agent.id }]"
          @click="handleSelectAgent(agent)"
        >
          <div class="agent-card-header">
            <el-avatar
              :size="48"
              :src="getAgentLogo(agent)"
              class="agent-logo"
            >
              <span class="agent-logo-text">{{ getAgentLogoText(agent) }}</span>
            </el-avatar>
            <div class="agent-card-title">
              <div class="agent-name">{{ agent.name }}</div>
              <div class="agent-tags">
                <el-tag
                  :type="agent.agent_type === 'plugin' ? 'warning' : 'success'"
                  size="small"
                >
                  {{ agent.agent_type === 'plugin' ? '插件' : agent.agent_type === 'knowledge_only' ? '知识库' : agent.agent_type }}
                </el-tag>
                <el-tag
                  type="info"
                  size="small"
                  style="margin-left: 4px;"
                >
                  {{ getChatTypeLabel(agent.chat_type) }}
                </el-tag>
              </div>
            </div>
            <el-icon v-if="selectedAgentId === agent.id" class="selected-icon">
              <CircleCheck />
            </el-icon>
          </div>
          <div v-if="agent.description" class="agent-description">
            {{ agent.description }}
          </div>
          <div class="agent-info">
            <div v-if="agent.knowledge_base" class="info-item">
              <el-icon><Document /></el-icon>
              <span>知识库：{{ agent.knowledge_base.name }}</span>
            </div>
            <div v-if="agent.llm_config" class="info-item">
              <el-icon><Cpu /></el-icon>
              <span>LLM：{{ agent.llm_config.name }}</span>
            </div>
          </div>
        </div>
        <el-empty
          v-if="!loading && filteredAgents.length === 0"
          description="暂无可用智能体"
          :image-size="80"
        />
      </div>
    </div>

  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { Search, Refresh, CircleCheck, Document, Cpu } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getAgentList, type AgentInfo, type AgentListReq } from '@/api/agent'

interface Props {
  modelValue: boolean
  treeId?: number | null // 服务目录ID
  package?: string // Package 名称
  currentNodeName?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  treeId: null,
  package: '',
  currentNodeName: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'confirm': [agent: AgentInfo]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const loading = ref(false)
const agentList = ref<AgentInfo[]>([])
const selectedAgentId = ref<number | null>(null)

const filterForm = reactive<{
  agent_type?: 'knowledge_only' | 'plugin'
  chat_type?: string
}>({})

// 过滤后的智能体列表
const filteredAgents = computed(() => {
  let result = agentList.value

  if (filterForm.agent_type) {
    result = result.filter(agent => agent.agent_type === filterForm.agent_type)
  }

  if (filterForm.chat_type) {
    result = result.filter(agent => agent.chat_type === filterForm.chat_type)
  }

  return result
})

// 获取聊天类型标签
function getChatTypeLabel(chatType: string): string {
  const labels: Record<string, string> = {
    function_gen: '函数生成',
    'chat-task': '任务对话'
  }
  return labels[chatType] || chatType
}

// 获取智能体 Logo（如果有则使用，否则使用默认生成的）
function getAgentLogo(agent: AgentInfo): string {
  if (agent.logo) {
    return agent.logo
  }
  // 生成默认 Logo（使用智能体 ID 生成唯一颜色）
  return generateDefaultLogo(agent.id, agent.name)
}

// 生成默认 Logo URL（使用智能体 ID 生成唯一颜色）
function generateDefaultLogo(agentId: number, agentName: string): string {
  // 使用智能体 ID 生成一个稳定的颜色
  const colors = [
    '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399',
    '#606266', '#303133', '#409EFF', '#67C23A', '#E6A23C'
  ]
  const colorIndex = agentId % colors.length
  const color = colors[colorIndex]
  
  // 生成 SVG data URL
  const svg = `
    <svg width="48" height="48" xmlns="http://www.w3.org/2000/svg">
      <rect width="48" height="48" fill="${color}" rx="8"/>
      <text x="24" y="32" font-family="Arial, sans-serif" font-size="20" font-weight="bold" fill="white" text-anchor="middle">${getAgentLogoText({ id: agentId, name: agentName } as AgentInfo)}</text>
    </svg>
  `.trim()
  
  return `data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(svg)))}`
}

// 获取智能体 Logo 文本（取名称首字符）
function getAgentLogoText(agent: AgentInfo): string {
  if (!agent.name) return 'A'
  // 取第一个字符（支持中文）
  const firstChar = agent.name.charAt(0)
  return firstChar.toUpperCase()
}

// 加载智能体列表
async function loadAgents() {
  loading.value = true
  try {
    const params: AgentListReq = {
      enabled: true,
      scope: 'market', // 显示市场中的公开智能体
      page: 1,
      page_size: 1000
    }
    const res = await getAgentList(params)
    agentList.value = res.agents || []
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    ElMessage.error(error.message || '加载智能体列表失败')
    agentList.value = []
  } finally {
    loading.value = false
  }
}

// 选择智能体（点击即确认）
function handleSelectAgent(agent: AgentInfo) {
  selectedAgentId.value = agent.id
  // 直接触发确认事件并关闭对话框
  emit('confirm', agent)
  handleClose()
}

// 搜索
function handleSearch() {
  // 过滤逻辑已在 computed 中实现
}

// 重置
function handleReset() {
  filterForm.agent_type = undefined
  filterForm.chat_type = undefined
}

// 关闭对话框
function handleClose() {
  visible.value = false
  selectedAgentId.value = null
  handleReset()
}

// 监听对话框打开，加载智能体列表
watch(visible, (newValue) => {
  if (newValue) {
    loadAgents()
  }
})
</script>

<style lang="scss">
.agent-select-dialog-wrapper {
  .el-dialog {
    max-width: 1400px;
  }
  
  .el-dialog__body {
    padding: 20px;
  }
}
</style>

<style scoped lang="scss">
.agent-select-dialog {
  .filter-section {
    margin-bottom: 20px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--el-border-color-light);
  }

  .filter-form {
    margin: 0;
  }

  .agent-cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 16px;
    max-height: 600px;
    overflow-y: auto;
    padding: 8px;
  }

  .agent-card {
    padding: 16px;
    border: 2px solid var(--el-border-color-light);
    border-radius: 12px;
    cursor: pointer;
    transition: all 0.3s;
    background: var(--el-bg-color);
    display: flex;
    flex-direction: column;
    gap: 12px;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      transform: translateY(-2px);
    }

    &.selected {
      border-color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
      box-shadow: 0 0 0 2px var(--el-color-primary-light-7);
      
      .agent-name {
        color: var(--el-color-primary);
        font-weight: 700;
      }
      
      .agent-description {
        color: var(--el-text-color-regular);
      }
      
      .agent-info {
        color: var(--el-text-color-secondary);
        border-top-color: var(--el-border-color-lighter);
      }
      
      .agent-tags {
        :deep(.el-tag) {
          background-color: var(--el-color-primary-light-8);
          border-color: var(--el-color-primary-light-6);
        }
      }
      
      .selected-icon {
        color: var(--el-color-primary);
      }
    }

    .agent-card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      .agent-logo {
        flex-shrink: 0;
        border: 2px solid var(--el-border-color-lighter);
        
        .agent-logo-text {
          font-size: 20px;
          font-weight: bold;
          color: white;
        }
      }

      .agent-card-title {
        flex: 1;
        min-width: 0;

        .agent-name {
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 4px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .agent-tags {
          display: flex;
          align-items: center;
          gap: 4px;
        }
      }

      .selected-icon {
        color: var(--el-color-primary);
        font-size: 24px;
        flex-shrink: 0;
      }
    }

    .agent-description {
      font-size: 13px;
      color: var(--el-text-color-regular);
      line-height: 1.5;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .agent-info {
      display: flex;
      flex-direction: column;
      gap: 8px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      padding-top: 8px;
      border-top: 1px solid var(--el-border-color-lighter);

      .info-item {
        display: flex;
        align-items: center;
        gap: 6px;

        .el-icon {
          font-size: 14px;
        }

        span {
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

