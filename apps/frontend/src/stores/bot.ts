import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Bot, BotDeployment, PublicBotDetail } from '../models/types';
import { api } from '../models/api';

export const useBotStore = defineStore('bot', () => {
  // 状态
  const bots = ref<Bot[]>([]);
  const deployments = ref<BotDeployment[]>([]);
  const activeBotId = ref<string | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // 搜索分页状态
  const searchResults = ref<PublicBotDetail[]>([]);
  const searchTotal = ref(0);
  const searchOffset = ref(0);
  const searchHasMore = ref(false);
  const searchLoading = ref(false);

  // 计算属性
  const activeBot = computed(() => bots.value.find((b) => b.id === activeBotId.value) ?? null);
  const activeBots = computed(() => bots.value.filter((b) => b.status === 'active'));

  // 加载 Bot 列表
  async function loadBots() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getBots();
      if (response.success && response.data) {
        bots.value = response.data;
      }
    } catch (err) {
      console.error('[bot store] 加载 Bot 列表失败:', err);
      error.value = '加载 Bot 列表失败';
    } finally {
      loading.value = false;
    }
  }

  // 加载部署列表
  async function loadDeployments() {
    try {
      const response = await api.getBotDeployments();
      if (response.success && response.data) {
        deployments.value = response.data;
      }
    } catch (err) {
      console.error('[bot store] 加载部署列表失败:', err);
    }
  }

  // 搜索公开 Bot（首次搜索，重置分页）
  async function searchPublicBots(query: string) {
    if (!query.trim()) {
      searchResults.value = [];
      searchTotal.value = 0;
      searchHasMore.value = false;
      return;
    }

    searchLoading.value = true;
    searchOffset.value = 0;

    try {
      const response = await api.searchBots(query, 20, 0);
      if (response.success && response.data) {
        searchResults.value = response.data.bots;
        searchTotal.value = response.data.total;
        searchHasMore.value =
          response.data.offset + response.data.bots.length < response.data.total;
      }
    } catch (err) {
      console.error('[bot store] 搜索 Bot 失败:', err);
      searchResults.value = [];
    } finally {
      searchLoading.value = false;
    }
  }

  // 加载更多搜索结果
  async function loadMoreSearchResults(query: string) {
    if (!searchHasMore.value || searchLoading.value) return;

    searchLoading.value = true;
    const nextOffset = searchOffset.value + 20;

    try {
      const response = await api.searchBots(query, 20, nextOffset);
      if (response.success && response.data) {
        searchResults.value = [...searchResults.value, ...response.data.bots];
        searchOffset.value = nextOffset;
        searchHasMore.value =
          response.data.offset + response.data.bots.length < response.data.total;
      }
    } catch (err) {
      console.error('[bot store] 加载更多搜索结果失败:', err);
    } finally {
      searchLoading.value = false;
    }
  }

  // 设置活跃 Bot
  function setActiveBot(botId: string | null) {
    activeBotId.value = botId;
  }

  // 清除错误
  function clearError() {
    error.value = null;
  }

  // 重置所有状态（用户切换时调用）
  function reset() {
    bots.value = [];
    deployments.value = [];
    activeBotId.value = null;
    loading.value = false;
    error.value = null;
    searchResults.value = [];
    searchTotal.value = 0;
    searchOffset.value = 0;
    searchHasMore.value = false;
    searchLoading.value = false;
  }

  return {
    bots,
    deployments,
    activeBotId,
    activeBot,
    activeBots,
    loading,
    error,
    searchResults,
    searchTotal,
    searchOffset,
    searchHasMore,
    searchLoading,
    loadBots,
    loadDeployments,
    searchPublicBots,
    loadMoreSearchResults,
    setActiveBot,
    clearError,
    reset,
  };
});
