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
        <n-form-item label="邮箱" :show-feedback="false">
          <n-input
            v-model:value="email"
            placeholder="请输入邮箱"
            size="large"
            :minlength="3"
            required
          />
        </n-form-item>

        <n-form-item label="密码" :show-feedback="false">
          <n-input
            v-model:value="password"
            type="password"
            placeholder="请输入密码"
            show-password-on="click"
            size="large"
            :minlength="6"
            required
          />
        </n-form-item>

        <n-alert v-if="auth.error" type="error" :bordered="false" class="mb-4">
          {{ auth.error }}
        </n-alert>

        <n-button
          type="primary"
          size="large"
          block
          :loading="auth.loading.value"
          @click="handleSubmit"
          class="!h-12 !font-medium"
        >
          {{ auth.loading.value ? '登录中...' : '登录' }}
        </n-button>
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
import { NButton, NInput, NFormItem, NAlert } from 'naive-ui';
import { useAuthController } from '../controllers/authController';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
import DynamicBackground from '../components/DynamicBackground.vue';

const auth = useAuthController();
const email = ref('');
const password = ref('');

const handleSubmit = async () => {
  await auth.handleLogin(email.value, password.value);
};
</script>

<style scoped></style>
