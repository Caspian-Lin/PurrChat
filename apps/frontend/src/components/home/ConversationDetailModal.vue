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
            class="w-16 h-16 roundrect overflow-hidden flex-shrink-0 flex items-center justify-center font-bold text-white text-3xl relative group/avatar cursor-pointer"
            style="background: var(--theme-gradient)"
            @click="canManageMembers && $refs.avatarInput?.click()"
          >
            <img
              v-if="conversation?.avatar_url"
              :src="conversation.avatar_url"
              alt="avatar"
              class="w-full h-full object-cover"
              referrerpolicy="no-referrer"
            />
            <template v-else>{{ conversation?.name?.charAt(0) || 'G' }}</template>
            <div
              v-if="canManageMembers"
              class="absolute inset-0 bg-black/40 opacity-0 group-hover/avatar:opacity-100 transition-opacity flex items-center justify-center"
            >
              <BsCamera :size="20" />
            </div>
          </div>
          <input
            v-if="conversation?.conversation_type === 'group' && canManageMembers"
            ref="avatarInput"
            type="file"
            accept="image/*"
            class="hidden"
            @change="handleAvatarChange"
          />
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
          <div class="flex-1 min-w-0">
            <!-- 群名编辑 -->
            <div
              v-if="
                conversation?.conversation_type === 'group' && canManageMembers && isEditingName
              "
              class="flex items-center gap-2"
            >
              <input
                v-model="editingName"
                v-focus
                type="text"
                maxlength="100"
                class="flex-1 px-2 py-1 text-lg font-semibold rounded-[var(--radius-sm)] bg-bg-quaternary text-text-primary outline-none focus:ring-1 focus:ring-[var(--theme-primary)]"
                @keydown.enter="handleSaveName"
                @keydown.escape="isEditingName = false"
              />
              <button
                class="p-1.5 rounded-lg hover:bg-hover-bg text-green-500 transition-colors"
                title="保存"
                @click="handleSaveName"
              >
                <BsCheck :size="18" />
              </button>
              <button
                class="p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary transition-colors"
                title="取消"
                @click="isEditingName = false"
              >
                <BsX :size="18" />
              </button>
            </div>
            <div
              v-else-if="conversation?.conversation_type === 'group' && canManageMembers"
              class="flex items-center gap-2 group/name cursor-pointer"
              @click="startEditingName"
            >
              <div class="font-semibold text-lg" style="color: var(--text-color)">
                {{ conversationName }}
              </div>
              <BsPencil
                :size="14"
                class="text-text-quaternary opacity-0 group-hover/name:opacity-100 transition-opacity"
              />
            </div>
            <div v-else class="font-semibold text-lg" style="color: var(--text-color)">
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
              {{ members.length }} 位成员（含 Bot）
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
            class="px-3 py-1.5 text-sm bg-accent-color text-white rounded-md hover:opacity-80 transition-colors"
            @click="showAddMemberModal = true"
          >
            添加成员
          </button>
        </div>
        <CustomScrollbar class="max-h-64 rounded-lg" style="background: var(--surface-color)">
          <div class="px-2 pt-2 pb-0.5">
            <BaseListItem
              v-for="member in members"
              :key="member.id"
              @click="handleShowUserProfile(member.user)"
            >
              <template #avatar>
                <div class="w-10 h-10 rounded-[var(--radius-md)] overflow-hidden">
                  <img
                    v-if="member.user?.avatar_url"
                    :src="member.user.avatar_url"
                    alt="avatar"
                    class="w-full h-full object-cover"
                  />
                  <div
                    v-else-if="member.user?.is_bot"
                    class="w-full h-full flex items-center justify-center text-white"
                    style="background: var(--theme-primary)"
                  >
                    <BsCpu :size="20" />
                  </div>
                  <div
                    v-else
                    class="w-full h-full flex items-center justify-center font-bold text-white"
                    style="background: var(--theme-gradient)"
                  >
                    {{ member.user?.username?.charAt(0) || 'U' }}
                  </div>
                </div>
              </template>
              <div class="flex items-center gap-2">
                <span class="font-medium text-text-primary">{{ member.user?.username }}</span>
                <span
                  v-if="member.user?.is_bot"
                  class="text-xs px-1.5 py-0.5 rounded-full text-white"
                  style="background: var(--theme-primary)"
                >
                  Bot
                </span>
                <span
                  v-else
                  class="text-xs px-1.5 py-0.5 rounded-full"
                  :class="
                    member.role === 'owner'
                      ? 'bg-amber-500/10 text-amber-600'
                      : member.role === 'admin'
                        ? 'bg-[var(--theme-primary)]/10 text-[var(--theme-primary)]'
                        : 'text-text-tertiary'
                  "
                >
                  {{ getRoleLabel(member.role) }}
                </span>
              </div>
              <template v-if="canManageMemberRole(member)" #actions>
                <div class="flex items-center gap-1.5">
                  <!-- 设置/取消管理员 -->
                  <button
                    v-if="member.role !== 'owner' && isOwner"
                    class="px-2.5 py-1 text-xs rounded-[var(--radius-sm)] transition-colors"
                    :class="
                      member.role === 'admin'
                        ? 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
                        : 'bg-[var(--theme-primary)]/10 text-[var(--theme-primary)] hover:bg-[var(--theme-primary)]/20'
                    "
                    @click.stop="handleToggleAdmin(member)"
                  >
                    {{ member.role === 'admin' ? '取消管理' : '设为管理' }}
                  </button>
                  <!-- 转让群主 -->
                  <button
                    v-if="member.role !== 'owner' && isOwner"
                    class="px-2.5 py-1 text-xs bg-amber-500/10 text-amber-600 rounded-[var(--radius-sm)] hover:bg-amber-500/20 transition-colors"
                    @click.stop="handleTransferOwner(member)"
                  >
                    转让群主
                  </button>
                  <!-- 移除 -->
                  <button
                    class="px-2.5 py-1 text-xs bg-red-500 text-white rounded-[var(--radius-sm)] hover:bg-red-600 transition-colors"
                    @click.stop="handleRemoveMember(member)"
                  >
                    移除
                  </button>
                </div>
              </template>
              <template v-else-if="canRemoveMember(member)" #actions>
                <button
                  class="px-2.5 py-1 text-xs bg-red-500 text-white rounded-[var(--radius-sm)] hover:bg-red-600 transition-colors"
                  @click.stop="handleRemoveMember(member)"
                >
                  移除
                </button>
              </template>
            </BaseListItem>
          </div>
        </CustomScrollbar>
      </div>

      <!-- 发送消息按钮 -->
      <div>
        <button
          class="w-full px-4 py-2 bg-accent-color text-white rounded-md hover:opacity-80 transition-colors"
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

      <!-- 退出群聊（非 owner 显示） -->
      <div v-if="conversation?.conversation_type === 'group' && !isOwner">
        <button
          class="w-full px-4 py-2 bg-red-500/10 text-red-500 rounded-md hover:bg-red-500/20 transition-colors"
          @click="handleLeaveGroup"
        >
          退出群聊
        </button>
      </div>

      <!-- 解散群聊（仅 owner 显示） -->
      <div v-if="conversation?.conversation_type === 'group' && isOwner">
        <button
          class="w-full px-4 py-2 bg-red-500/10 text-red-500 rounded-md hover:bg-red-500/20 transition-colors"
          @click="handleDisbandGroup"
        >
          解散群聊
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
import { ref, computed, watch, nextTick } from 'vue';
import { BsCamera, BsPencil, BsCheck, BsX, BsCpu } from 'vue-icons-plus/bs';
import BaseModal from '../common/BaseModal.vue';
import BaseListItem from '../common/BaseListItem.vue';
import AddMemberModal from './AddMemberModal.vue';
import CustomScrollbar from '../common/CustomScrollbar.vue';
import { api } from '../../models/api';
import { getUserAvatar, getUserUsername, getOtherUser } from '../../utils/userHelpers';
import { useMessage } from '../../composables/useMessage';
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
  'conversation-updated': [];
}>();

