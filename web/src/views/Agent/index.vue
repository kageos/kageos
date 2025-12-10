<template>
  <div class="agent-index">
    <el-card shadow="hover" class="index-card">
      <template #header>
        <div class="card-header">
          <el-button
            link
            :icon="ArrowLeft"
            @click="handleBack"
            class="back-button"
          >
            返回
          </el-button>
          <h2>Agent-Server 管理</h2>
        </div>
      </template>

      <div class="menu-grid">
        <el-card
          shadow="hover"
          class="menu-card"
          @click="navigateTo('/agent/agents')"
        >
          <div class="menu-content">
            <el-icon :size="48" class="menu-icon">
              <Operation />
            </el-icon>
            <h3>智能体管理</h3>
            <p>管理智能体配置，包括纯知识库类型和接口调用类型</p>
          </div>
        </el-card>

        <el-card
          shadow="hover"
          class="menu-card"
          @click="navigateTo('/agent/knowledge')"
        >
          <div class="menu-content">
            <el-icon :size="48" class="menu-icon">
              <Document />
            </el-icon>
            <h3>知识库管理</h3>
            <p>管理知识库配置和文档</p>
          </div>
        </el-card>

        <el-card
          shadow="hover"
          class="menu-card"
          @click="navigateTo('/agent/llm')"
        >
          <div class="menu-content">
            <el-icon :size="48" class="menu-icon">
              <Cpu />
            </el-icon>
            <h3>LLM 管理</h3>
            <p>管理大模型配置，支持多种 LLM 提供商</p>
          </div>
        </el-card>

        <el-card
          shadow="hover"
          class="menu-card"
          @click="navigateTo('/agent/plugins')"
        >
          <div class="menu-content">
            <el-icon :size="48" class="menu-icon">
              <Connection />
            </el-icon>
            <h3>插件管理</h3>
            <p>管理插件配置，支持插件复用和独立管理</p>
          </div>
        </el-card>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, Operation, Document, Cpu, Connection } from '@element-plus/icons-vue'

const router = useRouter()

function navigateTo(path: string) {
  router.push(path)
}

function handleBack() {
  router.back()
}
</script>

<style scoped>
.agent-index {
  padding: 20px;
}

.index-card {
  min-height: calc(100vh - 100px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-header h2 {
  flex: 1;
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.back-button {
  padding: 0;
}

.menu-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  margin-top: 20px;
}

.menu-card {
  cursor: pointer;
  transition: all 0.3s;
}

.menu-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.menu-content {
  text-align: center;
  padding: 20px;
}

.menu-icon {
  color: var(--el-color-primary);
  margin-bottom: 16px;
}

.menu-content h3 {
  margin: 0 0 12px 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.menu-content p {
  margin: 0;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.6;
}
</style>

