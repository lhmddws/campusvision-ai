<template>
  <div class="drawer-overlay" :class="{ open: visible }" @click="$emit('close')" />
  <div class="drawer" :class="{ open: visible }">
    <div class="drawer-header">
      <h3>&#9881; 配置面板</h3>
      <button class="drawer-close" @click="$emit('close')">&#10005;</button>
    </div>

    <div class="drawer-body">
      <!-- 1. 帧设置 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.frame }" @click="sections.frame = !sections.frame">
          <span>帧设置</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.frame }">
          <div class="cfg-row">
            <span class="cfg-label">JPEG质量 <span class="cfg-unit">(10-100)</span></span>
            <input type="range" min="10" max="100" v-model.number="form.jpeg_quality" />
            <span class="cfg-value">{{ form.jpeg_quality }}</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">帧宽度</span>
            <select v-model.number="form.frame_width">
              <option :value="320">320</option>
              <option :value="640">640</option>
              <option :value="854">854</option>
              <option :value="1280">1280</option>
            </select>
            <span class="cfg-value">{{ form.frame_width }}px</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">FPS <span class="cfg-unit">(1-30)</span></span>
            <input type="range" min="1" max="30" v-model.number="form.fps" />
            <span class="cfg-value">{{ form.fps }}</span>
          </div>
        </div>
      </div>

      <!-- 2. 检测设置 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.detection }" @click="sections.detection = !sections.detection">
          <span>检测设置</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.detection }">
          <div class="cfg-row">
            <span class="cfg-label">置信度阈值 <span class="cfg-unit">(0.1-0.99)</span></span>
            <input type="range" min="0.1" max="0.99" step="0.05" v-model.number="form.confidence_threshold" />
            <span class="cfg-value">{{ form.confidence_threshold.toFixed(2) }}</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">最小人脸 <span class="cfg-unit">(40-300px)</span></span>
            <input type="range" min="40" max="300" v-model.number="form.min_face_size" />
            <span class="cfg-value">{{ form.min_face_size }}px</span>
          </div>
        </div>
      </div>

      <!-- 3. 匹配设置 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.match }" @click="sections.match = !sections.match">
          <span>匹配设置</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.match }">
          <div class="cfg-row">
            <span class="cfg-label">匹配阈值 <span class="cfg-unit">(0.1-0.99)</span></span>
            <input type="range" min="0.1" max="0.99" step="0.05" v-model.number="form.match_threshold" />
            <span class="cfg-value">{{ form.match_threshold.toFixed(2) }}</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">缓存TTL <span class="cfg-unit">(1-86400s)</span></span>
            <input type="number" min="1" max="86400" v-model.number="form.cache_ttl" />
            <span class="cfg-value">{{ form.cache_ttl }}s</span>
          </div>
        </div>
      </div>

      <!-- 4. 方向设置 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.direction }" @click="sections.direction = !sections.direction">
          <span>方向设置</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.direction }">
          <div class="cfg-row">
            <span class="cfg-label">ROI线位置 <span class="cfg-unit">(0.1-0.9)</span></span>
            <input type="range" min="0.1" max="0.9" step="0.05" v-model.number="form.roi_line_x" />
            <span class="cfg-value">{{ form.roi_line_x.toFixed(2) }}</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">最小轨迹点 <span class="cfg-unit">(1-10)</span></span>
            <input type="range" min="1" max="10" v-model.number="form.min_track_points" />
            <span class="cfg-value">{{ form.min_track_points }}</span>
          </div>
        </div>
      </div>

      <!-- 5. 陌生人告警 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.stranger }" @click="sections.stranger = !sections.stranger">
          <span>陌生人告警</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.stranger }">
          <div class="cfg-toggle-row">
            <span class="cfg-label">启用</span>
            <div class="toggle-track" :class="{ active: form.stranger_alert_enabled }" @click="form.stranger_alert_enabled = !form.stranger_alert_enabled">
              <div class="toggle-thumb" />
            </div>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">告警阈值 <span class="cfg-unit">(0.1-0.9)</span></span>
            <input type="range" min="0.1" max="0.9" step="0.05" v-model.number="form.stranger_alert_threshold" />
            <span class="cfg-value">{{ form.stranger_alert_threshold.toFixed(2) }}</span>
          </div>
        </div>
      </div>

      <!-- 6. 夜间模式 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.night }" @click="sections.night = !sections.night">
          <span>夜间模式</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.night }">
          <div class="cfg-toggle-row">
            <span class="cfg-label">启用</span>
            <div class="toggle-track" :class="{ active: form.night_mode_enabled }" @click="form.night_mode_enabled = !form.night_mode_enabled">
              <div class="toggle-thumb" />
            </div>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">开始小时 <span class="cfg-unit">(0-23)</span></span>
            <input type="range" min="0" max="23" v-model.number="form.night_mode_start_hour" />
            <span class="cfg-value">{{ String(form.night_mode_start_hour).padStart(2, '0') }}:00</span>
          </div>
          <div class="cfg-row">
            <span class="cfg-label">结束小时 <span class="cfg-unit">(0-23)</span></span>
            <input type="range" min="0" max="23" v-model.number="form.night_mode_end_hour" />
            <span class="cfg-value">{{ String(form.night_mode_end_hour).padStart(2, '0') }}:00</span>
          </div>
        </div>
      </div>

      <!-- 7. 运动检测 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.motion }" @click="sections.motion = !sections.motion">
          <span>运动检测</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.motion }">
          <div class="cfg-row">
            <span class="cfg-label">运动阈值 <span class="cfg-unit">(0.01-0.5)</span></span>
            <input type="range" min="0.01" max="0.5" step="0.01" v-model.number="form.motion_threshold" />
            <span class="cfg-value">{{ form.motion_threshold.toFixed(2) }}</span>
          </div>
          <div class="cfg-toggle-row">
            <span class="cfg-label">动态抽帧</span>
            <div class="toggle-track" :class="{ active: form.dynamic_extraction }" @click="form.dynamic_extraction = !form.dynamic_extraction">
              <div class="toggle-thumb" />
            </div>
          </div>
        </div>
      </div>

      <!-- 8. 去重设置 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.dedup }" @click="sections.dedup = !sections.dedup">
          <span>去重设置</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.dedup }">
          <div class="cfg-row">
            <span class="cfg-label">去重窗口 <span class="cfg-unit">(1-60s)</span></span>
            <input type="range" min="1" max="60" v-model.number="form.dedup_window_seconds" />
            <span class="cfg-value">{{ form.dedup_window_seconds }}s</span>
          </div>
        </div>
      </div>

      <!-- 9. 人员管理 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.people }" @click="sections.people = !sections.people">
          <span>人员管理</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.people }">
          <div class="cfg-people-input-row">
            <input
              class="cfg-people-input"
              v-model="newPersonName"
              placeholder="输入姓名..."
              @keyup.enter="addPerson"
            />
            <button class="gen-btn primary" @click="addPerson" :disabled="!newPersonName.trim()">添加</button>
          </div>
          <div class="cfg-hint">支持CSV批量导入：姓名1, 姓名2, ...</div>
          <div class="cfg-people-list">
            <div class="cfg-person-tag" v-for="name in people" :key="name">
              <span class="cfg-person-name">{{ name }}</span>
              <button class="cfg-person-remove" @click="$emit('remove-person', name)">&#10005;</button>
            </div>
            <div v-if="!people || people.length === 0" class="cfg-empty">暂无人员</div>
          </div>
        </div>
      </div>

      <!-- 10. 人脸库 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.faces }" @click="sections.faces = !sections.faces">
          <span>人脸库</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.faces }">
          <div v-if="recognitionStatus" class="cfg-face-stats">
            <span>已录入: {{ recognitionStatus.enrolled_count || 0 }} 人</span>
            <span>置信度: {{ (recognitionStatus.confidence_threshold || 0).toFixed(2) }}</span>
          </div>
          <div class="cfg-face-list" v-if="faces && faces.length > 0">
            <div class="cfg-face-card" v-for="face in faces" :key="face.student_id || face.name">
              <img
                class="cfg-face-thumb"
                :src="face.image_url || 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%2240%22 height=%2240%22 viewBox=%220 0 40 40%22%3E%3Crect fill=%22%231a1a3a%22 width=%2240%22 height=%2240%22/%3E%3Ctext x=%2220%22 y=%2220%22 text-anchor=%22middle%22 dominant-baseline=%22central%22 fill=%22%23666688%22 font-size=%2214%22%3E?%3C/text%3E%3C/svg%3E'"
                alt="face"
              />
              <div class="cfg-face-info">
                <span class="cfg-face-name">{{ face.name }}</span>
                <span class="cfg-face-id" v-if="face.student_id">{{ face.student_id }}</span>
              </div>
              <button class="cfg-face-delete gen-btn" @click="$emit('delete-face', face.name)">删除</button>
            </div>
          </div>
          <div v-else class="cfg-empty">暂无录入人脸</div>
          <div class="cfg-face-enroll-row">
            <input class="cfg-people-input" v-model="enrollName" placeholder="姓名" />
            <input class="cfg-people-input" v-model="enrollStudentId" placeholder="学号" />
            <input type="file" ref="enrollFileInput" accept="image/*" @change="onEnrollFileChange" class="cfg-file-input" />
            <button class="gen-btn primary" @click="enrollFace" :disabled="!enrollName || !enrollStudentId || !enrollFile">录入</button>
          </div>
        </div>
      </div>

      <!-- 11. 摄像头管理 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.cameras }" @click="sections.cameras = !sections.cameras">
          <span>摄像头管理</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.cameras }">
          <div class="cfg-camera-card" v-for="(cam, camId) in cameras" :key="camId">
            <div class="cfg-camera-head">
              <span class="cfg-camera-dot" :style="{ background: cam.color || 'var(--gray)' }" />
              <span class="cfg-camera-id">{{ camId }}</span>
              <button
                class="gen-btn"
                @click="$emit('stop-webcam', camId)"
              >停止</button>
            </div>
            <div class="cfg-camera-fields">
              <label>标签</label>
              <input class="cfg-people-input" :value="cam.label" placeholder="Camera label" readonly />
            </div>
            <div class="cfg-camera-fields">
              <label>楼栋</label>
              <input class="cfg-people-input" :value="cam.building" placeholder="Building" readonly />
            </div>
          </div>
          <div class="cfg-camera-add-row">
            <input class="cfg-people-input" v-model="newCamId" placeholder="新摄像头ID" />
            <input class="cfg-people-input" v-model="newCamBuilding" placeholder="楼栋" />
            <input class="cfg-people-input" v-model="newCamLabel" placeholder="标签" />
            <div class="cfg-camera-actions">
              <button class="gen-btn" @click="startWebcam(0)" :disabled="!newCamId">Webcam</button>
            </div>
          </div>
        </div>
      </div>

      <!-- 12. 流水线状态 -->
      <div class="cfg-section">
        <div class="cfg-section-header" :class="{ collapsed: !sections.pipeline }" @click="sections.pipeline = !sections.pipeline">
          <span>流水线状态</span>
          <span class="chevron">&#9660;</span>
        </div>
        <div class="cfg-section-body" :class="{ hidden: !sections.pipeline }">
          <div class="pipeline-flow">
            <div class="pipeline-step">
              <span class="pipeline-dot green" />
              <span>摄像头</span>
            </div>
            <span class="pipeline-arrow">&#8594;</span>
            <div class="pipeline-step">
              <span class="pipeline-dot" :class="config.kafka ? 'green' : 'red'" />
              <span>Stream Gateway</span>
            </div>
            <span class="pipeline-arrow">&#8594;</span>
            <div class="pipeline-step">
              <span class="pipeline-dot" :class="config.kafka ? 'green' : 'red'" />
              <span>Kafka</span>
            </div>
            <span class="pipeline-arrow">&#8594;</span>
            <div class="pipeline-step">
              <span class="pipeline-dot" :class="config.kafka ? 'green' : 'red'" />
              <span>Face Recognition</span>
            </div>
            <span class="pipeline-arrow">&#8594;</span>
            <div class="pipeline-step">
              <span class="pipeline-dot amber" />
              <span>Event</span>
            </div>
            <span class="pipeline-arrow">&#8594;</span>
            <div class="pipeline-step">
              <span class="pipeline-dot amber" />
              <span>Dormitory Service</span>
            </div>
          </div>
          <div class="pipeline-info">
            <span v-if="config.kafka_brokers" class="pipeline-info-item">Kafka: {{ config.kafka_brokers }}</span>
            <span v-if="config.camera_source" class="pipeline-info-item">源: {{ config.camera_source }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="drawer-footer">
      <button @click="handleSave">保存配置</button>
      <button class="danger" @click="$emit('reset-config')">恢复默认</button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  config: { type: Object, default: () => ({}) },
  cameras: { type: Object, default: () => ({}) },
  people: { type: Array, default: () => [] },
  faces: { type: Array, default: () => [] },
  recognitionStatus: { type: Object, default: () => ({}) },
  behaviorConfig: { type: Object, default: () => ({}) },
})

