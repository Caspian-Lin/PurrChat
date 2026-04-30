<template>
  <div class="min-h-screen flex items-center justify-center relative">
    <DynamicBackground />
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div
      class="relative z-10 p-8 rounded-[var(--radius-xl)] w-full max-w-md elevated-lg"
      style="background: var(--background-color)"
    >
      <h1
        class="text-3xl font-bold font-display text-center mb-2"
        style="color: var(--theme-primary)"
      >
        PurrChat
      </h1>
      <p class="text-center mb-8" style="color: var(--text-secondary-color)">创建新账号</p>

      <form @submit.prevent="handleSubmit" class="space-y-5">
        <BaseFormItem label="用户名">
          <BaseInput v-model="username" type="text" placeholder="请输入用户名" required />
        </BaseFormItem>

        <BaseFormItem label="邮箱">
          <BaseInput v-model="email" type="email" placeholder="请输入邮箱" required />
        </BaseFormItem>

        <BaseFormItem label="电话号码">
          <BaseInput v-model="phone" type="tel" placeholder="请输入电话" required />
        </BaseFormItem>

        <BaseFormItem label="密码">
          <BaseInput
            v-model="password"
            type="password"
            placeholder="请输入密码（至少6位）"
            required
          />
        </BaseFormItem>

        <BaseFormItem label="确认密码">
          <BaseInput
            v-model="confirmPassword"
            type="password"
            placeholder="请再次输入密码"
            required
          />
        </BaseFormItem>

        <!-- Cloudflare Turnstile 人机验证 -->
        <div v-if="turnstileEnabled" class="flex justify-center">
          <div id="turnstile-container"></div>
        </div>

        <BaseAlert v-if="auth.error.value" type="error" class="mb-4">
          {{ auth.error }}
        </BaseAlert>

        <button
          type="submit"
          class="w-full h-12 font-medium"
          style="background: var(--theme-primary); color: #fff"
          :disabled="auth.loading.value"
        >
          {{ auth.loading.value ? '注册中...' : '注册' }}
        </button>
      </form>

      <div class="text-center mt-6 text-sm" style="color: var(--text-secondary-color)">
        已有账号？
        <router-link
          to="/login"
          class="font-medium hover:underline"
          style="color: var(--theme-primary)"
        >
          立即登录
        </router-link>
      </div>

      <!-- 隐私协议 + ICP 备案 -->
      <div class="mt-8 pt-5 text-center space-y-1.5" style="border-top: 1px solid var(--border-subtle-color)">
        <p class="text-xs" style="color: var(--text-tertiary-color)">
          <router-link to="/privacy" class="hover:underline" style="color: var(--text-tertiary-color)">隐私政策</router-link>
        </p>
        <p class="text-xs" style="color: var(--text-tertiary-color)">
          <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer" class="hover:underline" style="color: var(--text-tertiary-color)">京ICP备XXXXXXXX号</a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useAuthController } from '../controllers/authController';
import { api } from '../models/api';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
import DynamicBackground from '../components/DynamicBackground.vue';
import BaseInput from '../components/common/BaseInput.vue';
import BaseFormItem from '../components/common/BaseFormItem.vue';
import BaseAlert from '../components/common/BaseAlert.vue';

const auth = useAuthController();
const username = ref('');
const email = ref('');
const phone = ref('');
const password = ref('');
const confirmPassword = ref('');

// Turnstile 状态
const turnstileEnabled = ref(false);
const turnstileToken = ref('');

onMounted(async () => {
  try {
    const config = await api.getTurnstileConfig();
    if (config.enabled && config.site_key) {
      turnstileEnabled.value = true;
      loadTurnstileScript(config.site_key);
    }
  } catch {
    // Turnstile 不可用时，允许继续注册（后端也会跳过验证）
  }
});

function loadTurnstileScript(siteKey: string) {
  // 避免重复加载
  if (document.querySelector('script[src*="challenges.cloudflare.com"]')) {
    renderWidget(siteKey);
    return;
  }
  const script = document.createElement('script');
  script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
  script.async = true;
  script.onload = () => renderWidget(siteKey);
  document.head.appendChild(script);
}

function renderWidget(siteKey: string) {
  const container = document.getElementById('turnstile-container');
  if (!container) return;

  // @ts-ignore - Turnstile 全局对象
  if (window.turnstile) {
    container.innerHTML = '';
    // @ts-ignore
    window.turnstile.render(container, {
      sitekey: siteKey,
      callback: (token: string) => {
        turnstileToken.value = token;
      },
      'error-callback': () => {
        turnstileToken.value = '';
      },
      'expired-callback': () => {
        turnstileToken.value = '';
      },
    });
  }
}

const handleSubmit = async () => {
  // 清除之前的错误信息
  auth.clearError();

  // 验证用户名长度
  if (username.value.length < 3 || username.value.length > 20) {
    auth.error.value = '用户名长度必须在3-20个字符之间';
    return;
  }

  // 验证密码长度
  if (password.value.length < 6) {
    auth.error.value = '密码长度至少为6个字符';
    return;
  }

  // 验证两次密码是否一致
  if (password.value !== confirmPassword.value) {
    auth.error.value = '两次输入的密码不一致';
    return;
  }

  // 验证邮箱格式（如果提供了邮箱）
  if (email.value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    auth.error.value = '邮箱格式不正确';
    return;
  }

  // 验证手机号长度（如果提供了手机号）
  if (phone.value && phone.value.length > 20) {
    auth.error.value = '手机号长度不能超过20个字符';
    return;
  }

  await auth.handleRegister(
    username.value,
    password.value,
    email.value,
    phone.value,
    turnstileToken.value || undefined
  );
};
</script>

<style scoped></style>
