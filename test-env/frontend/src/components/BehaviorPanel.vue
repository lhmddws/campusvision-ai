<template>
  <div class="behavior-panel">
    <!-- ── Section 1: 行为分析状态 ── -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">⚡</span>
        <span>行为分析状态</span>
      </div>
      <div class="bp-section-body">
        <div class="bp-status-bar">
          <div class="bp-status-badge" :class="behaviorConfig.enabled ? 'active' : 'inactive'">
            <span class="bp-status-dot"></span>
            <span>{{ behaviorConfig.enabled ? '运行中' : '已停止' }}</span>
          </div>
        </div>
        <div class="bp-config-grid">
          <div class="bp-config-item">
            <span class="bp-config-label">ROI 线位置</span>
            <span class="bp-config-value">{{ behaviorConfig.roi_line_x ?? '—' }}%</span>
          </div>
          <div class="bp-config-item">
            <span class="bp-config-label">最小跟踪点</span>
            <span class="bp-config-value">{{ behaviorConfig.min_track_points ?? '—' }}</span>
          </div>
          <div class="bp-config-item">
            <span class="bp-config-label">运动阈值</span>
            <span class="bp-config-value">{{ behaviorConfig.motion_threshold ?? '—' }}</span>
          </div>
          <div class="bp-config-item">
            <span class="bp-config-label">动态提取</span>
            <span class="bp-config-value" :style="{ color: behaviorConfig.dynamic_extraction ? 'var(--green)' : 'var(--text-dim)' }">
              {{ behaviorConfig.dynamic_extraction ? '已启用' : '已禁用' }}
            </span>
          </div>
          <div class="bp-config-item">
            <span class="bp-config-label">去重窗口</span>
            <span class="bp-config-value">{{ behaviorConfig.dedup_window_seconds ?? '—' }}s</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Section 2: 行为事件 (SSE) ── -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">🔍</span>
        <span>行为事件</span>
        <span v-if="behaviorEvents.length" class="bp-section-count">{{ behaviorEvents.length }}</span>
      </div>
      <div class="bp-section-body">
        <div v-if="behaviorEvents.length === 0" class="bp-empty">暂无行为事件数据</div>
        <div v-else class="bp-events-two-col">
          <!-- Left: Event log -->
          <div class="bp-events-log">
            <div
              v-for="evt in visibleBehaviorEvents"
              :key="evt.id"
              class="bp-behavior-event"
              :class="'severity-' + severityMap[evt.event_type]"
            >
              <div class="bp-bevent-left">
                <span class="bp-bevent-time">{{ formatTime(evt.timestamp) }}</span>
                <span class="bp-bevent-badge" :style="{ background: typeColor(evt.event_type, 0.12), color: typeColor(evt.event_type) }">
                  {{ typeLabel(evt.event_type) }}
                </span>
              </div>
              <div class="bp-bevent-right">
                <span class="bp-bevent-detail">{{ evt.detail || evt.event_type }}</span>
                <span class="bp-bevent-camera">{{ getCamLabel(evt.camera_id) }}</span>
              </div>
            </div>
          </div>
          <!-- Right: Trend chart -->
          <div class="bp-events-chart">
            <div class="bp-chart-title">行为类型分布</div>
            <div class="bp-chart-bars">
              <div v-for="item in chartData" :key="item.type" class="bp-chart-row">
                <div class="bp-chart-row-label">{{ item.label }}</div>
                <div class="bp-chart-row-track">
                  <div
                    class="bp-chart-row-fill"
                    :style="{ width: item.pct + '%', background: item.color }"
                  ></div>
                </div>
                <div class="bp-chart-row-value">{{ item.count }}</div>
              </div>
            </div>
            <div v-if="chartData.every(c => c.count === 0)" class="bp-chart-empty">等待事件...</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Section 3: 行为统计 ── -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">📈</span>
        <span>行为统计</span>
      </div>
      <div class="bp-section-body">
        <div class="bp-event-stats-grid">
          <!-- Entry/Exit ratio -->
          <div class="bp-event-card">
            <div class="bp-event-card-label">出入比</div>
            <div class="bp-ratio-display">
              <div class="bp-ratio-bar">
                <div
                  class="bp-ratio-fill bp-ratio-entry"
                  :style="{ width: entryRatioPct + '%' }"
                ></div>
                <div
                  class="bp-ratio-fill bp-ratio-exit"
                  :style="{ width: exitRatioPct + '%' }"
                ></div>
              </div>
              <div class="bp-ratio-labels">
                <span class="bp-ratio-entry-label">
                  <span class="bp-ratio-dot entry"></span>
                  入 {{ entryCount }}
                </span>
                <span class="bp-ratio-exit-label">
                  <span class="bp-ratio-dot exit"></span>
                  出 {{ exitCount }}
                </span>
              </div>
            </div>
          </div>

          <!-- Night mode -->
          <div class="bp-event-card">
            <div class="bp-event-card-label">夜间模式</div>
            <div class="bp-night-status">
              <span class="bp-night-badge" :class="nightModeActive ? 'on' : 'off'">
                <span class="bp-night-dot"></span>
                {{ nightModeActive ? '进行中' : '未激活' }}
              </span>
              <div class="bp-night-range" v-if="behaviorConfig.night_mode_enabled">
                <span class="bp-night-range-icon">🌙</span>
                <span>{{ nightModeRange }}</span>
              </div>
            </div>
          </div>

          <!-- Stranger alerts -->
          <div class="bp-event-card">
            <div class="bp-event-card-label">陌生人告警</div>
            <div class="bp-stranger-status">
              <span class="bp-stranger-badge" :class="behaviorConfig.stranger_alert_enabled ? 'enabled' : 'disabled'">
                {{ behaviorConfig.stranger_alert_enabled ? '已启用' : '已禁用' }}
              </span>
              <div v-if="behaviorConfig.stranger_alert_enabled" class="bp-stranger-detail">
                <span class="bp-stranger-threshold">
                  阈值: {{ behaviorConfig.stranger_alert_threshold ?? '—' }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  behaviorConfig: { type: Object, default: () => ({}) },
  cameras: { type: Object, default: () => ({}) },
  stats: { type: Object, default: () => ({}) },
  behaviorEvents: { type: Array, default: () => [] },
})

