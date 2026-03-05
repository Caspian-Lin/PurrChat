<template>
  <Teleport to="body">
    <div class="fixed top-4 right-4 z-[9999] flex flex-col gap-2">
      <TransitionGroup name="message">
        <div
          v-for="msg in messages"
          :key="msg.id"
          :class="[
            'px-4 py-3 rounded-lg shadow-lg min-w-[200px] max-w-[400px]',
            msg.type === 'success' ? 'bg-green-500 text-white' : '',
            msg.type === 'error' ? 'bg-red-500 text-white' : '',
            msg.type === 'warning' ? 'bg-yellow-500 text-white' : '',
            msg.type === 'info' ? 'bg-blue-500 text-white' : '',
          ]"
        >
          {{ msg.content }}
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useMessage } from '../../composables/useMessage';

const { messages } = useMessage();
</script>

<style scoped>
.message-enter-active,
.message-leave-active {
  transition: all 0.3s ease;
}

.message-enter-from,
.message-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.message-leave-active {
  position: absolute;
  right: 0;
}
</style>
