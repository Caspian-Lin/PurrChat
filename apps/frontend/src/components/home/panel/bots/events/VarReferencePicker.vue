<template>
  <Teleport to="body">
    <div v-if="visible" class="var-picker-overlay" @click.self="$emit('close')">
      <div class="var-picker" :style="anchorStyle">
        <!-- 搜索 -->
        <div class="var-picker__search">
          <input
            v-model="search"
            class="var-picker__input"
            placeholder="搜索变量..."
            autofocus
          />
        </div>

        <!-- 变量列表 -->
        <div class="var-picker__list">
          <!-- 内置变量 -->
          <div class="var-picker__group">
            <div class="var-picker__group-title">内置变量</div>
            <button
              v-for="v in builtinVars"
              :key="v.key"
              class="var-picker__item"
              :class="{ 'var-picker__item--active': highlightedKey === v.key }"
              @click="select(v)"
              @mouseenter="highlightedKey = v.key"
            >
              <span class="var-picker__dot" :style="{ background: PORT_COLORS[v.dataType] }" />
              <span class="var-picker__var-key">{{ v.key }}</span>
              <span class="var-picker__var-label">{{ v.label }}</span>
              <span class="var-picker__var-type">{{ v.dataType }}</span>
            </button>
          </div>

          <!-- 上游节点输出 -->
          <div v-for="group in upstreamGroups" :key="group.nodeId" class="var-picker__group">
            <div class="var-picker__group-title">
              {{ group.nodeName }}
              <span class="var-picker__group-id">{{ group.nodeId }}</span>
            </div>
            <button
              v-for="port in group.ports"
              :key="port.ref"
              class="var-picker__item"
              :class="{ 'var-picker__item--active': highlightedKey === port.ref }"
              @click="select(port)"
              @mouseenter="highlightedKey = port.ref"
            >
              <span class="var-picker__dot" :style="{ background: PORT_COLORS[port.dataType] }" />
              <span class="var-picker__var-key">{{ port.ref }}</span>
              <span class="var-picker__var-label">{{ port.name }}</span>
              <span class="var-picker__var-type">{{ port.dataType }}</span>
            </button>
          </div>

          <!-- 无结果 -->
          <div v-if="!builtinVars.length && !upstreamGroups.length" class="var-picker__empty">
            无可用变量
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { PORT_COLORS, type PortDataType } from '../../../../../utils/portTypes';
import type { WorkflowEvent, FlowConnection } from '../../../../../models/types';

interface VariableItem {
  ref: string; // 人类可读的引用格式，如 {触发.用户消息}
  key: string; // 用于搜索匹配
  label: string;
  dataType: PortDataType;
  value: string; // 实际插入的引用字符串，如 $evt_xxx:out_input
}

interface Props {
  visible: boolean;
  /** 当前节点 ID（排除自身） */
  currentNodeId: string;
  /** 所有事件节点 */
  events: WorkflowEvent[];
  /** 所有连线 */
  connections: FlowConnection[];
  /** 锚点位置 */
  anchor?: { x: number; y: number };
}

const props = withDefaults(defineProps<Props>(), {
  anchor: () => ({ x: 0, y: 0 }),
});

const emit = defineEmits<{
  close: [];
  select: [item: VariableItem];
}>();

const search = ref('');
const highlightedKey = ref('');

// 锚点定位
const anchorStyle = computed(() => ({
  position: 'fixed' as const,
  left: `${props.anchor.x}px`,
  top: `${props.anchor.y + 24}px`, // 按钮下方
}));

// 打开时重置
watch(
  () => props.visible,
  () => {
    if (props.visible) {
      search.value = '';
      highlightedKey.value = '';
    }
  }
);

// 内置变量（总是可用）
const builtinVars = computed<VariableItem[]>(() => {
  const trigger = props.events.find((e) => e.type === 'trigger');
  const tid = trigger?.id || '';

  const vars: VariableItem[] = [
    {
      ref: '{input}',
      key: 'input input 用户消息',
      label: '用户消息',
      dataType: 'string',
      value: tid ? `$${tid}:out_input` : '{input}',
    },
    {
      ref: '{username}',
      key: 'username 发送者 用户名',
      label: '发送者',
      dataType: 'string',
      value: tid ? `$${tid}:out_username` : '{username}',
    },
    {
      ref: '{time}',
      key: 'time 时间',
      label: '时间',
      dataType: 'string',
      value: tid ? `$${tid}:out_time` : '{time}',
    },
    {
      ref: '{args}',
      key: 'args 参数',
      label: '参数',
      dataType: 'string',
      value: tid ? `$${tid}:out_args` : '{args}',
    },
  ];

  return filterBySearch(vars);
});