// ── Camera helpers ──
function getCamLabel(camId) {
  return props.cameras[camId]?.label || props.cameras[camId]?.building || camId
}

// ── Behavior event types ──
const EVENT_TYPES = {
  loiter:  { label: '滞留', color: 'var(--amber)' },
  running: { label: '奔跑', color: 'var(--red)' },
  crowd:   { label: '聚集', color: '#e67e22' },
  zone:    { label: '入侵', color: 'var(--red)' },
}

const severityMap = {
  loiter: 'warning',
  running: 'critical',
  crowd: 'warning',
  zone: 'critical',
}

function typeLabel(type) {
  return EVENT_TYPES[type]?.label || type
}

function typeColor(type, alpha) {
  const raw = EVENT_TYPES[type]?.color || 'var(--text-dim)'
  if (alpha == null) return raw
  // Convert CSS variable reference to rgba string
  const map = {
    'var(--amber)': `rgba(217,119,6,${alpha})`,
    'var(--red)': `rgba(220,38,38,${alpha})`,
    '#e67e22': `rgba(230,126,34,${alpha})`,
  }
  return map[raw] || raw
}

const visibleBehaviorEvents = computed(() => {
  return props.behaviorEvents.slice(0, 100)
})

const chartData = computed(() => {
  const counts = {}
  props.behaviorEvents.forEach(e => {
    counts[e.event_type] = (counts[e.event_type] || 0) + 1
  })
  const maxCount = Math.max(...Object.values(counts), 1)
  return Object.keys(EVENT_TYPES).map(type => ({
    type,
    label: EVENT_TYPES[type].label,
    count: counts[type] || 0,
    pct: ((counts[type] || 0) / maxCount) * 100,
    color: EVENT_TYPES[type].color,
  }))
})

