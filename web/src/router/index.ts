import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
    history: createWebHistory(),
    routes: [
        // 管理后台路由
        {
            path: '/admin',
            component: () => import('../views/admin/Layout.vue'),
            meta: { requiresAuth: true },
            children: [
                {
                    path: '',
                    redirect: '/admin/dashboard'
                },
                {
                    path: 'dashboard',
                    name: 'Dashboard',
                    component: () => import('../views/admin/Dashboard.vue'),
                    meta: { title: '概览' }
                },
                {
                    path: 'files',
                    name: 'Files',
                    component: () => import('../views/admin/Files.vue'),
                    meta: { title: '文件浏览' }
                },
                {
                    path: 'shares',
                    name: 'Shares',
                    component: () => import('../views/admin/Shares.vue'),
                    meta: { title: '分享管理' }
                },
                {
                    path: 'settings',
                    name: 'Settings',
                    component: () => import('../views/admin/Settings.vue'),
                    meta: { title: '系统设置' }
                }
            ]
        },
        {
            path: '/admin/login',
            name: 'Login',
            component: () => import('../views/admin/Login.vue'),
            meta: { title: '登录' }
        },
        // 分享页面路由
        {
            path: '/s/:code',
            name: 'Share',
            component: () => import('../views/share/ShareView.vue')
        },
        // 首页重定向
        {
            path: '/',
            redirect: '/admin'
        },
        // 404
        {
            path: '/:pathMatch(.*)*',
            component: () => import('../views/NotFound.vue')
        }
    ]
})

// 路由守卫 - 延迟导入 store 避免循环依赖
router.beforeEach(async (to, _from, next) => {
    // 设置页面标题
    document.title = to.meta.title ? `${to.meta.title} - WoOpen` : 'WoOpen'

    // 检查是否需要认证
    if (to.meta.requiresAuth) {
        // 延迟导入 store
        const { useAuthStore } = await import('../stores/auth')
        const authStore = useAuthStore()

        if (!authStore.isLoggedIn) {
            next('/admin/login')
            return
        }
    }

    next()
})

export default router
