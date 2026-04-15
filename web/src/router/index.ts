import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // 认证页面
    {
      path: '/login',
      name: 'login',
      component: () => import('../features/auth/pages/LoginPage.vue'),
      meta: {
        title: '登录',
        requireAuth: false
      }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../features/auth/pages/RegisterPage.vue'),
      meta: {
        title: '注册',
        requireAuth: false
      }
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('../features/auth/pages/ForgotPasswordPage.vue'),
      meta: {
        title: '忘记密码',
        requireAuth: false
      }
    },
    {
      path: '/create-test-user',
      name: 'create-test-user',
      component: () => import('../features/auth/pages/CreateTestUserPage.vue'),
      meta: {
        title: '创建测试用户',
        requireAuth: true
      }
    },

    // 用户设置页面
    {
      path: '/user/settings',
      name: 'user-settings',
      component: () => import('../features/user/pages/UserSettingsPage.vue'),
      meta: {
        title: '个人设置',
        requireAuth: true
      }
    },
    // 组织架构和用户管理页面
    {
      path: '/organization',
      name: 'organization-management',
      component: () => import('../features/organization/pages/OrganizationManagementPage.vue'),
      meta: {
        title: '组织架构和用户管理',
        requireAuth: true
      }
    },

    // 权限申请页面
    {
      path: '/permissions/apply',
      name: 'permission-apply',
      component: () => import('../features/permission/pages/PermissionApplyPage.vue'),
      meta: {
        title: '权限申请',
        requireAuth: true
      }
    },
    // 角色管理页面
    {
      path: '/permissions/roles',
      name: 'role-management',
      component: () => import('../features/permission/pages/RoleManagementPage.vue'),
      meta: {
        title: '角色管理',
        requireAuth: true
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
      component: () => import('../features/agent/pages/LLMManagementPage.vue'),
      meta: {
        title: 'LLM 管理',
        requireAuth: true
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

    // 工作空间页面（新架构）
    // 注意：/workspace/api/* 路径会被 Vite 代理到后端，不会被 Vue Router 匹配
    {
      path: '/workspace',
      name: 'workspace',
      component: () => import('../architecture/presentation/views/WorkspaceView.vue'),
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
      component: () => import('../architecture/presentation/views/WorkspaceView.vue'),
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
      component: () => import('../architecture/presentation/views/WorkspaceView.vue'),
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
      component: () => import('../views/Error/404.vue'),
      meta: {
        title: '页面不存在',
        requireAuth: false
      }
    }
  ],
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // 设置页面标题（Workspace页面会通过watch动态更新，这里只设置默认标题）
  if (to.meta?.title && !to.path.startsWith('/workspace')) {
    document.title = `${to.meta.title} - ${import.meta.env.VITE_APP_TITLE || 'AI Agent OS'}`
  }

  // 检查是否需要认证
  if (to.meta?.requireAuth !== false) {
    // 检查登录状态（不自动调用API）
    if (!authStore.token) {
      // 没有token，直接跳转到登录页
      next({ name: 'login', query: { redirect: to.fullPath } })
      return
    }
  }

  // 如果已登录用户访问登录/注册页面，重定向到 /workspace/自己的username（会弹出选择工作空间）
  if (authStore.isAuthenticated && (to.name === 'login' || to.name === 'register')) {
    const username = authStore.userName || 'me'
    next({ path: `/workspace/${username}`, replace: true })
    return
  }

  // 根路径 /：不再显示一站式首页，直接重定向到 /workspace/自己的username 并弹窗选择工作空间
  if (to.path === '/') {
    if (authStore.isAuthenticated) {
      const username = authStore.userName || 'me'
      next({ path: `/workspace/${username}`, replace: true })
      return
    }
    next({ path: '/login', query: { redirect: to.fullPath }, replace: true })
    return
  }

  // /workspace 无 user 时也重定向到 /workspace/自己的username
  if (to.path === '/workspace' && to.name === 'workspace') {
    if (authStore.isAuthenticated) {
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
