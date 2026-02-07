<template>
  <div class="min-h-screen flex items-center justify-center relative">
    <DynamicBackground />
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div
      class="relative z-10 bg-white dark:bg-gray-800 p-8 rounded-xl shadow-lg w-full max-w-md"
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
        创建新账号
      </p>

      <form @submit.prevent="handleSubmit" class="space-y-5">
        <n-form-item label="用户名" :show-feedback="false">
          <n-input
            v-model:value="username"
            placeholder="请输入用户名"
            size="large"
            :minlength="3"
            :maxlength="20"
            required
          />
        </n-form-item>

        <n-form-item label="邮箱" :show-feedback="false">
          <n-input
            v-model:value="email"
            placeholder="请输入邮箱"
            size="large"
            :minlength="3"
            required
          />
        </n-form-item>

        <n-form-item label="电话号码" :show-feedback="false">
          <n-input
            v-model:value="phone"
            placeholder="请输入电话"
            size="large"
            :minlength="11"
            required
          />
        </n-form-item>

        <n-form-item label="密码" :show-feedback="false">
          <n-input
            v-model:value="password"
            type="password"
            placeholder="请输入密码（至少6位）"
            show-password-on="click"
            size="large"
            :minlength="6"
            required
          />
        </n-form-item>

        <n-form-item label="确认密码" :show-feedback="false">
          <n-input
            v-model:value="confirmPassword"
            type="password"
            placeholder="请再次输入密码"
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
          {{ auth.loading.value ? '注册中...' : '注册' }}
        </n-button>
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { NButton, NInput, NFormItem, NAlert } from 'naive-ui';
import { useAuthController } from '../controllers/authController';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
import DynamicBackground from '../components/DynamicBackground.vue';

const auth = useAuthController();
const username = ref('');
const email = ref('');
const phone = ref('');
const password = ref('');
const confirmPassword = ref('');

const handleSubmit = async () => {
  if (password.value !== confirmPassword.value) {
    auth.error.value = '两次输入的密码不一致';
    return;
  }
  await auth.handleRegister(username.value, password.value, email.value, phone.value);
};
</script>

<style scoped></style>
