<template>
  <div class="cam-card" :class="{ large }">
    <div class="cam-feed">
      <img
        v-show="!imgError"
        :src="frameUrl"
        :class="{ loading: imgLoading }"
        @load="onImgLoad"
        @error="onImgError"
        alt="Camera feed"
      />

      <div v-if="imgError" class="no-signal">
        <svg viewBox="0 0 120 80" xmlns="http://www.w3.org/2000/svg">
          <rect x="15" y="18" width="90" height="44" rx="4" fill="none" stroke="currentColor" stroke-width="1.5"/>
          <circle cx="60" cy="40" r="9" fill="none" stroke="currentColor" stroke-width="1.5"/>
          <ellipse cx="60" cy="40" rx="3" ry="3" fill="currentColor" opacity="0.5"/>
          <line x1="36" y1="52" x2="47" y2="30" stroke="currentColor" stroke-width="1.5" opacity="0.6"/>
          <text x="60" y="73" text-anchor="middle" fill="currentColor" font-size="7" font-family="monospace" opacity="0.7">NO SIGNAL</text>
        </svg>
      </div>

      <slot name="overlay" />

      <div class="cam-info">
        <div class="cam-info-left">
          <span class="status-dot online"></span>
          <span class="cam-name">{{ cameraId }}</span>
          <span class="cam-label">{{ label }}</span>
        </div>
        <span class="cam-timestamp">{{ currentTime }}</span>
      </div>
    </div>

    <div class="cam-controls">
      <button class="ctrl-btn ctrl-entry" @click="emit('simulate', 'entry')">进入</button>
      <button class="ctrl-btn ctrl-exit" @click="emit('simulate', 'exit')">离开</button>
      <button class="ctrl-btn ctrl-idle" @click="emit('simulate', 'idle')">无人</button>
      <button class="ctrl-btn ctrl-webcam" title="切换摄像头">📷</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { api } from '../api/index.js'

const props = defineProps({
  cameraId: { type: String, required: true },
  label: { type: String, required: true },
  building: { type: String, required: true },
  color: { type: String, required: true },
  latestEvent: { type: Object, default: null },
  config: { type: Object, default: () => ({}) },
  large: { type: Boolean, default: false },
})

const emit = defineEmits(['simulate'])

// ── Frame refresh ──
const timestamp = ref(Date.now())
let frameTimer

onMounted(() => {
  frameTimer = setInterval(() => {
    timestamp.value = Date.now()
  }, 2000)
})

onUnmounted(() => {
  clearInterval(frameTimer)
  clearInterval(timeTimer)
})

const frameUrl = computed(() => {
  return `${api.frameUrl(props.cameraId)}?t=${timestamp.value}`
})

// ── Image state ──
const imgLoading = ref(true)
const imgError = ref(false)

watch(frameUrl, () => {
  imgLoading.value = true
  imgError.value = false
})

function onImgLoad() {
  imgLoading.value = false
  imgError.value = false
}

function onImgError() {
  imgError.value = true
  imgLoading.value = false
}

// ── Current time display ──
const currentTime = ref('')
let timeTimer

function updateTime() {
  const now = new Date()
  currentTime.value = now.toLocaleTimeString('zh-CN', { hour12: false })
}

onMounted(() => {
  updateTime()
  timeTimer = setInterval(updateTime, 1000)
})
</script>

<style scoped>
.cam-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  box-shadow: var(--shadow-sm);
  transition: box-shadow var(--transition-normal);
}

.cam-card:hover {
  box-shadow: var(--shadow-md);
}

.cam-card.large {
  grid-column: 1 / -1;
}

/* ── Camera feed ── */
.cam-feed {
  position: relative;
  flex: 1;
  min-height: 0;
  background: var(--bg-hover);
  overflow: hidden;
}

.cam-card.large .cam-feed {
  min-height: 320px;
}

.cam-feed img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 1;
  transition: opacity 0.3s ease;
}

.cam-feed img.loading {
  opacity: 0.4;
}

/* NO SIGNAL fallback */
.no-signal {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  background: var(--bg-page);
}

.no-signal svg {
  width: 80px;
  height: 60px;
}

/* ── Bottom info bar ── */
.cam-info {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(4px);
  border-top: 1px solid var(--border-color);
  z-index: 3;
}

.cam-info-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.cam-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: 0.3px;
}

.cam-label {
  font-size: 11px;
  color: var(--text-muted);
}

.cam-timestamp {
  font-size: 11px;
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.3px;
}

/* ── Controls bar ── */
.cam-controls {
  display: flex;
  gap: 4px;
  padding: 8px 10px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-card);
  flex-shrink: 0;
}

.ctrl-btn {
  flex: 1;
  padding: 6px 0;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-secondary);
  font-family: var(--font);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.3px;
  cursor: pointer;
  transition: all var(--transition-fast);
  text-align: center;
}

.ctrl-btn:hover {
  background: var(--bg-hover);
}

.ctrl-entry {
  color: var(--color-success);
  border-color: var(--color-success-bg);
}

.ctrl-entry:hover {
  background: var(--color-success-bg);
}

.ctrl-exit {
  color: var(--color-danger);
  border-color: var(--color-danger-bg);
}

.ctrl-exit:hover {
  background: var(--color-danger-bg);
}

.ctrl-idle {
  color: var(--text-muted);
}

.ctrl-idle:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}

.ctrl-webcam {
  flex: 0 0 auto;
  width: 32px;
  padding: 6px 0;
  color: var(--text-muted);
  font-size: 13px;
  line-height: 1;
}

.ctrl-webcam:hover {
  color: var(--text-secondary);
  background: var(--bg-hover);
}
</style>
