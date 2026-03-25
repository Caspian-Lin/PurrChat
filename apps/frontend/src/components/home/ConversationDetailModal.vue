<template>
  <BaseModal
    :show="show"
    :title="conversationName"
    class="max-w-md"
    @update:show="emit('update:show', $event)"
  >
    <div class="flex flex-col gap-4">
      <!-- 会话信息 -->
      <div class="p-4 rounded-lg" style="background: var(--surface-color)">
        <div class="flex items-center gap-4">
          <div
            v-if="conversation?.conversation_type === 'group'"
            class="w-16 h-16 roundrect overflow-hidden flex-shrink-0 flex items-center justify-center font-bold text-white text-3xl"
            style="background: var(--theme-gradient)"
          >
            {{ conversation?.name?.charAt(0) || 'G' }}
          </div>
          <div
            v-else-if="otherUser"
            class="w-16 h-16 roundrect overflow-hidden flex-shrink-0 cursor-pointer"
            @click="handleShowUserProfile(otherUser)"
          >
            <img
              v-if="getUserAvatar(otherUser)"
              :src="getUserAvatar(otherUser)"
              alt="avatar"
              class="w-full h-full object-cover"
            />
            <div
              v-else
              class="w-full h-full flex items-center justify-center font-bold text-white text-3xl"
              style="background: var(--theme-gradient)"
            >
              {{ getUserUsername(otherUser)?.charAt(0) || 'U' }}
            </div>
          </div>
          <div class="flex-1">
            <div class="font-semibold text-lg" style="color: var(--text-color)">
              {{ conversationName }}
            </div>
            <div class="flex items-center gap-2 mt-1">
              <div class="w-[12px] h-[12px] rounded-full bg-accent-color" />
              <div class="text-sm" style="color: var(--text-secondary-color)">
                {{ conversation?.conversation_type === 'group' ? 'GID' : 'CID' }}:
                {{ conversation?.id }}
              </div>
            </div>
            <div
              v-if="conversation?.conversation_type === 'group'"
              class="text-sm"
              style="color: var(--text-secondary-color)"
            >
              {{ members.length }} 位成员
            </div>
            <div v-else class="text-sm" style="color: var(--text-secondary-color)">私聊</div>
          </div>
        </div>

        <!-- 群聊额外信息（仅群聊显示） -->
        <div
          v-if="conversation?.conversation_type === 'group'"
          class="mt-4 p-4 rounded-lg"
          style="background: var(--surface-color)"
        >
          <div class="flex items-center gap-2 mb-3">
            <div class="w-[12px] h-[12px] rounded-full bg-accent-color" />
            <div class="text-sm font-medium" style="color: var(--text-secondary-color)">群主</div>
            <div class="text-sm" style="color: var(--text-color)">
              {{ groupOwner?.user?.username || '未知' }}
            </div>
          </div>
        </div>
      </div>

      <!-- 成员列表（仅群聊显示） -->
      <div v-if="conversation?.conversation_type === 'group'">
        <div class="flex items-center justify-between mb-2">
          <label class="text-sm font-medium" style="color: var(--text-color)"> 成员列表 </label>
          <button
            v-if="canManageMembers"
            class="px-3 py-1.5 text-sm bg-primary text-white rounded-md hover:bg-primary-dark transition-colors"
            @click="showAddMemberModal = true"
          >
            添加成员
          </button>
        </div>
        <CustomScrollbar class="max-h-64 rounded-lg" style="background: var(--surface-color)">
          <div class="h-full">
            <div
              v-for="member in members"
              :key="member.id"
              class="flex items-center gap-3 p-3 border-b"
              style="border-color: var(--border-color)"
            >
              <div
                class="w-10 h-10 roundrect overflow-hidden flex-shrink-0 cursor-pointer"
                @click="handleShowUserProfile(member.user)"
              >
                <img
                  v-if="member.user?.avatar_url"
                  :src="member.user.avatar_url"
                  alt="avatar"
                  class="w-full h-full object-cover"
                />
                <div
                  v-else
                  class="w-full h-full flex items-center justify-center font-bold text-white"
                  style="background: var(--theme-gradient)"
                >
                  {{ member.user?.username?.charAt(0) || 'U' }}
                </div>
              </div>
              <div class="flex-1">
                <div class="font-medium" style="color: var(--text-color)">
                  {{ member.user?.username }}
                </div>
                <div class="text-xs" style="color: var(--text-secondary-color)">
                  {{ getRoleLabel(member.role) }}
                </div>
              </div>
              <button
                v-if="canRemoveMember(member)"
                class="px-3 py-1.5 text-sm bg-red-500 text-white rounded-md hover:bg-red-600 transition-colors"
                @click="handleRemoveMember(member)"
              >
                移除
              </button>
            </div>
          </div>
        </CustomScrollbar>
      </div>

      <!-- 发送消息按钮 -->
      <div>
        <button
          class="w-full px-4 py-2 bg-primary text-white rounded-md hover:bg-primary-dark transition-colors"
          @click="handleSendMessage"
        >
          发送消息
        </button>
      </div>

      <!-- 导出消息 -->
      <div>
        <button
          class="w-full px-4 py-2 bg-bg-secondary text-text-primary rounded-md hover:bg-hover-bg transition-colors"
          @click="handleExportMessages"
        >
          导出历史消息
        </button>
      </div>

      <!-- 关闭按钮 -->
      <div>
        <button
          class="w-full px-4 py-2 bg-bg-secondary text-text-primary rounded-md hover:bg-hover-bg transition-colors"
          @click="emit('update:show', false)"
        >
          关闭
        </button>
      </div>
    </div>

    <!-- 添加成员modal（仅群聊） -->
    <AddMemberModal
      v-if="showAddMemberModal && conversation?.conversation_type === 'group'"
      :show="showAddMemberModal"
      :conversation-id="conversation?.id || ''"
      :current-members="members"
      @update:show="showAddMemberModal = $event"
      @member-added="handleMemberAdded"
    />
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import BaseModal from '../common/BaseModal.vue';
import AddMemberModal from './AddMemberModal.vue';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import { api } from '../../models/api';
import { getUserAvatar, getUserUsername, getOtherUser } from '../../utils/userHelpers';
import type { Conversation, Enrollment } from '../../models/types';