const emit = defineEmits([
  'close',
  'update-config',
  'reset-config',
  'add-person',
  'remove-person',
  'enroll-face',
  'delete-face',
  'start-webcam',
  'stop-webcam',
])

// ── Local form state (reactive copy of config fields) ──
const form = reactive({
  jpeg_quality: 80,
  frame_width: 640,
  frame_height: 480,
  fps: 15,
  confidence_threshold: 0.5,
  min_face_size: 80,
  match_threshold: 0.6,
  cache_ttl: 3600,
  roi_line_x: 0.5,
  min_track_points: 3,
  dedup_window_seconds: 5,
  stranger_alert_enabled: false,
  stranger_alert_threshold: 0.5,
  night_mode_enabled: false,
  night_mode_start_hour: 22,
  night_mode_end_hour: 6,
  motion_threshold: 0.05,
  dynamic_extraction: false,
  camera_source: '',
  webcam_device: 0,
  test_people: '',
})

// ── Section collapse state ──
const sections = reactive({
  frame: true,
  detection: true,
  match: true,
  direction: true,
  stranger: true,
  night: true,
  motion: true,
  dedup: true,
  people: true,
  faces: true,
  cameras: true,
  pipeline: true,
})

// ── People management ──
const newPersonName = ref('')

function addPerson() {
  const name = newPersonName.value.trim()
  if (name) {
    emit('add-person', name)
    newPersonName.value = ''
  }
}

