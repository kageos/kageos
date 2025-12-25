<template>
  <div class="permission-apply-wrapper">
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
        <div class="apply-layout">
          <!-- 左侧：申请说明 -->
          <div class="apply-sidebar">
            <el-card shadow="never" class="info-card">
              <template #header>
                <h3>权限申请说明</h3>
              </template>
              <div class="info-content">
                <p class="info-text">您可以选择申请以下资源的权限：</p>
                <ul class="info-list">
                  <li>
                    <strong>当前资源：</strong>
                    <code class="resource-path">{{ permissionInfo.resource_path }}</code>
                  </li>
                  <li v-if="permissionScopes.length > 1">
                    <strong>父级资源：</strong>您还可以申请父级目录或应用的权限
                  </li>
                </ul>
                <p class="info-text">选择权限后，点击"快捷选择"可以快速选择该资源的全部权限。</p>
              </div>
            </el-card>
          </div>

          <!-- 中间：权限选择区域 -->
          <div class="apply-main">
            <div class="permission-scopes">
              <h3 class="scopes-title">选择权限范围</h3>
              
              <div 
                v-for="(scope, scopeIndex) in permissionScopes" 
                :key="scope.resourcePath"
                class="permission-scope-card"
              >
                <div class="scope-header">
                  <div class="scope-title">
                    <el-icon><Document /></el-icon>
                    <span class="scope-name">{{ scope.displayName }}</span>
                    <el-tag size="small" :type="scope.resourceType === 'function' ? 'primary' : scope.resourceType === 'directory' ? 'success' : 'warning'">
                      {{ scope.resourceType === 'function' ? '函数' : scope.resourceType === 'directory' ? '目录' : '应用' }}
                    </el-tag>
                  </div>
                  <el-button 
                    v-if="scope.quickSelect"
                    link 
                    type="primary" 
                    size="small"
                    @click="handleQuickSelect(scopeIndex)"
                  >
                    {{ scope.quickSelect.label }}
                  </el-button>
                </div>
                
                <div class="scope-path">
                  <code>{{ scope.resourcePath }}</code>
                </div>
                
                <el-checkbox-group 
                  v-model="selectedPermissions[scopeIndex]"
                  class="permission-checkbox-group"
                >
                  <el-checkbox
                    v-for="permission in scope.permissions"
                    :key="permission.action"
                    :label="permission.action"
                    class="permission-checkbox"
                  >
                    <div class="permission-option">
                      <span class="permission-name">{{ permission.displayName }}</span>
                      <el-tag 
                        v-if="permission.isMinimal" 
                        size="small" 
                        type="info" 
                        class="minimal-tag"
                      >
                        最小粒度
                      </el-tag>
                      <code class="permission-code">{{ permission.action }}</code>
                    </div>
                  </el-checkbox>
                </el-checkbox-group>
              </div>
            </div>
          </div>

          <!-- 右侧：申请表单 -->
          <div class="apply-sidebar-right">
            <el-card shadow="never" class="form-card">
              <template #header>
                <h3>提交申请</h3>
              </template>
              <el-form
                ref="formRef"
                :model="formData"
                :rules="rules"
                label-width="80px"
                class="apply-form"
              >
                <el-form-item label="申请理由" prop="reason">
                  <el-input
                    v-model="formData.reason"
                    type="textarea"
                    :rows="6"
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
                    style="width: 100%"
                  >
                    提交申请
                  </el-button>
                  <el-button 
                    @click="handleCancel"
                    style="width: 100%; margin-top: 12px"
                  >
                    取消
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </div>
      </div>
    </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElText, ElIcon } from 'element-plus'
import { Document } from '@element-plus/icons-vue'
import { 
  getPermissionDisplayName, 
  getPermissionScopes,
  type PermissionScope
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

// 权限范围列表（包括当前资源和父级资源）
const permissionScopes = ref<PermissionScope[]>([])

// 每个范围选中的权限点（索引对应 permissionScopes）
const selectedPermissions = ref<string[][]>([])

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

// 检查是否至少选择了一个权限
const hasSelectedPermissions = computed(() => {
  return selectedPermissions.value.some(perms => perms.length > 0)
})

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

  // ⭐ 根据资源路径获取权限范围（包括当前资源和父级资源）
  try {
    await loadPermissionScopes(resourcePath, action)
  } catch (err: any) {
    console.error('加载权限范围失败:', err)
    // 如果加载失败，使用默认逻辑
    permissionScopes.value = getPermissionScopes(resourcePath)
    selectedPermissions.value = permissionScopes.value.map(() => [])
  }

  // ⭐ 设置默认选中的权限点（第一个范围的最小粒度权限）
  if (permissionScopes.value.length > 0 && selectedPermissions.value.length > 0) {
    const firstScope = permissionScopes.value[0]
    const minimalPermissions = firstScope.permissions
      .filter(p => p.isMinimal === true)
      .map(p => p.action)
    
    // 如果 URL 中指定了 action，也加入默认选中
    if (action && !minimalPermissions.includes(action)) {
      minimalPermissions.push(action)
    }
    
    selectedPermissions.value[0] = minimalPermissions
  }

  loading.value = false
})

