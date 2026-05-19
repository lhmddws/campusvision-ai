<template>
  <div class="cam-card" :class="[flashClass]">
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

      <div class="cam-badge">
        <span class="badge-dot" :style="{ background: color }"></span>
        <span class="badge-id">{{ cameraId }}</span>
        <span class="badge-label">{{ label }}</span>
      </div>

      <div class="rec-indicator">
        <span class="rec-dot"></span>
        REC
      </div>

      <div class="cam-timestamp">{{ currentTime }}</div>
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

// ── Flash effect ──
const flashClass = ref('')
let flashTimeout = null

watch(() => props.latestEvent?.action, (newAction, oldAction) => {
  if (!newAction || newAction === oldAction) return

  if (flashTimeout) clearTimeout(flashTimeout)

  if (newAction === 'entry') flashClass.value = 'flash'
  else if (newAction === 'exit') flashClass.value = 'flash-exit'
  else flashClass.value = 'flash-idle'

  flashTimeout = setTimeout(() => {
    flashClass.value = ''
    flashTimeout = null
  }, 1000)
})
</script>

<style scoped>
.cam-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: border-color 0.3s ease, box-shadow 0.3s ease;
  position: relative;
}

.cam-card.flash {
  border-color: var(--green);
  box-shadow: var(--glow-green);
}

.cam-card.flash-exit {
  border-color: var(--red);
  box-shadow: var(--glow-red);
}

.cam-card.flash-idle {
  border-color: var(--amber);
  box-shadow: var(--glow-amber);
}

.cam-feed {
  position: relative;
  flex: 1;
  min-height: 0;
  background: #000;
  overflow: hidden;
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

/* Scanning line overlay */
.cam-feed::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--green), transparent);
  opacity: 0.5;
  animation: scan 2s linear infinite;
  pointer-events: none;
  z-index: 2;
}

@keyframes scan {
  0% { top: 0; }
  100% { top: 100%; }
}

/* NO SIGNAL fallback */
.no-signal {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  background: #050510;
}

.no-signal svg {
  width: 80px;
  height: 60px;
}

/* Badge — top left */
.cam-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: rgba(0, 0, 0, 0.7);
  border-radius: var(--radius-sm);
  z-index: 3;
  font-size: 10px;
  letter-spacing: 0.5px;
}

.badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 4px currentColor;
}

.badge-id {
  color: var(--text-bright);
  font-weight: 600;
}

.badge-label {
  color: var(--text-dim);
}

/* REC indicator — top right */
.rec-indicator {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: rgba(0, 0, 0, 0.7);
  border-radius: var(--radius-sm);
  color: var(--red);
  font-size: 9px;
  letter-spacing: 1px;
  z-index: 3;
}

.rec-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--red);
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* Timestamp — bottom right */
.cam-timestamp {
  position: absolute;
  bottom: 8px;
  right: 8px;
  padding: 2px 6px;
  background: rgba(0, 0, 0, 0.7);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 10px;
  z-index: 3;
  letter-spacing: 0.5px;
}

/* Controls bar */
.cam-controls {
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  border-top: 1px solid var(--border);
  background: var(--bg-panel);
  flex-shrink: 0;
}

.ctrl-btn {
  flex: 1;
  padding: 5px 0;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text);
  font-family: var(--font);
  font-size: 10px;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all var(--transition);
  text-align: center;
}

.ctrl-btn:hover {
  background: rgba(255, 255, 255, 0.03);
}

.ctrl-entry {
  color: var(--green);
  border-color: rgba(0, 255, 136, 0.25);
}

.ctrl-entry:hover {
  background: rgba(0, 255, 136, 0.08);
  box-shadow: 0 0 8px rgba(0, 255, 136, 0.06);
}

.ctrl-exit {
  color: var(--red);
  border-color: rgba(255, 51, 85, 0.25);
}

.ctrl-exit:hover {
  background: rgba(255, 51, 85, 0.08);
  box-shadow: 0 0 8px rgba(255, 51, 85, 0.06);
}

.ctrl-idle {
  color: var(--text-dim);
  border-color: var(--border);
}

.ctrl-idle:hover {
  background: rgba(255, 255, 255, 0.03);
  color: var(--text);
}

.ctrl-webcam {
  flex: 0 0 auto;
  width: 30px;
  padding: 5px 0;
  color: var(--text-dim);
  font-size: 12px;
  line-height: 1;
}

.ctrl-webcam:hover {
  color: var(--text-bright);
  background: rgba(255, 255, 255, 0.03);
}
</style>
