import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { featureFlags } from '@/architecture/shared/config/features'
import { setAppRouter } from '@/architecture/shared/routing/navigation'
import { onLocaleChanged, translate } from '@/architecture/shared/i18n'

function updateDocumentTitle(titleKey: unknown, path: string) {
  if (typeof titleKey !== 'string' || path.startsWith('/workspace')) {
    return
  }
  document.title = `${translate(titleKey)} - ${import.meta.env.VITE_APP_TITLE || 'kageos'}`
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // 认证页面
    {
      path: '/login',
      name: 'login',
      component: () => import('@/architecture/presentation/features/auth/pages/LoginPage.vue'),
      meta: {
        titleKey: 'route.login',
        requireAuth: false
      }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/architecture/presentation/features/auth/pages/RegisterPage.vue'),
      meta: {
        titleKey: 'route.register',
        requireAuth: false
      }
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('@/architecture/presentation/features/auth/pages/ForgotPasswordPage.vue'),
      meta: {
        titleKey: 'route.forgotPassword',
        requireAuth: false
      }
    },
    {
      path: '/auth/oauth/callback',
      name: 'oauth-callback',
      component: () => import('@/architecture/presentation/features/auth/pages/OAuthCallbackPage.vue'),
      meta: {
        titleKey: 'route.login',
        requireAuth: false
      }
    },
    {
      path: '/auth/oauth/register',
      name: 'oauth-register',
      component: () => import('@/architecture/presentation/features/auth/pages/OAuthRegisterPage.vue'),
      meta: {
        titleKey: 'route.oauthRegister',
        requireAuth: false
      }
    },
    {
      path: '/public/s/:shareId',
      name: 'public-share',
      component: () => import('@/architecture/presentation/features/public/pages/PublicSharePage.vue'),
      meta: {
        titleKey: 'route.publicShare',
        requireAuth: false
      }
    },
    {
      path: '/m/action',
      name: 'mobile-action',
      component: () => import('@/architecture/presentation/features/mobile/pages/MobileActionPage.vue'),
      meta: {
        titleKey: 'route.mobileAction',
        requireAuth: true
      }
    },
    {
      path: '/m',
      name: 'mobile-ask',
      component: () => import('@/architecture/presentation/features/mobile/pages/MobileAskPage.vue'),
      meta: {
        titleKey: 'route.mobileAsk',
        requireAuth: true
      }
    },

    // 用户设置页面
    {
      path: '/user/settings',
      name: 'user-settings',
      component: () => import('@/architecture/presentation/features/user/pages/UserSettingsPage.vue'),
      meta: {
        titleKey: 'route.userSettings',
        requireAuth: true
      }
    },
    // LLM 配置
    {
      path: '/agent',
      redirect: '/agent/llm',
      meta: {
        titleKey: 'route.llmConfig',
        requireAuth: true
      }
    },
    {
      path: '/agent/llm',
      name: 'llm-management',
      component: () => import('@/architecture/presentation/features/agent/pages/LLMManagementPage.vue'),
      meta: {
        titleKey: 'route.llmManagement',
        requireAuth: true,
        feature: 'llmManagement'
      }
    },
    {
      path: '/agent/openapi',
      redirect: {
        path: '/system/settings',
        query: { tab: 'openapi' }
      },
      meta: {
        titleKey: 'route.openapiConfig',
        requireAuth: true,
        feature: 'openapiTokens'
      }
    },
    {
      path: '/system/settings',
      name: 'system-settings',
      component: () => import('@/architecture/presentation/features/system/pages/SystemSettingsPage.vue'),
      meta: {
        titleKey: 'route.systemSettings',
        requireAuth: true
      }
    },
    {
      path: '/connectors',
      redirect: {
        path: '/system/settings',
        query: { tab: 'connectors' }
      },
      meta: {
        titleKey: 'route.connectorManagement',
        requireAuth: true,
        feature: 'connectorSettings'
      }
    },
    {
      path: '/connectors/providers',
      redirect: {
        path: '/system/settings',
        query: { tab: 'connectors' }
      },
      meta: {
        titleKey: 'route.connectorManagement',
        requireAuth: true,
        feature: 'connectorSettings'
      }
    },
    {
      path: '/permissions',
      alias: ['/permissions/access', '/permissions/apply'],
      name: 'permissions',
      component: () => import('@/architecture/presentation/features/access/pages/PermissionPage.vue'),
      meta: {
        titleKey: 'access.title',
        requireAuth: true
      }
    },

    // 根路径：直接走工作空间链路，后续由全局守卫补齐登录态和 username
    {
      path: '/',
      name: 'home',
      redirect: '/workspace',
      meta: {
        titleKey: 'route.home',
        requireAuth: false
      }
    },

    // 工作空间页面（统一架构）
    // 注意：/workspace/api/* 路径会被 Vite 代理到后端，不会被 Vue Router 匹配
    {
      path: '/workspace',
      name: 'workspace',
      component: () => import('@/architecture/presentation/views/WorkspaceBootstrapView.vue'),
      meta: {
        titleKey: 'route.workspace',
        requireAuth: true
      }
    },
    {
      path: '/workspace/workstation',
      redirect: '/workspace',
      meta: {
        titleKey: 'route.workspace',
        requireAuth: true
      }
    },
    // 兼容旧的 /workspace/:user 入口，统一准备并进入默认空间。
    {
      path: '/workspace/:user',
      name: 'workspace-user',
      component: () => import('@/architecture/presentation/views/WorkspaceBootstrapView.vue'),
      meta: {
        titleKey: 'route.workspace',
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
        titleKey: 'route.workspace',
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
        titleKey: 'route.notFound',
        requireAuth: false
      }
    }
  ],
})

setAppRouter(router)

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
  updateDocumentTitle(to.meta?.titleKey, to.path)

  // 检查是否需要认证
  if (to.meta?.requireAuth !== false) {
    // 先尝试用 refresh token 无感恢复登录态，失败才进入登录页
    if (!hasAuthSession) {
      // 没有token，直接跳转到登录页
      next({ name: 'login', query: { redirect: to.fullPath } })
      return
    }
  }

  // 已登录用户访问登录/注册页面时进入默认空间准备页。
  if (hasAuthSession && (to.name === 'login' || to.name === 'register')) {
    next({ path: '/workspace', replace: true })
    return
  }

  // 根路径直接进入默认空间准备页。
  if (to.path === '/') {
    if (hasAuthSession) {
      next({ path: '/workspace', replace: true })
      return
    }
    next({ path: '/login', query: { redirect: to.fullPath }, replace: true })
    return
  }

  next()
})

export default router

onLocaleChanged(() => {
  const route = router.currentRoute.value
  updateDocumentTitle(route.meta?.titleKey, route.path)
})
