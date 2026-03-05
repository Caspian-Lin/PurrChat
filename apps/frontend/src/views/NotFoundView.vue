<template>
  <div
    class="min-h-screen flex items-center justify-center relative"
    style="background: var(--theme-gradient)"
  >
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div class="text-center text-white">
      <div class="text-9xl font-bold mb-4 opacity-80">404</div>
      <h1 class="text-4xl font-bold mb-4">页面未找到</h1>
      <p class="text-xl mb-8 opacity-90">抱歉，您访问的页面不存在</p>
      <router-link to="/">
        <BaseButton size="large" class="!bg-white !text-purple-600 hover:!bg-gray-100">
          返回首页
        </BaseButton>
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseButton from '../components/common/BaseButton.vue';
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
</script>

<style scoped></style>