// ── Face enrollment ──
const enrollName = ref('')
const enrollStudentId = ref('')
const enrollFile = ref(null)
const enrollFileInput = ref(null)

function onEnrollFileChange(e) {
  enrollFile.value = e.target.files[0] || null
}

function enrollFace() {
  if (!enrollName.value.trim() || !enrollStudentId.value.trim() || !enrollFile.value) return
  emit('enroll-face', enrollName.value.trim(), enrollStudentId.value.trim(), enrollFile.value)
  enrollName.value = ''
  enrollStudentId.value = ''
  enrollFile.value = null
  if (enrollFileInput.value) {
    enrollFileInput.value.value = ''
  }
}

// ── Camera management ──
const newCamId = ref('')
const newCamBuilding = ref('')
const newCamLabel = ref('')

function startWebcam(deviceIndex) {
  if (!newCamId.value) return
  emit('start-webcam', newCamId.value, deviceIndex)
}

// ── Save config ──
function handleSave() {
  const updates = {}
  const keys = [
    'jpeg_quality', 'frame_width', 'frame_height', 'fps',
    'confidence_threshold', 'min_face_size',
    'match_threshold', 'cache_ttl',
    'roi_line_x', 'min_track_points',
    'stranger_alert_enabled', 'stranger_alert_threshold',
    'night_mode_enabled', 'night_mode_start_hour', 'night_mode_end_hour',
    'motion_threshold', 'dynamic_extraction',
    'dedup_window_seconds',
    'camera_source', 'webcam_device',
  ]
  for (const key of keys) {
    if (form[key] !== undefined) {
      updates[key] = form[key]
    }
  }
  emit('update-config', updates)
}

