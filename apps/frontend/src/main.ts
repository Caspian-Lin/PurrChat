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

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

// 初始化主题
const themeStore = useThemeStore();
themeStore.initTheme();

app.mount('#app');
