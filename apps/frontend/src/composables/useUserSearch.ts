import { ref, computed } from 'vue';
import { api } from '../models/api';
import type { User } from '../models/types';

export const useUserSearch = () => {
  const searchQuery = ref('');
  const searchResults = ref<User[]>([]);
  const showSearchResults = ref(false);

  /**
   * 搜索用户
   */
  const searchUsers = async () => {
    if (!searchQuery.value.trim()) return;

    try {
      const response = await api.searchUsers(searchQuery.value);
      if (response.success && response.data) {
        searchResults.value = response.data;
        showSearchResults.value = true;
      }
    } catch (error) {
      console.error('Failed to search users:', error);
    }
  };

  /**
   * 计算属性：按匹配字段分组搜索结果
   */
  const groupedResults = computed(() => {
    const query = searchQuery.value.toLowerCase();
    const uidMatches: User[] = [];
    const emailMatches: User[] = [];
    const phoneMatches: User[] = [];

    for (const user of searchResults.value) {
      const uidStr = user.uid.toString().toLowerCase();
      const email = user.email?.toLowerCase() || '';
      const phone = user.phone?.toLowerCase() || '';

      // 检查UID是否匹配
      if (uidStr.includes(query)) {
        uidMatches.push(user);
      }
      // 检查邮箱是否匹配
      if (email.includes(query)) {
        emailMatches.push(user);
      }
      // 检查手机号是否匹配
      if (phone.includes(query)) {
        phoneMatches.push(user);
      }
    }

    return {
      uidMatches,
      emailMatches,
      phoneMatches,
    };
  });

  /**
   * 处理搜索框失焦事件
   */
  const handleSearchBlur = () => {
    // 延迟关闭，以便点击搜索结果
    setTimeout(() => {
      showSearchResults.value = false;
    }, 200);
  };

  /**
   * 高亮匹配的文本
   * @param text - 原始文本
   * @param query - 搜索查询
   * @returns 带高亮标记的HTML字符串
   */
  const highlightMatch = (text: string, query: string): string => {
    if (!text || !query) return text;
    const regex = new RegExp(`(${query})`, 'gi');
    return text.replace(
      regex,
      '<span class="bg-yellow-300 dark:bg-yellow-600 text-black dark:text-white px-1 rounded">$1</span>'
    );
  };

  /**
   * 清空搜索结果
   */
  const clearSearchResults = () => {
    searchResults.value = [];
    searchQuery.value = '';
    showSearchResults.value = false;
  };

  return {
    searchQuery,
    searchResults,
    showSearchResults,
    groupedResults,
    searchUsers,
    handleSearchBlur,
    highlightMatch,
    clearSearchResults,
  };
};
