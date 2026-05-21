<template>
  <div class="event-panel">
    <!-- ── Header ── -->
    <div class="panel-header">
      <div class="header-left">
        <span class="panel-title">实时事件日志</span>
        <span class="event-count-badge">{{ eventStats.total }}</span>
      </div>
      <div class="header-right">
        <div class="mini-chart" v-if="eventStats.total > 0">
          <div
            class="mini-bar"
            :style="{ height: barHeight(eventStats.entry), background: 'var(--color-success)' }"
            :title="`entry: ${eventStats.entry}`"
          ></div>
          <div
            class="mini-bar"
            :style="{ height: barHeight(eventStats.exit), background: 'var(--text-muted)' }"
            :title="`exit: ${eventStats.exit}`"
          ></div>
          <div
            class="mini-bar"
            :style="{ height: barHeight(eventStats.idle), background: 'var(--color-warning)' }"
            :title="`idle: ${eventStats.idle}`"
          ></div>
        </div>
        <button class="icon-btn" @click="$emit('toggle-stats')" :class="{ active: statsVisible }" title="切换统计面板">
          📊
        </button>
      </div>
    </div>

    <!-- ── Toolbar ── -->
    <div class="panel-toolbar">
      <button class="tool-btn" @click="$emit('simulate-random')">随机</button>
      <button class="tool-btn" @click="$emit('simulate-preset', 'rush_hour')">高峰</button>
      <button class="tool-btn" @click="$emit('simulate-preset', 'night')">夜间</button>
      <button class="tool-btn clear-btn" @click="$emit('clear-events')">清空</button>
    </div>

    <!-- ── Event List ── -->
    <div class="event-list" ref="listRef">
      <div v-if="events.length === 0" class="empty-state">
        等待事件数据...
      </div>
      <TransitionGroup name="event-fade" tag="div" class="event-items">
        <div
          v-for="(evt, idx) in visibleEvents"
          :key="evt.id || evt.time || idx"
          class="event-item"
          :class="[`type-${evt.event_type}`]"
        >
          <div class="event-left">
            <span class="event-time">{{ formatTime(evt.time) }}</span>
            <span class="event-badge" :class="`badge-${evt.event_type}`">
              {{ badgeLabel(evt.event_type) }}
            </span>
          </div>
          <div class="event-right">
            <span class="event-detail">{{ evt.detail || evt.event_type }}</span>
            <span class="event-camera">{{ evt.camera_id || evt.building || '' }}</span>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  events: { type: Array, default: () => [] },
  eventStats: { type: Object, default: () => ({ entry: 0, exit: 0, idle: 0, total: 0 }) },
  statsVisible: { type: Boolean, default: false },
})

defineEmits(['simulate-random', 'simulate-preset', 'clear-events', 'toggle-stats'])

const listRef = ref(null)

const MAX_DISPLAY = 200

const visibleEvents = computed(() => {
  return props.events.slice(0, MAX_DISPLAY)
})

function formatTime(ts) {
  if (!ts) return '--:--:--'
  const d = typeof ts === 'number' || typeof ts === 'string'
    ? new Date(ts)
    : ts instanceof Date ? ts : new Date()
  if (Number.isNaN(d.getTime())) return '--:--:--'
  return d.toTimeString().slice(0, 8)
}

function badgeLabel(type) {
  const map = {
    entry: '入',
    exit: '出',
    idle: '闲',
    motion: '动',
    stranger: '陌',
  }
  return map[type] || type
}

function barHeight(count) {
  const max = Math.max(props.eventStats.entry, props.eventStats.exit, props.eventStats.idle, 1)
  const pct = (count / max) * 100
  return pct < 4 && count > 0 ? '4px' : `${Math.max(pct, 0)}%`
}
</script>

<style scoped>
.event-panel {
  width: 320px;
  min-width: 280px;
  background: var(--bg-card);
  border-left: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  flex-shrink: 0;
}

