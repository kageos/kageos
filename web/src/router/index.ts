import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
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

    // 首页 - workspace页面（支持路径参数）
    {
      path: '/workspace',
      name: 'workspace',
      component: () => import('../views/Workspace/index.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
      }
    },
    {
      path: '/workspace/:path+',
      name: 'workspace-path',
      component: () => import('../views/Workspace/index.vue'),
      meta: {
        title: '工作空间',
        requireAuth: true
      }
    },
    // 重定向根路径到workspace
    {
      path: '/',
      redirect: '/workspace'
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

  // 设置页面标题
  if (to.meta?.title) {
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