// 通过 connections 反向遍历找到所有上游节点
const upstreamGroups = computed(() => {
  // 构建邻接表：nodeId → set of upstream nodeIds (通过连线可达)
  const upstream = buildUpstreamMap(props.events, props.connections, props.currentNodeId);

  // 排除 trigger 节点（已在内置变量中列出）
  const triggerId = props.events.find((e) => e.type === 'trigger')?.id;
  const groups: {
    nodeId: string;
    nodeName: string;
    ports: VariableItem[];
  }[] = [];

  for (const [nodeId, _] of upstream) {
    if (nodeId === triggerId) continue;
    if (nodeId === props.currentNodeId) continue;

    const evt = props.events.find((e) => e.id === nodeId);
    if (!evt) continue;

    const outputPorts = (evt.ports || []).filter((p) => p.direction === 'output');
    const items: VariableItem[] = outputPorts
      .map((p) => ({
        ref: `{${evt.name}.${p.name}}`,
        key: `${evt.name} ${p.name} ${p.id} ${nodeId}`,
        label: p.name,
        dataType: p.dataType,
        value: `$${nodeId}:${p.id}`,
      }))
      .filter((item) => !search.value || item.key.includes(search.value.toLowerCase()));

    if (items.length > 0) {
      groups.push({
        nodeId,
        nodeName: evt.name || evt.type,
        ports: items,
      });
    }
  }

  return groups;
});

/** 从当前节点出发，反向遍历 connections 找到所有上游节点 */
function buildUpstreamMap(
  events: WorkflowEvent[],
  connections: FlowConnection[],
  currentNodeId: string
): Map<string, Set<string>> {
  const eventIds = new Set(events.map((e) => e.id));

  // 构建反向邻接表：targetNodeId → [sourceNodeId]
  const reverseAdj = new Map<string, string[]>();
  for (const conn of connections) {
    if (!reverseAdj.has(conn.targetNodeId)) reverseAdj.set(conn.targetNodeId, []);
    reverseAdj.get(conn.targetNodeId)!.push(conn.sourceNodeId);
  }

  // BFS 反向遍历
  const upstream = new Map<string, Set<string>>();
  const visited = new Set<string>([currentNodeId]);
  const queue = [currentNodeId];

  while (queue.length > 0) {
    const nodeId = queue.shift()!;
    const sources = reverseAdj.get(nodeId) || [];
    for (const srcId of sources) {
      if (visited.has(srcId) || !eventIds.has(srcId)) continue;
      visited.add(srcId);

      // 记录直接上游关系
      if (!upstream.has(srcId)) upstream.set(srcId, new Set());
      upstream.get(srcId)!.add(nodeId);

      queue.push(srcId);
    }
  }

  return upstream;
}

function filterBySearch(vars: VariableItem[]): VariableItem[] {
  if (!search.value) return vars;
  const q = search.value.toLowerCase();
  return vars.filter((v) => v.key.includes(q) || v.label.toLowerCase().includes(q));
}

function select(item: VariableItem) {
  emit('select', item);
  emit('close');
}
</script>

<style scoped>
.var-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
}

.var-picker {
  width: 320px;
  max-height: 360px;
  background: var(--strong-background-color, #fff);
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.1));
  box-shadow: var(--shadow-lg, 0 8px 32px rgba(28, 25, 23, 0.12));
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 200;
}

.var-picker__search {
  padding: 8px;
  border-bottom: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.06));
  flex-shrink: 0;
}

.var-picker__input {
  width: 100%;
  padding: 6px 10px;
  font-size: 12px;
  border-radius: var(--radius-xs, 4px);
  border: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.08));
  background: var(--surface-secondary-color, #f4f1ec);
  color: var(--text-color, #1c1917);
  outline: none;
  box-sizing: border-box;
}

.var-picker__input:focus {
  border-color: var(--theme-primary, #5a8f4e);
}

.var-picker__list {
  overflow-y: auto;
  flex: 1;
  padding: 4px 0;
}

.var-picker__group {
  padding: 2px 0;
}

.var-picker__group + .var-picker__group {
  border-top: 1px solid var(--border-subtle-color, rgba(0, 0, 0, 0.04));
}

.var-picker__group-title {
  padding: 4px 12px 2px;
  font-size: 10px;
  font-weight: 600;
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  letter-spacing: 0.3px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.var-picker__group-id {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 9px;
  font-weight: 400;
  color: var(--text-tertiary-color, #a8a29e);
  opacity: 0.6;
}

.var-picker__item {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 5px 12px;
  border: none;
  background: none;
  cursor: pointer;
  text-align: left;
  transition: background 0.1s;
  color: var(--text-color, #1c1917);
}

.var-picker__item:hover,
.var-picker__item--active {
  background: color-mix(in srgb, var(--theme-primary, #5a8f4e) 6%, transparent);
}

.var-picker__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.var-picker__var-key {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 11px;
  color: var(--text-color, #1c1917);
  white-space: nowrap;
}

.var-picker__var-label {
  font-size: 11px;
  color: var(--text-secondary-color, #57534e);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.var-picker__var-type {
  font-size: 9px;
  padding: 1px 5px;
  border-radius: var(--radius-xs, 4px);
  background: var(--surface-tertiary-color, #e8e4de);
  color: var(--text-tertiary-color, #a8a29e);
  text-transform: uppercase;
  flex-shrink: 0;
}

.var-picker__empty {
  padding: 20px 12px;
  text-align: center;
  font-size: 12px;
  color: var(--text-tertiary-color, #a8a29e);
}
</style>
