<template>
  <div class="min-h-screen flex items-center justify-center relative">
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div
      class="p-8 rounded-[var(--radius-xl)] w-full max-w-md relative z-10 elevated-lg"
      style="background: var(--background-color)"
    >
      <h1
        class="text-3xl font-bold font-display text-center mb-2"
        style="color: var(--theme-primary)"
      >
        PurrChat
      </h1>
      <p class="text-center mb-8" style="color: var(--text-secondary-color)">欢迎回来</p>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <BaseFormItem label="邮箱">
          <BaseInput v-model="email" type="email" placeholder="请输入邮箱" required />
        </BaseFormItem>

        <BaseFormItem label="密码">
          <BaseInput v-model="password" type="password" placeholder="请输入密码" required />
        </BaseFormItem>

        <BaseAlert v-if="auth.error.value" type="error" class="mb-4">
          {{ auth.error }}
        </BaseAlert>

        <button
          type="submit"
          class="w-full h-12 font-medium"
          style="background: var(--theme-primary); color: #fff"
          :disabled="auth.loading.value"
        >
          {{ auth.loading.value ? '登录中...' : '登录' }}
        </button>
      </form>

      <div class="text-center mt-6 text-sm" style="color: var(--text-secondary-color)">
        还没有账号？
        <router-link
          to="/register"
          class="font-medium hover:underline"
          style="color: var(--theme-primary)"
        >
          立即注册
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
    <DynamicBackground />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthController } from '../controllers/authController';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
import DynamicBackground from '../components/DynamicBackground.vue';
import BaseInput from '../components/common/BaseInput.vue';
import BaseFormItem from '../components/common/BaseFormItem.vue';
import BaseAlert from '../components/common/BaseAlert.vue';

const auth = useAuthController();
const email = ref('');
const password = ref('');

const handleSubmit = async () => {
  // 清除之前的错误信息
  auth.clearError();

  console.log('=== 登录开始 ===');
  console.log('邮箱:', email.value);
  console.log('密码:', password.value ? '***' : '');
  console.log('auth.loading:', auth.loading.value);

  try {
    const result = await auth.handleLogin(email.value, password.value);
    console.log('登录结果:', result);
  } catch (error) {
    console.error('登录异常:', error);
  }

  console.log('=== 登录结束 ===');
};
</script>

<style scoped></style>
