<template>
  <div class="stats-panel" v-show="true">
    <!-- ── 系统概况 ── -->
    <div class="stats-section">
      <div class="section-title">系统概况</div>
      <div class="kp-grid">
        <div class="kp-card">
          <div class="kp-value">{{ uptime || '--:--:--' }}</div>
          <div class="kp-label">运行时长</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.frames_generated ?? 0 }}</div>
          <div class="kp-label">总帧数</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.events_total ?? 0 }}</div>
          <div class="kp-label">总事件</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.active_cameras ?? cameraCount }}</div>
          <div class="kp-label">摄像头</div>
        </div>
        <div class="kp-card" :class="kafkaConnected ? 'kp-accent-green' : 'kp-accent-red'">
          <div class="kp-value" :class="kafkaConnected ? 'text-green' : 'text-red'">
            {{ kafkaConnected ? '已连接' : '断开' }}
          </div>
          <div class="kp-label">Kafka 状态</div>
        </div>
      </div>
    </div>

    <!-- ── 事件类型 ── -->
    <div class="stats-section">
      <div class="section-title">事件类型</div>
      <div class="kp-grid">
        <div class="kp-card kp-accent-green">
          <div class="kp-value text-green">{{ stats.event_type_counts?.entry ?? 0 }}</div>
          <div class="kp-label">进入 (entry)</div>
        </div>
        <div class="kp-card kp-accent-red">
          <div class="kp-value text-red">{{ stats.event_type_counts?.exit ?? 0 }}</div>
          <div class="kp-label">离开 (exit)</div>
        </div>
        <div class="kp-card kp-accent-amber">
          <div class="kp-value text-amber">{{ stats.event_type_counts?.idle ?? 0 }}</div>
          <div class="kp-label">徘徊 (idle)</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.events_per_min ?? 0 }}</div>
          <div class="kp-label">事件/分钟</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.peak_events_per_min ?? 0 }}</div>
          <div class="kp-label">峰值/分钟</div>
        </div>
      </div>
    </div>

    <!-- ── Kafka 吞吐 ── -->
    <div class="stats-section">
      <div class="section-title">Kafka 吞吐</div>
      <div class="kp-grid">
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.kafka_frames_sent ?? 0 }}</div>
          <div class="kp-label">已发送帧</div>
        </div>
        <div class="kp-card kp-accent-blue">
          <div class="kp-value">{{ stats.kafka_events_sent ?? 0 }}</div>
          <div class="kp-label">已发送事件</div>
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
/* ── Panel ── */
.stats-panel {
  width: 280px;
  min-width: 240px;
  background: var(--bg-card);
  border-left: 1px solid var(--border-color);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

/* ── Section ── */
.stats-section {
  padding: 14px 14px;
  border-bottom: 1px solid var(--border-color);
}

.section-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 10px;
}

/* ── KPI Grid ── */
.kp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 8px;
}

.kp-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  box-shadow: var(--shadow-sm);
  position: relative;
}

/* ── Color accent bars (left) ── */
.kp-accent-blue {
  border-left: 4px solid var(--color-primary);
}
.kp-accent-green {
  border-left: 4px solid var(--color-success);
}
.kp-accent-red {
  border-left: 4px solid var(--color-danger);
}
.kp-accent-amber {
  border-left: 4px solid var(--color-warning);
}

.kp-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
  font-family: var(--font);
}

.kp-label {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Text color helpers ── */
.text-green {
  color: var(--color-success) !important;
}
.text-red {
  color: var(--color-danger) !important;
}
.text-amber {
  color: var(--color-warning) !important;
}

/* ── Camera stats ── */
.camera-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.no-cameras {
  font-size: 12px;
  color: var(--text-muted);
  text-align: center;
  padding: 16px 0;
}

.cam-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
}

.cam-name {
  font-size: 12px;
  color: var(--text-secondary);
  min-width: 44px;
  flex-shrink: 0;
  font-weight: 500;
}

.cam-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  min-width: 28px;
  text-align: right;
  font-family: var(--font);
}

.cam-bar-track {
  flex: 1;
  height: 4px;
  background: var(--bg-hover);
  border-radius: 2px;
  overflow: hidden;
}

.cam-bar-fill {
  height: 100%;
  background: var(--color-primary-light);
  border-radius: 2px;
  transition: width 0.4s ease;
  min-width: 2px;
}
</style>
