<template>
  <main class="h-full overflow-y-auto bg-bg-tertiary text-text-primary">
    <div class="mx-auto max-w-5xl px-4 py-8 sm:px-8 lg:px-12">
      <header
        class="flex flex-wrap items-start justify-between gap-5 border-b border-border-subtle pb-6"
      >
        <div class="flex items-start gap-3 max-w-2xl">
          <button
            class="mt-1 flex-shrink-0 p-1.5 rounded-lg hover:bg-hover-bg text-text-tertiary hover:text-text-primary transition-colors"
            title="返回 Bot Studio"
            @click="$router.push('/bots')"
          >
            <BsArrowLeft :size="18" />
          </button>
          <div>
            <p class="text-sm font-medium text-text-secondary">Bot Studio / Developer</p>
            <h1 class="mt-2 text-2xl font-semibold tracking-[-0.02em]">OneBot API 支持矩阵</h1>
            <p class="mt-2 text-sm leading-6 text-text-secondary">
              此页面直接读取服务端协议 Registry，展示当前可用能力与兼容边界。
            </p>
          </div>
        </div>
        <button
          class="rounded-[var(--radius-sm,8px)] bg-bg-quaternary px-4 py-2 text-sm font-medium transition-colors hover:bg-hover-bg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--theme-primary)]"
          @click="load"
        >
          刷新
        </button>
      </header>

      <section v-if="loading" class="space-y-3 py-8" aria-label="正在加载能力矩阵">
        <div
          v-for="width in [85, 68, 74, 52]"
          :key="width"
          class="h-16 animate-pulse rounded-[var(--radius-md,12px)] bg-bg-quaternary"
          :style="{ width: `${width}%` }"
        />
      </section>

      <section v-else-if="error" class="py-16 text-center" role="alert">
        <h2 class="text-lg font-semibold">无法加载支持矩阵</h2>
        <p class="mx-auto mt-2 max-w-md text-sm text-text-secondary">{{ error }}</p>
        <button
          class="mt-5 rounded-[var(--radius-sm,8px)] px-4 py-2 text-sm font-medium text-white"
          style="background: var(--theme-primary)"
          @click="load"
        >
          重试
        </button>
      </section>

      <template v-else-if="catalog">
        <section class="grid gap-3 py-6 text-sm sm:grid-cols-3" aria-label="协议概览">
          <div class="rounded-[var(--radius-md,12px)] bg-bg-secondary p-4">
            <span class="text-text-tertiary">Profile</span>
            <p class="mt-1 font-medium">{{ catalog.profile.version }}</p>
          </div>
          <div class="rounded-[var(--radius-md,12px)] bg-bg-secondary p-4">
            <span class="text-text-tertiary">消息格式</span>
            <p class="mt-1 font-medium">{{ catalog.profile.message_format }}</p>
          </div>
          <div class="rounded-[var(--radius-md,12px)] bg-bg-secondary p-4">
            <span class="text-text-tertiary">标识符</span>
            <p class="mt-1 font-medium">{{ catalog.profile.id_format }}</p>
          </div>
        </section>

        <section
          class="sticky top-0 z-10 -mx-4 border-y border-border-subtle bg-bg-tertiary px-4 py-4 sm:-mx-8 sm:px-8 lg:-mx-12 lg:px-12"
        >
          <label class="sr-only" for="capability-search">搜索 Action 或 Event</label>
          <input
            id="capability-search"
            v-model.trim="query"
            class="w-full rounded-[var(--radius-sm,8px)] border border-border bg-bg-primary px-3 py-2 text-sm outline-none focus:border-[var(--theme-primary)]"
            placeholder="搜索 Action、Event 或兼容说明"
            type="search"
          />
          <div class="mt-3 flex flex-wrap gap-2" aria-label="筛选支持状态">
            <button
              v-for="option in statusOptions"
              :key="option.value"
              class="rounded-full px-3 py-1 text-xs font-medium transition-colors"
              :class="
                selectedStatus === option.value
                  ? 'bg-[var(--theme-primary)] text-white'
                  : 'bg-bg-quaternary text-text-secondary hover:bg-hover-bg'
              "
              @click="selectedStatus = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </section>

        <section class="py-7">
          <div class="flex items-baseline justify-between gap-4">
            <h2 class="text-lg font-semibold">Actions</h2>
            <span class="text-sm text-text-tertiary">{{ actions.length }} 项</span>
          </div>
          <div
            v-if="actions.length"
            class="mt-3 divide-y divide-border-subtle rounded-[var(--radius-md,12px)] bg-bg-secondary px-4"
          >
            <article
              v-for="action in actions"
              :id="`action-${action.name}`"
              :key="action.name"
              class="scroll-mt-36 py-5"
            >
              <div class="flex flex-wrap items-center gap-2">
                <code class="font-medium text-text-primary">{{ action.name }}</code
                ><StatusBadge :status="action.status" /><span class="text-xs text-text-tertiary">{{
                  sourceLabel(action.source)
                }}</span>
              </div>
              <p
                v-if="action.compatibility_note"
                class="mt-2 text-sm leading-6 text-text-secondary"
              >
                {{ action.compatibility_note }}
              </p>
              <dl class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-text-tertiary">
                <div>
                  <dt class="inline">传输：</dt>
                  <dd class="inline">{{ action.transports.join(' · ') }}</dd>
                </div>
                <div v-if="action.required_capability">
                  <dt class="inline">权限：</dt>
                  <dd class="inline">{{ action.required_capability }}</dd>
                </div>
                <div>
                  <dt class="inline">版本：</dt>
                  <dd class="inline">{{ action.version }}</dd>
                </div>
              </dl>
              <pre
                v-if="action.request_example"
                class="mt-3 overflow-x-auto rounded-[var(--radius-sm,8px)] bg-bg-primary p-3 text-xs leading-5 text-text-secondary"
                >{{ pretty(action.request_example) }}</pre
              >
            </article>
          </div>
          <p v-else class="py-10 text-center text-sm text-text-tertiary">
            没有符合当前筛选条件的 Action。
          </p>
        </section>

        <section class="border-t border-border-subtle py-7">
          <div class="flex items-baseline justify-between gap-4">
            <h2 class="text-lg font-semibold">Events</h2>
            <span class="text-sm text-text-tertiary">{{ events.length }} 项</span>
          </div>
          <div
            v-if="events.length"
            class="mt-3 divide-y divide-border-subtle rounded-[var(--radius-md,12px)] bg-bg-secondary px-4"
          >
            <article
              v-for="event in events"
              :id="`event-${event.post_type}-${event.detail_type}`"
              :key="`${event.post_type}-${event.detail_type}`"
              class="scroll-mt-36 py-5"
            >
              <div class="flex flex-wrap items-center gap-2">
                <code class="font-medium">{{ event.post_type }}.{{ event.detail_type }}</code
                ><StatusBadge :status="event.status" /><span class="text-xs text-text-tertiary">{{
                  sourceLabel(event.source)
                }}</span>
              </div>
              <p v-if="event.compatibility_note" class="mt-2 text-sm leading-6 text-text-secondary">
                {{ event.compatibility_note }}
              </p>
              <dl class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-text-tertiary">
                <div>
                  <dt class="inline">传输：</dt>
                  <dd class="inline">{{ event.transports.join(' · ') }}</dd>
                </div>
                <div v-if="event.required_capability">
                  <dt class="inline">权限：</dt>
                  <dd class="inline">{{ event.required_capability }}</dd>
                </div>
              </dl>
            </article>
          </div>
          <p v-else class="py-10 text-center text-sm text-text-tertiary">
            没有符合当前筛选条件的 Event。
          </p>
        </section>
      </template>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue';
