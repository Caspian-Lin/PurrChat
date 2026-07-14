<template>
  <section>
    <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
      <BsBook :size="16" class="text-text-tertiary" />
      接入指南
    </h3>

    <!-- 快速接入 -->
    <div class="space-y-3 mb-6">
      <!-- 步骤 1：获取 Token -->
      <div class="flex gap-3">
        <div
          class="flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium"
          :style="{ background: 'var(--theme-primary)' }"
        >
          <span class="text-white">1</span>
        </div>
        <div class="flex-1 min-w-0 pt-0.5">
          <p class="text-sm text-text-primary">在上方「API Token」区域生成 Token</p>
        </div>
      </div>

      <!-- 步骤 2：连接 WebSocket -->
      <div class="flex gap-3">
        <div
          class="flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium"
          :style="{ background: 'var(--theme-primary)' }"
        >
          <span class="text-white">2</span>
        </div>
        <div class="flex-1 min-w-0 pt-0.5">
          <p class="text-sm text-text-primary mb-1">连接 WebSocket</p>
          <pre
            class="text-xs text-text-secondary bg-bg-quaternary rounded-[var(--radius-sm,8px)] p-2.5 overflow-x-auto"
          ><code>GET {{ wsEndpoint }}
Authorization: Bearer purr_bot_xxxx</code></pre>
        </div>
      </div>

      <!-- 步骤 3：收发事件 -->
      <div class="flex gap-3">
        <div
          class="flex-shrink-0 w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium"
          :style="{ background: 'var(--theme-primary)' }"
        >
          <span class="text-white">3</span>
        </div>
        <div class="flex-1 min-w-0 pt-0.5">
          <p class="text-sm text-text-primary mb-1">接收事件并调用 Action</p>
          <pre
            class="text-xs text-text-secondary bg-bg-quaternary rounded-[var(--radius-sm,8px)] p-2.5 overflow-x-auto"
          ><code>// 服务端推送事件
{{ eventExample }}

// 客户端调用 Action
{{ actionExample }}</code></pre>
        </div>
      </div>
    </div>

    <!-- HTTP Action 入口 -->
    <div class="mb-6 p-3 rounded-[var(--radius-sm,8px)] bg-bg-quaternary">
      <p class="text-xs text-text-secondary mb-1">HTTP Action 端点</p>
      <pre class="text-xs text-text-tertiary overflow-x-auto"><code>POST {{ httpEndpoint }}
Authorization: Bearer purr_bot_xxxx</code></pre>
    </div>

    <!-- 能力矩阵 -->
    <details class="rounded-[var(--radius-sm,8px)] bg-bg-quaternary">
      <summary class="cursor-pointer px-3 py-2.5 text-sm text-text-primary font-medium select-none">
        支持的 Actions & Events
        <span v-if="catalog" class="text-xs text-text-tertiary ml-1">
          ({{ catalog.actions.length }} Actions · {{ catalog.events.length }} Events)
        </span>
      </summary>
      <div class="px-3 pb-3 space-y-4">
        <div v-if="loading" class="text-xs text-text-quaternary py-4 text-center">加载中...</div>
        <div v-else-if="error" class="text-xs text-red-400 py-2">{{ error }}</div>
        <template v-else-if="catalog">
          <!-- Actions -->
          <div>
            <p class="text-xs font-medium text-text-secondary mb-2">Actions</p>
            <div class="space-y-1.5">
              <div
                v-for="action in catalog.actions.slice(0, 10)"
                :key="action.name"
                class="flex items-center gap-2 text-xs"
              >
                <code class="text-text-primary">{{ action.name }}</code>
                <span class="text-text-quaternary">{{ action.status }}</span>
                <span v-if="action.required_capability" class="text-text-quaternary">
                  · {{ action.required_capability }}
                </span>
              </div>
              <p v-if="catalog.actions.length > 10" class="text-xs text-text-quaternary">
                +{{ catalog.actions.length - 10 }} 更多…
              </p>
            </div>
          </div>
          <!-- Events -->
          <div>
            <p class="text-xs font-medium text-text-secondary mb-2">Events</p>
            <div class="space-y-1.5">
              <div
                v-for="event in catalog.events.slice(0, 10)"
                :key="`${event.post_type}-${event.detail_type}`"
                class="flex items-center gap-2 text-xs"
              >
                <code class="text-text-primary">{{ event.post_type }}.{{ event.detail_type }}</code>
                <span class="text-text-quaternary">{{ event.status }}</span>
              </div>
              <p v-if="catalog.events.length > 10" class="text-xs text-text-quaternary">
                +{{ catalog.events.length - 10 }} 更多…
              </p>
            </div>
          </div>
        </template>
      </div>
    </details>

    <!-- 完整文档链接 -->
    <a
      :href="developerUrl"
      target="_blank"
      class="mt-4 inline-flex items-center gap-1.5 text-xs text-[var(--theme-primary)] hover:underline"
    >
      <BsBoxArrowUpRight :size="12" />
      查看完整 API 能力矩阵
    </a>
  </section>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { BsBook, BsBoxArrowUpRight } from 'vue-icons-plus/bs';
import { api } from '../../../../models/api';
import { getApiBaseUrl } from '../../../../config/app';
import type { BotApiCapabilities } from '../../../../models/types';

const props = defineProps<{ botId: string }>();

const catalog = ref<BotApiCapabilities>();
const loading = ref(true);
const error = ref('');

const apiBase = getApiBaseUrl();
const wsEndpoint = computed(() => `${apiBase.replace(/^http/, 'ws')}/api/bot/v1/ws`);
const httpEndpoint = computed(() => `${apiBase}/api/bot/v1/actions/:action`);
const developerUrl = '/bot-studio/developer/api';

const eventExample = JSON.stringify(
  {
    post_type: 'message',
    detail_type: 'private',
    self: { platform: 'purrchat', user_id: props.botId },
    message: [{ type: 'text', data: { text: 'Hello' } }],
  },
  null,
  0
);

const actionExample = JSON.stringify(
  {
    action: 'send_message',
    params: {
      detail_type: 'private',
      user_id: 'target-user-id',
      message: [{ type: 'text', data: { text: 'Hi!' } }],
    },
  },
  null,
  0
);

onMounted(async () => {
  try {
    catalog.value = await api.getBotApiCapabilities();
  } catch {
    error.value = '无法加载能力矩阵';
  } finally {
    loading.value = false;
  }
});
</script>
