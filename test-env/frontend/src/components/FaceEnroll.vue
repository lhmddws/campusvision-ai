<template>
  <div class="face-enroll">
    <!-- Section 1: 人脸录入 -->
    <div class="fe-section">
      <div class="fe-section-header">
        <span class="fe-section-icon">📝</span>
        <span>人脸录入</span>
      </div>
      <div class="fe-section-body">
        <div class="enroll-form">
          <div class="enroll-form-row">
            <div class="enroll-field">
              <label class="enroll-label">姓名</label>
              <input
                v-model="formName"
                type="text"
                class="enroll-input"
                placeholder="输入姓名"
                @keyup.enter="submitEnroll"
              />
            </div>
            <div class="enroll-field">
              <label class="enroll-label">学号</label>
              <input
                v-model="formStudentId"
                type="text"
                class="enroll-input"
                placeholder="输入学号"
                @keyup.enter="submitEnroll"
              />
            </div>
          </div>
          <div class="enroll-form-row">
            <div class="enroll-field enroll-file-field">
              <label class="enroll-label">照片</label>
              <div class="enroll-file-wrap">
                <label class="enroll-file-btn">
                  <span>选择文件</span>
                  <input
                    ref="fileInputRef"
                    type="file"
                    accept="image/*"
                    hidden
                    @change="handleFileChange"
                  />
                </label>
                <span class="enroll-file-name">{{ selectedFileName || '未选择文件' }}</span>
              </div>
            </div>
          </div>
          <div v-if="previewUrl" class="enroll-preview-wrap">
            <img :src="previewUrl" class="enroll-preview" alt="preview" />
            <button class="enroll-preview-clear" @click="clearPreview">✕</button>
          </div>
          <button
            class="enroll-submit"
            :disabled="!canSubmit || submitting"
            @click="submitEnroll"
          >
            <span v-if="submitting" class="loading-spinner"></span>
            <span v-else>录入</span>
          </button>
        </div>
      </div>
    </div>

    <!-- Section 2: 已录入人脸 -->
    <div class="fe-section">
      <div class="fe-section-header">
        <span class="fe-section-icon">👤</span>
        <span>已录入人脸</span>
        <span class="fe-count-badge">{{ faces.length }}</span>
      </div>
      <div class="fe-section-body">
        <div v-if="faces.length === 0" class="fe-empty">
          <svg class="fe-empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
          </svg>
          <span>暂无录入的人脸数据</span>
        </div>
        <div v-else class="fe-face-grid">
          <div
            v-for="face in faces"
            :key="face.student_id || face.name"
            class="fe-face-card"
          >
            <div class="fe-face-img-wrap">
              <img
                :src="api.getFaceImageUrl(face.name)"
                class="fe-face-img"
                alt="face"
                @error="handleImgError($event)"
              />
            </div>
            <div class="fe-face-info">
              <div class="fe-face-name">{{ face.name }}</div>
              <div class="fe-face-id">{{ face.student_id }}</div>
              <div class="fe-face-date">{{ formatDate(face.enrolled_at) }}</div>
            </div>
            <button class="fe-face-delete" @click="deleteFace(face)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Section 3: 实时识别 (SSE-driven) -->
    <div class="fe-section">
      <div class="fe-section-header">
        <span class="fe-section-icon">🎯</span>
        <span>实时识别</span>
        <span class="fe-live-dot" :class="{ active: recognitionEvents.length > 0 }"></span>
      </div>
      <div class="fe-section-body fe-recog-body">
        <div v-if="recognitionEvents.length === 0" class="fe-empty">
          <span>Waiting for recognition...</span>
        </div>
        <div v-else class="fe-recog-table">
          <div class="fe-recog-row fe-recog-header">
            <span class="fe-recog-cell fe-col-cam">摄像头</span>
            <span class="fe-recog-cell fe-col-person">目标</span>
            <span class="fe-recog-cell fe-col-conf">置信度</span>
            <span class="fe-recog-cell fe-col-time">时间</span>
          </div>
          <div
            v-for="evt in recognitionEvents"
            :key="evt.id"
            class="fe-recog-row"
          >
            <span class="fe-recog-cell fe-col-cam">
              <span
                class="fe-cam-dot"
                :style="{ background: getCamColor(evt.camera_id) }"
              ></span>
              {{ getCamLabel(evt.camera_id) }}
            </span>
            <span class="fe-recog-cell fe-col-person">
              <span v-if="evt.is_stranger" class="fe-stranger">陌生人</span>
              <span v-else class="fe-known-name">{{ evt.name }}</span>
            </span>
            <span class="fe-recog-cell fe-col-conf">
              <span class="fe-conf-badge" :style="{ color: confColor(evt.confidence) }">
                <span class="fe-conf-dot"></span>
                {{ (evt.confidence * 100).toFixed(0) }}%
              </span>
            </span>
            <span class="fe-recog-cell fe-col-time fe-time-cell">{{ formatTimestamp(evt.timestamp) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { api } from '../api/index.js'

const props = defineProps({
  faces: { type: Array, default: () => [] },
  cameras: { type: Object, default: () => ({}) },
  recognitionEvents: { type: Array, default: () => [] },
})

const emit = defineEmits(['refresh'])

// ── Enrollment form ──
const formName = ref('')
const formStudentId = ref('')
const selectedFile = ref(null)
const selectedFileName = ref('')
const previewUrl = ref('')
const fileInputRef = ref(null)
const submitting = ref(false)

const canSubmit = computed(() => formName.value.trim() && formStudentId.value.trim() && selectedFile.value)

function handleFileChange(e) {
  const file = e.target.files[0]
  if (!file) return
  selectedFile.value = file
  selectedFileName.value = file.name
  const reader = new FileReader()
  reader.onload = (ev) => { previewUrl.value = ev.target.result }
  reader.readAsDataURL(file)
}

function clearPreview() {
  previewUrl.value = ''
  selectedFile.value = null
  selectedFileName.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

async function submitEnroll() {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  try {
    await api.enrollFace(formName.value.trim(), formStudentId.value.trim(), selectedFile.value)
    formName.value = ''
    formStudentId.value = ''
    clearPreview()
    emit('refresh')
  } catch (e) {
    console.error('Enroll error:', e)
  } finally {
    submitting.value = false
  }
}

function handleImgError(e) {
  e.target.src = ''
  e.target.style.display = 'none'
}

// ── Face delete ──
async function deleteFace(face) {
  try {
    await api.deleteFace(face.name)
    emit('refresh')
  } catch (e) {
    console.error('Delete error:', e)
  }
}

// ── Date formatting ──
function formatDate(dateStr) {
  if (!dateStr) return '—'
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
  } catch {
    return dateStr
  }
}

function getCamColor(camId) {
  return props.cameras[camId]?.color || 'var(--text-dim)'
}

function getCamLabel(camId) {
  return props.cameras[camId]?.label || props.cameras[camId]?.building || camId
}

function confColor(conf) {
  if (conf > 0.7) return 'var(--green)'
  if (conf > 0.4) return 'var(--amber)'
  return 'var(--red)'
}

function formatTimestamp(ts) {
  if (!ts) return '—'
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return String(ts)
  }
}


</script>

<style scoped>
.face-enroll {
  padding: 8px 14px 14px;
  overflow-y: auto;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* ── Sections ── */
.fe-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}

.fe-section-header {
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

.fe-section-icon {
  font-size: 12px;
}

.fe-section-body {
  padding: 10px 12px;
}

.fe-count-badge {
  margin-left: auto;
  background: rgba(0,255,136,0.1);
  border: 1px solid rgba(0,255,136,0.2);
  border-radius: 10px;
  padding: 1px 8px;
  font-size: 10px;
  color: var(--green);
}

/* ── Live indicator dot ── */
.fe-live-dot {
  margin-left: auto;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-dim);
  opacity: 0.4;
  transition: all 0.3s ease;
}
.fe-live-dot.active {
  background: var(--green);
  opacity: 1;
  box-shadow: 0 0 6px rgba(0,255,136,0.5);
}

/* ── Empty ── */
.fe-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 24px 0;
  color: var(--text-dim);
  font-size: 11px;
}

