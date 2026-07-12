<template>
  <div class="space-y-5">
    <div>
      <h3 class="text-sm font-semibold text-text-primary">确认 Bot 权限</h3>
      <p class="mt-1 text-sm leading-6 text-text-secondary">
        {{ botName }} 将在「{{
          targetLabel
        }}」中使用以下能力。你可以关闭非核心权限，之后也能随时调整。
      </p>
    </div>

    <div v-if="requestedCapabilities.length" class="space-y-2" aria-label="Bot 请求的权限">
      <label
        v-for="capability in requestedCapabilities"
        :key="capability"
        class="flex items-start gap-3 rounded-[var(--radius-md,12px)] bg-bg-secondary px-3.5 py-3"
        :class="isCore(capability) ? 'cursor-default' : 'cursor-pointer hover:bg-hover-bg'"
      >
        <input
          :checked="selectedCapabilities.includes(capability)"
          type="checkbox"
          class="mt-1 h-4 w-4 accent-[var(--theme-primary)]"
          :disabled="isCore(capability)"
          :aria-describedby="`capability-${capability}`"
          @change="toggleCapability(capability)"
        />
        <span class="min-w-0 flex-1">
          <span class="flex flex-wrap items-center gap-2 text-sm font-medium text-text-primary">
            <span aria-hidden="true">{{ getMeta(capability).icon }}</span>
            {{ getMeta(capability).label }}
            <span
              v-if="isCore(capability)"
              class="rounded-full bg-[var(--theme-primary)]/10 px-2 py-0.5 text-[11px] font-medium text-[var(--theme-primary)]"
            >
              运行必需
            </span>
            <span
              v-else-if="getMeta(capability).sensitive"
              class="rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:text-amber-300"
            >
              敏感权限
            </span>
          </span>
          <span
            :id="`capability-${capability}`"
            class="mt-1 block text-xs leading-5 text-text-secondary"
          >
            {{ getMeta(capability).description }}
          </span>
        </span>
      </label>
    </div>

    <div v-else class="rounded-[var(--radius-md,12px)] bg-bg-secondary px-4 py-3">
      <p class="text-sm font-medium text-text-primary">这个 Bot 还没有可授权的运行权限</p>
      <p class="mt-1 text-xs leading-5 text-text-secondary">
        请让创建者先保存固定回复配置或发布工作流，再回来安装。为避免 Bot
        安装后无法响应，当前不能继续。
      </p>
    </div>

    <label
      v-if="hasSelectedSensitiveCapability"
      class="flex cursor-pointer items-start gap-3 rounded-[var(--radius-md,12px)] bg-amber-500/10 px-4 py-3"
    >
      <input
        v-model="sensitiveAcknowledged"
        type="checkbox"
        class="mt-1 h-4 w-4 accent-[var(--theme-primary)]"
      />
      <span class="text-xs leading-5 text-text-primary">
        我了解：所选敏感权限可能让对话内容发送到第三方服务，相关数据将受该服务的隐私政策约束。
      </span>
    </label>

    <p class="text-xs leading-5 text-text-tertiary">
      PurrChat 只会向 Bot 提供你在这里授予的能力。群聊安装会影响该群中的所有成员。
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import {
  CAPABILITY_META,
  Capability,
  isSensitiveCapability,
  type CapabilityMeta,
} from '@purrchat/workflow-types';

const props = defineProps<{
  botName: string;
  targetLabel: string;
  requestedCapabilities: string[];
  initialCapabilities?: string[];
}>();

const emit = defineEmits<{
  change: [capabilities: string[], canConfirm: boolean];
}>();

const coreCapabilities = new Set<string>([Capability.ReadTrigger, Capability.Send]);
const selectedCapabilities = ref<string[]>([]);
const sensitiveAcknowledged = ref(false);

const hasSelectedSensitiveCapability = computed(() =>
  selectedCapabilities.value.some(isSensitiveCapability)
);
const canConfirm = computed(
  () =>
    props.requestedCapabilities.length > 0 &&
    (!hasSelectedSensitiveCapability.value || sensitiveAcknowledged.value)
);

function resetSelection() {
  selectedCapabilities.value = [
    ...new Set(
      (props.initialCapabilities ?? props.requestedCapabilities).filter((capability) =>
        props.requestedCapabilities.includes(capability)
      )
    ),
  ];
  for (const capability of props.requestedCapabilities) {
    if (isCore(capability) && !selectedCapabilities.value.includes(capability)) {
      selectedCapabilities.value.push(capability);
    }
  }
  sensitiveAcknowledged.value = false;
}

function isCore(capability: string) {
  return coreCapabilities.has(capability);
}

function getMeta(capability: string): CapabilityMeta {
  return (
    CAPABILITY_META[capability as keyof typeof CAPABILITY_META] ?? {
      label: capability,
      icon: '·',
      description: 'Bot 声明的扩展能力',
    }
  );
}

function toggleCapability(capability: string) {
  if (isCore(capability)) return;
  selectedCapabilities.value = selectedCapabilities.value.includes(capability)
    ? selectedCapabilities.value.filter((item) => item !== capability)
    : [...selectedCapabilities.value, capability];
}

watch(() => [props.requestedCapabilities, props.initialCapabilities], resetSelection, {
  immediate: true,
  deep: true,
});

watch(
  [selectedCapabilities, canConfirm],
  () => emit('change', [...selectedCapabilities.value], canConfirm.value),
  { immediate: true, deep: true }
);
</script>
