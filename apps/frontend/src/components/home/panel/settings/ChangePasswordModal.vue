<template>
  <BaseModal :show="show" title="修改密码" @update:show="emit('update:show', $event)">
    <div class="space-y-5">
      <div>
        <label class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary-color)">
          当前密码
        </label>
        <BaseInput v-model="form.oldPassword" type="password" placeholder="输入当前密码" />
      </div>
      <div>
        <label class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary-color)">
          新密码
        </label>
        <BaseInput
          v-model="form.newPassword"
          type="password"
          placeholder="输入新密码（至少 6 位）"
        />
      </div>
      <div>
        <label class="block text-sm font-medium mb-1.5" style="color: var(--text-secondary-color)">
          确认新密码
        </label>
        <BaseInput
          v-model="confirmPassword"
          type="password"
          placeholder="再次输入新密码"
          @keyup.enter="handleSubmit"
        />
      </div>

      <p v-if="errorMsg" class="text-sm" style="color: var(--color-error)">
        {{ errorMsg }}
      </p>
    </div>

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
        class="px-4 py-2 text-sm text-white rounded-[var(--radius-sm)] transition-opacity disabled:opacity-50"
        :style="{ backgroundColor: 'var(--theme-primary)' }"
        @click="handleSubmit"
      >
        {{ submitting ? '修改中...' : '确认修改' }}
      </button>
    </template>
  </BaseModal>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue';
import BaseModal from '../../../common/BaseModal.vue';
import BaseInput from '../../../common/BaseInput.vue';
import { api } from '../../../../models/api';

interface Props {
  show: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  'update:show': [value: boolean];
  success: [];
}>();

const form = reactive({
  oldPassword: '',
  newPassword: '',
});
const confirmPassword = ref('');
const errorMsg = ref('');
const submitting = ref(false);

const isFormValid = computed(() => {
  return (
    form.oldPassword.length >= 6 &&
    form.newPassword.length >= 6 &&
    form.newPassword === confirmPassword.value
  );
});

// 关闭时重置表单
watch(
  () => props.show,
  (val) => {
    if (!val) {
      form.oldPassword = '';
      form.newPassword = '';
      confirmPassword.value = '';
      errorMsg.value = '';
    }
  }
);

async function handleSubmit() {
  if (form.newPassword !== confirmPassword.value) {
    errorMsg.value = '两次输入的新密码不一致';
    return;
  }
  if (form.newPassword.length < 6) {
    errorMsg.value = '新密码至少 6 位';
    return;
  }
  if (form.oldPassword === form.newPassword) {
    errorMsg.value = '新密码不能与当前密码相同';
    return;
  }

  submitting.value = true;
  errorMsg.value = '';

  try {
    const resp = await api.changePassword({
      old_password: form.oldPassword,
      new_password: form.newPassword,
    });
    if (resp.success) {
      emit('success');
    } else {
      errorMsg.value = resp.message || '修改失败';
    }
  } catch {
    errorMsg.value = '网络错误，请重试';
  } finally {
    submitting.value = false;
  }
}
</script>
