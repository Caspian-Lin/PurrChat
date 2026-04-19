<template>
  <BasePanel
    panel-id="bots"
    :initial-sidebar-width="300"
    :min-sidebar-width="220"
    :max-sidebar-width="420"
  >
    <template #sidebar>
      <div class="flex flex-col h-full">
        <!-- 顶部操作栏 -->
        <div
          class="flex items-center gap-2 px-3 pt-5 pb-3 bg-bg-secondary border-b border-border-subtle flex-shrink-0"
        >
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="新建 Bot"
            @click="showCreateModal = true"
          >
            <BsPlusLg :size="20" />
          </button>
          <button
            class="relative p-2 flex items-center justify-center hover:bg-hover-bg transition-colors text-text-tertiary hover:text-text-primary"
            title="搜索公开 Bot"
            @click="showSearch = !showSearch"
          >
            <BsSearch :size="18" />
          </button>
        </div>

        <!-- 搜索栏 -->
        <div v-if="showSearch" class="px-3 py-2 border-b border-border-subtle flex-shrink-0">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索公开 Bot..."
            class="w-full px-3 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all"
            @input="handleSearch"
          />
        </div>

        <!-- Bot 列表 -->
        <div class="flex-1 min-h-0">
          <BotList
            :bots="searchQuery ? botStore.searchResults : botStore.bots"
            :active-bot-id="botStore.activeBotId"
            :loading="botStore.loading || botStore.searchLoading"
            :is-search="!!searchQuery"
            :has-more="botStore.searchHasMore"
            @select="handleSelectBot"
            @delete="handleDeleteBot"
            @create-conversation="handleCreateConversation"
            @deploy-to-group="handleDeployToGroup"
            @load-more="handleLoadMore"
          />
        </div>
      </div>
    </template>

    <!-- 编辑器 -->
    <BotEditor
      v-if="botStore.activeBot"
      :key="botStore.activeBotId"
      :bot="botStore.activeBot"
      @update="handleUpdateBot"
      @back="botStore.setActiveBot(null)"
    />

    <!-- 空状态 -->
    <div v-else class="flex-1 flex items-center justify-center">
      <div class="text-center space-y-3">
        <BsCpu :size="48" class="mx-auto text-text-quaternary" />
        <h3 class="text-lg font-medium text-text-secondary">Bot Studio</h3>
        <p class="text-sm text-text-tertiary max-w-[280px]">
          创建和管理你的 Bot，配置触发规则和回复方式，部署到对话中。
        </p>
        <button
          class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-[var(--radius-sm,8px)] text-white transition-colors"
          style="background: var(--theme-primary)"
          @click="showCreateModal = true"
        >
          <BsPlusLg :size="16" />
          创建 Bot
        </button>
      </div>
    </div>
  </BasePanel>

  <!-- 创建 Bot 弹窗 -->
  <CreateBotModal
    v-if="showCreateModal"
    @create="handleCreateBot"
    @close="showCreateModal = false"
  />

  <!-- 部署到群聊弹窗 -->
  <DeployToGroupModal
    v-if="deployToGroupBotId"
    :bot-id="deployToGroupBotId"
    @deployed="() => {}"
    @close="deployToGroupBotId = null"
  />
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { BsPlusLg, BsSearch, BsCpu } from 'vue-icons-plus/bs';
import BasePanel from './BasePanel.vue';
import BotList from './bots/BotList.vue';
import BotEditor from './bots/BotEditor.vue';
import CreateBotModal from './bots/CreateBotModal.vue';
import DeployToGroupModal from './bots/DeployToGroupModal.vue';
import { useBotStore } from '../../../stores/bot';
import { useBots } from '../../../composables/useBots';
import { useRouter } from 'vue-router';

const botStore = useBotStore();
const { createBot, deleteBot, updateBot, createBotConversation } = useBots();
const router = useRouter();

const showCreateModal = ref(false);
const showSearch = ref(false);
const searchQuery = ref('');
const deployToGroupBotId = ref<string | null>(null);

onMounted(async () => {
  await botStore.loadBots();
});

async function handleCreateBot(data: { name: string; description?: string }) {
  const bot = await createBot({
    name: data.name,
    description: data.description || '',
  });
  if (bot) {
    showCreateModal.value = false;
    botStore.setActiveBot(bot.id);
  }
}

async function handleDeleteBot(botId: string) {
  await deleteBot(botId);
}

async function handleUpdateBot(botId: string, data: any) {
  await updateBot(botId, data);
}

function handleSelectBot(botId: string) {
  botStore.setActiveBot(botId);
}

async function handleCreateConversation(botId: string) {
  const conversation = await createBotConversation(botId);
  if (conversation) {
    router.push('/chat');
  }
}

function handleDeployToGroup(botId: string) {
  deployToGroupBotId.value = botId;
}

async function handleSearch() {
  await botStore.searchPublicBots(searchQuery.value);
}

async function handleLoadMore() {
  await botStore.loadMoreSearchResults(searchQuery.value);
}
</script>
