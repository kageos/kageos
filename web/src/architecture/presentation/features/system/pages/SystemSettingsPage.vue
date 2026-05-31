<template>
  <div class="system-settings-page">
    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('route.systemSettings') }}</h2>
            <p>Email delivery and self-service registration are managed by the system owner.</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Connection" @click="router.push('/connectors/providers')">
              {{ t('workspace.connectorConfig') }}
            </el-button>
            <el-button :icon="Refresh" @click="loadSettings">{{ t('common.refresh') }}</el-button>
            <el-button type="primary" :icon="Check" :loading="saving" @click="saveSettings">
              {{ t('connectorProvider.save') }}
            </el-button>
          </div>
        </div>
      </template>

      <div v-loading="loading" class="settings-body">
        <el-alert
          v-if="form.registration_mode === 'admin_only'"
          title="Self-service registration is disabled. Users must be created by system."
          type="info"
          show-icon
          :closable="false"
        />

        <el-form ref="formRef" :model="form" label-width="180px" class="settings-form">
          <el-divider content-position="left">Registration</el-divider>
          <el-form-item label="Registration mode">
            <el-radio-group v-model="form.registration_mode">
              <el-radio-button value="admin_only">Admin only</el-radio-button>
              <el-radio-button value="email_code">Email verification</el-radio-button>
              <el-radio-button value="debug_code">Debug code</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-divider content-position="left">Email service</el-divider>
          <el-form-item label="Email mode">
            <el-radio-group v-model="form.email.mode">
              <el-radio-button value="smtp">SMTP</el-radio-button>
              <el-radio-button value="log">Log</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="SMTP host">
            <el-input v-model="form.email.host" placeholder="smtp.example.com" />
          </el-form-item>
          <el-form-item label="SMTP port">
            <el-input-number v-model="form.email.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="Username">
            <el-input v-model="form.email.username" placeholder="SMTP account username" />
          </el-form-item>
          <el-form-item label="Password">
            <el-input
              v-model="form.email.password"
              type="password"
              show-password
              :placeholder="form.email.password_set ? 'Already configured; leave blank to keep current password' : 'SMTP password'"
            />
          </el-form-item>
          <el-form-item label="From">
            <el-input v-model="form.email.from" placeholder="noreply@example.com" />
          </el-form-item>
          <el-form-item label="From name">
            <el-input v-model="form.email.from_name" placeholder="Kageos" />
          </el-form-item>

          <el-divider content-position="left">Test email</el-divider>
          <el-form-item label="Recipient">
            <div class="test-row">
              <el-input v-model="testEmail" placeholder="admin@example.com" />
              <el-button :icon="Message" :loading="testing" @click="sendTestEmail">Send test</el-button>
            </div>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Check, Connection, Message, Refresh } from '@element-plus/icons-vue'
import {
  getSystemSettings,
  updateSystemSettings,
  testSystemEmail,
  type SystemSettings
} from '@/architecture/presentation/context/api/system-settings'

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const router = useRouter()
const { t } = useI18n()

const form = reactive<SystemSettings>({
  registration_mode: 'admin_only',
  email: {
    mode: 'smtp',
    host: '',
    port: 587,
    username: '',
    password: '',
    password_set: false,
    from: '',
    from_name: 'Kageos',
  },
})

function applySettings(settings: SystemSettings) {
  form.registration_mode = settings.registration_mode
  form.email = {
    ...settings.email,
    password: '',
  }
}

async function loadSettings() {
  loading.value = true
  try {
    applySettings(await getSystemSettings())
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to load settings')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  try {
    applySettings(await updateSystemSettings(JSON.parse(JSON.stringify(form))))
    ElMessage.success('Settings saved')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to save settings')
  } finally {
    saving.value = false
  }
}

async function sendTestEmail() {
  if (!testEmail.value.trim()) {
    ElMessage.warning('Enter a recipient email first')
    return
  }
  testing.value = true
  try {
    await testSystemEmail(testEmail.value.trim())
    ElMessage.success('Test email sent')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to send test email')
  } finally {
    testing.value = false
  }
}

onMounted(loadSettings)
</script>

<style scoped>
.system-settings-page {
  min-height: 100vh;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.settings-card {
  max-width: 980px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.card-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
}

.card-header p {
  margin: 0;
  color: var(--el-text-color-secondary);
}

.header-actions,
.test-row {
  display: flex;
  gap: 10px;
}

.settings-body {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.settings-form {
  max-width: 760px;
}

.test-row {
  width: 100%;
}
</style>