// ⭐ 加载权限范围（包括当前资源和父级资源）
const loadPermissionScopes = async (resourcePath: string, defaultAction?: string) => {
  // 解析资源路径，判断资源类型
  // 格式：/user/app/dir1/dir2/function
  const pathParts = resourcePath.split('/').filter(Boolean)
  
  if (pathParts.length < 2) {
    // 路径格式错误
    permissionScopes.value = getPermissionScopes(resourcePath)
    selectedPermissions.value = permissionScopes.value.map(() => [])
    return
  }

  const user = pathParts[0]
  const app = pathParts[1]
  
  let resourceType: 'function' | 'directory' | 'app' | undefined
  let templateType: string | undefined
  
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
        // 根据节点类型和模板类型获取权限范围
        if (node.type === 'function') {
          resourceType = 'function'
          templateType = node.template_type
        } else if (node.type === 'package') {
          resourceType = 'directory'
        }
      }
    }
  } catch (err) {
    // 获取服务树失败，使用默认逻辑
    console.warn('获取服务树失败，使用默认逻辑:', err)
  }
  
  // 如果无法从服务树获取，根据路径长度判断
  if (!resourceType) {
    if (pathParts.length === 2) {
      resourceType = 'app'
    } else {
      resourceType = 'function'  // 默认按函数处理
    }
  }
  
  // 获取权限范围
  permissionScopes.value = getPermissionScopes(resourcePath, resourceType, templateType)
  selectedPermissions.value = permissionScopes.value.map(() => [])
}

// ⭐ 快捷选择（选择某个范围的全部权限）
const handleQuickSelect = (scopeIndex: number) => {
  const scope = permissionScopes.value[scopeIndex]
  if (scope.quickSelect) {
    selectedPermissions.value[scopeIndex] = [...scope.quickSelect.actions]
    ElMessage.success(`已选择：${scope.quickSelect.label}`)
  }
}

