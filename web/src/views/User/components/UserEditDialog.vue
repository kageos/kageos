<!--
  用户编辑对话框 - 编辑用户组织架构和 Leader
  
  需求：
  - 可以修改用户的部门（通过组织架构树选择）
  - 可以修改用户的 Leader（通过用户选择器）
  - 可以清空部门和 Leader
-->
<template>
  <el-dialog
    v-model="dialogVisible"
    title="编辑用户组织架构"
    width="600px"
    :close-on-click-modal="false"
  >
    <div v-if="userInfo" class="user-edit-content">
      <!-- 用户基本信息 -->
      <div class="user-info-section">
        <div class="section-title">用户信息</div>
        <div class="user-basic">
          <UserDisplay :user-info="userInfo" mode="simple" size="large" />
          <div class="user-details">
            <div class="detail-item">
              <span class="label">用户名：</span>
              <span class="value">{{ userInfo.username }}</span>
            </div>
            <div v-if="userInfo.email" class="detail-item">
              <span class="label">邮箱：</span>
              <span class="value">{{ userInfo.email }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 组织架构选择 -->
      <div class="form-section">
        <div class="section-title">组织架构</div>
        <el-form :model="formData" label-width="120px">
          <el-form-item label="所属部门">
            <DepartmentSelector
              v-model="formData.department_full_path"
              :department-tree="departmentTreeData"
            />
          </el-form-item>

          <el-form-item label="直接上级">
            <el-autocomplete
              v-model="formData.leader_username"
              :fetch-suggestions="searchLeaderUsers"
              placeholder="输入用户名搜索 Leader（可选）"
              clearable
              style="width: 100%"
              :trigger-on-focus="false"
              value-key="username"
              @select="handleLeaderSelect"
            >
              <template #default="{ item }">
                <div class="user-option">
                  <UserDisplay :user-info="item" mode="simple" size="small" />
                </div>
              </template>
            </el-autocomplete>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          保存
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { UserInfo } from '@/types'
import type { Department } from '@/api/department'
import { assignUserOrganization, searchUsersFuzzy } from '@/api/user'
import UserDisplay from '@/architecture/presentation/widgets/UserDisplay.vue'
import DepartmentSelector from '@/components/DepartmentSelector.vue'

interface Props {
  modelValue: boolean
  userInfo: UserInfo | null
  departmentTree: Department[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'success': []
}>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formData = ref({
  department_full_path: '' as string | null,
  leader_username: '' as string | null
})

const submitting = ref(false)

// 部门树数据
const departmentTreeData = computed(() => {
  return props.departmentTree || []
})

// 监听 userInfo 变化，初始化表单数据
watch(() => props.userInfo, (newUserInfo) => {
  if (newUserInfo) {
    formData.value = {
      department_full_path: newUserInfo.department_full_path || null,
      leader_username: newUserInfo.leader_username || null
    }
  }
}, { immediate: true })

// 搜索 Leader 用户
async function searchLeaderUsers(queryString: string, cb: (results: UserInfo[]) => void) {
  if (!queryString || queryString.trim().length < 1) {
    cb([])
    return
  }
  
  try {
    const res = await searchUsersFuzzy(queryString.trim(), 10)
    // 排除当前用户
    const filtered = res.users.filter(u => u.username !== props.userInfo?.username)
    cb(filtered)
  } catch (error) {
    console.error('搜索用户失败:', error)
    cb([])
  }
}

// 处理 Leader 选择
function handleLeaderSelect(item: UserInfo) {
  formData.value.leader_username = item.username
}

// 提交
async function handleSubmit() {
  if (!props.userInfo) return
  
  submitting.value = true
  try {
    await assignUserOrganization({
      username: props.userInfo.username,
      department_full_path: formData.value.department_full_path || null,
      leader_username: formData.value.leader_username || null
    })
    
    ElMessage.success('更新成功')
    emit('success')
  } catch (error: any) {
    ElMessage.error(error.message || '更新失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.user-edit-content {
  padding: 10px 0;
}

.user-info-section,
.form-section {
  margin-bottom: 24px;
  
  &:last-child {
    margin-bottom: 0;
  }
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.user-basic {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-item {
  font-size: 14px;
  
  .label {
    color: var(--el-text-color-secondary);
    margin-right: 8px;
  }
  
  .value {
    color: var(--el-text-color-primary);
  }
}


.user-option {
  display: flex;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

