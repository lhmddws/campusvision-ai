<template>
  <div class="app" @keydown="handleKeydown" tabindex="0" ref="appRef">
    <HeaderBar
      :cameras="cameras"
      :kafka-connected="kafkaConnected"
      :uptime="uptime"
      :event-stats="eventStats"
      :active-tab="activeTab"
      @toggle-dash="toggleDash"
      @toggle-settings="toggleSettings"
      @tab-select="(tab) => activeTab = tab"
    />

    <div class="main">
      <div class="content-area">
        <!-- Camera Grid (always visible) -->
        <CameraGrid
          v-show="activeTab === 'cameras' || activeTab === 'faces' || activeTab === 'behavior'"
          :cameras="cameras"
          :latest-events="latestEvents"
          :people="people"
          :config="config"
          :active-tab="activeTab"
          @simulate="handleSimulate"
        />

        <!-- Face Enrollment Panel -->
        <FaceEnroll
          v-show="activeTab === 'faces'"
          :faces="faces"
          @refresh="loadFaces"
        />

        <!-- Behavior Panel -->
        <BehaviorPanel
          v-show="activeTab === 'behavior'"
          :behavior-config="behaviorConfig"
          :cameras="cameras"
          :stats="stats"
        />
      </div>

      <!-- Stats Panel -->
      <StatsPanel
        v-show="statsVisible"
        :stats="stats"
        :cameras="cameras"
        :uptime="uptime"
        :kafka-connected="kafkaConnected"
      />

      <!-- Event Panel -->
      <EventPanel
        :events="events"
        :event-stats="eventStats"
        :stats-visible="statsVisible"
        @simulate-random="handleRandom"
        @simulate-preset="handlePreset"
        @clear-events="handleClearEvents"
        @toggle-stats="toggleStats"
      />
    </div>

    <StatusBar
      :kafka-connected="kafkaConnected"
      :uptime="uptime"
      :event-count="events.length"
      :cameras="cameras"
      :stats="stats"
      @simulate-random="handleRandom"
      @simulate-preset="handlePreset"
      @clear-events="handleClearEvents"
    />

    <!-- Config Drawer -->
    <ConfigDrawer
      :visible="settingsVisible"
      :config="config"
      :cameras="cameras"
      :people="people"
      :faces="faces"
      :recognition-status="recognitionStatus"
      :behavior-config="behaviorConfig"
      @close="toggleSettings"
      @update-config="handleUpdateConfig"
      @reset-config="handleResetConfig"
      @add-person="handleAddPerson"
      @remove-person="handleRemovePerson"
      @enroll-face="handleEnrollFace"
      @delete-face="handleDeleteFace"
      @start-webcam="handleWebcamStart"
      @stop-webcam="handleWebcamStop"
    />

    <!-- Toast Container -->
    <div class="toast-container" id="toast-container">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        :class="['toast', toast.type, { leaving: toast.leaving }]"
      >{{ toast.message }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { api } from './api/index.js'
import HeaderBar from './components/HeaderBar.vue'
import StatusBar from './components/StatusBar.vue'
import CameraGrid from './components/CameraGrid.vue'
import EventPanel from './components/EventPanel.vue'
import StatsPanel from './components/StatsPanel.vue'
import ConfigDrawer from './components/ConfigDrawer.vue'
import FaceEnroll from './components/FaceEnroll.vue'
import BehaviorPanel from './components/BehaviorPanel.vue'

const appRef = ref(null)

// ── State ──
const cameras = ref({})
const config = ref({})
const events = ref([])
const stats = ref({})
const people = ref([])
const faces = ref([])
const recognitionStatus = ref({})
const behaviorConfig = ref({})
const toasts = ref([])
const kafkaConnected = ref(false)
const uptime = ref('')
const latestEvents = ref({})
const eventStats = ref({ entry: 0, exit: 0, idle: 0, total: 0 })

const settingsVisible = ref(false)
const statsVisible = ref(false)
const activeTab = ref('cameras')

let pollTimer = null
let toastId = 0

// ── Toast ──
function showToast(message, type = 'success') {
  const id = ++toastId
  toasts.value.push({ id, message, type, leaving: false })
  setTimeout(() => {
    const t = toasts.value.find(t => t.id === id)
    if (t) t.leaving = true
    setTimeout(() => {
      toasts.value = toasts.value.filter(t => t.id !== id)
    }, 300)
  }, 3000)
}

// ── API calls ──
async function loadHealth() {
  try {
    const h = await api.health()
    kafkaConnected.value = h.kafka
    cameras.value = h.cameras || {}
    config.value = h.config || {}
  } catch (e) {
    console.error('health fetch error:', e)
  }
}

async function loadEvents() {
  try {
    events.value = await api.getEvents(200)
  } catch (e) { /* ignore */ }
}

async function loadStats() {
  try {
    stats.value = await api.getStats()
    const etc = stats.value.event_type_counts || {}
    eventStats.value = {
      entry: etc.entry || 0,
      exit: etc.exit || 0,
      idle: etc.idle || 0,
      total: stats.value.events_total || events.value.length,
    }
  } catch (e) { /* ignore */ }
}

async function loadConfig() {
  try {
    const c = await api.getConfig()
    config.value = c.config || {}
    cameras.value = c.cameras || cameras.value
  } catch (e) { /* ignore */ }
}

async function loadPeople() {
  try {
    const p = await api.getPeople()
    people.value = p.people || []
  } catch (e) { /* ignore */ }
}

async function loadFaces() {
  try {
    const f = await api.getFaces()
    faces.value = f.faces || []
  } catch (e) { /* ignore */ }
}

async function loadRecognitionStatus() {
  try {
    recognitionStatus.value = await api.recognitionStatus()
  } catch (e) { /* ignore */ }
}

async function loadBehaviorConfig() {
  try {
    behaviorConfig.value = await api.behaviorStatus()
  } catch (e) { /* ignore */ }
}

async function pollAll() {
  await Promise.all([
    loadHealth(),
    loadEvents(),
    loadStats(),
  ])
}

// ── Uptime ──
function updateUptime() {
  const s = stats.value.uptime_sec || 0
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  uptime.value = `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${sec.toString().padStart(2, '0')}`
}

let uptimeTimer = null

// ── Actions ──
const lastSimulation = {}

async function handleSimulate(cameraId, action) {
  const now = Date.now()
  const key = `${cameraId}-${action}`
  if (lastSimulation[key] && now - lastSimulation[key] < 300) return
  lastSimulation[key] = now

  try {
    const result = await api.simulate(cameraId, { action })

    if (action !== 'idle') {
      latestEvents.value[cameraId] = { action, time: Date.now() }
    }

    if (result.event) {
      events.value.unshift(result.event)
      events.value = events.value.slice(0, 300)
    }
  } catch (e) {
    showToast(`模拟失败: ${e.message}`, 'error')
  }
}

async function handleRandom() {
  try {
    const result = await api.randomScenario(8)
    showToast(`已生成 ${result.generated} 个随机事件`, 'success')
    await loadEvents()
    await loadStats()
  } catch (e) {
    showToast(`随机事件失败: ${e.message}`, 'error')
  }
}

async function handlePreset(preset) {
  try {
    const result = await api.presetScenario(preset)
    if (result.events_cleared) {
      events.value = []
      showToast('事件日志已清空', 'info')
    } else {
      showToast(`场景 ${preset} 已执行，生成 ${result.generated} 个事件`, 'success')
    }
    await loadEvents()
    await loadStats()
  } catch (e) {
    showToast(`场景失败: ${e.message}`, 'error')
  }
}

function handleClearEvents() {
  handlePreset('clear_log')
}

function toggleSettings() {
  settingsVisible.value = !settingsVisible.value
  if (settingsVisible.value) {
    loadConfig()
    loadPeople()
    loadFaces()
    loadRecognitionStatus()
    loadBehaviorConfig()
  }
}

function toggleDash() {
  statsVisible.value = !statsVisible.value
}

function toggleStats() {
  statsVisible.value = !statsVisible.value
}

async function handleUpdateConfig(updates) {
  try {
    const result = await api.updateConfig(updates)
    config.value = result.config
    showToast('配置已更新', 'success')
  } catch (e) {
    showToast(`配置更新失败: ${e.message}`, 'error')
  }
}

async function handleResetConfig() {
  try {
    const result = await api.resetConfig()
    config.value = result.config
    showToast('配置已恢复默认', 'info')
  } catch (e) {
    showToast(`重置失败: ${e.message}`, 'error')
  }
}

async function handleAddPerson(name) {
  try {
    const result = await api.addPerson(name)
    people.value = result.people
    showToast(`已添加: ${name}`, 'success')
  } catch (e) {
    showToast(`添加失败: ${e.message}`, 'error')
  }
}

async function handleRemovePerson(name) {
  try {
    const result = await api.removePerson(name)
    people.value = result.people
    showToast(`已删除: ${name}`, 'info')
  } catch (e) {
    showToast(`删除失败: ${e.message}`, 'error')
  }
}

async function handleEnrollFace(name, studentId, imageFile) {
  try {
    await api.enrollFace(name, studentId, imageFile)
    showToast(`人脸录入成功: ${name}`, 'success')
    await loadFaces()
  } catch (e) {
    showToast(`人脸录入失败: ${e.message}`, 'error')
  }
}

async function handleDeleteFace(name) {
  try {
    await api.deleteFace(name)
    showToast(`已删除人脸: ${name}`, 'info')
    await loadFaces()
  } catch (e) {
    showToast(`删除失败: ${e.message}`, 'error')
  }
}

async function handleWebcamStart(cameraId, deviceIndex) {
  try {
    await api.webcamStart(cameraId, deviceIndex)
    showToast(`Webcam ${cameraId} 已启动`, 'success')
  } catch (e) {
    showToast(`Webcam 启动失败: ${e.message}`, 'error')
  }
}

async function handleWebcamStop(cameraId) {
  try {
    await api.webcamStop(cameraId)
    showToast(`Webcam ${cameraId} 已停止`, 'info')
  } catch (e) {
    showToast(`停止失败: ${e.message}`, 'error')
  }
}

// ── Keyboard shortcuts ──
function handleKeydown(e) {
  if (e.ctrlKey && e.key === 'r') { e.preventDefault(); handleRandom() }
  if (e.ctrlKey && e.key === 't') { e.preventDefault(); handlePreset('rush_hour') }
  if (e.key === 'f' || e.key === 'F') { toggleDash() }
}

// ── Lifecycle ──
onMounted(async () => {
  await pollAll()
  await loadPeople()
  await loadFaces()
  await loadRecognitionStatus()
  await loadBehaviorConfig()

  pollTimer = setInterval(pollAll, 5000)

  uptimeTimer = setInterval(() => {
    stats.value.uptime_sec = (stats.value.uptime_sec || 0) + 1
    updateUptime()
  }, 1000)

  updateUptime()
  appRef.value?.focus()
})

onUnmounted(() => {
  clearInterval(pollTimer)
  clearInterval(uptimeTimer)
})
</script>

<style scoped>
.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  outline: none;
}
.main {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
</style>
