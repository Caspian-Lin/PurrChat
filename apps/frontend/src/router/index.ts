import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('../views/HomeView.vue'),
    meta: { requiresAuth: true },
    redirect: '/chat',
    children: [
      {
        path: 'chat',
        name: 'Chat',
        component: () => import('../components/home/panel/ChatPanel.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'friends',
        name: 'Friends',
        component: () => import('../components/home/panel/FriendsPanel.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'ai',
        name: 'Ai',
        component: () => import('../components/home/panel/AiPanel.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'bots',
        name: 'Bots',
        component: () => import('../components/home/panel/BotStudioPanel.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('../components/home/panel/SettingsPanel.vue'),
        meta: { requiresAuth: true },
      },
    ],
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
    meta: { requiresGuest: true },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/RegisterView.vue'),
    meta: { requiresGuest: true },
  },
  {
    path: '/ws-debug',
    name: 'WebSocketDebug',
    component: () => import('../views/WebSocketDebugView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/bots/:botId/mechanisms/:mechanismId/special-mode',
    name: 'SpecialModeEditor',
    component: () => import('../views/SpecialModeEditorView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/privacy',
    name: 'PrivacyPolicy',
    component: () => import('../views/PrivacyPolicyView.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('../views/NotFoundView.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

// 路由守卫
router.beforeEach((to, from, next) => {
  const auth = useAuthStore();
  console.log('[router] 路由守卫', {
    to: to.path,
    requiresAuth: to.meta.requiresAuth,
    requiresGuest: to.meta.requiresGuest,
    isAuthenticated: auth.isAuthenticated,
    token: auth.token ? '存在' : '不存在',
  });

  // 在登录页面和注册页面之间切换时，清空错误信息
  if (
    (to.path === '/login' || to.path === '/register') &&
    (from.path === '/login' || from.path === '/register')
  ) {
    auth.error = null;
  }

  // 需要认证的路由
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    console.log('[router] 需要认证但未登录，跳转到登录页');
    next('/login');
    return;
  }

  // 需要未认证的路由（已登录用户不能访问）
  if (to.meta.requiresGuest && auth.isAuthenticated) {
    console.log('[router] 已登录用户访问需要未认证的路由，跳转到首页');
    next('/');
    return;
  }

  console.log('[router] 路由守卫通过，继续导航');
  next();
});

export default router;
