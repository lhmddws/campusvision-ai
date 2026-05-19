<template>
  <div class="behavior-panel">
    <!-- Section 1: 行为分析状态 -->
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

    <!-- Section 2: 实时轨迹 -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">🔄</span>
        <span>实时轨迹</span>
      </div>
      <div class="bp-section-body">
        <div v-if="camIds.length === 0" class="bp-empty">暂无摄像头数据</div>
        <div v-else class="bp-traj-list">
          <div
            v-for="camId in camIds"
            :key="camId"
            class="bp-traj-card"
          >
            <div class="bp-traj-header">
              <span class="bp-cam-dot" :style="{ background: getCamColor(camId) }"></span>
              <span class="bp-traj-cam-name">{{ getCamLabel(camId) }}</span>
              <span class="bp-traj-building">{{ getCamBuilding(camId) }}</span>
            </div>

            <!-- ROI line visualization -->
            <div class="bp-roi-bar">
              <div class="bp-roi-track">
                <div
                  class="bp-roi-line"
                  :style="{ left: roiPercent + '%' }"
                ></div>
                <span class="bp-roi-label" :style="{ left: roiPercent + '%' }">ROI</span>
              </div>
            </div>

            <div class="bp-traj-meta">
              <div class="bp-traj-stat">
                <span class="bp-traj-stat-label">活动</span>
                <span class="bp-traj-stat-value" :style="{ color: trajActivityColor(camId) }">
                  {{ getTrajActivity(camId) }}
                </span>
              </div>
              <div class="bp-traj-stat">
                <span class="bp-traj-stat-label">轨迹点</span>
                <span class="bp-traj-stat-value bp-traj-points">{{ getTrajPoints(camId) }}</span>
              </div>
              <div class="bp-traj-stat">
                <span class="bp-traj-stat-label">方向</span>
                <span class="bp-traj-direction" :class="getTrajDirection(camId).cls">
                  {{ getTrajDirection(camId).arrow }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Section 3: 运动检测 -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">📡</span>
        <span>运动检测</span>
      </div>
      <div class="bp-section-body">
        <div v-if="camIds.length === 0" class="bp-empty">暂无摄像头数据</div>
        <div v-else class="bp-motion-list">
          <div
            v-for="camId in camIds"
            :key="camId"
            class="bp-motion-card"
          >
            <div class="bp-motion-top">
              <span class="bp-cam-dot" :style="{ background: getCamColor(camId) }"></span>
              <span class="bp-motion-cam-name">{{ getCamLabel(camId) }}</span>
            </div>

            <!-- Motion level bar -->
            <div class="bp-motion-bar-wrap">
              <div class="bp-motion-bar-track">
                <div
                  class="bp-motion-bar-fill"
                  :style="{ width: getMotionLevel(camId) + '%', background: getMotionColor(camId) }"
                ></div>
                <div
                  class="bp-motion-threshold"
                  :style="{ left: motionThresholdPct + '%' }"
                  title="阈值"
                ></div>
              </div>
              <span class="bp-motion-pct">{{ getMotionLevel(camId) }}%</span>
            </div>

            <div class="bp-motion-meta">
              <div class="bp-motion-tag" :class="motionDetectedStatus(camId).cls">
                {{ motionDetectedStatus(camId).label }}
              </div>
              <div class="bp-motion-tag" :class="dynamicExtractStatus(camId).cls">
                {{ dynamicExtractStatus(camId).label }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Section 4: 行为事件统计 -->
    <div class="bp-section">
      <div class="bp-section-header">
        <span class="bp-section-icon">📈</span>
        <span>行为事件统计</span>
      </div>
      <div class="bp-section-body">
        <div class="bp-event-stats-grid">
          <!-- Event rate gauge -->
          <div class="bp-event-card">
            <div class="bp-event-card-label">事件速率</div>
            <div class="bp-gauge-wrap">
              <svg class="bp-gauge" viewBox="0 0 100 50">
                <path
                  d="M 10 50 A 40 40 0 0 1 90 50"
                  fill="none"
                  stroke="var(--border)"
                  stroke-width="8"
                  stroke-linecap="round"
                />
                <path
                  :d="gaugeArc"
                  fill="none"
                  :stroke="gaugeColor"
                  stroke-width="8"
                  stroke-linecap="round"
                />
              </svg>
              <div class="bp-gauge-value" :style="{ color: gaugeColor }">
                {{ eventRate }}<span class="bp-gauge-unit">/min</span>
              </div>
            </div>
          </div>

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
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  behaviorConfig: { type: Object, default: () => ({}) },
  cameras: { type: Object, default: () => ({}) },
  stats: { type: Object, default: () => ({}) },
})

// ── Helpers ──
const camIds = computed(() => Object.keys(props.cameras))

function getCamColor(camId) {
  return props.cameras[camId]?.color || 'var(--text-dim)'
}

