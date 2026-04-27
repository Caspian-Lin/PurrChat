<template>
  <BaseModal :show="show" title="注销账号" @update:show="emit('update:show', $event)">
    <!-- 警告说明 -->
    <div
      class="rounded-[var(--radius-sm)] p-4 mb-5"
      style="background: var(--color-error-bg, rgba(220, 38, 38, 0.08)); border: 1px solid rgba(220, 38, 38, 0.2)"
    >
      <div class="flex items-start gap-3">
        <BsExclamationTriangle
          class="shrink-0 mt-0.5"
          :size="18"
          style="color: var(--color-error)"
        />
        <div class="text-sm" style="color: var(--text-secondary-color)">
          <p class="font-medium mb-1.5" style="color: var(--color-error)">此操作不可逆</p>
          <p class="mb-2">注销后，您的所有数据将被永久删除，包括：</p>
          <ul class="list-disc list-inside space-y-0.5 text-xs" style="color: var(--text-tertiary-color)">
            <li>个人资料和账号信息</li>
            <li>所有好友关系</li>
            <li>创建的 Bot 及其部署</li>
            <li>聊天记录中的发送者身份</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 密码确认 -->
    <div class="mb-4">
      <label class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary-color)">
        请输入密码以确认注销
      </label>
      <BaseInput
        v-model="password"
        type="password"
        placeholder="输入当前密码"
        @keyup.enter="handleSubmit"
      />
    </div>

    <p v-if="errorMsg" class="text-sm mb-4" style="color: var(--color-error)">
      {{ errorMsg }}
    </p>

    <template #footer>
      <button
        class="px-4 py-2 text-sm rounded-[var(--radius-sm)] transition-colors"
        style="color: var(--text-secondary-color); background: var(--surface-tertiary-color)"
        @click="emit('update:show', false)"
      >
        取消
      </button>
      <button
        :disabled="submitting || !isFormValid"
        class="px-4 py-2 text-sm text-white rounded-[var(--radius-sm)] transition-all disabled:opacity-50"
        :style="{
          backgroundColor: isArmed ? 'var(--color-error)' : 'var(--color-error)',
          opacity: isArmed ? 1 : undefined,
        }"
        @click="handleSubmit"
      >
        <span v-if="submitting">注销中...</span>
        <span v-else-if="countdown > 0">再次点击以确认注销 ({{ countdown }}s)</span>
        <span v-else>确认注销</span>
      </button>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue';
import { BsExclamationTriangle } from 'vue-icons-plus/bs';
import BaseModal from '../../../common/BaseModal.vue';
import BaseInput from '../../../common/BaseInput.vue';
import { useAuthStore } from '../../../../stores/auth';
import { useRouter } from 'vue-router';

interface Props {
  show: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
}>();

const authStore = useAuthStore();
const router = useRouter();

const password = ref('');
const errorMsg = ref('');
const submitting = ref(false);
const isArmed = ref(false);
const countdown = ref(0);
let countdownTimer: ReturnType<typeof setInterval> | null = null;

const isFormValid = computed(() => password.value.length >= 6);

// 关闭时重置状态
watch(
  () => props.show,
  (val) => {
    if (!val) {
      resetState();
    }
  }
);

function resetState() {
  password.value = '';
  errorMsg.value = '';
  submitting.value = false;
  isArmed.value = false;
  countdown.value = 0;
  if (countdownTimer) {
    clearInterval(countdownTimer);
    countdownTimer = null;
  }
}

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
});

function startCountdown() {
  isArmed.value = true;
  countdown.value = 3;
  countdownTimer = setInterval(() => {
    countdown.value--;
    if (countdown.value <= 0) {
      isArmed.value = false;
      if (countdownTimer) {
        clearInterval(countdownTimer);
        countdownTimer = null;
      }
    }
  }, 1000);
}

async function handleSubmit() {
  if (!isArmed.value && !submitting.value) {
    // 第一次点击：启动倒计时
    startCountdown();
    return;
  }

  if (!isArmed.value) return;

  // 倒计时内第二次点击：执行注销
  submitting.value = true;
  errorMsg.value = '';

  const success = await authStore.deleteAccount(password.value);
  if (success) {
    resetState();
    emit('update:show', false);
    router.push('/login');
  } else {
    errorMsg.value = authStore.error || '注销失败';
    submitting.value = false;
    isArmed.value = false;
    countdown.value = 0;
    if (countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }
}
</script>
