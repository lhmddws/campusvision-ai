<template>
  <div class="event-panel">
    <!-- ── Header ── -->
    <div class="panel-header">
      <div class="header-left">
        <span class="panel-title">■ 实时 事件 日志</span>
        <span class="event-count-badge">{{ eventStats.total }}</span>
      </div>
      <div class="header-right">
        <div class="mini-chart" v-if="eventStats.total > 0">
          <div
            class="mini-bar entry-bar"
            :style="{ height: barHeight(eventStats.entry) }"
            :title="`entry: ${eventStats.entry}`"
          ></div>
          <div
            class="mini-bar exit-bar"
            :style="{ height: barHeight(eventStats.exit) }"
            :title="`exit: ${eventStats.exit}`"
          ></div>
          <div
            class="mini-bar idle-bar"
            :style="{ height: barHeight(eventStats.idle) }"
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
      <TransitionGroup name="event-slide" tag="div" class="event-items">
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
  background: var(--bg-panel);
  border-left: 1px solid var(--border);
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
  border-bottom: 1px solid var(--border);
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
  font-weight: 400;
  color: var(--text-bright);
  letter-spacing: 0.5px;
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
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text-dim);
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

.mini-bar.entry-bar {
  background: var(--green);
}

.mini-bar.exit-bar {
  background: var(--red);
}

.mini-bar.idle-bar {
  background: var(--amber);
}

/* ── Icon button ── */
.icon-btn {
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-dim);
  width: 26px;
  height: 26px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition);
  line-height: 1;
}

.icon-btn:hover {
  color: var(--text);
  border-color: var(--border-bright);
}

.icon-btn.active {
  color: var(--green);
  border-color: rgba(0, 255, 136, 0.3);
  background: rgba(0, 255, 136, 0.06);
}

/* ── Toolbar ── */
.panel-toolbar {
  display: flex;
  gap: 4px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.tool-btn {
  flex: 1;
  padding: 4px 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-dim);
  font-family: var(--font);
  font-size: 10px;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all var(--transition);
}

.tool-btn:hover {
  background: rgba(0, 255, 136, 0.06);
  border-color: rgba(0, 255, 136, 0.3);
  color: var(--green);
}

.tool-btn:active {
  transform: scale(0.96);
}

.tool-btn.clear-btn:hover {
  border-color: rgba(255, 51, 85, 0.3);
  color: var(--red);
  background: rgba(255, 51, 85, 0.06);
}

/* ── Event List ── */
.event-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
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
  color: var(--text-dim);
  font-size: 12px;
  letter-spacing: 0.5px;
}

/* ── Event Item ── */
.event-item {
  display: flex;
  gap: 8px;
  padding: 6px 12px;
  border-bottom: 1px solid var(--border);
  transition: background var(--transition);
  align-items: flex-start;
}

.event-item:hover {
  background: var(--bg-card-hover);
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
  color: var(--text-dim);
  font-family: var(--font);
  line-height: 1;
  white-space: nowrap;
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
  font-weight: 500;
}

.badge-entry {
  background: var(--green-bg);
  color: var(--green);
  border: 1px solid rgba(0, 255, 136, 0.2);
}

.badge-exit {
  background: var(--red-bg);
  color: var(--red);
  border: 1px solid rgba(255, 51, 85, 0.2);
}

.badge-idle {
  background: var(--amber-bg);
  color: var(--amber);
  border: 1px solid rgba(255, 170, 51, 0.2);
}

.badge-motion {
  background: var(--blue-bg);
  color: var(--blue);
  border: 1px solid rgba(51, 153, 255, 0.2);
}

.badge-stranger {
  background: var(--purple-bg);
  color: var(--purple);
  border: 1px solid rgba(170, 102, 255, 0.2);
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
  color: var(--text);
  line-height: 1.3;
  word-break: break-word;
}

.event-camera {
  font-size: 9px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
}

/* ── Type label on left border ── */
.type-entry { border-left: 2px solid var(--green); }
.type-exit { border-left: 2px solid var(--red); }
.type-idle { border-left: 2px solid var(--amber); }
.type-motion { border-left: 2px solid var(--blue); }
.type-stranger { border-left: 2px solid var(--purple); }

/* ── Slide-in animation ── */
.event-slide-enter-active {
  transition: all 0.3s ease-out;
}

.event-slide-leave-active {
  transition: all 0.2s ease-in;
}

.event-slide-enter-from {
  opacity: 0;
  transform: translateX(-20px);
}

.event-slide-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.event-slide-move {
  transition: transform 0.3s ease;
}
</style>
