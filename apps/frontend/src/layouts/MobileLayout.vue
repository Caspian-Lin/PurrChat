<template>
  <div class="mobile-layout">
    <!-- 主内容区（全屏，排除底部导航高度） -->
    <div class="mobile-content">
      <router-view />
    </div>

    <!-- 底部 Tab 导航 -->
    <BottomNav />
  </div>
</template>

<script setup lang="ts">
import BottomNav from '../components/mobile/BottomNav.vue';
</script>

<style scoped>
.mobile-layout {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: var(--background-color);
}

.mobile-content {
  flex: 1;
  overflow: hidden;
  /* 为底部导航留出空间 */
  padding-bottom: 56px;
}
</style>