// ── Sync form from props ──
function syncFormFromConfig(cfg) {
  const keys = [
    'jpeg_quality', 'frame_width', 'frame_height', 'fps',
    'confidence_threshold', 'min_face_size',
    'match_threshold', 'cache_ttl',
    'roi_line_x', 'min_track_points',
    'stranger_alert_enabled', 'stranger_alert_threshold',
    'night_mode_enabled', 'night_mode_start_hour', 'night_mode_end_hour',
    'motion_threshold', 'dynamic_extraction',
    'dedup_window_seconds',
    'camera_source', 'webcam_device',
  ]
  for (const key of keys) {
    if (cfg[key] !== undefined) {
      form[key] = cfg[key]
    }
  }
}

function reloadForm() {
  if (props.config && Object.keys(props.config).length > 0) {
    syncFormFromConfig(props.config)
  }
}

// ── Watch visible to reload form from props ──
watch(() => props.visible, (newVal) => {
  if (newVal) {
    reloadForm()
  }
})

// Watch config changes while open
watch(() => props.config, (newConfig) => {
  if (props.visible && newConfig && Object.keys(newConfig).length > 0) {
    syncFormFromConfig(newConfig)
  }
}, { deep: true })
</script>

<style scoped>
/* ── Person management ── */
.cfg-people-input-row {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
}

