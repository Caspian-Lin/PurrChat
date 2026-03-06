<template>
  <div class="min-h-screen flex items-center justify-center relative">
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div
      class="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-lg w-full max-w-md relative z-10"
      style="background: var(--background-color)"
    >
      <h1
        class="text-3xl font-bold text-center mb-2"
        style="
          background: var(--theme-gradient);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        "
      >
        PurrChat
      </h1>
      <p
        class="text-center text-gray-600 dark:text-gray-400 mb-8"
        style="color: var(--text-secondary-color)"
      >
        欢迎回来
      </p>

      <form @submit.prevent="handleSubmit" class="space-y-6">
        <BaseFormItem label="邮箱">
          <BaseInput v-model="email" type="email" placeholder="请输入邮箱" required />
        </BaseFormItem>

        <BaseFormItem label="密码">
          <BaseInput v-model="password" type="password" placeholder="请输入密码" required />
        </BaseFormItem>

        <BaseAlert v-if="auth.error" type="error" class="mb-4">
          {{ auth.error }}
        </BaseAlert>

        <button type="submit" class="w-full !h-12 !font-medium" block :disabled="auth.loading">
          {{ auth.loading ? '登录中...' : '登录' }}
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
  console.log('=== 登录开始 ===');
  console.log('邮箱:', email.value);
  console.log('密码:', password.value ? '***' : '');
  console.log('auth.loading:', auth.loading);

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