function getCamLabel(camId) {
  return props.cameras[camId]?.label || props.cameras[camId]?.building || camId
}

function getCamBuilding(camId) {
  return props.cameras[camId]?.building || ''
}

const roiPercent = computed(() => {
  const v = props.behaviorConfig.roi_line_x
  return v != null ? Math.min(Math.max(Number(v), 0), 100) : 50
})

const motionThresholdPct = computed(() => {
  const v = props.behaviorConfig.motion_threshold
  if (v == null) return 50
  return Math.min(Math.max(Number(v) * 100, 0), 100)
})

// ── Simulated trajectory state ──
const trajState = ref({})
let trajTimer = null

const activities = ['运动中', '静止', '无人']
const directions = ['entry', 'exit']

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function initTrajState() {
  const state = {}
  camIds.value.forEach(id => {
    state[id] = {
      activity: activities[randomInt(0, 2)],
      points: randomInt(0, 24),
      direction: directions[randomInt(0, 1)],
      motionLevel: randomInt(0, 100),
    }
  })
  trajState.value = state
}

function updateTrajState() {
  const state = { ...trajState.value }
  camIds.value.forEach(id => {
    if (!state[id]) {
      state[id] = {
        activity: activities[randomInt(0, 2)],
        points: randomInt(0, 24),
        direction: directions[randomInt(0, 1)],
        motionLevel: randomInt(0, 100),
      }
      return
    }
    // Slight random walk
    const actRoll = Math.random()
    if (actRoll < 0.15) {
      state[id].activity = activities[randomInt(0, 2)]
    }
    state[id].points = Math.max(0, state[id].points + randomInt(-3, 5))
    if (Math.random() < 0.2) {
      state[id].direction = state[id].direction === 'entry' ? 'exit' : 'entry'
    }
    state[id].motionLevel = Math.min(100, Math.max(0, state[id].motionLevel + randomInt(-15, 15)))
  })
  trajState.value = state
}

function getTrajActivity(camId) {
  return trajState.value[camId]?.activity || '无人'
}

function trajActivityColor(camId) {
  const act = getTrajActivity(camId)
  if (act === '运动中') return 'var(--green)'
  if (act === '静止') return 'var(--amber)'
  return 'var(--text-dim)'
}

function getTrajPoints(camId) {
  return trajState.value[camId]?.points ?? 0
}

function getTrajDirection(camId) {
  const dir = trajState.value[camId]?.direction
  if (dir === 'entry') return { arrow: '→ 进入', cls: 'entry' }
  return { arrow: '← 离开', cls: 'exit' }
}

// ── Simulated motion detection ──
function getMotionLevel(camId) {
  return trajState.value[camId]?.motionLevel ?? 0
}

function getMotionColor(camId) {
  const level = getMotionLevel(camId)
  if (level > 60) return 'var(--green)'
  if (level > 25) return 'var(--amber)'
  return 'var(--text-dim)'
}

function motionDetectedStatus(camId) {
  const level = getMotionLevel(camId)
  if (level > 30) return { label: '已检测', cls: 'detected' }
  if (level > 10) return { label: '微弱', cls: 'weak' }
  return { label: '未检测', cls: 'none' }
}

function dynamicExtractStatus(camId) {
  const enabled = props.behaviorConfig.dynamic_extraction
  const active = getMotionLevel(camId) > 30
  if (!enabled) return { label: '动态提取: 关', cls: 'disabled' }
  return active ? { label: '动态提取: 开', cls: 'active' } : { label: '动态提取: 待机', cls: 'standby' }
}

// ── Event stats ──
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

const eventRate = computed(() => {
  const uptimeSec = props.stats?.uptime_sec || 1
  const rate = (totalEvents.value / Math.max(uptimeSec, 1)) * 60
  return Math.round(rate * 10) / 10
})

const gaugeColor = computed(() => {
  if (eventRate.value > 20) return 'var(--red)'
  if (eventRate.value > 8) return 'var(--amber)'
  return 'var(--green)'
})

const gaugeArc = computed(() => {
  const maxRate = 30
  const pct = Math.min(eventRate.value / maxRate, 1)
  const angle = pct * 180
  const rad = (angle * Math.PI) / 180
  const x = 50 - 40 * Math.cos(rad)
  const y = 50 - 40 * Math.sin(rad)
  const largeArc = angle > 90 ? 1 : 0
  return `M 10 50 A 40 40 0 ${largeArc} 1 ${x.toFixed(2)} ${y.toFixed(2)}`
})

const nightModeActive = computed(() => {
  if (!props.behaviorConfig.night_mode_enabled) return false
  const now = new Date()
  const hour = now.getHours()
  const start = props.behaviorConfig.night_mode_start_hour ?? 22
  const end = props.behaviorConfig.night_mode_end_hour ?? 6
  if (start <= end) {
    return hour >= start && hour < end
  }
  // Crosses midnight
  return hour >= start || hour < end
})

