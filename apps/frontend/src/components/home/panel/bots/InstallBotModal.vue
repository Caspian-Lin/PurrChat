<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="$emit('close')">
    <div class="absolute inset-0 bg-black/30" aria-hidden="true" />
    <section
      class="relative w-full max-w-lg overflow-hidden rounded-[var(--radius-lg,16px)] bg-bg-primary shadow-lg"
      role="dialog"
      aria-modal="true"
      aria-labelledby="install-bot-title"
    >
      <header class="flex items-center justify-between border-b border-border-subtle px-6 py-4">
        <div>
          <h2 id="install-bot-title" class="text-base font-semibold text-text-primary">
            {{ installation ? '重新授权' : '安装' }} {{ bot.name }}
          </h2>
          <p class="mt-0.5 text-xs text-text-tertiary">安装到我的私聊</p>
        </div>
        <button
          class="rounded-lg p-1.5 text-text-tertiary transition-colors hover:bg-hover-bg hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--theme-primary)]"
          aria-label="关闭"
          :disabled="installing"
          @click="$emit('close')"
        >
          <BsX :size="18" />
        </button>
      </header>

      <div class="max-h-[65vh] overflow-y-auto px-6 py-5">
        <BotPermissionReview
          :bot-name="bot.name"
          target-label="我的私聊"
          :requested-capabilities="bot.requested_capabilities ?? []"
          :initial-capabilities="installation?.granted_capabilities"
          @change="handlePermissionChange"
        />
      </div>

      <footer class="flex items-center justify-end gap-3 border-t border-border-subtle px-6 py-4">
        <button
          class="rounded-[var(--radius-sm,8px)] px-4 py-2 text-sm text-text-secondary transition-colors hover:bg-hover-bg"
          :disabled="installing"
          @click="$emit('close')"
        >
          取消
        </button>
        <button
          class="inline-flex min-w-24 items-center justify-center gap-2 rounded-[var(--radius-sm,8px)] bg-[var(--theme-primary)] px-4 py-2 text-sm font-medium text-white transition-opacity disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="installing || !canConfirm"
          @click="confirmInstall"
        >
          <span
            v-if="installing"
            class="h-4 w-4 animate-spin rounded-full border-2 border-white/40 border-t-white"
          />
          {{ installing ? '保存中' : installation ? '保存授权并开始对话' : '授权并开始对话' }}
        </button>
      </footer>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { BsX } from 'vue-icons-plus/bs';
import type { Bot, BotDeployment } from '../../../../models/types';
import { useBots } from '../../../../composables/useBots';
import { useAuthController } from '../../../../controllers/authController';
import BotPermissionReview from './BotPermissionReview.vue';

const props = defineProps<{ bot: Bot; installation?: BotDeployment | null }>();
const emit = defineEmits<{
  installed: [];
  close: [];
}>();

const { installBot, updateInstallation } = useBots();
const auth = useAuthController();
const selectedCapabilities = ref<string[]>([]);
const canConfirm = ref(false);
const installing = ref(false);

function handlePermissionChange(capabilities: string[], allowed: boolean) {
  selectedCapabilities.value = capabilities;
  canConfirm.value = allowed;
}

async function confirmInstall() {
  const userId = auth.currentUser?.id;
  if (!userId || !canConfirm.value) return;

  installing.value = true;
  const diagnosticsConsent = selectedCapabilities.value.includes('network:external')
    ? 'granted'
    : 'denied';
  const installation = props.installation
    ? await updateInstallation(props.installation.id, {
        status: 'active',
        granted_capabilities: selectedCapabilities.value,
        diagnostics_consent: diagnosticsConsent,
      })
    : await installBot(props.bot.id, {
        target_type: 'user',
        target_id: userId,
        granted_capabilities: selectedCapabilities.value,
        diagnostics_consent: diagnosticsConsent,
      });
  installing.value = false;
  if (installation) emit('installed');
}
</script>