.cfg-people-input {
  flex: 1;
  padding: 6px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 11px;
  font-family: var(--font);
}
.cfg-people-input:focus {
  border-color: rgba(0, 255, 136, 0.3);
}

.cfg-hint {
  font-size: 10px;
  color: var(--text-dim);
  margin-bottom: 8px;
}

.cfg-people-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.cfg-person-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  font-size: 11px;
}

.cfg-person-name {
  color: var(--text);
}

.cfg-person-remove {
  background: transparent;
  border: none;
  color: var(--red-dim);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 2px;
  transition: color var(--transition);
}
.cfg-person-remove:hover {
  color: var(--red);
}

.cfg-empty {
  color: var(--text-dim);
  font-size: 11px;
  padding: 8px 0;
}

/* ── Face library ── */
.cfg-face-stats {
  display: flex;
  gap: 12px;
  font-size: 11px;
  color: var(--text-dim);
  margin-bottom: 8px;
}

.cfg-face-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 8px;
}

.cfg-face-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.cfg-face-thumb {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.cfg-face-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.cfg-face-name {
  font-size: 12px;
  color: var(--text-bright);
}

.cfg-face-id {
  font-size: 10px;
  color: var(--text-dim);
}

.cfg-face-delete {
  flex-shrink: 0;
}

.cfg-face-enroll-row {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
  padding-top: 6px;
  border-top: 1px solid var(--border);
}

.cfg-file-input {
  font-size: 10px;
  color: var(--text-dim);
  font-family: var(--font);
  max-width: 120px;
}

.cfg-file-input::file-selector-button {
  padding: 3px 8px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: transparent;
  color: var(--text);
  font-family: var(--font);
  font-size: 10px;
  cursor: pointer;
  transition: all var(--transition);
}
.cfg-file-input::file-selector-button:hover {
  border-color: rgba(0, 255, 136, 0.3);
  color: var(--green);
}

/* ── Camera management ── */
.cfg-camera-card {
  padding: 8px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  margin-bottom: 6px;
}

.cfg-camera-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.cfg-camera-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.cfg-camera-id {
  flex: 1;
  font-size: 12px;
  color: var(--text-bright);
  font-weight: 500;
}

.cfg-camera-fields {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.cfg-camera-fields label {
  font-size: 10px;
  color: var(--text-dim);
  min-width: 32px;
  flex-shrink: 0;
}
.cfg-camera-fields .cfg-people-input {
  flex: 1;
}

.cfg-camera-add-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-top: 6px;
  border-top: 1px solid var(--border);
}
.cfg-camera-add-row .cfg-people-input {
  min-width: 80px;
  flex: 1;
}

.cfg-camera-actions {
  display: flex;
  gap: 4px;
}

/* ── Pipeline flow ── */
.pipeline-flow {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 2px;
  padding: 8px 0;
}

.pipeline-step {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  color: var(--text);
  white-space: nowrap;
}

.pipeline-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--gray);
  flex-shrink: 0;
}
.pipeline-dot.green {
  background: var(--green);
  box-shadow: 0 0 4px var(--green);
}
.pipeline-dot.red {
  background: var(--red);
  box-shadow: 0 0 4px var(--red);
}
.pipeline-dot.amber {
  background: var(--amber);
  box-shadow: 0 0 4px var(--amber);
}

.pipeline-arrow {
  color: var(--text-dim);
  font-size: 11px;
}

.pipeline-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 6px;
  border-top: 1px solid var(--border);
}

.pipeline-info-item {
  font-size: 10px;
  color: var(--text-dim);
}
</style>
