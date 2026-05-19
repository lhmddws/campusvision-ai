<template>
  <header class="header-bar">
    <!-- Left: Logo + Title -->
    <div class="header-left">
      <div class="logo-box">CV</div>
      <span class="title">Campus <span class="title-green">Vision</span> AI</span>
    </div>

    <!-- Middle: Tab buttons -->
    <nav class="header-tabs">
      <button
        :class="['tab-btn', { active: activeTab === 'cameras' }]"
        @click="$emit('tab-select', 'cameras')"
      >
        <span class="tab-icon">📷</span>监控
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'faces' }]"
        @click="$emit('tab-select', 'faces')"
      >
        <span class="tab-icon">👤</span>人脸
      </button>
      <button
        :class="['tab-btn', { active: activeTab === 'behavior' }]"
        @click="$emit('tab-select', 'behavior')"
      >
        <span class="tab-icon">🚶</span>行为
      </button>
    </nav>

    <!-- Right: Status + actions -->
    <div class="header-right">
      <!-- Kafka status -->
      <div class="kafka-indicator" :class="{ connected: kafkaConnected }">
        <span class="status-dot"></span>
        <span class="status-text">{{ kafkaConnected ? '已连接' : '未连接' }}</span>
      </div>

      <!-- Uptime -->
      <div class="uptime-display" title="运行时间">
        <span class="uptime-icon">⏱</span>
        <span>{{ uptime }}</span>
      </div>

      <!-- Building stats -->
      <div class="building-stats" v-if="Object.keys(cameras).length">
        <span
          v-for="cam in cameras"
          :key="cam.building"
          class="building-pill"
          :style="{ borderColor: cam.color || 'var(--border-bright)' }"
        >
          <span class="building-dot" :style="{ background: cam.color || 'var(--text-dim)' }"></span>
          {{ cam.building }}
        </span>
      </div>

      <!-- Event stats summary -->
      <div class="event-summary">
        <span class="stat-item entry">{{ eventStats.entry }}</span>
        <span class="stat-item exit">{{ eventStats.exit }}</span>
        <span class="stat-item idle">{{ eventStats.idle }}</span>
        <span class="stat-item total">{{ eventStats.total }}</span>
      </div>

      <!-- Action buttons -->
      <div class="header-actions">
        <button class="icon-btn" title="统计面板" @click="$emit('toggle-dash')">📊</button>
        <button class="icon-btn" title="设置" @click="$emit('toggle-settings')">⚙</button>
      </div>
    </div>
  </header>
</template>

<script setup>
defineProps({
  cameras: { type: Object, default: () => ({}) },
  kafkaConnected: { type: Boolean, default: false },
  uptime: { type: String, default: '00:00:00' },
  eventStats: {
    type: Object,
    default: () => ({ entry: 0, exit: 0, idle: 0, total: 0 }),
  },
  activeTab: {
    type: String,
    default: 'cameras',
    validator: (v) => ['cameras', 'faces', 'behavior'].includes(v),
  },
})

defineEmits(['toggle-dash', 'toggle-settings', 'tab-select'])
</script>

<style scoped>
.header-bar {
  height: var(--header-h);
  min-height: var(--header-h);
  display: flex;
  align-items: center;
  padding: 0 14px;
  gap: 16px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
  z-index: 10;
  user-select: none;
}

/* ── Left ── */
.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.logo-box {
  width: 30px;
  height: 30px;
  border: 1.5px solid var(--green);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: var(--green);
  letter-spacing: 1px;
  box-shadow: var(--glow-green);
  background: rgba(0, 255, 136, 0.06);
}

.title {
  font-size: 13px;
  color: var(--text-bright);
  letter-spacing: 1px;
  white-space: nowrap;
}

.title-green {
  color: var(--green);
}

/* ── Tabs ── */
.header-tabs {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.header-tabs .tab-btn {
  padding: 5px 14px;
  border: 1px solid transparent;
  border-bottom: none;
  border-radius: 4px 4px 0 0;
  background: transparent;
  color: var(--text-dim);
  font-family: var(--font);
  font-size: 11px;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all var(--transition);
  position: relative;
  bottom: 0;
  white-space: nowrap;
}

.header-tabs .tab-btn:hover {
  color: var(--text);
  background: rgba(255, 255, 255, 0.02);
}

.header-tabs .tab-btn.active {
  color: var(--green);
  border-color: var(--border);
  background: var(--bg-card);
}

.header-tabs .tab-btn .tab-icon {
  margin-right: 4px;
}

/* ── Right ── */
.header-right {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-left: auto;
  flex-shrink: 0;
}

/* Kafka status */
.kafka-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  color: var(--text-dim);
  white-space: nowrap;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--red);
  box-shadow: var(--glow-red);
  transition: all var(--transition);
}

.kafka-indicator.connected .status-dot {
  background: var(--green);
  box-shadow: var(--glow-green);
}

.status-text {
  letter-spacing: 0.3px;
}

/* Uptime */
.uptime-display {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text-dim);
  white-space: nowrap;
}

.uptime-icon {
  font-size: 10px;
}

/* Building stats */
.building-stats {
  display: flex;
  gap: 6px;
}

.building-pill {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border: 1px solid var(--border-bright);
  border-radius: 10px;
  font-size: 9px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
  white-space: nowrap;
}

.building-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* Event summary */
.event-summary {
  display: flex;
  gap: 6px;
}

.stat-item {
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  min-width: 14px;
  text-align: center;
}

.stat-item.entry {
  color: var(--green);
}

.stat-item.exit {
  color: var(--red);
}

.stat-item.idle {
  color: var(--amber);
}

.stat-item.total {
  color: var(--text-dim);
}

.stat-item.total::before {
  content: '/';
  margin-right: 3px;
  color: var(--text-dim);
}

/* Action buttons */
.header-actions {
  display: flex;
  gap: 2px;
}

.icon-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--text-dim);
  font-size: 14px;
  cursor: pointer;
  transition: all var(--transition);
}

.icon-btn:hover {
  background: rgba(255, 255, 255, 0.03);
  border-color: var(--border);
  color: var(--text);
}
</style>
