<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Message, Check, Loading } from '@element-plus/icons-vue'
import { register as registerApi, sendEmailCode } from '@/api/auth'
import type { RegisterRequest } from '@/types'

const router = useRouter()

// 表单数据
const registerForm = reactive<RegisterRequest>({
  username: '',
  email: '',
  password: '',
  code: ''
})

// 表单引用
const registerFormRef = ref()

// 加载状态
const loading = ref(false)
const sendingCode = ref(false)
const countdown = ref(0)

// 表单验证规则
const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度在 2 到 50 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度在 6 到 50 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { len: 6, message: '验证码长度为 6 位', trigger: 'blur' }
  ]
}

// 处理注册
const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()
    loading.value = true

    await registerApi(registerForm)

    ElMessage.success('注册成功！请登录')
    await router.push('/login')
  } catch (error: any) {
    console.error('注册失败:', error)
    const message = error?.response?.data?.message || error?.message || '注册失败，请重试'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 发送验证码
const sendVerificationCode = async () => {
  if (!registerForm.email) {
    ElMessage.warning('请先输入邮箱地址')
    return
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(registerForm.email)) {
    ElMessage.warning('请输入正确的邮箱地址')
    return
  }

  try {
    sendingCode.value = true
    await sendEmailCode(registerForm.email)
    ElMessage.success('验证码已发送到您的邮箱')

    // 开始倒计时
    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (error: any) {
    console.error('发送验证码失败:', error)
    const message = error?.response?.data?.message || error?.message || '发送验证码失败，请重试'
    ElMessage.error(message)
  } finally {
    sendingCode.value = false
  }
}

// 跳转到登录页
const goToLogin = () => {
  router.push('/login')
}

// 处理回车键
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    handleRegister()
  }
}
</script>

<template>
  <div class="register-container" @keypress="handleKeyPress">
    <!-- 左侧品牌展示 -->
    <div class="register-brand">
      <div class="brand-content">
        <div class="brand-logo">
          <img alt="AI Agent OS" class="logo" src="@/assets/logo.svg" />
        </div>
        <h1 class="brand-title">创建新账号</h1>
        <p class="brand-subtitle">
          加入 AI Agent OS 平台<br />
          开始您的智能代理管理之旅
        </p>
        <div class="brand-steps">
          <div class="step-item">
            <div class="step-number">1</div>
            <div class="step-content">
              <h4>注册账号</h4>
              <p>填写基本信息创建账号</p>
            </div>
          </div>
          <div class="step-item">
            <div class="step-number">2</div>
            <div class="step-content">
              <h4>验证邮箱</h4>
              <p>验证邮箱确保安全</p>
            </div>
          </div>
          <div class="step-item">
            <div class="step-number">3</div>
            <div class="step-content">
              <h4>开始使用</h4>
              <p>创建您的第一个应用</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧注册表单 -->
    <div class="register-form-section">
      <div class="register-card">
        <div class="register-header">
          <h2>创建账号</h2>
          <p>填写您的信息来注册新账号</p>
        </div>

        <el-form
          ref="registerFormRef"
          :model="registerForm"
          :rules="rules"
          label-width="0"
          size="large"
          class="register-form"
        >
          <el-form-item prop="username">
            <el-input
              v-model="registerForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              clearable
              size="large"
            />
          </el-form-item>

          <el-form-item prop="email">
            <el-input
              v-model="registerForm.email"
              placeholder="请输入邮箱地址"
              :prefix-icon="Message"
              clearable
              size="large"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="registerForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              show-password
              clearable
              size="large"
            />
          </el-form-item>

          <el-form-item prop="code">
            <div class="code-input-group">
              <el-input
                v-model="registerForm.code"
                placeholder="请输入验证码"
                maxlength="6"
                clearable
                size="large"
              />
              <el-button
                type="primary"
                size="large"
                :disabled="countdown > 0 || sendingCode"
                :loading="sendingCode"
                @click="sendVerificationCode"
                class="code-btn"
              >
                {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
              </el-button>
            </div>
          </el-form-item>

          <el-form-item class="register-btn-item">
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="register-btn"
              @click="handleRegister"
            >
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              立即注册
            </el-button>
          </el-form-item>

          <div class="register-footer">
            <span class="register-tip">已有账号？</span>
            <el-button type="text" @click="goToLogin" class="login-link">
              立即登录
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-container {
  min-height: 100vh;
  display: flex;
  background: #f0f2f5;
}

.register-brand {
  flex: 1;
  background: linear-gradient(135deg, #4ade80 0%, #22c55e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 60px;
  position: relative;
  overflow: hidden;
}

.register-brand::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,224L48,208C96,192,192,160,288,160C384,160,480,192,576,197.3C672,203,768,181,864,186.7C960,192,1056,235,1152,240C1248,245,1344,213,1392,197.3L1440,181.3L1440,0L1392,0C1344,0,1248,0,1152,0C1056,0,960,0,864,0C768,0,672,0,576,0C480,0,384,0,288,0C192,0,96,0,48,0L0,0Z"></path></svg>') no-repeat bottom;
  background-size: cover;
}

.brand-content {
  text-align: center;
  color: white;
  z-index: 1;
  position: relative;
}

.brand-logo {
  margin-bottom: 24px;
}

.logo {
  width: 80px;
  height: 80px;
  filter: brightness(0) invert(1);
}

.brand-title {
  font-size: 36px;
  font-weight: 700;
  margin: 0 0 16px 0;
  color: white;
  letter-spacing: -0.5px;
}

.brand-subtitle {
  font-size: 16px;
  line-height: 1.6;
  margin: 0 0 48px 0;
  color: rgba(255, 255, 255, 0.9);
  max-width: 400px;
}

.brand-steps {
  text-align: left;
  max-width: 320px;
  margin: 0 auto;
}

.step-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 24px;
  color: rgba(255, 255, 255, 0.9);
}