import { BsArrowLeft } from 'vue-icons-plus/bs';
import { api } from '../models/api';
import type {
  BotApiActionCapability,
  BotApiCapabilities,
  BotApiEventCapability,
  BotApiStatus,
} from '../models/types';

const catalog = ref<BotApiCapabilities>();
const loading = ref(true);
const error = ref('');
const query = ref('');
const selectedStatus = ref<BotApiStatus | 'all'>('all');
const statusOptions: { value: BotApiStatus | 'all'; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'stable', label: 'Stable' },
  { value: 'beta', label: 'Beta' },
  { value: 'partial', label: 'Partial' },
  { value: 'rejected', label: 'Rejected' },
];

const matches = (item: BotApiActionCapability | BotApiEventCapability, name: string) => {
  const value =
    `${name} ${item.category} ${item.compatibility_note ?? ''} ${item.source}`.toLowerCase();
  return (
    (selectedStatus.value === 'all' || item.status === selectedStatus.value) &&
    value.includes(query.value.toLowerCase())
  );
};
const actions = computed(
  () => catalog.value?.actions.filter((item) => matches(item, item.name)) ?? []
);
const events = computed(
  () =>
    catalog.value?.events.filter((item) =>
      matches(item, `${item.post_type}.${item.detail_type}`)
    ) ?? []
);

const StatusBadge = defineComponent({
  props: { status: { type: String, required: true } },
  setup: (props) => () =>
    h(
      'span',
      {
        class: 'rounded-full bg-bg-quaternary px-2 py-0.5 text-xs font-medium text-text-secondary',
      },
      props.status
    ),
});
function sourceLabel(source: string) {
  return (
    (
      {
        onebot_core: 'OneBot Core',
        go_cqhttp_extension: 'go-cqhttp Extension',
        napcat_extension: 'NapCat Extension',
        purrchat_extension: 'PurrChat Extension',
      } as Record<string, string>
    )[source] ?? source
  );
}
function pretty(value: Record<string, unknown>) {
  return JSON.stringify(value, null, 2);
}
async function load() {
  loading.value = true;
  error.value = '';
  try {
    catalog.value = await api.getBotApiCapabilities();
  } catch {
    error.value = '请检查网络连接或稍后重试。';
  } finally {
    loading.value = false;
  }
}
onMounted(load);
</script>