const nightModeRange = computed(() => {
  const s = props.behaviorConfig.night_mode_start_hour ?? 22
  const e = props.behaviorConfig.night_mode_end_hour ?? 6
  return `${String(s).padStart(2, '0')}:00 - ${String(e).padStart(2, '0')}:00`
})

// ── Lifecycle ──
onMounted(() => {
  initTrajState()
  trajTimer = setInterval(updateTrajState, 2500)
})

onUnmounted(() => {
  if (trajTimer) clearInterval(trajTimer)
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

.bp-section-body {
  padding: 10px 12px;
}

.bp-empty {
  text-align: center;
  padding: 20px 0;
  color: var(--text-dim);
  font-size: 11px;
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

/* ── Trajectory cards ── */
.bp-traj-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bp-traj-card {
  padding: 8px 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.bp-traj-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.bp-cam-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.bp-traj-cam-name {
  font-size: 11px;
  color: var(--text-bright);
  font-weight: 500;
}

.bp-traj-building {
  font-size: 9px;
  color: var(--text-dim);
  margin-left: auto;
}

/* ROI bar */
.bp-roi-bar {
  margin-bottom: 6px;
}

.bp-roi-track {
  position: relative;
  height: 16px;
  background: var(--bg-deep);
  border: 1px solid var(--border);
  border-radius: 3px;
  overflow: visible;
}

.bp-roi-line {
  position: absolute;
  top: -2px;
  bottom: -2px;
  width: 2px;
  background: var(--amber);
  box-shadow: 0 0 6px rgba(255,170,51,0.4);
  transform: translateX(-50%);
}

.bp-roi-label {
  position: absolute;
  top: -1px;
  font-size: 8px;
  color: var(--amber);
  transform: translateX(-50%);
  letter-spacing: 0.3px;
}

/* Trajectory meta */
.bp-traj-meta {
  display: flex;
  gap: 12px;
}

.bp-traj-stat {
  display: flex;
  align-items: center;
  gap: 5px;
}

.bp-traj-stat-label {
  font-size: 9px;
  color: var(--text-dim);
}

.bp-traj-stat-value {
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.bp-traj-points {
  color: var(--blue);
}

.bp-traj-direction {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
}

.bp-traj-direction.entry {
  color: var(--green);
  background: rgba(0,255,136,0.08);
}

.bp-traj-direction.exit {
  color: var(--red);
  background: rgba(255,51,85,0.08);
}

/* ── Motion detection ── */
.bp-motion-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.bp-motion-card {
  padding: 8px 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.bp-motion-top {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.bp-motion-cam-name {
  font-size: 11px;
  color: var(--text-bright);
}

/* Motion bar */
.bp-motion-bar-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.bp-motion-bar-track {
  flex: 1;
  position: relative;
  height: 8px;
  background: var(--bg-deep);
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: visible;
}

.bp-motion-bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease, background 0.5s ease;
}

.bp-motion-threshold {
  position: absolute;
  top: -3px;
  bottom: -3px;
  width: 2px;
  background: var(--red);
  opacity: 0.7;
  transform: translateX(-50%);
  pointer-events: none;
}

.bp-motion-pct {
  min-width: 30px;
  text-align: right;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  color: var(--text-dim);
}

/* Motion tags */
.bp-motion-meta {
  display: flex;
  gap: 6px;
}

.bp-motion-tag {
  font-size: 9px;
  padding: 1px 7px;
  border-radius: 3px;
}

.bp-motion-tag.detected {
  color: var(--green);
  background: rgba(0,255,136,0.08);
  border: 1px solid rgba(0,255,136,0.2);
}

.bp-motion-tag.weak {
  color: var(--amber);
  background: rgba(255,170,51,0.08);
  border: 1px solid rgba(255,170,51,0.2);
}

.bp-motion-tag.none {
  color: var(--text-dim);
  background: transparent;
  border: 1px solid var(--border);
}

.bp-motion-tag.active {
  color: var(--green);
  background: rgba(0,255,136,0.08);
  border: 1px solid rgba(0,255,136,0.2);
}

.bp-motion-tag.standby {
  color: var(--blue);
  background: rgba(51,153,255,0.08);
  border: 1px solid rgba(51,153,255,0.2);
}

.bp-motion-tag.disabled {
  color: var(--text-dim);
  background: transparent;
  border: 1px solid var(--border);
}

/* ── Event stats grid ── */
.bp-event-stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
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

/* Gauge */
.bp-gauge-wrap {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.bp-gauge {
  width: 100%;
  max-width: 100px;
  height: auto;
}

.bp-gauge-value {
  position: absolute;
  font-size: 18px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  bottom: 4px;
  display: flex;
  align-items: baseline;
  gap: 2px;
}

.bp-gauge-unit {
  font-size: 9px;
  font-weight: 400;
  color: var(--text-dim);
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