const message = useMessage();
const showAddMemberModal = ref(false);
const members = ref<Enrollment[]>([]);
const isEditingName = ref(false);
const editingName = ref('');
const avatarInput = ref<HTMLInputElement | null>(null);

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

// 检查当前用户是否是群主
const isOwner = computed(() => {
  if (!props.currentUserId || !props.conversation) return false;
  const currentMember = members.value.find((m) => m.user_id === props.currentUserId);
  return currentMember?.role === 'owner';
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
    const memberRes = await api.getConversationMembers(props.conversation.id);
    members.value = memberRes.success && memberRes.data ? memberRes.data : [];
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

// 编辑群名
function startEditingName() {
  editingName.value = props.conversation?.name || '';
  isEditingName.value = true;
  nextTick(() => {
    // v-focus 指令处理聚焦
  });
}

// 保存群名
async function handleSaveName() {
  if (!props.conversation?.id || !editingName.value.trim()) return;
  try {
    const response = await api.updateConversation(props.conversation.id, {
      name: editingName.value.trim(),
    });
    if (response.success) {
      isEditingName.value = false;
      message.success('群名称已更新');
      emit('conversation-updated');
    }
  } catch (error) {
    console.error('Failed to update conversation name:', error);
    message.error('更新群名称失败');
  }
}

// 更换群头像
async function handleAvatarChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file || !props.conversation?.id) return;

  // 这里复用头像上传逻辑
  try {
    const { storageApi } = await import('../../models/api');
    const uploadResponse = await storageApi.requestUpload({
      file_name: file.name,
      file_size: file.size,
      content_type: file.type,
      category: 'avatar',
      usage: 'group-avatar',
    });

    if (!uploadResponse.success || !uploadResponse.data) {
      message.error('上传申请失败');
      return;
    }

    await fetch(uploadResponse.data.upload_url, {
      method: uploadResponse.data.method,
      body: file,
    });

    const confirmResponse = await storageApi.confirmUpload({
      upload_id: uploadResponse.data.upload_id,
      object_key: uploadResponse.data.object_key,
    });

    if (!confirmResponse.success || !confirmResponse.data) {
      message.error('上传确认失败');
      return;
    }

    const updateResponse = await api.updateConversation(props.conversation.id, {
      avatar_url: confirmResponse.data.public_url,
    });

    if (updateResponse.success) {
      message.success('群头像已更新');
      emit('conversation-updated');
    }
  } catch (error) {
    console.error('Failed to update group avatar:', error);
    message.error('更新群头像失败');
  }

  // 清空 input，允许重复选择同一文件
  (event.target as HTMLInputElement).value = '';
}

