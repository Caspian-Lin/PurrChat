import { createApp } from 'vue';
import { createPinia } from 'pinia';
import './style.css';
// Vue Flow 全局样式
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import App from './App.vue';
import router from './router';
import { useThemeStore } from './stores/theme';
import { useAuthStore } from './stores/auth';

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

// 初始化主题
const themeStore = useThemeStore();
themeStore.initTheme();

// 启动时后台验证 Cookie 有效性（user 已从 localStorage 恢复，路由守卫可立即放行）
const authStore = useAuthStore();
if (authStore.user) {
  authStore.fetchUser();
}

app.mount('#app');
