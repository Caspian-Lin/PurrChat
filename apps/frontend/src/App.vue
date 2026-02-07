<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <n-message-provider>
      <div id="app">
        <router-view />
      </div>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import {
  NConfigProvider,
  NMessageProvider,
  darkTheme,
  lightTheme as naiveLightTheme,
} from 'naive-ui';
import { useThemeStore } from './stores/theme';
import { themeColors } from './config/theme';

const themeStore = useThemeStore();

// NaiveUI主题
const naiveTheme = computed(() => {
  return themeStore.mode === 'dark' ? darkTheme : naiveLightTheme;
});

// NaiveUI主题覆盖
const themeOverrides = computed(() => {
  const colorConfig = themeColors[themeStore.color];
  return {
    common: {
      primaryColor: colorConfig.primary,
      primaryColorHover: colorConfig.secondary,
      primaryColorPressed: colorConfig.secondary,
      primaryColorSuppl: colorConfig.primary,
    },
  };
});
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family:
    -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
  background: var(--background-color);
  color: var(--text-color);
}

#app {
  width: 100%;
  height: 100%;
}
</style>
