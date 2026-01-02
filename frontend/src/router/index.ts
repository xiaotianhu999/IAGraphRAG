import { createRouter, createWebHistory } from 'vue-router'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { useAuthStore } from '@/stores/auth'
import { validateToken } from '@/api/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      redirect: "/platform/knowledge-bases",
    },
    {
      path: "/login",
      name: "login",
      component: () => import("../views/auth/Login.vue"),
      meta: { requiresAuth: false, requiresInit: false }
    },
    {
      path: "/init",
      name: "init",
      component: () => import("../views/auth/InitWizard.vue"),
      meta: { requiresAuth: false, requiresInit: false }
    },
    {
      path: "/knowledgeBase",
      name: "home",
      component: () => import("../views/knowledge/KnowledgeBase.vue"),
      meta: { requiresInit: true, requiresAuth: true }
    },
    {
      path: "/platform",
      name: "Platform",
      redirect: "/platform/knowledge-bases",
      component: () => import("../views/platform/index.vue"),
      meta: { requiresInit: true, requiresAuth: true },
      children: [
        {
          path: "tenant",
          redirect: "/platform/settings"
        },
        {
          path: "settings",
          name: "settings",
          component: () => import("../views/settings/Settings.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresMenuPermission: 'settings' }
        },
        {
          path: "knowledge-bases",
          name: "knowledgeBaseList",
          component: () => import("../views/knowledge/KnowledgeBaseList.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresMenuPermission: 'knowledge-bases' }
        },
        {
          path: "knowledge-bases/:kbId",
          name: "knowledgeBaseDetail",
          component: () => import("../views/knowledge/KnowledgeBase.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresMenuPermission: 'knowledge-bases' }
        },
        {
          path: "creatChat",
          name: "globalCreatChat",
          component: () => import("../views/creatChat/creatChat.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "knowledge-bases/:kbId/creatChat",
          name: "kbCreatChat",
          component: () => import("../views/creatChat/creatChat.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "chat/:chatid",
          name: "chat",
          component: () => import("../views/chat/index.vue"),
          meta: { requiresInit: true, requiresAuth: true }
        },
        {
          path: "admin/dashboard",
          name: "adminDashboard",
          component: () => import("../views/admin/Dashboard.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresAdmin: true }
        },
        {
          path: "admin/tenants",
          name: "adminTenants",
          component: () => import("../views/admin/TenantManager.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresSuperAdmin: true }
        },
        {
          path: "admin/users",
          name: "adminUsers",
          component: () => import("../views/admin/UserManager.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresAdmin: true }
        },
        {
          path: "admin/audit-logs",
          name: "adminAuditLogs",
          component: () => import("../views/admin/AuditLog.vue"),
          meta: { requiresInit: true, requiresAuth: true, requiresAdmin: true }
        },
      ],
    },
  ],
});

// 路由守卫：检查认证状态和系统初始化状态
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // 检查系统初始化状态 (仅在访问非初始化页面时检查)
  if (to.path !== '/init') {
    const isInitialized = await authStore.checkInitStatus()
    if (!isInitialized) {
      next('/init')
      return
    }
  }
  
  // 如果访问的是登录页面或初始化页面，直接放行
  if (to.meta.requiresAuth === false || to.meta.requiresInit === false) {
    // 如果已登录用户访问登录页面，重定向到知识库列表页面
    if (to.path === '/login' && authStore.isLoggedIn) {
      next('/platform/knowledge-bases')
      return
    }
    next()
    return
  }

  // 检查用户认证状态
  if (to.meta.requiresAuth !== false) {
    if (!authStore.isLoggedIn) {
      // 未登录，跳转到登录页面
      next('/login')
      return
    }

    // 检查超级管理员权限
    if (to.meta.requiresSuperAdmin && !authStore.canAccessAllTenants) {
      next('/platform/knowledge-bases')
      return
    }

    // 检查管理员权限 (超级管理员或租户管理员)
    if (to.meta.requiresAdmin && !authStore.isAdmin) {
      next('/platform/knowledge-bases')
      return
    }

    // 检查菜单权限（针对普通用户）
    if (to.meta.requiresMenuPermission && !authStore.isAdmin) {
      const menuConfig = authStore.user?.menu_config || []
      const requiredPermission = to.meta.requiresMenuPermission as string
      
      if (!menuConfig.includes(requiredPermission)) {
        console.warn(`User lacks menu permission: ${requiredPermission}`)
        // 重定向到用户有权限的默认页面
        next('/platform/creatChat')
        return
      }
    }

    // 验证Token有效性
    // try {
    //   const { valid } = await validateToken()
    //   if (!valid) {
    //     // Token无效，清空认证信息并跳转到登录页面
    //     authStore.logout()
    //     next('/login')
    //     return
    //   }
    // } catch (error) {
    //   console.error('Token验证失败:', error)
    //   authStore.logout()
    //   next('/login')
    //   return
    // }
  }

  next()
});

// 路由切换后确保菜单配置正确
router.afterEach(() => {
  const authStore = useAuthStore()
  if (authStore.isLoggedIn) {
    authStore.ensureMenuConfig()
  }
})

export default router
