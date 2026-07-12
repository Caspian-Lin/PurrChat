<template>
  <section>
    <h3 class="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
      <BsKey :size="16" class="text-text-tertiary" />
      API Token
    </h3>

    <!-- 创建 Token -->
    <div class="mb-4 flex gap-2">
      <input
        v-model="newCredentialName"
        type="text"
        maxlength="40"
        placeholder="Token 名称（如 production / staging）"
        class="flex-1 px-3 py-2 text-sm rounded-[var(--radius-sm,8px)] bg-bg-quaternary text-text-primary placeholder:text-text-quaternary outline-none focus:ring-1 focus:ring-[var(--theme-primary)] transition-all"
        @keydown.enter="handleCreate"
      />
      <button
        class="flex items-center gap-1.5 px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] text-white transition-colors disabled:opacity-50"
        style="background: var(--theme-primary)"
        :disabled="!newCredentialName.trim() || creating"
        @click="handleCreate"
      >
        <BsPlus :size="14" />
        生成 Token
      </button>
    </div>

    <!-- Token 列表 -->
    <div v-if="loading" class="text-sm text-text-quaternary py-4 text-center">加载中...</div>
    <div v-else-if="credentials.length === 0" class="text-sm text-text-quaternary py-4 text-center">
      还没有 Token，创建一个来接入外部 Bot 平台
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="cred in credentials"
        :key="cred.id"
        class="flex items-center gap-3 px-3 py-2.5 rounded-[var(--radius-sm,8px)] bg-bg-quaternary"
      >
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-sm text-text-primary font-medium">{{ cred.name }}</span>
            <span
              v-if="cred.revoked_at"
              class="text-[10px] px-1.5 py-0.5 rounded-full bg-red-500/10 text-red-500"
            >
              已撤销
            </span>
          </div>
          <div class="flex items-center gap-2 mt-0.5">
            <code class="text-xs text-text-tertiary font-mono">{{ cred.token_prefix }}...</code>
            <span class="text-[10px] text-text-quaternary">
              {{ formatTime(cred.created_at) }} 创建
            </span>
            <span v-if="cred.last_used_at" class="text-[10px] text-text-quaternary">
              · {{ formatTime(cred.last_used_at) }} 使用
            </span>
          </div>
        </div>
        <button
          v-if="!cred.revoked_at"
          class="px-2 py-1 text-xs rounded-[var(--radius-sm,8px)] text-text-tertiary hover:text-text-primary hover:bg-hover-bg transition-colors"
          @click="handleRotate(cred.id)"
        >
          轮换
        </button>
        <button
          v-if="!cred.revoked_at"
          class="px-2 py-1 text-xs rounded-[var(--radius-sm,8px)] text-red-400 hover:text-red-500 hover:bg-red-500/5 transition-colors"
          @click="handleRevoke(cred.id, cred.name)"
        >
          撤销
        </button>
      </div>
    </div>

    <!-- 一次性 Token 展示 -->
    <Transition name="token-modal">
      <div
        v-if="tokenDisplay"
        class="fixed inset-0 z-50 flex items-center justify-center"
        @click.self="tokenDisplay = null"
      >
        <div class="absolute inset-0 bg-black/30 backdrop-blur-sm" />
        <div
          class="relative w-full max-w-lg mx-4 bg-bg-primary rounded-[var(--radius-lg,16px)] shadow-lg overflow-hidden"
        >
          <div class="px-6 py-5 space-y-4">
            <div class="flex items-center gap-2">
              <BsExclamationTriangle :size="18" class="text-amber-500" />
              <h3 class="text-sm font-semibold text-text-primary">Token 已生成</h3>
            </div>
            <p class="text-xs text-text-secondary">
              这是唯一一次完整展示，请立即复制保存。关闭后将无法再次查看。
            </p>
            <div
              class="flex items-center gap-2 px-3 py-2.5 rounded-[var(--radius-sm,8px)] bg-bg-quaternary"
            >
              <code class="flex-1 text-sm text-text-primary font-mono break-all">
                {{ tokenDisplay }}
              </code>
              <button
                class="flex-shrink-0 p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
                title="复制"
                @click="copyToken"
              >
                <BsClipboard :size="14" />
              </button>
            </div>
            <p v-if="copied" class="text-xs text-green-500">已复制到剪贴板</p>
          </div>
          <div class="flex justify-end px-6 py-4 border-t border-border-subtle">
            <button
              class="px-4 py-2 text-sm rounded-[var(--radius-sm,8px)] text-white"
              style="background: var(--theme-primary)"
              @click="tokenDisplay = null"
            >
              我已保存
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { BsKey, BsPlus, BsClipboard, BsExclamationTriangle } from 'vue-icons-plus/bs';
import { api } from '../../../../models/api';
import type { BotAPICredential } from '../../../../models/types';

const props = defineProps<{ botId: string }>();

const credentials = ref<BotAPICredential[]>([]);
const loading = ref(true);
const creating = ref(false);
const newCredentialName = ref('');
const tokenDisplay = ref<string | null>(null);
const copied = ref(false);

onMounted(async () => {
  await loadCredentials();
});

async function loadCredentials() {
  loading.value = true;
  try {
    const res = await api.listBotCredentials(props.botId);
    if (res.data) {
      credentials.value = res.data;
    }
  } catch {
    // ignore
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  if (!newCredentialName.value.trim() || creating.value) return;
  creating.value = true;
  try {
    const res = await api.createBotCredential(props.botId, {
      name: newCredentialName.value.trim(),
    });
    if (res.data) {
      tokenDisplay.value = res.data.token;
      copied.value = false;
      newCredentialName.value = '';
      await loadCredentials();
    }
  } catch (err: any) {
    const msg = err?.response?.data?.error?.message || err?.response?.data?.error;
    alert(typeof msg === 'string' ? msg : '创建 Token 失败');
  } finally {
    creating.value = false;
  }
}

async function handleRotate(credentialId: string) {
  if (!confirm('轮换将生成新 Token 并使旧 Token 立即失效，已连接的 WebSocket 也会断开。继续？'))
    return;
  try {
    const res = await api.rotateBotCredential(props.botId, credentialId);
    if (res.data) {
      tokenDisplay.value = res.data.token;
      copied.value = false;
      await loadCredentials();
    }
  } catch (err: any) {
    const msg = err?.response?.data?.error?.message || err?.response?.data?.error;
    alert(typeof msg === 'string' ? msg : '轮换 Token 失败');
  }
}

async function handleRevoke(credentialId: string, name: string) {
  if (!confirm(`确认撤销 Token「${name}」？此操作不可撤销。`)) return;
  try {
    await api.revokeBotCredential(props.botId, credentialId);
    await loadCredentials();
  } catch (err: any) {
    const msg = err?.response?.data?.error?.message || err?.response?.data?.error;
    alert(typeof msg === 'string' ? msg : '撤销 Token 失败');
  }
}

async function copyToken() {
  if (!tokenDisplay.value) return;
  try {
    await navigator.clipboard.writeText(tokenDisplay.value);
    copied.value = true;
  } catch {
    // ignore
  }
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diff = now.getTime() - d.getTime();
  if (diff < 60000) return '刚刚';
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`;
  return d.toLocaleDateString();
}
</script>

<style scoped>
.token-modal-enter-active {
  transition: all 200ms cubic-bezier(0.25, 1, 0.5, 1);
}
.token-modal-leave-active {
  transition: all 150ms cubic-bezier(0.16, 1, 0.3, 1);
}
.token-modal-enter-from,
.token-modal-leave-to {
  opacity: 0;
}
</style>