// 提交申请
const handleSubmit = async () => {
  if (!formRef.value) return

  // 检查是否至少选择了一个权限
  if (!hasSelectedPermissions.value) {
    ElMessage.warning('请至少选择一个权限')
    return
  }

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true

  try {
    // ⭐ 按资源分组提交权限申请（每个资源一个请求）
    const requests: Promise<any>[] = []
    const requestInfos: Array<{ resourcePath: string; actions: string[] }> = []
    
    for (let i = 0; i < permissionScopes.value.length; i++) {
      const scope = permissionScopes.value[i]
      const selected = selectedPermissions.value[i] || []
      
      if (selected.length > 0) {
        requestInfos.push({
          resourcePath: scope.resourcePath,
          actions: selected,
        })
        
        requests.push(
          applyPermission({
            resource_path: scope.resourcePath,
            actions: selected,
            reason: formData.value.reason,
          })
        )
      }
    }
    
    // 等待所有请求完成
    const results = await Promise.allSettled(requests)
    
    // 统计成功和失败的数量
    let successCount = 0
    let failCount = 0
    const failMessages: string[] = []
    
    results.forEach((result, index) => {
      if (result.status === 'fulfilled') {
        successCount++
      } else {
        failCount++
        const info = requestInfos[index]
        failMessages.push(`${info.resourcePath}: ${result.reason?.message || '申请失败'}`)
      }
    })
    
    // 显示结果
    if (failCount === 0) {
      ElMessage.success(`权限申请成功，已提交 ${successCount} 个资源的权限申请`)
    } else {
      ElMessage.warning(`权限申请部分成功：成功 ${successCount} 个，失败 ${failCount} 个`)
      if (failMessages.length > 0) {
        console.error('失败的申请:', failMessages)
      }
    }
    
    // 延迟后返回上一页
    setTimeout(() => {
      router.back()
    }, 1500)
  } catch (err: any) {
    // 显示详细的错误信息
    const errorMessage = err?.response?.data?.msg || err?.message || '提交申请失败'
    ElMessage.error(errorMessage)
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
.permission-apply-wrapper {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--el-bg-color-page);
  padding: 24px;
  box-sizing: border-box;
}

.permission-apply {
  max-width: 1600px;
  margin: 0 auto;
  padding-bottom: 40px;

  .apply-card {
    border-radius: 12px;
    border: none;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    background: var(--el-bg-color);

    :deep(.el-card__header) {
      padding: 20px 24px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      background: var(--el-fill-color-lighter);
      border-radius: 12px 12px 0 0;
    }

    :deep(.el-card__body) {
      padding: 24px;
    }

    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      h2 {
        margin: 0;
        font-size: 22px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .loading-container {
      padding: 20px;
    }

    .error-container {
      padding: 20px;
    }

    .apply-content {
      .apply-layout {
        display: grid;
        grid-template-columns: 280px 1fr 320px;
        gap: 24px;
        align-items: start;
      }

      .apply-sidebar {
        position: sticky;
        top: 24px;

        .info-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }

          .info-content {
            .info-text {
              margin: 0 0 12px 0;
              font-size: 14px;
              color: var(--el-text-color-regular);
              line-height: 1.6;
            }

            .info-list {
              margin: 12px 0;
              padding-left: 20px;
              list-style: disc;

              li {
                margin: 8px 0;
                line-height: 1.6;
                font-size: 14px;
                color: var(--el-text-color-regular);

                .resource-path {
                  display: block;
                  margin-top: 4px;
                  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                  font-size: 12px;
                  color: var(--el-text-color-primary);
                  background: var(--el-fill-color-lighter);
                  padding: 4px 8px;
                  border-radius: 4px;
                  border: 1px solid var(--el-border-color-lighter);
                  word-break: break-all;
                }
              }
            }
          }
        }
      }

      .apply-main {
        min-width: 0; // 防止 grid 溢出

        .permission-scopes {
          .scopes-title {
            margin: 0 0 20px 0;
            font-size: 18px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .permission-scope-card {
            margin-bottom: 24px;
            padding: 20px;
            border: 1px solid var(--el-border-color-light);
            border-radius: 12px;
            background: var(--el-bg-color);
            transition: all 0.3s ease;
            box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);

            &:hover {
              box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
              border-color: var(--el-border-color);
            }

            .scope-header {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 12px;

              .scope-title {
                display: flex;
                align-items: center;
                gap: 8px;

                .scope-name {
                  font-weight: 500;
                  color: var(--el-text-color-primary);
                }
              }
            }

            .scope-path {
              margin-bottom: 16px;
              padding: 10px 14px;
              background: var(--el-fill-color-lighter);
              border-radius: 6px;
              border: 1px solid var(--el-border-color-lighter);

              code {
                font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                font-size: 13px;
                color: var(--el-text-color-primary);
              }
            }

            .permission-checkbox-group {
              display: grid;
              grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
              gap: 12px;
              width: 100%;

              :deep(.el-checkbox) {
                margin: 0;
                height: auto;
                align-items: flex-start;
                
                .el-checkbox__input {
                  margin-top: 2px;
                }
                
                .el-checkbox__label {
                  width: 100%;
                  padding-left: 8px;
                  line-height: 1.5;
                }
              }

              :deep(.el-checkbox.is-checked) {
                .permission-checkbox {
                  border-color: var(--el-color-primary);
                  background-color: var(--el-color-primary-light-9);
                }
              }

              .permission-checkbox {
                width: 100%;
                margin: 0;
                padding: 12px 14px;
                border: 1px solid var(--el-border-color-lighter);
                border-radius: 8px;
                transition: all 0.2s ease;
                background: var(--el-fill-color-lighter);
                min-height: 70px;
                display: flex;
                flex-direction: column;
                justify-content: center;
                box-sizing: border-box;

                &:hover {
                  border-color: var(--el-color-primary-light-7);
                  background-color: var(--el-fill-color);
                  transform: translateY(-1px);
                  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
                }

                .permission-option {
                  display: flex;
                  flex-direction: column;
                  align-items: flex-start;
                  gap: 6px;
                  width: 100%;

                  .permission-name {
                    font-weight: 500;
                    color: var(--el-text-color-primary);
                    font-size: 14px;
                    line-height: 1.4;
                    word-break: break-word;
                  }

                  .permission-code {
                    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                    font-size: 11px;
                    color: var(--el-text-color-secondary);
                    background: var(--el-fill-color);
                    padding: 2px 6px;
                    border-radius: 4px;
                    border: 1px solid var(--el-border-color-lighter);
                    align-self: flex-start;
                    word-break: break-all;
                  }

                  .minimal-tag {
                    align-self: flex-start;
                    margin-top: 2px;
                  }
                }
              }
            }
          }
        }
      }

      .apply-sidebar-right {
        position: sticky;
        top: 24px;

        .form-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }
        }

        .apply-form {
          .form-item-tip {
            margin-top: 8px;
          }

          :deep(.el-form-item__label) {
            font-weight: 500;
            color: var(--el-text-color-primary);
          }

          :deep(.el-textarea__inner) {
            border-radius: 8px;
            border-color: var(--el-border-color);
            background: var(--el-fill-color-lighter);
            transition: all 0.2s ease;

            &:focus {
              border-color: var(--el-color-primary);
              background: var(--el-bg-color);
            }
          }

          :deep(.el-button) {
            border-radius: 8px;
            padding: 10px 20px;
          }
        }
      }
    }
  }
}
</style>