interface Props {
  show: boolean;
  conversation: Conversation | null;
  currentUserId: string | undefined;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  'show-user-profile': [user: any];
  'members-changed': [];
  'start-chat': [conversation: Conversation];
}>();

const showAddMemberModal = ref(false);
const members = ref<Enrollment[]>([]);

// 获取会话名称
const conversationName = computed(() => {
  if (!props.conversation) return '';
  if (props.conversation.conversation_type === 'group') {
    return props.conversation.name || '群聊';
  }
  return getUserUsername(getOtherUser(props.conversation, props.currentUserId)) || '私聊';
});

// 获取私聊的对方用户
const otherUser = computed(() => {
  if (!props.conversation || props.conversation.conversation_type === 'group') {
    return null;
  }
  return getOtherUser(props.conversation, props.currentUserId);
});

// 检查当前用户是否可以管理成员（仅群聊）
const canManageMembers = computed(() => {
  if (
    !props.currentUserId ||
    !props.conversation ||
    props.conversation.conversation_type !== 'group'
  ) {
    return false;
  }

  const currentMember = members.value.find((m) => m.user_id === props.currentUserId);
  return currentMember?.role === 'owner' || currentMember?.role === 'admin';
});

// 获取群主（仅群聊）
const groupOwner = computed(() => {
  if (!props.conversation || props.conversation.conversation_type !== 'group') {
    return null;
  }
  return members.value.find((m) => m.role === 'owner') || null;
});

// 检查是否可以移除某个成员（仅群聊）
const canRemoveMember = (member: Enrollment) => {
  if (!canManageMembers.value) {
    return false;
  }

  // 不能移除owner
  if (member.role === 'owner') {
    return false;
  }

  // owner可以移除任何人（除了自己），admin只能移除member
  const currentMember = members.value.find((m) => m.user_id === props.currentUserId);
  if (currentMember?.role === 'admin' && member.role === 'admin') {
    return false;
  }

  return true;
};

// 获取角色标签
const getRoleLabel = (role: string) => {
  const labels: Record<string, string> = {
    owner: '群主',
    admin: '管理员',
    member: '成员',
  };
  return labels[role] || role;
};

// 加载成员列表（仅群聊）
const loadMembers = async () => {
  if (!props.conversation?.id || props.conversation.conversation_type !== 'group') {
    return;
  }

  try {
    const response = await api.getConversationMembers(props.conversation.id);
    if (response.success && response.data) {
      members.value = response.data;
    }
  } catch (error) {
    console.error('Failed to load members:', error);
  }
};

// 显示用户资料
const handleShowUserProfile = (user: any) => {
  if (user) {
    emit('show-user-profile', user);
  }
};

// 移除成员（仅群聊）
const handleRemoveMember = async (member: Enrollment) => {
  if (!props.conversation?.id || !confirm(`确定要移除 ${member.user?.username} 吗？`)) {
    return;
  }

  try {
    const response = await api.removeMemberFromConversation({
      conversation_id: props.conversation.id,
      user_id: member.user_id,
    });

    if (response.success) {
      await loadMembers();
      emit('members-changed');
    }
  } catch (error) {
    console.error('Failed to remove member:', error);
    alert('移除成员失败，请重试');
  }
};

// 成员添加成功（仅群聊）
const handleMemberAdded = async () => {
  await loadMembers();
  emit('members-changed');
};

// 发送消息
const handleSendMessage = () => {
  if (props.conversation) {
    emit('start-chat', props.conversation);
    emit('update:show', false);
  }
};

// 导出消息
const handleExportMessages = async () => {
  if (!props.conversation?.id) {
    return;
  }

  try {
    const response = await api.exportMessages(props.conversation.id);
    if (response.success && response.data) {
      // 将消息导出为JSONL格式
      const jsonl = response.data.map((msg) => JSON.stringify(msg)).join('\n');

      // 创建下载链接
      const blob = new Blob([jsonl], { type: 'application/jsonl' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `messages_${props.conversation.id}_${Date.now()}.jsonl`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
  } catch (error) {
    console.error('Failed to export messages:', error);
    alert('导出消息失败，请重试');
  }
};

// 监听show变化，加载成员（仅群聊）
watch(
  () => props.show,
  (newShow) => {
    if (newShow && props.conversation?.conversation_type === 'group') {
      loadMembers();
    }
  }
);
</script>

<style scoped>
.roundrect {
  border-radius: 8px;
}
</style>
