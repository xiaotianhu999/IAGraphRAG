import { reactive, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import i18n from '@/i18n'

type MenuChild = Record<string, any>

interface MenuItem {
  title: string
  titleKey?: string
  icon: string
  path: string
  childrenPath?: string
  children?: MenuChild[]
}

const createMenuChildren = () => reactive<MenuChild[]>([])

export const useMenuStore = defineStore('menuStore', () => {
  const allMenus: MenuItem[] = [
    { title: '', titleKey: 'menu.dashboard', icon: 'setting', path: 'admin/dashboard' },
    { title: '', titleKey: 'menu.knowledgeBase', icon: 'zhishiku', path: 'knowledge-bases' },
    {
      title: '',
      titleKey: 'menu.chat',
      icon: 'prefixIcon',
      path: 'creatChat',
      childrenPath: 'chat',
      children: createMenuChildren()
    },
    { title: '', titleKey: 'menu.settings', icon: 'setting', path: 'settings' },
    { title: '', titleKey: 'menu.tenantManagement', icon: 'setting', path: 'admin/tenants' },
    { title: '', titleKey: 'menu.userManagement', icon: 'setting', path: 'admin/users' },
    { title: '', titleKey: 'menu.auditLog', icon: 'setting', path: 'admin/audit-logs' },
    { title: '', titleKey: 'menu.logout', icon: 'logout', path: 'logout' }
  ]

  const menuArr = reactive<MenuItem[]>([...allMenus])

  const setMenuConfig = (allowedPaths: string[], isSuperAdmin: boolean, role?: string) => {
    const filtered = allMenus.filter(item => {
      if (item.path === 'logout') return true
      // 租户管理仅限超级管理员
      if (item.path === 'admin/tenants') return isSuperAdmin
      // 系统仪表盘允许超级管理员或租户管理员
      if (item.path === 'admin/dashboard') return isSuperAdmin || role === 'admin'
      // 用户管理和审计日志：超级管理员总是可见，租户管理员需检查配置
      if (item.path === 'admin/users' || item.path === 'admin/audit-logs') {
        if (isSuperAdmin) return true
        if (role === 'admin') return allowedPaths.includes(item.path)
        return false
      }
      // 其他菜单（知识库、对话、设置）：
      // 超级管理员和租户管理员可见所有功能
      // 普通用户根据配置决定权限
      if (isSuperAdmin || role === 'admin') return true
      
      // 普通用户：严格按照租户配置的 menu_config 控制访问权限
      // 如果配置为空，默认只允许 AI对话 功能
      if (allowedPaths.length === 0) {
        return item.path === 'creatChat'
      }
      return allowedPaths.includes(item.path)
    })
    menuArr.splice(0, menuArr.length, ...filtered)
    applyMenuTranslations()
  }

  const isFirstSession = ref(false)
  const firstQuery = ref('')
  const firstMentionedItems = ref<any[]>([])

  const applyMenuTranslations = () => {
    menuArr.forEach(item => {
      if (item.titleKey) {
        item.title = i18n.global.t(item.titleKey)
      }
    })
  }

  applyMenuTranslations()

  watch(
    () => i18n.global.locale.value,
    () => {
      applyMenuTranslations()
    }
  )

  const clearMenuArr = () => {
    menuArr.splice(0, menuArr.length, ...allMenus)
    applyMenuTranslations()
  }

  const updatemenuArr = (obj: any) => {
    const chatMenu = menuArr[1]
    if (!chatMenu.children) {
      chatMenu.children = createMenuChildren()
    }
    const exists = chatMenu.children.some((item: MenuChild) => item.id === obj.id)
    if (!exists) {
      chatMenu.children.push(obj)
    }
  }

  const updataMenuChildren = (item: MenuChild) => {
    const chatMenu = menuArr[1]
    if (!chatMenu.children) {
      chatMenu.children = createMenuChildren()
    }
    chatMenu.children.unshift(item)
  }

  const updatasessionTitle = (sessionId: string, title: string) => {
    const chatMenu = menuArr[1]
    chatMenu.children?.forEach((item: MenuChild) => {
      if (item.id === sessionId) {
        item.title = title
        item.isNoTitle = false
      }
    })
  }

  const changeIsFirstSession = (payload: boolean) => {
    isFirstSession.value = payload
  }

  const changeFirstQuery = (payload: string, mentionedItems: any[] = []) => {
    firstQuery.value = payload
    firstMentionedItems.value = mentionedItems
  }

  return {
    menuArr,
    isFirstSession,
    firstQuery,
    firstMentionedItems,
    setMenuConfig,
    clearMenuArr,
    updatemenuArr,
    updataMenuChildren,
    updatasessionTitle,
    changeIsFirstSession,
    changeFirstQuery
  }
})
