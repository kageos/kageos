<template>
  <div class="agent-index">
    <el-card shadow="hover" class="index-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <div class="header-icon">
              <el-icon :size="28"><Operation /></el-icon>
            </div>
            <div>
              <h2>Agent-Server 管理</h2>
              <p class="header-description">统一管理智能体、知识库、LLM配置和插件，构建强大的AI应用生态</p>
            </div>
          </div>
        </div>
      </template>

      <!-- 功能模块卡片区 -->
      <div class="modules-section">
        <div class="section-header">
          <h3 class="section-title">
            <el-icon class="section-icon"><Grid /></el-icon>
            功能模块
          </h3>
          <p class="section-description">选择下方模块进入对应的管理页面</p>
        </div>
        <div class="modules-grid">
          <!-- 智能体管理 -->
          <el-card
            shadow="hover"
            class="module-card module-card--agents"
            @click="navigateTo('/agent/agents')"
          >
            <div class="module-card__header">
              <div class="module-card__icon-wrapper">
                <div class="module-card__icon">
                  <el-icon :size="32">
                    <Operation />
                  </el-icon>
                </div>
                <div class="module-card__badge">
                  <el-badge :value="stats.agents.total" :max="99" />
                </div>
              </div>
            </div>
            <div class="module-card__body">
              <h4 class="module-card__title">智能体管理</h4>
              <p class="module-card__description">
                管理智能体配置，包括纯知识库类型和插件调用类型，支持智能体的创建、编辑、启用和禁用
              </p>
              <div class="module-card__stats">
                <div class="module-card__stat-item">
                  <el-icon><CircleCheck /></el-icon>
                  <span>已启用: {{ stats.agents.enabled }}</span>
                </div>
                <div class="module-card__stat-item">
                  <el-icon><Document /></el-icon>
                  <span>知识库类型: {{ stats.agents.knowledgeOnly }}</span>
                </div>
                <div class="module-card__stat-item">
                  <el-icon><Connection /></el-icon>
                  <span>插件类型: {{ stats.agents.plugin }}</span>
                </div>
              </div>
            </div>
            <div class="module-card__footer">
              <el-button type="primary" :icon="ArrowRight" @click.stop="navigateTo('/agent/agents')">
                进入管理
              </el-button>
            </div>
          </el-card>

          <!-- 知识库管理 -->
          <el-card
            shadow="hover"
            class="module-card module-card--knowledge"
            @click="navigateTo('/agent/knowledge')"
          >
            <div class="module-card__header">
              <div class="module-card__icon-wrapper">
                <div class="module-card__icon">
                  <el-icon :size="32">
                    <Document />
                  </el-icon>
                </div>
                <div class="module-card__badge">
                  <el-badge :value="stats.knowledge.total" :max="99" />
                </div>
              </div>
            </div>
            <div class="module-card__body">
              <h4 class="module-card__title">知识库管理</h4>
              <p class="module-card__description">
                管理知识库配置和文档，支持文档的添加、编辑、删除和树形结构展示
              </p>
              <div class="module-card__stats">
                <div class="module-card__stat-item">
                  <el-icon><CircleCheck /></el-icon>
                  <span>激活: {{ stats.knowledge.active }}</span>
                </div>
                <div class="module-card__stat-item">
                  <el-icon><Document /></el-icon>
                  <span>文档总数: {{ stats.knowledge.documents }}</span>
                </div>
              </div>
            </div>
            <div class="module-card__footer">
              <el-button type="primary" :icon="ArrowRight" @click.stop="navigateTo('/agent/knowledge')">
                进入管理
              </el-button>
            </div>
          </el-card>

          <!-- LLM 管理 -->
          <el-card
            shadow="hover"
            class="module-card module-card--llm"
            @click="navigateTo('/agent/llm')"
          >
            <div class="module-card__header">
              <div class="module-card__icon-wrapper">
                <div class="module-card__icon">
                  <el-icon :size="32">
                    <Cpu />
                  </el-icon>
                </div>
                <div class="module-card__badge">
                  <el-badge :value="stats.llm.total" :max="99" />
                </div>
              </div>
            </div>
            <div class="module-card__body">
              <h4 class="module-card__title">LLM 管理</h4>
              <p class="module-card__description">
                管理大模型配置，支持多种 LLM 提供商，可设置默认配置
              </p>
              <div class="module-card__stats">
                <div class="module-card__stat-item">
                  <el-icon><Star /></el-icon>
                  <span>默认配置: {{ stats.llm.default }}</span>
                </div>
                <div class="module-card__stat-item">
                  <el-icon><Shop /></el-icon>
                  <span>提供商数: {{ stats.llm.providers }}</span>
                </div>
              </div>
            </div>
            <div class="module-card__footer">
              <el-button type="primary" :icon="ArrowRight" @click.stop="navigateTo('/agent/llm')">
                进入管理
              </el-button>
            </div>
          </el-card>

        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Operation,
  Document,
  Cpu,
  Connection,
  ArrowRight,
  Grid,
  CircleCheck,
  Star,
  Shop
} from '@element-plus/icons-vue'
import {
  getAgentList,
  getKnowledgeList,
  getLLMList
} from '@/api/agent'

const router = useRouter()

const stats = ref({
  agents: {
    total: 0,
    enabled: 0,
    knowledgeOnly: 0,
    plugin: 0
  },
  knowledge: {
    total: 0,
    active: 0,
    documents: 0
  },
  llm: {
    total: 0,
    default: 0,
    providers: 0
  },
})

