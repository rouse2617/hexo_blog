import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/chat'
  },
  {
    path: '/chat',
    name: 'Chat',
    component: () => import('@/views/Chat.vue'),
    meta: { title: '智能对话' }
  },
  {
    path: '/hosts',
    name: 'Hosts',
    component: () => import('@/views/Hosts.vue'),
    meta: { title: '主机管理' }
  },
  {
    path: '/tools',
    name: 'Tools',
    component: () => import('@/views/Tools.vue'),
    meta: { title: '工具列表' }
  },
  {
    path: '/scripts',
    name: 'Scripts',
    component: () => import('@/views/Scripts.vue'),
    meta: { title: '脚本管理' }
  },
  {
    path: '/tasks',
    name: 'Tasks',
    component: () => import('@/views/Tasks.vue'),
    meta: { title: '任务历史' }
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/Settings.vue'),
    meta: { title: '安全设置' }
  },
  {
    path: '/console',
    name: 'Console',
    component: () => import('@/views/Console.vue'),
    meta: { title: '控制台' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title || 'AI Pro'} - 智能运维助手`
  next()
})

export default router
