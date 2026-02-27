import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'hub-market',
      component: () => import('../views/HubMarket.vue'),
      meta: {
        title: '应用中心'
      }
    },
    {
      path: '/directory/:path+',
      name: 'hub-directory-detail',
      component: () => import('../views/HubDirectoryDetail.vue'),
      meta: {
        title: '目录详情'
      }
    },
    {
      path: '/manage',
      name: 'hub-directory-manage',
      component: () => import('../views/HubDirectoryManage.vue'),
      meta: {
        title: '我的目录',
        requireAuth: true
      }
    },
    {
      path: '/pub-keys',
      name: 'pub-key-manage',
      component: () => import('../views/PubKeyManage.vue'),
      meta: {
        title: '发布密钥',
        requireAuth: true
      }
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('../views/Error/404.vue'),
      meta: {
        title: '页面不存在'
      }
    }
  ],
})

// 路由守卫
router.beforeEach((to, from, next) => {
  // 设置页面标题
  if (to.meta?.title) {
    document.title = `${to.meta.title} - Hub 应用中心`
  }

  // 🔥 处理从 OS 传递过来的 token（跨站点登录）
  // OS 跳转到 Hub 时，会通过 URL 参数传递 token
  const tokenFromUrl = to.query.token as string
  if (tokenFromUrl) {
    // 保存 token 到 localStorage
    localStorage.setItem('token', tokenFromUrl)
    
    // 清除 URL 中的 token 参数（安全考虑，避免 token 泄露）
    const newQuery = { ...to.query }
    delete newQuery.token
    
    // 使用 replace 避免在历史记录中留下带 token 的 URL
    next({
      path: to.path,
      query: newQuery,
      replace: true
    })
    return
  }

  next()
})

export default router

