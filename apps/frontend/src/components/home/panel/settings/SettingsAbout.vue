<template>
  <section id="settings-about" class="settings-section">
    <h2 class="settings-section__title">关于 PurrChat</h2>

    <!-- 品牌标识 -->
    <div class="text-center py-8">
      <p class="text-3xl font-bold" style="color: var(--theme-primary)">PurrChat</p>
      <p class="text-text-tertiary text-sm mt-2">Intimate · Refined · Alive</p>
      <p class="text-text-tertiary text-xs mt-1">探索人与 AI 的情感连接和灵感边界</p>
    </div>

    <!-- 版本信息 -->
    <div class="space-y-3 p-4 rounded-[var(--radius-sm,8px)]" style="background: var(--surface-secondary-color)">
      <div class="flex justify-between">
        <span class="text-sm text-text-secondary">版本</span>
        <span class="text-sm text-text-primary">{{ version }}</span>
      </div>
      <div class="flex justify-between">
        <span class="text-sm text-text-secondary">构建时间</span>
        <span class="text-sm text-text-primary">{{ buildDate }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { APP_VERSION } from '../../../../config/settings';

const version = APP_VERSION;
const buildDate = import.meta.env.VITE_BUILD_DATE || 'dev';
</script>