.step-number {
  width: 32px;
  height: 32px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.step-content {
  flex: 1;
}

.step-content h4 {
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: white;
}

.step-content p {
  font-size: 13px;
  margin: 0;
  color: rgba(255, 255, 255, 0.8);
  line-height: 1.4;
}

.register-form-section {
  width: 520px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  padding: 60px;
}

.register-card {
  width: 100%;
  max-width: 450px;
}

.register-header {
  text-align: center;
  margin-bottom: 40px;
}

.register-header h2 {
  font-size: 28px;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 8px 0;
}

.register-header p {
  font-size: 14px;
  color: #6b7280;
  margin: 0;
}

.register-form {
  margin-bottom: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 24px;
}

:deep(.el-input__wrapper) {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  padding: 0 16px;
}

:deep(.el-input__inner) {
  height: 48px;
  font-size: 15px;
}

.code-input-group {
  display: flex;
  gap: 12px;
}

.code-input-group .el-input {
  flex: 1;
}

.code-btn {
  white-space: nowrap;
  min-width: 120px;
}

.register-btn-item {
  margin-bottom: 32px;
}

.register-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 8px;
}

.register-footer {
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.register-tip {
  font-size: 14px;
  color: #6b7280;
}

.login-link {
  font-size: 14px;
  font-weight: 500;
  color: #3b82f6;
  padding: 0;
}

.login-link:hover {
  color: #2563eb;
}

/* Element Plus 样式覆盖 */
:deep(.el-form-item__error) {
  padding-top: 4px;
  font-size: 12px;
}

:deep(.el-input__prefix) {
  color: #9ca3af;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .register-brand {
    padding: 0 40px;
  }

  .register-form-section {
    width: 480px;
    padding: 40px;
  }
}

@media (max-width: 768px) {
  .register-container {
    flex-direction: column;
  }

  .register-brand {
    width: 100%;
    padding: 60px 20px;
    min-height: auto;
  }

  .brand-content {
    max-width: 400px;
  }

  .brand-title {
    font-size: 28px;
  }

  .brand-subtitle {
    font-size: 14px;
  }

  .register-form-section {
    width: 100%;
    padding: 40px 20px;
  }

  .register-card {
    max-width: 100%;
  }
}

@media (max-width: 480px) {
  .brand-steps {
    display: none;
  }

  .register-form-section {
    padding: 30px 20px;
  }

  .code-input-group {
    flex-direction: column;
    gap: 12px;
  }

  .code-btn {
    width: 100%;
    min-width: auto;
  }
}
</style>