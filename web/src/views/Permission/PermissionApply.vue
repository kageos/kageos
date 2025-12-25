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
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElText } from 'element-plus'
import { 
  getPermissionDisplayName, 
  getAvailablePermissions, 
  getDefaultSelectedPermissions 
} from '@/utils/permission'
import { applyPermission } from '@/api/permission'
import type { FormInstance, FormRules } from 'element-plus'
import { getAppWithServiceTree } from '@/api/app'
import { useAuthStore } from '@/stores/auth'

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

// 可用的权限点列表
const availablePermissions = ref<Array<{ action: string; displayName: string; isMinimal?: boolean }>>([])

// 表单数据
const formRef = ref<FormInstance>()
const formData = ref({
  actions: [] as string[],  // ⭐ 改为数组，支持多选
  reason: '',
})

// 表单验证规则
const rules: FormRules = {
  actions: [
    { 
      type: 'array', 
      required: true, 
      message: '请至少选择一个操作类型', 
      trigger: 'change',
      min: 1
    },
  ],
  reason: [
    { required: true, message: '请填写申请理由', trigger: 'blur' },
    { min: 10, message: '申请理由至少需要10个字符', trigger: 'blur' },
  ],
}

// 初始化权限信息
onMounted(async () => {
  // 从 URL 参数中获取权限信息
  const resource = route.query.resource as string
  const action = route.query.action as string  // 可选，用于默认选中

  if (!resource) {
    error.value = '缺少必要的参数：resource'
    loading.value = false
    return
  }

  const resourcePath = decodeURIComponent(resource)
  permissionInfo.value = {
    resource_path: resourcePath,
    action: action || '',
    action_display: action ? getPermissionDisplayName(action) : '',
    error_message: '',
  }

  // ⭐ 根据资源路径获取可用的权限点
  try {
    await loadAvailablePermissions(resourcePath, action)
  } catch (err: any) {
    console.error('加载可用权限失败:', err)
    // 如果加载失败，使用默认权限点
    availablePermissions.value = getAvailablePermissions(resourcePath)
  }

  // ⭐ 设置默认选中的权限点（最小粒度）
  const defaultSelected = getDefaultSelectedPermissions(availablePermissions.value)
  // 如果 URL 中指定了 action，也加入默认选中
  if (action && !defaultSelected.includes(action)) {
    defaultSelected.push(action)
  }
  formData.value.actions = defaultSelected

  loading.value = false
})

// ⭐ 加载可用的权限点
const loadAvailablePermissions = async (resourcePath: string, defaultAction?: string) => {
  // 解析资源路径，判断资源类型
  // 格式：/user/app/dir1/dir2/function
  const pathParts = resourcePath.split('/').filter(Boolean)
  
  if (pathParts.length < 2) {
    // 路径格式错误
    availablePermissions.value = getAvailablePermissions(resourcePath)
    return
  }

  const user = pathParts[0]
  const app = pathParts[1]
  
  // 尝试从服务树获取资源信息
  try {
    const response = await getAppWithServiceTree(user, app)
    
    if (response && response.tree) {
      // 在服务树中查找匹配的节点
      const findNode = (nodes: any[], path: string): any => {
        for (const node of nodes) {
          if (node.full_code_path === path) {
            return node
          }
          if (node.children && node.children.length > 0) {
            const found = findNode(node.children, path)
            if (found) return found
          }
        }
        return null
      }

      const node = findNode(response.tree, resourcePath)
      
      if (node) {
        // 根据节点类型和模板类型获取可用权限点
        if (node.type === 'function') {
          availablePermissions.value = getAvailablePermissions(
            resourcePath,
            'function',
            node.template_type
          )
        } else if (node.type === 'package') {
          availablePermissions.value = getAvailablePermissions(
            resourcePath,
            'directory'
          )
        } else {
          availablePermissions.value = getAvailablePermissions(resourcePath)
        }
      } else {
        // 节点未找到，根据路径长度判断
        if (pathParts.length === 2) {
          // /user/app - 应用
          availablePermissions.value = getAvailablePermissions(resourcePath, 'app')
        } else if (pathParts.length >= 3) {
          // /user/app/dir... - 可能是目录或函数，默认按函数处理
          availablePermissions.value = getAvailablePermissions(resourcePath, 'function')
        } else {
          availablePermissions.value = getAvailablePermissions(resourcePath)
        }
      }
    } else {
      // 无法获取服务树，使用默认逻辑
      if (pathParts.length === 2) {
        availablePermissions.value = getAvailablePermissions(resourcePath, 'app')
      } else {
        availablePermissions.value = getAvailablePermissions(resourcePath, 'function')
      }
    }
  } catch (err) {
    // 获取服务树失败，使用默认逻辑
    console.warn('获取服务树失败，使用默认权限点:', err)
    if (pathParts.length === 2) {
      availablePermissions.value = getAvailablePermissions(resourcePath, 'app')
    } else {
      availablePermissions.value = getAvailablePermissions(resourcePath, 'function')
    }
  }
}

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
    // ⭐ 批量申请多个权限点
    const promises = formData.value.actions.map(action =>
      applyPermission({
        resource_path: permissionInfo.value.resource_path,
        action: action,
        reason: formData.value.reason,
      })
    )

    await Promise.all(promises)

    const selectedNames = formData.value.actions
      .map(action => getPermissionDisplayName(action))
      .join('、')
    
    ElMessage.success(`权限申请成功：${selectedNames}`)
    
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

        .permission-checkbox-group {
          display: flex;
          flex-direction: column;
          gap: 12px;
          width: 100%;
        }

        .permission-checkbox {
          width: 100%;
          margin: 0;
          padding: 12px;
          border: 1px solid var(--el-border-color);
          border-radius: 8px;
          transition: all 0.2s ease;

          &:hover {
            border-color: var(--el-color-primary);
            background-color: var(--el-fill-color-light);
          }

          .permission-option {
            display: flex;
            align-items: center;
            gap: 12px;
            width: 100%;

            .permission-name {
              font-weight: 500;
              color: var(--el-text-color-primary);
              flex: 1;
            }

            .minimal-tag {
              margin-left: auto;
            }

            .permission-code {
              font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
              font-size: 12px;
              color: var(--el-text-color-secondary);
              background: var(--el-fill-color-light);
              padding: 2px 6px;
              border-radius: 4px;
            }
          }
        }

        .form-item-tip {
          margin-top: 8px;
        }
      }
    }
  }
}
</style>