function formatTime(ts) {
  if (!ts) return '--:--:--'
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return '--:--:--'
  return d.toTimeString().slice(0, 8)
}

// ── Entry/Exit stats (from props) ──
const entryCount = computed(() => {
  return props.stats?.event_type_counts?.entry ?? 0
})

const exitCount = computed(() => {
  return props.stats?.event_type_counts?.exit ?? 0
})

const totalEvents = computed(() => {
  return entryCount.value + exitCount.value
})

const entryRatioPct = computed(() => {
  if (totalEvents.value === 0) return 50
  return (entryCount.value / totalEvents.value) * 100
})

const exitRatioPct = computed(() => {
  if (totalEvents.value === 0) return 50
  return (exitCount.value / totalEvents.value) * 100
})

// ── Night mode ──
const nightModeActive = computed(() => {
  if (!props.behaviorConfig.night_mode_enabled) return false
  const now = new Date()
  const hour = now.getHours()
  const start = props.behaviorConfig.night_mode_start_hour ?? 22
  const end = props.behaviorConfig.night_mode_end_hour ?? 6
  if (start <= end) {
    return hour >= start && hour < end
  }
  return hour >= start || hour < end
})

const nightModeRange = computed(() => {
  const s = props.behaviorConfig.night_mode_start_hour ?? 22
  const e = props.behaviorConfig.night_mode_end_hour ?? 6
  return `${String(s).padStart(2, '0')}:00 - ${String(e).padStart(2, '0')}:00`
})


</script>

<style scoped>
.behavior-panel {
  padding: 8px 14px 14px;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* ── Sections ── */
.bp-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.bp-section-header {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 8px 12px;
  font-size: 11px;
  color: var(--text-bright);
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
  user-select: none;
}

.bp-section-icon {
  font-size: 12px;
}

.bp-section-count {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 16px;
  padding: 0 5px;
  border-radius: 8px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  color: var(--text-dim);
  font-size: 9px;
  font-variant-numeric: tabular-nums;
}

.bp-section-body {
  padding: 10px 12px;
}

.bp-empty {
  text-align: center;
  padding: 32px 0;
  color: var(--text-dim);
  font-size: 11px;
  letter-spacing: 0.3px;
}

/* ── Status badge ── */
.bp-status-bar {
  margin-bottom: 10px;
}

.bp-status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 14px;
  border-radius: 14px;
  font-size: 11px;
  letter-spacing: 0.5px;
}

.bp-status-badge.active {
  background: rgba(0,255,136,0.1);
  border: 1px solid rgba(0,255,136,0.25);
  color: var(--green);
}

.bp-status-badge.inactive {
  background: rgba(255,255,255,0.03);
  border: 1px solid var(--border);
  color: var(--text-dim);
}

.bp-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.bp-status-badge.active .bp-status-dot {
  background: var(--green);
  box-shadow: 0 0 6px rgba(0,255,136,0.4);
}

.bp-status-badge.inactive .bp-status-dot {
  background: var(--gray);
}

/* ── Config grid ── */
.bp-config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 6px;
}

.bp-config-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 8px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.bp-config-label {
  font-size: 9px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
}

.bp-config-value {
  font-size: 12px;
  color: var(--text-bright);
  font-variant-numeric: tabular-nums;
}

/* ── Events two-column layout ── */
.bp-events-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  min-height: 140px;
}

.bp-events-log {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 260px;
  overflow-y: auto;
  padding-right: 4px;
}

.bp-events-log::-webkit-scrollbar {
  width: 4px;
}

.bp-events-log::-webkit-scrollbar-track {
  background: transparent;
}

.bp-events-log::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 2px;
}

