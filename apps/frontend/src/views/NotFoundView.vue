<template>
  <div
    class="min-h-screen flex items-center justify-center relative"
    style="background: var(--theme-gradient)"
  >
    <div class="absolute top-4 right-4 z-10">
      <ThemeSwitcher />
    </div>
    <div class="text-center text-white">
      <div class="text-9xl font-bold mb-4 opacity-80 font-display">404</div>
      <h1 class="text-4xl font-bold mb-4 font-display">页面未找到</h1>
      <p class="text-xl mb-8 opacity-90">抱歉，您访问的页面不存在</p>
      <router-link
        to="/"
        class="inline-block px-6 py-3 bg-white rounded-[var(--radius-sm)] hover:bg-gray-100 transition-colors font-medium"
        style="color: var(--theme-primary)"
      >
        返回首页
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import ThemeSwitcher from '../components/ThemeSwitcher.vue';
</script>

<style scoped></style>
