<template>
  <Teleport to="body">
    <Transition name="preview">
      <div
        v-if="show"
        class="fixed inset-0 z-50 flex items-center justify-center"
        style="background: rgba(0, 0, 0, 0.85)"
        @click.self="handleClose"
        @keydown.esc="handleClose"
        tabindex="0"
      >
        <!-- 关闭按钮 -->
        <button
          class="absolute top-4 right-4 z-10 p-2 rounded-full hover:bg-white/20 text-white transition-colors"
          @click="handleClose"
        >
          <BsX class="text-2xl" />
        </button>

        <!-- 图片容器 -->
        <div class="max-w-[90vw] max-h-[85vh] flex flex-col items-center">
          <img
            :src="imageUrl"
            :alt="fileName"
            class="max-w-full max-h-[80vh] object-contain rounded-lg select-none"
            draggable="false"
          />

          <!-- 底部工具栏 -->
          <div class="flex items-center gap-4 mt-4">
            <span class="text-white/60 text-sm truncate max-w-[300px]">{{ fileName }}</span>
            <button
              class="flex items-center gap-2 px-4 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white text-sm transition-colors"
              @click="emit('download')"
            >
              <BsDownload class="text-lg" />
              下载
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { watch } from 'vue';
import { BsX, BsDownload } from 'vue-icons-plus/bs';

interface Props {
  show: boolean;
  imageUrl: string;
  fileName: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  download: [];
}>();

const handleClose = () => {
  emit('update:show', false);
};

// 打开时聚焦以支持 ESC 键关闭
watch(
  () => props.show,
  (val) => {
    if (val) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
  }
);
</script>

<style scoped>
.preview-enter-active,
.preview-leave-active {
  transition: opacity 0.2s ease;
}
.preview-enter-from,
.preview-leave-to {
  opacity: 0;
}
</style>