/* ── Behavior event item ── */
.bp-behavior-event {
  display: flex;
  gap: 8px;
  padding: 5px 8px;
  border-radius: var(--radius-sm);
  border-left: 2px solid transparent;
  background: var(--bg-panel);
  transition: background var(--transition-fast);
}

.bp-behavior-event:hover {
  background: var(--bg-hover);
}

.bp-behavior-event.severity-warning {
  border-left-color: var(--amber);
}

.bp-behavior-event.severity-critical {
  border-left-color: var(--red);
}

.bp-bevent-left {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  flex-shrink: 0;
  min-width: 42px;
}

.bp-bevent-time {
  font-size: 9px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
  line-height: 1;
  white-space: nowrap;
}

.bp-bevent-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 9px;
  font-weight: 600;
  line-height: 1.4;
  white-space: nowrap;
}

.bp-bevent-right {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.bp-bevent-detail {
  font-size: 10px;
  color: var(--text-bright);
  line-height: 1.3;
  word-break: break-word;
}

.bp-bevent-camera {
  font-size: 9px;
  color: var(--text-dim);
}

/* ── Chart ── */
.bp-events-chart {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 6px 8px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.bp-chart-title {
  font-size: 10px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
}

.bp-chart-bars {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.bp-chart-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bp-chart-row-label {
  font-size: 10px;
  color: var(--text-bright);
  min-width: 28px;
  flex-shrink: 0;
}

.bp-chart-row-track {
  flex: 1;
  height: 12px;
  background: var(--bg-deep);
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
}

.bp-chart-row-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.4s ease;
  min-width: 0;
}

.bp-chart-row-value {
  font-size: 10px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
  min-width: 20px;
  text-align: right;
}

.bp-chart-empty {
  text-align: center;
  color: var(--text-dim);
  font-size: 10px;
  padding: 16px 0;
}

/* ── Event stats grid ── */
.bp-event-stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.bp-event-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.bp-event-card-label {
  font-size: 10px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
}

/* Ratio */
.bp-ratio-display {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bp-ratio-bar {
  display: flex;
  height: 8px;
  background: var(--bg-deep);
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
}

.bp-ratio-fill {
  height: 100%;
  transition: width 0.5s ease;
}

.bp-ratio-entry {
  background: var(--green);
}

.bp-ratio-exit {
  background: var(--red);
}

.bp-ratio-labels {
  display: flex;
  justify-content: space-between;
  font-size: 9px;
}

.bp-ratio-entry-label {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--green);
}

.bp-ratio-exit-label {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--red);
}

.bp-ratio-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.bp-ratio-dot.entry {
  background: var(--green);
}

.bp-ratio-dot.exit {
  background: var(--red);
}

/* Night mode */
.bp-night-status {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bp-night-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 10px;
  align-self: flex-start;
}

.bp-night-badge.on {
  background: rgba(51,153,255,0.1);
  border: 1px solid rgba(51,153,255,0.25);
  color: var(--blue);
}

.bp-night-badge.off {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
}

.bp-night-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
}

.bp-night-badge.on .bp-night-dot {
  background: var(--blue);
  box-shadow: 0 0 6px rgba(51,153,255,0.4);
}

.bp-night-badge.off .bp-night-dot {
  background: var(--gray);
}

.bp-night-range {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-dim);
}

.bp-night-range-icon {
  font-size: 10px;
}

/* Stranger alerts */
.bp-stranger-status {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bp-stranger-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 10px;
  align-self: flex-start;
}

.bp-stranger-badge.enabled {
  background: rgba(255,170,51,0.1);
  border: 1px solid rgba(255,170,51,0.25);
  color: var(--amber);
}

.bp-stranger-badge.disabled {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-dim);
}

.bp-stranger-detail {
  font-size: 9px;
  color: var(--text-dim);
}

.bp-stranger-threshold {
  font-variant-numeric: tabular-nums;
}
</style>
