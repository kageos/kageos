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
    class="user-edit-dialog"
    width="600px"
    :close-on-click-modal="false"
  >
    <template #header>
      <div class="dialog-header">
        <span class="dialog-kicker">Member Editor</span>
        <h3>编辑用户组织架构</h3>
        <p>调整部门归属和直属上级，保存后会立即反映到当前组织成员列表。</p>
      </div>
    </template>

    <div v-if="userInfo" class="user-edit-content">
      <section class="user-hero-card">
        <div class="user-basic">
          <UserDisplay :user-info="userInfo" mode="simple" size="large" />
          <div class="user-details">
            <div class="detail-primary">
              <strong>{{ userInfo.nickname || userInfo.username }}</strong>
              <span>@{{ userInfo.username }}</span>
            </div>
            <div v-if="userInfo.email" class="detail-item">
              <span class="value">{{ userInfo.email }}</span>
            </div>
          </div>
        </div>

        <div class="user-current-grid">
          <article class="current-card">
            <span class="current-label">当前部门</span>
            <strong>{{ userInfo.department_full_name_path || userInfo.department_name || '未分配部门' }}</strong>
          </article>
          <article class="current-card">
            <span class="current-label">直属上级</span>
            <strong>{{ userInfo.leader_display_name || userInfo.leader_username || '未分配 Leader' }}</strong>
          </article>
        </div>
      </section>

      <section class="form-card">
        <div class="section-title">
          <span>组织调整</span>
          <p>这里修改的是用户的组织归属，不会改动用户基础账号信息。</p>
        </div>

        <el-form :model="formData" label-width="120px">
          <el-form-item label="所属部门">
            <DepartmentSelector
              v-model="formData.department_full_path"
              :department-tree="departmentTreeData"
            />
          </el-form-item>

          <el-form-item label="直接上级">
            <UserWidget
              :value="leaderFieldValue"
              :field="leaderField"
              mode="edit"
              field-path="leader_username"
              @update:modelValue="handleLeaderChange"
            />
          </el-form-item>
        </el-form>
      </section>
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
import { assignUserOrganization } from '@/api/user'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import DepartmentSelector from '@/shared/components/DepartmentSelector.vue'
import UserWidget from '@/shared/components/UserWidget.vue'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import type { FieldValue } from '@/architecture/runtime/types/field'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'

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

const leaderField = createWidgetFieldConfig({
  code: 'leader_username',
  name: '直接上级',
  widgetType: WidgetType.USER
})

const leaderFieldValue = computed(() =>
  createStringFieldValue(leaderField, formData.value.leader_username, { emptyRaw: '' })
)

const handleLeaderChange = (value: FieldValue) => {
  formData.value.leader_username = extractStringFieldRaw(value) || null
}

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

// 注意：搜索和选择逻辑已由 UserWidget 组件内部处理，不再需要这些函数

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
    dialogVisible.value = false
    emit('success')
  } catch (error: any) {
    ElMessage.error(error.message || '更新失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.user-edit-dialog {
  --user-edit-ink: var(--text-primary);
  --user-edit-muted: var(--text-secondary);
  --user-edit-line: color-mix(in srgb, var(--border-base) 82%, var(--color-primary) 18%);
  --user-edit-surface: linear-gradient(
    180deg,
    color-mix(in srgb, var(--bg-primary) 90%, var(--color-primary) 10%),
    color-mix(in srgb, var(--bg-secondary) 92%, var(--color-primary) 8%)
  );
  --user-edit-card: color-mix(in srgb, var(--bg-primary) 84%, var(--bg-secondary) 16%);
  --user-edit-accent-soft: color-mix(in srgb, var(--color-primary) 10%, transparent);
  --user-edit-kicker: color-mix(in srgb, var(--color-primary) 68%, var(--text-secondary) 32%);
}

.dialog-header {
  display: flex;
  flex-direction: column;
  gap: 8px;

  h3 {
    margin: 0;
    font-size: 24px;
    color: var(--user-edit-ink);
  }

  p {
    margin: 0;
    line-height: 1.7;
    color: var(--user-edit-muted);
    font-size: 13px;
  }
}

.dialog-kicker {
  display: inline-flex;
  align-items: center;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-size: 11px;
  font-weight: 700;
  color: var(--user-edit-kicker);
}

.user-edit-content {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.user-hero-card,
.form-card {
  padding: 18px;
  border-radius: 20px;
  background: var(--user-edit-surface);
  border: 1px solid var(--user-edit-line);
}

.section-title {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 16px;

  span {
    font-size: 14px;
    font-weight: 700;
    color: var(--user-edit-ink);
  }

  p {
    margin: 0;
    font-size: 12px;
    line-height: 1.6;
    color: var(--user-edit-muted);
  }
}

.user-basic {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.user-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-primary {
  display: flex;
  flex-direction: column;
  gap: 4px;

  strong {
    font-size: 20px;
    color: var(--user-edit-ink);
  }

  span {
    font-size: 13px;
    color: var(--user-edit-muted);
  }
}

.detail-item {
  font-size: 14px;

  .value {
    color: var(--text-regular);
  }
}

.user-current-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.current-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border-radius: 16px;
  background: var(--user-edit-card);
  border: 1px solid var(--user-edit-line);

  strong {
    word-break: break-word;
    color: var(--user-edit-ink);
  }
}

.current-label {
  font-size: 12px;
  color: var(--user-edit-muted);
}

.form-card {
  :deep(.el-form-item) {
    margin-bottom: 20px;
  }

  :deep(.el-form-item__label) {
    font-weight: 600;
    color: var(--text-regular);
  }

  :deep(.el-input__wrapper),
  :deep(.el-select__wrapper),
  :deep(.el-textarea__inner) {
    border-radius: 14px;
    box-shadow: none;
    background: var(--bg-primary);
    border: 1px solid var(--border-base);
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.user-edit-dialog) {
  border-radius: 28px;
  overflow: hidden;
  background: var(--bg-primary);
}

:deep(.user-edit-dialog .el-dialog__header) {
  padding: 24px 24px 12px;
}

:deep(.user-edit-dialog .el-dialog__body) {
  padding: 0 24px 8px;
}

:deep(.user-edit-dialog .el-dialog__footer) {
  padding: 12px 24px 24px;
}

@media (max-width: 720px) {
  .user-current-grid {
    grid-template-columns: 1fr;
  }

  .user-basic {
    flex-direction: column;
  }
}
</style>
