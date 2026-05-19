<template>
  <div class="stats-panel" v-show="true">
    <!-- ── 系统概况 ── -->
    <div class="stats-section">
      <div class="section-title">系统概况</div>
      <div class="kv-grid">
        <div class="dash-kv">
          <span class="kv-label">运行时长</span>
          <span class="kv-value">{{ uptime || '--:--:--' }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">总帧数</span>
          <span class="kv-value">{{ stats.frames_generated ?? 0 }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">总事件</span>
          <span class="kv-value">{{ stats.events_total ?? 0 }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">摄像头</span>
          <span class="kv-value">{{ stats.active_cameras ?? cameraCount }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">Kafka</span>
          <span class="kv-value" :class="kafkaConnected ? 'status-ok' : 'status-err'">
            {{ kafkaConnected ? '已连接' : '断开' }}
          </span>
        </div>
      </div>
    </div>

    <!-- ── 事件类型 ── -->
    <div class="stats-section">
      <div class="section-title">事件类型</div>
      <div class="kv-grid">
        <div class="dash-kv type-entry-box">
          <span class="kv-label">进入 (entry)</span>
          <span class="kv-value type-entry-val">{{ stats.event_type_counts?.entry ?? 0 }}</span>
        </div>
        <div class="dash-kv type-exit-box">
          <span class="kv-label">离开 (exit)</span>
          <span class="kv-value type-exit-val">{{ stats.event_type_counts?.exit ?? 0 }}</span>
        </div>
        <div class="dash-kv type-idle-box">
          <span class="kv-label">徘徊 (idle)</span>
          <span class="kv-value type-idle-val">{{ stats.event_type_counts?.idle ?? 0 }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">事件/分钟</span>
          <span class="kv-value">{{ stats.events_per_min ?? 0 }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">峰值/分钟</span>
          <span class="kv-value">{{ stats.peak_events_per_min ?? 0 }}</span>
        </div>
      </div>
    </div>

    <!-- ── Kafka 吞吐 ── -->
    <div class="stats-section">
      <div class="section-title">Kafka 吞吐</div>
      <div class="kv-grid">
        <div class="dash-kv">
          <span class="kv-label">已发送帧</span>
          <span class="kv-value">{{ stats.kafka_frames_sent ?? 0 }}</span>
        </div>
        <div class="dash-kv">
          <span class="kv-label">已发送事件</span>
          <span class="kv-value">{{ stats.kafka_events_sent ?? 0 }}</span>
        </div>
      </div>
    </div>

    <!-- ── 各摄像头统计 ── -->
    <div class="stats-section">
      <div class="section-title">各摄像头统计</div>
      <div class="camera-stats">
        <div v-if="cameraKeys.length === 0" class="no-cameras">暂无摄像头数据</div>
        <div
          v-for="cam in cameraKeys"
          :key="cam"
          class="cam-row"
        >
          <span class="cam-name">{{ cameras[cam]?.name || cam }}</span>
          <span class="cam-count">{{ getCameraEventCount(cam) }}</span>
          <div class="cam-bar-track">
            <div
              class="cam-bar-fill"
              :style="{ width: cameraBarWidth(cam) }"
            ></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  stats: { type: Object, default: () => ({}) },
  cameras: { type: Object, default: () => ({}) },
  uptime: { type: String, default: '' },
  kafkaConnected: { type: Boolean, default: false },
})

const cameraKeys = computed(() => {
  return Object.keys(props.cameras)
})

const cameraCount = computed(() => {
  return cameraKeys.value.length
})

function getCameraEventCount(cameraId) {
  if (props.stats.camera_event_counts) {
    return props.stats.camera_event_counts[cameraId] ?? 0
  }
  return 0
}

const maxCamEvents = computed(() => {
  const counts = props.stats.camera_event_counts || {}
  const vals = Object.values(counts)
  return vals.length > 0 ? Math.max(...vals, 1) : 1
})

function cameraBarWidth(cameraId) {
  const count = getCameraEventCount(cameraId)
  const max = maxCamEvents.value
  const pct = (count / max) * 100
  return `${Math.max(pct, 0)}%`
}
</script>

<style scoped>
.stats-panel {
  width: 280px;
  min-width: 240px;
  background: var(--bg-panel);
  border-left: 1px solid var(--border);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

/* ── Section ── */
.stats-section {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}

.section-title {
  font-size: 10px;
  font-weight: 400;
  color: var(--text-dim);
  letter-spacing: 1px;
  text-transform: uppercase;
  margin-bottom: 8px;
}

/* ── KV Grid ── */
.kv-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
  gap: 6px;
}

.dash-kv {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.kv-label {
  font-size: 9px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
  text-transform: uppercase;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kv-value {
  font-size: 16px;
  font-weight: 500;
  color: var(--text-bright);
  font-family: var(--font);
  line-height: 1.2;
}

/* ── Status colors ── */
.status-ok {
  color: var(--green) !important;
}

.status-err {
  color: var(--red) !important;
}

/* ── Event type colored cards ── */
.type-entry-box {
  border-left: 2px solid var(--green);
}

.type-entry-val {
  color: var(--green) !important;
}

.type-exit-box {
  border-left: 2px solid var(--red);
}

.type-exit-val {
  color: var(--red) !important;
}

.type-idle-box {
  border-left: 2px solid var(--amber);
}

.type-idle-val {
  color: var(--amber) !important;
}

/* ── Camera stats ── */
.camera-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.no-cameras {
  font-size: 11px;
  color: var(--text-dim);
  text-align: center;
  padding: 12px 0;
}

.cam-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 4px;
}

.cam-name {
  font-size: 11px;
  color: var(--text);
  min-width: 48px;
  flex-shrink: 0;
}

.cam-count {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-bright);
  min-width: 28px;
  text-align: right;
  font-family: var(--font);
}

.cam-bar-track {
  flex: 1;
  height: 4px;
  background: var(--bg-deep);
  border-radius: 2px;
  overflow: hidden;
}

.cam-bar-fill {
  height: 100%;
  background: var(--green);
  border-radius: 2px;
  transition: width 0.4s ease;
  min-width: 2px;
}
</style>
