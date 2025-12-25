<template>
  <div class="permission-apply">
    <el-card shadow="hover" class="apply-card">
      <template #header>
        <div class="card-header">
          <h2>权限申请</h2>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-else-if="error" class="error-container">
        <el-alert
          :title="error"
          type="error"
          :closable="false"
          show-icon
        />
      </div>

      <div v-else class="apply-content">
        <!-- 权限信息展示 -->
        <el-descriptions :column="1" border class="permission-info">
          <el-descriptions-item label="资源路径">
            <code>{{ permissionInfo.resource_path }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="操作类型">
            <el-tag type="info">{{ permissionInfo.action }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="操作名称">
            <strong>{{ permissionInfo.action_display }}</strong>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 申请说明 -->
        <el-alert
          title="权限申请说明"
          type="info"
          :closable="false"
          class="apply-tip"
        >
          <template #default>
            <p>您正在申请以下权限：</p>
            <ul>
              <li><strong>资源：</strong>{{ permissionInfo.resource_path }}</li>
              <li><strong>操作：</strong>{{ permissionInfo.action_display }}（{{ permissionInfo.action }}）</li>
            </ul>
            <p>提交申请后，系统管理员将审核您的权限申请。</p>
          </template>
        </el-alert>

        <!-- 申请表单 -->
        <el-form
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-width="120px"
          class="apply-form"
        >
          <el-form-item label="申请理由" prop="reason">
            <el-input
              v-model="formData.reason"
              type="textarea"
              :rows="4"
              placeholder="请填写申请权限的理由，以便管理员审核"
              maxlength="500"
              show-word-limit
            />
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              :loading="submitting"
              @click="handleSubmit"
            >
              提交申请
            </el-button>
            <el-button @click="handleCancel">取消</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getPermissionDisplayName } from '@/utils/permission'
import { applyPermission } from '@/api/permission'
import type { FormInstance, FormRules } from 'element-plus'

const route = useRoute()
const router = useRouter()

// 权限信息
const permissionInfo = ref({
  resource_path: '',
  action: '',
  action_display: '',
  error_message: '',
})

// 加载状态
const loading = ref(true)
const error = ref('')
const submitting = ref(false)

// 表单数据
const formRef = ref<FormInstance>()
const formData = ref({
  reason: '',
})

// 表单验证规则
const rules: FormRules = {
  reason: [
    { required: true, message: '请填写申请理由', trigger: 'blur' },
    { min: 10, message: '申请理由至少需要10个字符', trigger: 'blur' },
  ],
}

// 初始化权限信息
onMounted(() => {
  // 从 URL 参数中获取权限信息
  const resource = route.query.resource as string
  const action = route.query.action as string

  if (!resource || !action) {
    error.value = '缺少必要的参数：resource 或 action'
    loading.value = false
    return
  }

  permissionInfo.value = {
    resource_path: decodeURIComponent(resource),
    action: decodeURIComponent(action),
    action_display: getPermissionDisplayName(action),
    error_message: '',
  }

  loading.value = false
})

// 提交申请
const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true

  try {
    const resp = await applyPermission({
      resource_path: permissionInfo.value.resource_path,
      action: permissionInfo.value.action,
      reason: formData.value.reason,
    })

    ElMessage.success(resp.message || '权限申请已提交，等待管理员审核')
    
    // 延迟后返回上一页
    setTimeout(() => {
      router.back()
    }, 1500)
  } catch (err: any) {
    ElMessage.error(err.message || '提交申请失败')
  } finally {
    submitting.value = false
  }
}

// 取消申请
const handleCancel = () => {
  router.back()
}
</script>

<style scoped lang="scss">
.permission-apply {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;

  .apply-card {
    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      h2 {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
      }
    }

    .loading-container {
      padding: 20px;
    }

    .error-container {
      padding: 20px;
    }

    .apply-content {
      .permission-info {
        margin-bottom: 20px;

        code {
          background: var(--el-fill-color-light);
          padding: 2px 6px;
          border-radius: 4px;
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
        }
      }

      .apply-tip {
        margin-bottom: 20px;

        ul {
          margin: 10px 0;
          padding-left: 20px;

          li {
            margin: 8px 0;
          }
        }
      }

      .apply-form {
        margin-top: 20px;
      }
    }
  }
}
</style>