// 检查是否可以管理成员角色（仅 owner）
function canManageMemberRole(member: Enrollment) {
  return isOwner.value && member.user_id !== props.currentUserId;
}

// 切换管理员角色
async function handleToggleAdmin(member: Enrollment) {
  if (!props.conversation?.id) return;
  const newRole = member.role === 'admin' ? 'member' : 'admin';
  const actionText = newRole === 'admin' ? '设为管理员' : '取消管理员';

  if (!confirm(`确定要将 ${member.user?.username} ${actionText}吗？`)) return;

  try {
    const response = await api.updateMemberRole({
      conversation_id: props.conversation.id,
      user_id: member.user_id,
      role: newRole,
    });
    if (response.success) {
      message.success(`${member.user?.username} 已${actionText}`);
      await loadMembers();
    }
  } catch (error) {
    console.error('Failed to update member role:', error);
    message.error(`${actionText}失败`);
  }
}

// 转让群主
async function handleTransferOwner(member: Enrollment) {
  if (!props.conversation?.id) return;
  if (!confirm(`确定要将群主转让给 ${member.user?.username} 吗？转让后你将变为普通成员。`)) return;
  if (!confirm('再次确认：此操作不可撤销！')) return;

  try {
    const response = await api.updateMemberRole({
      conversation_id: props.conversation.id,
      user_id: member.user_id,
      role: 'owner',
    });
    if (response.success) {
      message.success('群主已转让给 ' + member.user?.username);
      await loadMembers();
      emit('conversation-updated');
    }
  } catch (error) {
    console.error('Failed to transfer owner:', error);
    message.error('转让群主失败');
  }
}

// 退出群聊
async function handleLeaveGroup() {
  if (!props.conversation?.id || !confirm('确定要退出群聊吗？')) return;

  try {
    const response = await api.removeMemberFromConversation({
      conversation_id: props.conversation.id,
      user_id: props.currentUserId!,
    });
    if (response.success) {
      message.success('已退出群聊');
      emit('conversation-updated');
      emit('update:show', false);
    }
  } catch (error) {
    console.error('Failed to leave group:', error);
    message.error('退出群聊失败');
  }
}

// 解散群聊
async function handleDisbandGroup() {
  if (!props.conversation?.id) return;
  if (!confirm('确定要解散此群聊吗？所有成员将被移除，此操作不可撤销！')) return;

  try {
    const response = await api.deleteConversation(props.conversation.id);
    if (response.success) {
      message.success('群聊已解散');
      emit('conversation-updated');
      emit('update:show', false);
    }
  } catch (error) {
    console.error('Failed to disband group:', error);
    message.error('解散群聊失败');
  }
}

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

<style scoped></style>
