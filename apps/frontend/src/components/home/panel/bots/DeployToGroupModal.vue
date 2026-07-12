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
        <h2 class="text-base font-semibold text-text-primary">安装到会话</h2>
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

        <!-- 会话列表 -->
        <div v-else-if="conversations.length > 0" class="space-y-1 max-h-[300px] overflow-y-auto">
          <button
            v-for="conv in conversations"
            :key="conv.id"
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-sm,8px)] hover:bg-hover-bg transition-colors text-left group"
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
              <div class="text-xs text-text-tertiary">
                <span v-if="conv.conversation_type === 'direct'">好友私聊</span>
                <span v-else>{{ conv.member_count }} 位成员</span>
              </div>
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

        <!-- 空状态 -->
        <div v-else class="text-center py-8">
          <BsPeopleFill :size="32" class="mx-auto text-text-quaternary mb-3" />
          <p class="text-sm text-text-tertiary">没有可安装的会话</p>
          <p class="text-xs text-text-quaternary mt-1">
            Bot 已安装到所有你的会话，或你还没有任何会话
          </p>
        </div>
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
import { ref, onMounted } from 'vue';
import { BsX, BsPeopleFill, BsCheckLg } from 'vue-icons-plus/bs';
import { useBots } from '../../../../composables/useBots';
import type { DeployableConversation } from '../../../../models/types';

interface Props {
  botId: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  deployed: [conversationId: string];
  close: [];
}>();

const { deployBot, getDeployableConversations } = useBots();

const loading = ref(true);
const conversations = ref<DeployableConversation[]>([]);
const deploying = ref<string | null>(null);
const deployed = ref<string | null>(null);

onMounted(async () => {
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
</script>
