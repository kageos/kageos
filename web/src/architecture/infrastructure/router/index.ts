import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { featureFlags } from '@/architecture/infrastructure/config/features'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // 认证页面
    {
      path: '/login',
      name: 'login',
      component: () => import('@/architecture/presentation/features/auth/pages/LoginPage.vue'),
      meta: {
        title: '登录',
        requireAuth: false
      }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/architecture/presentation/features/auth/pages/RegisterPage.vue'),
      meta: {
        title: '注册',
        requireAuth: false
      }
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('@/architecture/presentation/features/auth/pages/ForgotPasswordPage.vue'),
      meta: {
        title: '忘记密码',
        requireAuth: false
      }
    },
    {
      path: '/create-test-user',
      name: 'create-test-user',
      component: () => import('@/architecture/presentation/features/auth/pages/CreateTestUserPage.vue'),
      meta: {
        title: '创建测试用户',
        requireAuth: true
      }
    },

    // 用户设置页面
    {
      path: '/user/settings',
      name: 'user-settings',
      component: () => import('@/architecture/presentation/features/user/pages/UserSettingsPage.vue'),
      meta: {
        title: '个人设置',
        requireAuth: true
      }
    },
    // 组织架构和用户管理页面
    {
      path: '/organization',
      name: 'organization-management',
      component: () => import('@/architecture/presentation/features/organization/pages/OrganizationManagementPage.vue'),
      meta: {
        title: '组织架构和用户管理',
        requireAuth: true,
        feature: 'organization'
      }
    },

    // LLM 配置
    {
      path: '/agent',
      redirect: '/agent/llm',
      meta: {
        title: 'LLM 配置',
        requireAuth: true
      }
    },
    {
      path: '/agent/llm',
      name: 'llm-management',
      component: () => import('@/architecture/presentation/features/agent/pages/LLMManagementPage.vue'),
      meta: {
        title: 'LLM 管理',
        requireAuth: true,
        feature: 'llmManagement'
      }
    },

    // 根路径：直接走工作空间链路，后续由全局守卫补齐登录态和 username
    {
      path: '/',
      name: 'home',
      redirect: '/workspace',
      meta: {
        title: '首页',
        requireAuth: false
      }
    },

    // 工作空间页面（统一架构）
    // 注意：/workspace/api/* 路径会被 Vite 代理到后端，不会被 Vue Router 匹配
    {
      path: '/workspace',
      name: 'workspace',
      component: () => import('@/architecture/presentation/views/WorkspaceView.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
      }
    },
    {
      path: '/workspace/workstation',
      redirect: '/workspace',
      meta: {
        title: '工作空间',
        requireAuth: true
      }
    },
    // 仅 user、无 app：进入工作空间并弹出「选择工作空间」（须在 /workspace/:user/:app 前匹配）
    {
      path: '/workspace/:user',
      name: 'workspace-user',
      component: () => import('@/architecture/presentation/views/WorkspaceView.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
      }
    },
    {
      // 匹配 /workspace/:user/:app 等页面路由，但不匹配 /workspace/api/*
      // 使用更精确的路径匹配，排除 /api 开头的路径
      path: '/workspace/:user/:app/:path*',
      name: 'workspace-path',
      component: () => import('@/architecture/presentation/views/WorkspaceView.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
      },
      // 路由守卫：排除 /api 路径
      beforeEnter: (to, from, next) => {
        // 如果路径包含 /api，说明是 API 请求，不应该被 Vue Router 处理
        // 这种情况应该由 Vite 代理处理，但为了安全，我们在这里也做检查
        if (to.path.startsWith('/workspace/api')) {
          // 这不应该发生，因为 Vite 代理应该已经处理了
          // 但为了安全，我们返回 404
          next({ name: 'not-found' })
          return
        }
        next()
      }
    },

    // 404页面
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/architecture/presentation/views/Error/404.vue'),
      meta: {
        title: '页面不存在',
        requireAuth: false
      }
    }
  ],
})

async function restoreAccessTokenIfPossible(authStore: ReturnType<typeof useAuthStore>): Promise<boolean> {
  if (authStore.token) {
    return true
  }

  const refreshToken = authStore.refreshToken || localStorage.getItem('refresh_token') || ''
  if (!refreshToken) {
    return false
  }

  try {
    await authStore.refreshUserToken()
    return !!authStore.token
  } catch {
    await authStore.logout({
      callApi: false,
      notify: false,
      redirectToLogin: false,
    })
    return false
  }
}

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  const hasAuthSession = await restoreAccessTokenIfPossible(authStore)
  const routeFeature = to.meta?.feature

  if (typeof routeFeature === 'string' && !featureFlags[routeFeature as keyof typeof featureFlags]) {
    next({ path: '/workspace', replace: true })
    return
  }

  // 设置页面标题（Workspace页面会通过watch动态更新，这里只设置默认标题）
  if (to.meta?.title && !to.path.startsWith('/workspace')) {
    document.title = `${to.meta.title} - ${import.meta.env.VITE_APP_TITLE || 'AI Agent OS'}`
  }

  // 检查是否需要认证
  if (to.meta?.requireAuth !== false) {
    // 先尝试用 refresh token 无感恢复登录态，失败才进入登录页
    if (!hasAuthSession) {
      // 没有token，直接跳转到登录页
      next({ name: 'login', query: { redirect: to.fullPath } })
      return
    }
  }

  // 如果已登录用户访问登录/注册页面，重定向到 /workspace/自己的username（会弹出选择工作空间）
  if (hasAuthSession && (to.name === 'login' || to.name === 'register')) {
    const username = authStore.userName || 'me'
    next({ path: `/workspace/${username}`, replace: true })
    return
  }

  // 根路径 /：不再显示一站式首页，直接重定向到 /workspace/自己的username 并弹窗选择工作空间
  if (to.path === '/') {
    if (hasAuthSession) {
      const username = authStore.userName || 'me'
      next({ path: `/workspace/${username}`, replace: true })
      return
    }
    next({ path: '/login', query: { redirect: to.fullPath }, replace: true })
    return
  }

  // /workspace 无 user 时也重定向到 /workspace/自己的username
  if (to.path === '/workspace' && to.name === 'workspace') {
    if (hasAuthSession) {
      const username = authStore.userName || 'me'
      next({ path: `/workspace/${username}`, replace: true })
      return
    }
    next()
    return
  }

  next()
})

export default router
