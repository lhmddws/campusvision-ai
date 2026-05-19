<template>
  <footer class="status-bar">
    <!-- Left: status info -->
    <div class="status-left">
      <div class="kafka-indicator" :class="{ connected: kafkaConnected }">
        <span class="status-dot"></span>
        <span class="status-label">Kafka</span>
        <span class="status-text">{{ kafkaConnected ? '已连接' : '未连接' }}</span>
      </div>

      <span class="sep">|</span>

      <div class="uptime-display">
        <span class="info-label">运行</span>
        <span>{{ uptime }}</span>
      </div>

      <span class="sep">|</span>

      <div class="event-count">
        <span class="info-label">已生成:</span>
        <span class="count-num">{{ eventCount }}</span>
        <span class="info-label">个事件</span>
      </div>

      <!-- Camera frame stats -->
      <template v-if="stats && stats.frames_total !== undefined">
        <span class="sep">|</span>
        <div class="frame-count">
          <span class="info-label">帧:</span>
          <span class="count-num">{{ stats.frames_total }}</span>
        </div>
      </template>
    </div>

    <!-- Right: actions -->
    <div class="status-right">
      <button class="gen-btn" @click="$emit('simulate-preset', 'all_entry')">全部进入</button>
      <button class="gen-btn" @click="$emit('simulate-preset', 'all_exit')">全部离开</button>
      <button class="gen-btn" @click="$emit('clear-events')">清空日志</button>

      <select
        class="scenario-select"
        v-model="selectedScenario"
        @change="handleScenario"
      >
        <option value="" disabled>场景预设...</option>
        <option value="rush_hour">高峰时段</option>
        <option value="night_mode">夜间模式</option>
        <option value="stranger">陌生人闯入</option>
      </select>

      <button class="gen-btn primary" @click="$emit('simulate-random')">🚀 流量</button>
    </div>
  </footer>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  kafkaConnected: { type: Boolean, default: false },
  uptime: { type: String, default: '00:00:00' },
  eventCount: { type: Number, default: 0 },
  cameras: { type: Object, default: () => ({}) },
  stats: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['simulate-random', 'simulate-preset', 'clear-events'])

const selectedScenario = ref('')

function handleScenario() {
  if (selectedScenario.value) {
    emit('simulate-preset', selectedScenario.value)
    selectedScenario.value = ''
  }
}
</script>

<style scoped>
.status-bar {
  height: var(--status-h);
  min-height: var(--status-h);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  background: var(--bg-panel);
  border-top: 1px solid var(--border);
  z-index: 10;
  user-select: none;
  gap: 12px;
}

/* ── Left ── */
.status-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  overflow: hidden;
}

.kafka-indicator {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  color: var(--text-dim);
  white-space: nowrap;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--red);
  box-shadow: var(--glow-red);
  transition: all var(--transition);
  flex-shrink: 0;
}

.kafka-indicator.connected .status-dot {
  background: var(--green);
  box-shadow: var(--glow-green);
}

.status-label {
  color: var(--text);
  letter-spacing: 0.3px;
}

.status-text {
  color: var(--text-dim);
}

.sep {
  color: var(--border-bright);
  font-size: 11px;
}

.uptime-display,
.event-count,
.frame-count {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-dim);
  white-space: nowrap;
}

.info-label {
  color: var(--text-dim);
}

.count-num {
  color: var(--text-bright);
  font-variant-numeric: tabular-nums;
  min-width: 20px;
}

/* ── Right ── */
.status-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.scenario-select {
  padding: 3px 10px;
  border: 1px solid var(--border-bright);
  border-radius: 14px;
  background: transparent;
  color: var(--text-dim);
  font-family: var(--font);
  font-size: 10px;
  cursor: pointer;
  letter-spacing: 0.3px;
  transition: all var(--transition);
  outline: none;
  min-width: 100px;
}

.scenario-select:hover {
  border-color: rgba(0, 255, 136, 0.3);
  color: var(--text);
}

.scenario-select:focus {
  border-color: rgba(0, 255, 136, 0.3);
}

.scenario-select option {
  background: var(--bg-panel);
  color: var(--text);
}
</style>
