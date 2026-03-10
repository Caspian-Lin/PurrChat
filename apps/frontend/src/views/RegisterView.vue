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

        <BaseAlert v-if="auth.error.value" type="error" class="mb-4">
          {{ auth.error }}
        </BaseAlert>

        <button
          type="submit"
          block
          :disabled="auth.loading.value"
          class="!h-12 w-full !font-medium"
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
    </div>
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
const username = ref('');
const email = ref('');
const phone = ref('');
const password = ref('');
const confirmPassword = ref('');

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

  await auth.handleRegister(username.value, password.value, email.value, phone.value);
};
</script>

<style scoped></style>
