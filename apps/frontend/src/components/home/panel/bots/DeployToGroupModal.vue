<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center" @click.self="$emit('close')">
    <!-- 遮罩 -->
    <div class="absolute inset-0 bg-black/30 backdrop-blur-sm" />

    <!-- 弹窗 -->
    <div
      class="relative w-full max-w-md mx-4 bg-bg-primary rounded-[var(--radius-lg,16px)] shadow-lg overflow-hidden"
    >
      <!-- 头部 -->
      <div class="flex items-center justify-between px-6 py-4 border-b border-border-subtle">
        <h2 class="text-base font-semibold text-text-primary">安装到群聊</h2>
        <button
          class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
          @click="$emit('close')"
        >
          <BsX :size="18" />
        </button>
      </div>

      <!-- 内容 -->
      <div class="px-6 py-4">
        <!-- 加载中 -->
        <div v-if="loading" class="flex items-center justify-center py-8">
          <div
            class="w-6 h-6 border-2 border-text-quaternary border-t-[var(--theme-primary)] rounded-full animate-spin"
          />
        </div>

        <template v-else>
          <!-- 已安装的群聊 -->
          <div v-if="installedGroups.length > 0" class="mb-4">
            <p class="text-xs font-medium text-text-tertiary mb-2">已安装</p>
            <div class="space-y-1">
              <div
                v-for="dep in installedGroups"
                :key="dep.id"
                class="flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-sm,8px)] bg-bg-secondary"
              >
                <div class="flex-1 min-w-0">
                  <div class="text-sm text-text-primary truncate">
                    {{ dep.target_name || dep.target_id }}
                  </div>
                </div>
                <button
                  class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-red-500 transition-colors"
                  title="卸载"
                  :disabled="uninstalling === dep.target_id"
                  @click="handleUndeploy(dep.target_id)"
                >
                  <div
                    v-if="uninstalling === dep.target_id"
                    class="w-3.5 h-3.5 border-2 border-text-quaternary border-t-[var(--theme-primary)] rounded-full animate-spin"
                  />
                  <BsTrash v-else :size="14" />
                </button>
              </div>
            </div>
          </div>

          <!-- 可安装的群聊 -->
          <div v-if="conversations.length > 0">
            <p
              v-if="installedGroups.length > 0"
              class="text-xs font-medium text-text-tertiary mb-2"
            >
              可安装
            </p>
            <div class="space-y-1 max-h-[260px] overflow-y-auto">
              <button
                v-for="conv in conversations"
                :key="conv.id"
                class="w-full flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors text-left"
                :disabled="deploying === conv.id"
                @click="handleDeploy(conv.id)"
              >
                <div
                  class="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 text-white text-xs font-bold"
                  style="background: var(--theme-primary)"
                >
                  {{ conv.name.charAt(0) }}
                </div>
                <div class="flex-1 min-w-0">
                  <div class="text-sm text-text-primary truncate">{{ conv.name }}</div>
                  <div class="text-xs text-text-tertiary">{{ conv.member_count }} 位成员</div>
                </div>
                <div
                  v-if="deploying === conv.id"
                  class="w-4 h-4 border-2 border-text-quaternary border-t-[var(--theme-primary)] rounded-full animate-spin flex-shrink-0"
                />
                <BsCheckLg
                  v-else-if="deployed === conv.id"
                  :size="16"
                  class="text-green-500 flex-shrink-0"
                />
              </button>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else-if="installedGroups.length === 0" class="text-center py-8">
            <BsPeopleFill :size="32" class="mx-auto text-text-quaternary mb-3" />
            <p class="text-sm text-text-tertiary">没有可安装的群聊</p>
            <p class="text-xs text-text-quaternary mt-1">
              Bot 已安装到所有你的群聊，或你还没有加入群聊
            </p>
          </div>
        </template>
      </div>

      <!-- 底部 -->
      <div class="flex justify-end px-6 py-4 border-t border-border-subtle">
        <button
          class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-secondary hover:bg-hover-bg transition-colors"
          @click="$emit('close')"
        >
          关闭
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { BsX, BsPeopleFill, BsCheckLg, BsTrash } from 'vue-icons-plus/bs';
import { useBots } from '../../../../composables/useBots';
import { useBotStore } from '../../../../stores/bot';
import type { DeployableConversation } from '../../../../models/types';

interface Props {
  botId: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  deployed: [conversationId: string];
  close: [];
}>();

const { deployBot, undeployBot, getDeployableConversations } = useBots();
const botStore = useBotStore();

const loading = ref(true);
const conversations = ref<DeployableConversation[]>([]);
const deploying = ref<string | null>(null);
const deployed = ref<string | null>(null);
const uninstalling = ref<string | null>(null);

const installedGroups = computed(() =>
  botStore.deployments.filter((d) => d.app_id === props.botId && d.target_type === 'conversation')
);

onMounted(async () => {
  await botStore.loadDeployments();
  conversations.value = await getDeployableConversations(props.botId);
  loading.value = false;
});

async function handleDeploy(conversationId: string) {
  deploying.value = conversationId;
  const success = await deployBot(props.botId, conversationId);
  if (success) {
    deployed.value = conversationId;
    conversations.value = conversations.value.filter((c) => c.id !== conversationId);
    emit('deployed', conversationId);
  }
  deploying.value = null;
}

async function handleUndeploy(conversationId: string) {
  uninstalling.value = conversationId;
  await undeployBot(props.botId, conversationId);
  uninstalling.value = null;
}
</script>