.fe-empty-icon {
  width: 36px;
  height: 36px;
  opacity: 0.4;
}

/* ── Enroll form ── */
.enroll-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.enroll-form-row {
  display: flex;
  gap: 8px;
}

.enroll-field {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.enroll-label {
  font-size: 10px;
  color: var(--text-dim);
  letter-spacing: 0.3px;
}

.enroll-input {
  padding: 6px 10px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 11px;
  font-family: var(--font);
  transition: border-color var(--transition);
}

.enroll-input:focus {
  border-color: rgba(0,255,136,0.3);
}

.enroll-input::placeholder {
  color: var(--text-dim);
  opacity: 0.5;
}

.enroll-file-field {
  flex: 1;
}

.enroll-file-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}

.enroll-file-btn {
  display: inline-flex;
  align-items: center;
  padding: 5px 12px;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 10px;
  font-family: var(--font);
  cursor: pointer;
  transition: all var(--transition);
  white-space: nowrap;
}

.enroll-file-btn:hover {
  border-color: rgba(0,255,136,0.3);
  color: var(--green);
}

.enroll-file-name {
  font-size: 10px;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.enroll-preview-wrap {
  position: relative;
  display: inline-block;
  margin-top: 4px;
}

.enroll-preview {
  width: 80px;
  height: 80px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
}

.enroll-preview-clear {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--bg-panel);
  color: var(--text-dim);
  font-size: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition);
}

