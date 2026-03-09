import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // 🔥 测试页面（放在最前面，避免被其他路由匹配）
    {
      path: '/test/form-renderer',
      name: 'test-form-renderer',
      component: () => import('../views/Test/FormRendererTest.vue'),
      meta: {
        title: '表单渲染器测试',
        requireAuth: false
      }
    },

    // 认证页面
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/Auth/Login.vue'),
      meta: {
        title: '登录',
        requireAuth: false
      }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/Auth/Register.vue'),
      meta: {
        title: '注册',
        requireAuth: false
      }
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: () => import('../views/Auth/ForgotPassword.vue'),
      meta: {
        title: '忘记密码',
        requireAuth: false
      }
    },
    {
      path: '/create-test-user',
      name: 'create-test-user',
      component: () => import('../views/Auth/CreateTestUser.vue'),
      meta: {
        title: '创建测试用户',
        requireAuth: true
      }
    },

    // 用户设置页面
    {
      path: '/user/settings',
      name: 'user-settings',
      component: () => import('../views/User/Settings.vue'),
      meta: {
        title: '个人设置',
        requireAuth: true
      }
    },
    // 组织架构和用户管理页面
    {
      path: '/organization',
      name: 'organization-management',
      component: () => import('../views/Organization/index.vue'),
      meta: {
        title: '组织架构和用户管理',
        requireAuth: true
      }
    },

    // 权限申请页面
    {
      path: '/permissions/apply',
      name: 'permission-apply',
      component: () => import('../views/Permission/PermissionApply.vue'),
      meta: {
        title: '权限申请',
        requireAuth: true
      }
    },
    // 角色管理页面
    {
      path: '/permissions/roles',
      name: 'role-management',
      component: () => import('../views/Permission/RoleManagement.vue'),
      meta: {
        title: '角色管理',
        requireAuth: true
      }
    },

    // LLM 与工作台管理
    {
      path: '/agent',
      name: 'agent-index',
      component: () => import('../views/Agent/index.vue'),
      meta: {
        title: 'LLM 与工作台',
        requireAuth: true
      }
    },
    {
      path: '/agent/llm',
      name: 'llm-management',
      component: () => import('../views/Agent/LLMManagement.vue'),
      meta: {
        title: 'LLM 管理',
        requireAuth: true
      }
    },

    // 首页 - 官网
    {
      path: '/',
      name: 'home',
      component: () => import('../views/Home.vue'),
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
    // 智能工作台管理（模式列表与配置；带 full_code_path 时同页显示工作台对话）
    {
      path: '/workspace/workstation',
      name: 'workspace-workstation',
      component: () => import('../architecture/presentation/views/WorkstationView.vue'),
      meta: {
        title: '智能工作台管理',
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

  // 如果已登录用户访问登录/注册页面，重定向到工作空间
  if (authStore.isAuthenticated && (to.name === 'login' || to.name === 'register')) {
    next({ name: 'workspace' })
    return
  }

  next()
})

export default router
