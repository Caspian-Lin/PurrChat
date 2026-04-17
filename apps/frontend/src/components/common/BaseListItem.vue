<template>
  <div
    :class="[
      'flex items-center gap-3 px-3 py-3 cursor-pointer transition-all duration-[var(--duration-fast)] rounded-[var(--radius-md)] group',
      selected ? 'bg-selected-bg' : 'hover:bg-hover-bg',
    ]"
    @click="$emit('click', $event)"
  >
    <!-- 头像区域 -->
    <div v-if="$slots.avatar" class="flex-shrink-0">
      <slot name="avatar" />
    </div>

    <!-- 内容区域 -->
    <div class="flex-1 min-w-0">
      <slot />
    </div>

    <!-- 操作按钮区域 -->
    <div
      v-if="$slots.actions"
      class="flex items-center gap-1 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity duration-[var(--duration-fast)]"
    >
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineEmits<{
  click: [event: MouseEvent];
}>();

defineProps<{
  selected?: boolean;
}>();
</script>
