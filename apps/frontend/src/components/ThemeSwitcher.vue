<template>
  <n-dropdown trigger="click" :options="dropdownOptions" @select="handleSelect">
    <n-button circle size="large" quaternary>
      <template #icon>
        <n-icon :component="themeIcon" />
      </template>
    </n-button>
  </n-dropdown>
</template>

<script setup lang="ts">
import { computed, h } from 'vue';
import { NButton, NDropdown, NIcon, type DropdownOption } from 'naive-ui';
import { SunnyOutline, MoonOutline } from '@vicons/ionicons5';
import { useThemeStore } from '../stores/theme';
import { themeColors } from '../config/theme';

const themeStore = useThemeStore();

const themeIcon = computed(() => {
  return themeStore.mode === 'light' ? SunnyOutline : MoonOutline;
});

const dropdownOptions = computed<DropdownOption[]>(() => [
  {
    type: 'group',
    label: '主题模式',
    key: 'theme-mode',
    children: [
      {
        label: '浅色',
        key: 'light',
        icon: () => h(NIcon, null, { default: () => h(SunnyOutline) }),
        props: {
          style: {
            color: themeStore.mode === 'light' ? 'var(--theme-primary)' : undefined,
          },
        },
      },
      {
        label: '深色',
        key: 'dark',
        icon: () => h(NIcon, null, { default: () => h(MoonOutline) }),
        props: {
          style: {
            color: themeStore.mode === 'dark' ? 'var(--theme-primary)' : undefined,
          },
        },
      },
    ],
  },
  {
    type: 'divider',
    key: 'd1',
  },
  {
    type: 'group',
    label: '主题色',
    key: 'theme-color',
    children: Object.entries(themeColors).map(([key, color]) => ({
      label: key.charAt(0).toUpperCase() + key.slice(1),
      key: `color-${key}`,
      icon: () =>
        h('div', {
          style: {
            width: '20px',
            height: '20px',
            borderRadius: '4px',
            background: color.gradient,
          },
        }),
      props: {
        style: {
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
        },
      },
    })),
  },
]);

const handleSelect = (key: string) => {
  if (key === 'light' || key === 'dark') {
    themeStore.setMode(key);
  } else if (key.startsWith('color-')) {
    const colorKey = key.replace('color-', '') as keyof typeof themeColors;
    themeStore.setColor(colorKey);
  }
};
</script>

<style scoped></style>
