<template>
  <div class="p-4 flex items-center gap-2 relative" style="border-color: var(--border-color)">
    <BaseInput
      v-model="searchQuery"
      placeholder="搜索用户 (UID/手机号/邮箱)"
      @keyup.enter="searchUsers"
      @focus="showSearchResults = true"
      @blur="handleSearchBlur"
      class="flex-1"
    />
    <BaseButton type="primary" @click="searchUsers"> 搜索 </BaseButton>

    <!-- 搜索结果弹窗 -->
    <div
      v-if="showSearchResults && searchResults.length > 0"
      class="absolute top-full left-4 right-4 mt-1 z-50 rounded-lg shadow-xl max-h-96 overflow-y-auto"
      style="background: var(--card-background); border: 1px solid var(--border-color)"
    >
      <!-- 按UID匹配的用户 -->
      <div
        v-if="groupedResults.uidMatches.length > 0"
        class="border-b"
        style="border-color: var(--border-color)"
      >
        <div class="px-4 py-2 font-semibold text-sm" style="color: var(--text-secondary-color)">
          UID 匹配
        </div>
        <div
          v-for="user in groupedResults.uidMatches"
          :key="user.id"
          class="flex items-center gap-4 p-4 cursor-pointer transition-colors hover:bg-hover-bg"
          @click="$emit('select-user', user)"
        >
          <div class="w-10 h-10 roundrect overflow-hidden flex-shrink-0">
            <img
              v-if="user.avatar_url"
              :src="user.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
              style="background: var(--theme-gradient)"
            >
              {{ user.username?.charAt(0) || 'U' }}
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-semibold truncate" style="color: var(--text-color)">
              {{ user.username }}
            </div>
            <div class="text-sm" style="color: var(--text-secondary-color)">
              UID: <span v-html="highlightMatch(user.uid.toString(), searchQuery)"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- 按邮箱匹配的用户 -->
      <div
        v-if="groupedResults.emailMatches.length > 0"
        class="border-b"
        style="border-color: var(--border-color)"
      >
        <div class="px-4 py-2 font-semibold text-sm" style="color: var(--text-secondary-color)">
          邮箱匹配
        </div>
        <div
          v-for="user in groupedResults.emailMatches"
          :key="user.id"
          class="flex items-center gap-4 p-4 cursor-pointer transition-colors hover:bg-hover-bg"
          @click="$emit('select-user', user)"
        >
          <div class="w-10 h-10 roundrect overflow-hidden flex-shrink-0">
            <img
              v-if="user.avatar_url"
              :src="user.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
              style="background: var(--theme-gradient)"
            >
              {{ user.username?.charAt(0) || 'U' }}
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-semibold truncate" style="color: var(--text-color)">
              {{ user.username }}
            </div>
            <div class="text-sm truncate" style="color: var(--text-secondary-color)">
              <span v-html="highlightMatch(user.email || '', searchQuery)"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- 按手机号匹配的用户 -->
      <div v-if="groupedResults.phoneMatches.length > 0">
        <div class="px-4 py-2 font-semibold text-sm" style="color: var(--text-secondary-color)">
          手机号匹配
        </div>
        <div
          v-for="user in groupedResults.phoneMatches"
          :key="user.id"
          class="flex items-center gap-4 p-4 cursor-pointer transition-colors hover:bg-hover-bg"
          @click="$emit('select-user', user)"
        >
          <div class="w-10 h-10 roundrect overflow-hidden flex-shrink-0">
            <img
              v-if="user.avatar_url"
              :src="user.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-sm"
              style="background: var(--theme-gradient)"
            >
              {{ user.username?.charAt(0) || 'U' }}
            </div>
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-semibold truncate" style="color: var(--text-color)">
              {{ user.username }}
            </div>
            <div class="text-sm" style="color: var(--text-secondary-color)">
              <span v-html="highlightMatch(user.phone || '', searchQuery)"></span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import BaseButton from '../common/BaseButton.vue';
import BaseInput from '../common/BaseInput.vue';
import { useUserSearch } from '../../composables/useUserSearch';
import type { User } from '../../models/types';

const {
  searchQuery,
  searchResults,
  showSearchResults,
  groupedResults,
  searchUsers,
  handleSearchBlur,
  highlightMatch,
} = useUserSearch();

defineEmits<{
  'select-user': [user: User];
}>();
</script>