// 加载统计数据
async function loadStats() {
  try {
    // 并行加载所有统计数据
    const [agentsRes, knowledgeRes, llmRes] = await Promise.all([
      getAgentList({ page: 1, page_size: 1 }),
      getKnowledgeList({ page: 1, page_size: 1 }),
      getLLMList({ page: 1, page_size: 1 })
    ])

    // 更新智能体统计（响应拦截器已解包，直接使用 data）
    stats.value.agents.total = agentsRes.total || 0
    // 获取详细统计需要加载更多数据
    const agentsDetailRes = await getAgentList({ page: 1, page_size: 1000 })
    if (agentsDetailRes.agents) {
      stats.value.agents.enabled = agentsDetailRes.agents.filter(a => a.enabled).length
      stats.value.agents.knowledgeOnly = agentsDetailRes.agents.filter(a => a.agent_type === 'knowledge_only').length
      stats.value.agents.plugin = agentsDetailRes.agents.filter(a => a.agent_type === 'plugin').length
    }

    // 更新知识库统计（响应拦截器已解包）
    stats.value.knowledge.total = knowledgeRes.total || 0
    const knowledgeDetailRes = await getKnowledgeList({ page: 1, page_size: 1000 })
    if (knowledgeDetailRes.knowledge_bases) {
      stats.value.knowledge.active = knowledgeDetailRes.knowledge_bases.filter(k => k.status === 'active').length
      stats.value.knowledge.documents = knowledgeDetailRes.knowledge_bases.reduce(
        (sum, k) => sum + (k.document_count || 0),
        0
      )
    }

    // 更新LLM统计（响应拦截器已解包）
    stats.value.llm.total = llmRes.total || 0
    if (llmRes.configs) {
      stats.value.llm.default = llmRes.configs.filter(l => l.is_default).length
      const providerSet = new Set(llmRes.configs.map(l => l.provider))
      stats.value.llm.providers = providerSet.size
    }

  } catch (error: any) {
    console.error('加载统计数据失败:', error)
    // 静默失败，不影响页面展示
  }
}

function navigateTo(path: string) {
  router.push(path)
}

onMounted(() => {
  loadStats()
})
</script>

<style scoped lang="scss">
.agent-index {
  padding: 20px;
}

.index-card {
  min-height: calc(100vh - 100px);
}

.card-header {
  .header-left {
    display: flex;
    align-items: flex-start;
    gap: 16px;
  }

  .header-icon {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  h2 {
    margin: 0 0 8px 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .header-description {
    margin: 0;
    font-size: 14px;
    color: var(--el-text-color-regular);
    line-height: 1.6;
  }
}

// 功能模块卡片区
.modules-section {
  .section-header {
    margin-bottom: 24px;
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin: 0 0 8px 0;

    .section-icon {
      color: var(--el-color-primary);
    }
  }

  .section-description {
    margin: 0;
    font-size: 14px;
    color: var(--el-text-color-regular);
  }

  .modules-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 24px;
  }

  .module-card {
    cursor: pointer;
    transition: all 0.3s;
    border: 1px solid var(--el-border-color-lighter);
    overflow: hidden;
    position: relative;

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: var(--el-color-primary);
      transform: scaleX(0);
      transform-origin: left;
      transition: transform 0.3s;
    }

    &:hover {
      transform: translateY(-4px);
      box-shadow: var(--el-box-shadow);
      border-color: var(--el-color-primary-light-8);

      &::before {
        transform: scaleX(1);
      }

      .module-card__icon {
        transform: scale(1.1);
      }

      .module-card__arrow {
        transform: translateX(4px);
      }
    }

    &__header {
      padding: 24px 24px 0 24px;
      position: relative;
    }

    &__icon-wrapper {
      position: relative;
      display: inline-block;
    }

    &__icon {
      width: 64px;
      height: 64px;
      border-radius: 12px;
      background: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
      display: flex;
      align-items: center;
      justify-content: center;
      transition: transform 0.3s;
    }

    &__badge {
      position: absolute;
      top: -8px;
      right: -8px;
    }

    &__body {
      padding: 20px 24px;
    }

    &__title {
      font-size: 18px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin: 0 0 12px 0;
    }

    &__description {
      font-size: 14px;
      color: var(--el-text-color-regular);
      line-height: 1.6;
      margin: 0 0 20px 0;
    }

    &__stats {
      display: flex;
      flex-direction: column;
      gap: 10px;
      padding: 16px;
      background: var(--el-fill-color-lighter);
      border-radius: 8px;
      margin-bottom: 16px;
    }

    &__stat-item {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      color: var(--el-text-color-secondary);

      .el-icon {
        color: var(--el-color-primary);
        font-size: 16px;
      }
    }

    &__footer {
      padding: 0 24px 24px 24px;
    }

    // 不同模块的图标颜色
    &--agents .module-card__icon {
      background: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
    }

    &--knowledge .module-card__icon {
      background: var(--el-color-success-light-9);
      color: var(--el-color-success);
    }

    &--llm .module-card__icon {
      background: var(--el-color-info-light-9);
      color: var(--el-color-info);
    }

    &--plugins .module-card__icon {
      background: var(--el-color-warning-light-9);
      color: var(--el-color-warning);
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .agent-index {
    padding: 16px;
  }

  .modules-grid {
    grid-template-columns: 1fr;
  }

  .card-header {
    .header-left {
      flex-direction: column;
      gap: 12px;
    }
  }
}
</style>
