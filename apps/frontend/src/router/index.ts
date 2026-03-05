import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { useAuthController } from '../controllers/authController';

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
router.beforeEach((to, _from, next) => {
  const auth = useAuthController();

  // 需要认证的路由
  if (to.meta.requiresAuth && !auth.isAuthenticated.value) {
    next('/login');
    return;
  }

  // 需要未认证的路由（已登录用户不能访问）
  if (to.meta.requiresGuest && auth.isAuthenticated.value) {
    next('/');
    return;
  }

  next();
});

export default router;