/* ── Header ── */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
  gap: 8px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.panel-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.3px;
  white-space: nowrap;
}

.event-count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 18px;
  padding: 0 6px;
  border-radius: 9px;
  background: var(--bg-page);
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  font-size: 10px;
  font-family: var(--font);
  line-height: 1;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

/* ── Mini bar chart ── */
.mini-chart {
  display: flex;
  align-items: flex-end;
  gap: 2px;
  height: 20px;
}

.mini-bar {
  width: 6px;
  border-radius: 2px 2px 0 0;
  min-height: 2px;
  transition: height 0.3s ease;
}

/* ── Icon button ── */
.icon-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  width: 26px;
  height: 26px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-fast);
  line-height: 1;
}

.icon-btn:hover {
  color: var(--text-primary);
  border-color: var(--border-color);
  background: var(--bg-hover);
}

.icon-btn.active {
  color: var(--color-primary);
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

/* ── Toolbar ── */
.panel-toolbar {
  display: flex;
  gap: 4px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.tool-btn {
  flex: 1;
  padding: 4px 0;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-family: var(--font);
  font-size: 10px;
  letter-spacing: 0.3px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tool-btn:hover {
  background: var(--bg-hover);
  border-color: var(--border-color);
  color: var(--text-primary);
}

.tool-btn:active {
  transform: scale(0.96);
}

.tool-btn.clear-btn:hover {
  color: var(--color-danger);
  border-color: var(--color-danger);
  background: var(--color-danger-bg);
}

/* ── Event List ── */
.event-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.event-list::-webkit-scrollbar {
  width: 5px;
}

.event-list::-webkit-scrollbar-track {
  background: transparent;
}

.event-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.event-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-muted);
}

.event-items {
  display: flex;
  flex-direction: column;
}

/* ── Empty State ── */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 120px;
  color: var(--text-muted);
  font-size: 12px;
  letter-spacing: 0.3px;
}

/* ── Event Item ── */
.event-item {
  display: flex;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-light);
  transition: background var(--transition-fast);
  align-items: flex-start;
  background: var(--bg-card);
}

.event-item:nth-child(even) {
  background: var(--bg-page);
}

.event-item:hover {
  background: var(--bg-hover);
}

.event-left {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  flex-shrink: 0;
  min-width: 56px;
}

.event-time {
  font-size: 10px;
  color: var(--text-muted);
  font-family: var(--font);
  line-height: 1;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.event-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 18px;
  border-radius: 3px;
  font-size: 10px;
  font-family: var(--font);
  line-height: 1;
  font-weight: 600;
  border: none;
}

/* ── Flat style badges with semantic colors ── */
.badge-entry {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.badge-exit {
  background: var(--bg-hover);
  color: var(--text-muted);
}

.badge-idle {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.badge-motion {
  background: var(--color-primary-bg);
  color: var(--color-primary-light);
}

.badge-stranger {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.event-right {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.event-detail {
  font-size: 11px;
  color: var(--text-primary);
  line-height: 1.3;
  word-break: break-word;
}

.event-camera {
  font-size: 9px;
  color: var(--text-muted);
  letter-spacing: 0.3px;
}

/* ── Type label on left border ── */
.type-entry { border-left: 2px solid var(--color-success); }
.type-exit { border-left: 2px solid var(--text-muted); }
.type-idle { border-left: 2px solid var(--color-warning); }
.type-motion { border-left: 2px solid var(--color-primary-light); }
.type-stranger { border-left: 2px solid var(--color-danger); }

/* ── Simple fade transition ── */
.event-fade-enter-active {
  transition: all 0.25s ease-out;
}

.event-fade-leave-active {
  transition: all 0.2s ease-in;
}

.event-fade-enter-from {
  opacity: 0;
  transform: translateY(-6px);
}

.event-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.event-fade-move {
  transition: transform 0.25s ease;
}
</style>