.enroll-preview-clear:hover {
  color: var(--red);
  border-color: rgba(255,51,85,0.3);
}

.enroll-submit {
  align-self: flex-start;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 20px;
  background: rgba(0,255,136,0.08);
  border: 1px solid rgba(0,255,136,0.25);
  border-radius: var(--radius);
  color: var(--green);
  font-size: 11px;
  font-family: var(--font);
  cursor: pointer;
  transition: all var(--transition);
  letter-spacing: 0.5px;
}

.enroll-submit:hover:not(:disabled) {
  background: rgba(0,255,136,0.14);
  box-shadow: 0 0 10px rgba(0,255,136,0.08);
}

.enroll-submit:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

/* ── Face grid ── */
.fe-face-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 8px;
}

.fe-face-card {
  display: flex;
  flex-direction: column;
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  transition: border-color var(--transition);
}

.fe-face-card:hover {
  border-color: var(--border-bright);
}

.fe-face-img-wrap {
  width: 100%;
  aspect-ratio: 1;
  background: var(--bg-deep);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.fe-face-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.fe-face-info {
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.fe-face-name {
  font-size: 11px;
  color: var(--text-bright);
  font-weight: 500;
}

.fe-face-id {
  font-size: 10px;
  color: var(--text-dim);
}

.fe-face-date {
  font-size: 9px;
  color: var(--text-dim);
  opacity: 0.7;
}

.fe-face-delete {
  margin: 0 8px 8px;
  padding: 3px 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--red-dim);
  font-size: 10px;
  font-family: var(--font);
  cursor: pointer;
  transition: all var(--transition);
}

.fe-face-delete:hover {
  background: rgba(255,51,85,0.08);
  border-color: rgba(255,51,85,0.3);
  color: var(--red);
}

/* ── Recognition table (SSE-driven) ── */
.fe-recog-body {
  max-height: 320px;
  overflow-y: auto;
  padding: 6px 12px;
}

.fe-recog-table {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.fe-recog-row {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 4px;
  border-radius: var(--radius-sm);
  transition: background var(--transition);
}

.fe-recog-row:hover:not(.fe-recog-header) {
  background: rgba(255,255,255,0.02);
}

.fe-recog-header {
  font-size: 9px;
  color: var(--text-dim);
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
  padding-bottom: 5px;
  margin-bottom: 2px;
  user-select: none;
}

.fe-recog-cell {
  font-size: 10px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fe-col-cam {
  width: 22%;
  display: flex;
  align-items: center;
  gap: 4px;
}

.fe-col-person {
  width: 22%;
}

.fe-col-conf {
  width: 26%;
  display: flex;
  align-items: center;
  gap: 6px;
}

.fe-col-time {
  width: 30%;
  text-align: right;
}

.fe-cam-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  flex-shrink: 0;
}

.fe-known-name {
  color: var(--green);
  font-weight: 500;
}

.fe-stranger {
  display: inline-block;
  color: var(--red);
  font-size: 9px;
  padding: 1px 6px;
  border: 1px solid rgba(255,51,85,0.3);
  border-radius: 3px;
  line-height: 1.4;
}

.fe-conf-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.fe-conf-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.fe-time-cell {
  font-size: 9px;
  color: var(--text-dim);
  font-variant-numeric: tabular-nums;
}

/* ── Misc ── */
.fe-loading-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid var(--border);
  border-top-color: var(--green);
  border-radius: 50%;
  animation: fe-spin 0.8s linear infinite;
  vertical-align: middle;
}

@keyframes fe-spin {
  to { transform: rotate(360deg); }
}
</style>
