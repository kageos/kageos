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

    // Agent-Server 管理页面
    {
      path: '/agent',
      name: 'agent-index',
      component: () => import('../views/Agent/index.vue'),
      meta: {
        title: 'Agent-Server 管理',
        requireAuth: true
      }
    },
    {
      path: '/agent/agents',
      name: 'agent-management',
      component: () => import('../views/Agent/AgentManagement.vue'),
      meta: {
        title: '智能体管理',
        requireAuth: true
      }
    },
    {
      path: '/agent/knowledge',
      name: 'knowledge-management',
      component: () => import('../views/Agent/KnowledgeManagement.vue'),
      meta: {
        title: '知识库管理',
        requireAuth: true
      }
    },
    {
      path: '/agent/knowledge/:id',
      name: 'knowledge-detail',
      component: () => import('../views/Agent/KnowledgeDetail.vue'),
      meta: {
        title: '知识库详情',
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
      path: '/workspace/:path+',
      name: 'workspace-path',
      component: () => import('../architecture/presentation/views/WorkspaceView.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
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

  // 🔥 调试日志
  console.log('[Router Guard] 导航:', {
    from: from.path,
    to: to.path,
    name: to.name,
    requireAuth: to.meta?.requireAuth,
    hasToken: !!authStore.token
  })

  // 设置页面标题（Workspace页面会通过watch动态更新，这里只设置默认标题）
  if (to.meta?.title && !to.path.startsWith('/workspace')) {
    document.title = `${to.meta.title} - ${import.meta.env.VITE_APP_TITLE || 'AI Agent OS'}`
  }

  // 检查是否需要认证
  if (to.meta?.requireAuth !== false) {
    // 检查登录状态（不自动调用API）
    if (!authStore.token) {
      // 没有token，直接跳转到登录页
      console.log('[Router Guard] 未登录，跳转到登录页')
      next({ name: 'login', query: { redirect: to.fullPath } })
      return
    }
  }

  // 如果已登录用户访问登录/注册页面，重定向到工作空间
  if (authStore.isAuthenticated && (to.name === 'login' || to.name === 'register')) {
    console.log('[Router Guard] 已登录用户访问登录页，跳转到工作空间')
    next({ name: 'workspace' })
    return
  }

  console.log('[Router Guard] 允许导航')
  next()
})

export default router
