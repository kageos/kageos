<script setup lang="ts">
import { useRouter } from 'vue-router'
import { House } from '@element-plus/icons-vue'

const router = useRouter()

const goHome = () => {
  router.push('/')
}
</script>

<template>
  <div class="not-found-container">
    <div class="not-found-content">
      <div class="error-image">
        <el-icon size="120" color="#909399">
          <WarningFilled />
        </el-icon>
      </div>

      <h1>404</h1>
      <h2>页面不存在</h2>
      <p>抱歉，您访问的页面不存在或已被删除</p>

      <div class="actions">
        <el-button type="primary" :icon="House" @click="goHome">
          返回首页
        </el-button>
        <el-button @click="$router.go(-1)">
          返回上页
        </el-button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.not-found-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
}

.not-found-content {
  text-align: center;
  background: white;
  padding: 60px 40px;
  border-radius: 16px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  max-width: 500px;
  width: 100%;
}

.error-image {
  margin-bottom: 30px;
}

h1 {
  font-size: 72px;
  font-weight: bold;
  color: #303133;
  margin: 0 0 16px 0;
}

h2 {
  font-size: 24px;
  color: #606266;
  margin: 0 0 16px 0;
}

p {
  font-size: 16px;
  color: #909399;
  margin: 0 0 40px 0;
  line-height: 1.6;
}

.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

@media (max-width: 768px) {
  .not-found-content {
    padding: 40px 20px;
  }

  h1 {
    font-size: 56px;
  }

  h2 {
    font-size: 20px;
  }

  .actions {
    flex-direction: column;
  }
}
</style>