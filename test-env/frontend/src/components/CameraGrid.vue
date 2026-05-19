<template>
  <div class="camera-grid">
    <div class="grid-container">
      <CameraCard
        v-for="(cam, id) in cameras"
        :key="id"
        :camera-id="id"
        :label="cam.label"
        :building="cam.building"
        :color="cam.color"
        :latest-event="latestEvents[id] || null"
        :config="config"
        @simulate="(action) => emit('simulate', id, action)"
      >
        <template v-if="activeTab === 'behavior'" #overlay>
          <div class="behavior-on-card">
            <div class="beh-title">行为分析</div>
            <div class="beh-item">
              <span class="beh-dot motion"></span>
              <span>移动</span>
              <span class="beh-val">{{ behaviorStats[id]?.motion ?? '--' }}</span>
            </div>
            <div class="beh-item">
              <span class="beh-dot stay"></span>
              <span>停留</span>
              <span class="beh-val">{{ behaviorStats[id]?.stay ?? '--' }}</span>
            </div>
            <div class="beh-item">
              <span class="beh-dot crowd"></span>
              <span>人数</span>
              <span class="beh-val">{{ behaviorStats[id]?.count ?? '--' }}</span>
            </div>
          </div>
        </template>
      </CameraCard>
    </div>

    <!-- Face Recognition Overlay -->
    <div v-if="activeTab === 'faces'" class="face-overlay">
      <div class="face-overlay-header">人脸识别结果</div>
      <div class="face-overlay-body">
        <div
          v-for="(cam, id) in cameras"
          :key="id"
          class="face-cam-row"
          :style="{ borderLeftColor: cam.color }"
        >
          <span class="face-cam-id" :style="{ color: cam.color }">{{ id }}</span>
          <span class="face-cam-label">{{ cam.label }}</span>
          <span class="face-result">
            <template v-if="cameraFaces[id] && cameraFaces[id].length > 0">
              <span class="face-name-tag" v-for="p in cameraFaces[id]" :key="p">{{ p }}</span>
              <span class="face-confidence">99%</span>
            </template>
            <span v-else class="face-none">未检测到人脸</span>
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import CameraCard from './CameraCard.vue'

const props = defineProps({
  cameras: { type: Object, required: true },
  latestEvents: { type: Object, default: () => ({}) },
  people: { type: Array, default: () => [] },
  config: { type: Object, default: () => ({}) },
  activeTab: { type: String, default: 'cameras' },
})

const emit = defineEmits(['simulate'])

// Deterministic per-camera face assignment based on camera id
const cameraFaces = computed(() => {
  const entries = Object.entries(props.cameras)
  const result = {}
  for (const [id] of entries) {
    const hash = id.charCodeAt(id.length - 1) || 0
    const count = hash % Math.min(props.people.length + 1, 4)
    const start = hash % Math.max(props.people.length, 1)
    const subset = []
    for (let i = 0; i < count; i++) {
      subset.push(props.people[(start + i) % props.people.length])
    }
    result[id] = subset
  }
  return result
})

// Simulated behavior stats per camera
const behaviorStats = computed(() => {
  const result = {}
  for (const id in props.cameras) {
    const hash = id.charCodeAt(id.length - 1) || 0
    result[id] = {
      motion: (hash * 3) % 12 + 1,
      stay: (hash * 7) % 8 + 1,
      count: (hash * 5) % 6 + 1,
    }
  }
  return result
})
</script>

<style scoped>
.camera-grid {
  flex: 1;
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.grid-container {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  gap: 10px;
  padding: 10px;
  min-height: 0;
}

/* ── Face Recognition Overlay ── */
.face-overlay {
  flex-shrink: 0;
  background: rgba(7, 7, 13, 0.88);
  backdrop-filter: blur(6px);
  border-top: 1px solid var(--border);
  padding: 10px 16px;
}

.face-overlay-header {
  font-size: 11px;
  color: var(--green);
  letter-spacing: 1px;
  margin-bottom: 8px;
  font-weight: 600;
}

.face-overlay-body {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.face-cam-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-left: 2px solid var(--border);
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.02);
  font-size: 11px;
}

.face-cam-id {
  font-weight: 700;
  font-size: 11px;
  min-width: 28px;
  letter-spacing: 0.5px;
}

.face-cam-label {
  color: var(--text-dim);
  font-size: 10px;
  min-width: 50px;
}

.face-result {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.face-name-tag {
  display: inline-block;
  padding: 1px 6px;
  background: rgba(0, 255, 136, 0.08);
  border: 1px solid rgba(0, 255, 136, 0.2);
  border-radius: 3px;
  color: var(--green);
  font-size: 10px;
  letter-spacing: 0.3px;
}

.face-confidence {
  color: var(--text-dim);
  font-size: 9px;
  margin-left: 2px;
}

.face-none {
  color: var(--text-dim);
  font-style: italic;
  font-size: 10px;
}

/* ── Behavior Overlay (per card) ── */
.behavior-on-card {
  position: absolute;
  top: 32px;
  right: 8px;
  z-index: 3;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(4px);
  border: 1px solid var(--amber);
  border-radius: var(--radius-sm);
  padding: 6px 10px;
  min-width: 80px;
  font-size: 9px;
  pointer-events: none;
}

.beh-title {
  font-size: 9px;
  color: var(--amber);
  letter-spacing: 0.8px;
  margin-bottom: 4px;
  font-weight: 600;
  text-transform: uppercase;
}

.beh-item {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text);
  line-height: 1.6;
}

.beh-dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  flex-shrink: 0;
}

.beh-dot.motion {
  background: var(--blue);
  box-shadow: 0 0 4px var(--blue);
}

.beh-dot.stay {
  background: var(--amber);
  box-shadow: 0 0 4px var(--amber);
}

.beh-dot.crowd {
  background: var(--purple);
  box-shadow: 0 0 4px var(--purple);
}

.beh-val {
  margin-left: auto;
  color: var(--text-dim);
  font-size: 8px;
  font-weight: 600;
}
</style>
