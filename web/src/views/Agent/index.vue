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
              <h2>LLM 与工作台</h2>
              <p class="header-description">管理 LLM 配置，工作台对话时选择使用的模型</p>
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
  Cpu,
  ArrowRight,
  Grid,
  Star,
  Shop
} from '@element-plus/icons-vue'
import { getLLMList } from '@/api/agent'

const router = useRouter()

const stats = ref({
  llm: {
    total: 0,
    default: 0,
    providers: 0
  },
})

// 加载统计数据
async function loadStats() {
  try {
    const llmRes = await getLLMList({ page: 1, page_size: 100 }) as { configs?: { is_default?: boolean; provider?: string }[]; total?: number }
    stats.value.llm.total = llmRes.total || 0
    if (llmRes.configs) {
      stats.value.llm.default = llmRes.configs.filter(l => l.is_default).length
      const providerSet = new Set(llmRes.configs.map(l => l.provider).filter(Boolean))
      stats.value.llm.providers = providerSet.size
    }
  } catch (error: unknown) {
    console.error('加载统计数据失败:', error)
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
