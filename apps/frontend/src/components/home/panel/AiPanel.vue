<template>
  <BasePanel panel-id="ai" :initial-sidebar-width="280" :min-sidebar-width="220" :max-sidebar-width="400">
    <template #sidebar>
      <div class="flex flex-col h-full">
        <!-- 顶部操作栏 -->
        <div
          class="flex items-center gap-2 px-3 pt-5 pb-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
        >
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-primary hover:text-text-primary"
            title="新建对话"
            @click="handleNewConversation"
          >
            <BsPlusLg :size="20" />
          </button>
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="管理 AI 配置"
            @click="handleAddConfig"
          >
            <BsGear :size="20" />
          </button>
        </div>

        <!-- 配置列表和会话列表 -->
        <div class="flex-1 min-h-0">
          <AiConfigList
            :configs="aiStore.configs"
            :active-config-id="aiStore.activeConfigId"
            :conversations="currentConversations"
            :active-conversation-id="aiStore.activeConversationId"
            @select-config="handleSelectConfig"
            @edit-config="handleEditConfig"
            @delete-config="handleDeleteConfig"
            @select-conversation="handleSelectConversation"
            @delete-conversation="handleDeleteConversation"
          />
        </div>
      </div>
    </template>

    <!-- 对话窗口 -->
    <AiChatWindow
      v-if="aiStore.activeConversation && aiStore.activeConfig"
      ref="chatWindowRef"
      :config="aiStore.activeConfig"
      :conversation="aiStore.activeConversation"
      :messages="aiStore.activeMessages"
      :is-streaming="isStreaming"
      :error="error"
      @send-message="handleSendMessage"
      @stop-generation="stopGeneration"
      @clear-error="clearError"
    />

    <!-- 未选择配置的空状态 -->
    <div
      v-else-if="!aiStore.hasConfigs"
      class="flex-1 flex flex-col items-center justify-center text-text-tertiary gap-4"
    >
      <BsRobot :size="64" class="opacity-30" />
      <h3 class="text-2xl font-semibold text-text-primary">开始使用 AI 对话</h3>
      <p>请先添加一个 AI 模型配置</p>
      <button
        class="px-6 py-2 bg-[var(--theme-primary)] hover:opacity-80 transition-opacity text-white font-semibold rounded-md"
        @click="handleAddConfig"
      >
        添加配置
      </button>
    </div>

    <!-- 未创建对话的空状态 -->
    <div v-else class="flex-1 flex flex-col items-center justify-center text-text-tertiary gap-4">
      <BsChatLeftText :size="64" class="opacity-30" />
      <h3 class="text-2xl font-semibold text-text-primary">AI 对话</h3>
      <p>选择一个配置并创建新对话</p>
      <button
        class="px-6 py-2 bg-[var(--theme-primary)] hover:opacity-80 transition-opacity text-white font-semibold rounded-md"
        @click="handleNewConversation"
      >
        新建对话
      </button>
    </div>
  </BasePanel>

  <!-- 配置弹窗 -->
  <AiConfigModal
    v-model:show="showConfigModal"
    :editing-config="editingConfig"
    @config-saved="handleConfigSaved"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue';
import { useAiStore } from '../../../stores/ai';
import { useAiChat } from '../../../composables/useAiChat';
import { useAuthStore } from '../../../stores/auth';
import AiConfigList from '../AiConfigList.vue';
import AiChatWindow from '../AiChatWindow.vue';
import AiConfigModal from '../AiConfigModal.vue';
import BasePanel from './BasePanel.vue';
import type { AiConfig } from '../../../models/types';
import { BsPlusLg, BsGear, BsRobot, BsChatLeftText } from 'vue-icons-plus/bs';

const aiStore = useAiStore();
const authStore = useAuthStore();
const { isStreaming, error, sendMessage, stopGeneration, clearError } = useAiChat();

const chatWindowRef = ref<InstanceType<typeof AiChatWindow> | null>(null);
const showConfigModal = ref(false);
const editingConfig = ref<AiConfig | null>(null);

// 当前配置下的会话列表
const currentConversations = computed(() => {
  if (!aiStore.activeConfigId) return [];
  return aiStore.conversations.filter((c) => c.configId === aiStore.activeConfigId);
});

// 选择配置
const handleSelectConfig = (configId: string) => {
  aiStore.setActiveConfig(configId);
  // 切换配置后，如果没有激活的会话或会话不属于新配置，取消选择
  if (aiStore.activeConversation && aiStore.activeConversation.configId !== configId) {
    aiStore.setActiveConversation(null);
  }
  // 如果新配置有会话，自动选中第一个
  const convs = aiStore.conversations.filter((c) => c.configId === configId);
  if (convs.length > 0 && !aiStore.activeConversation) {
    aiStore.setActiveConversation(convs[0]!.id);
  }
};

// 新建对话
const handleNewConversation = () => {
  if (!aiStore.activeConfig) {
    handleAddConfig();
    return;
  }
  aiStore.createConversation(aiStore.activeConfig.id);
};

// 选择会话
const handleSelectConversation = (conversationId: string) => {
  aiStore.setActiveConversation(conversationId);
};

// 删除会话
const handleDeleteConversation = (conversationId: string) => {
  aiStore.deleteConversation(conversationId);
};

// 添加配置
const handleAddConfig = () => {
  editingConfig.value = null;
  showConfigModal.value = true;
};

// 编辑配置
const handleEditConfig = (config: AiConfig) => {
  editingConfig.value = config;
  showConfigModal.value = true;
};

// 删除配置
const handleDeleteConfig = (configId: string) => {
  aiStore.deleteConfig(configId);
};

// 配置保存完成
const handleConfigSaved = (data: {
  name: string;
  apiUrl: string;
  apiKey: string;
  model: string;
  temperature: number;
  maxTokens?: number;
}) => {
  if (editingConfig.value) {
    aiStore.updateConfig(editingConfig.value.id, data);
  } else {
    const newConfig = aiStore.addConfig(data);
    if (!aiStore.activeConfigId) {
      aiStore.setActiveConfig(newConfig.id);
    }
  }
  showConfigModal.value = false;
  editingConfig.value = null;
};

// 获取表单数据（从 AiConfigModal 的内部表单获取）
// 由于 AiConfigModal 使用内部 form state，我们通过 editingConfig + ref 获取
// 这里需要改为从 modal 组件中获取表单数据

// 发送消息
const handleSendMessage = async (content: string) => {
  await sendMessage(content);
};

// 监听错误变化，滚动到底部显示错误
watch(error, () => {
  if (error.value) {
    // 滚动到底部
    chatWindowRef.value?.scrollToBottom();
  }
});

// 初始化
onMounted(() => {
  aiStore.initStore(authStore.currentUser?.id);
  // 首次使用且无配置时，自动弹出引导
  if (!aiStore.hasConfigs) {
    showConfigModal.value = true;
  }
});
</script>

<style scoped></style>
