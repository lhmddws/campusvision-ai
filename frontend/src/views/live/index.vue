<template>
  <div class="live-page">
    <!-- Header -->
    <div class="live-header">
      <div class="live-header-left">
        <el-icon :size="20" class="live-header-icon"><VideoCamera /></el-icon>
        <span class="live-header-title">实时预览</span>
        <el-tag v-if="wsConnected" type="success" size="small" effect="dark" class="conn-tag">
          已连接
        </el-tag>
        <el-tag v-else type="danger" size="small" effect="dark" class="conn-tag">
          连接断开
        </el-tag>
      </div>
      <div class="live-header-right">
        <span class="camera-count">共 {{ cameraList.length }} 个摄像头</span>
        <el-button size="small" :icon="Refresh" @click="handleRefresh">
          刷新
        </el-button>
      </div>
    </div>

    <!-- Camera Grid -->
    <div v-if="cameraList.length > 0" class="camera-grid">
      <div
        v-for="cam in cameraList"
        :key="cam.camera_id"
        class="camera-panel"
        @click="openFullscreen(cam)"
      >
        <div class="camera-panel-header">
          <div class="camera-panel-info">
            <span class="camera-panel-name">{{ cam.name }}</span>
            <span class="camera-panel-building">{{ cam.building }}栋</span>
          </div>
          <span class="status-indicator" :class="statusClass(cam.camera_id)">
            {{ statusText(cam.camera_id) }}
          </span>
        </div>
        <div class="camera-panel-body">
          <FrameCanvas
            :camera-id="cam.camera_id"
            :frame-data="frameDataByCameraId[cam.camera_id] ?? null"
            :bboxes="bboxesByCameraId[cam.camera_id] ?? []"
          />
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state">
      <el-empty description="暂无摄像头，请先添加摄像头" />
    </div>

    <!-- Fullscreen Modal -->
    <el-dialog
      v-model="fullscreenVisible"
      fullscreen
      :title="fullscreenCamera?.name ?? '实时画面'"
      destroy-on-close
      class="fullscreen-dialog"
    >
      <div class="fullscreen-body">
        <FrameCanvas
          v-if="fullscreenCamera"
          :key="fullscreenCamera.camera_id"
          :camera-id="fullscreenCamera.camera_id"
          :frame-data="frameDataByCameraId[fullscreenCamera.camera_id] ?? null"
          :bboxes="bboxesByCameraId[fullscreenCamera.camera_id] ?? []"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts">
export default { name: 'Live' };
</script>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { Refresh, VideoCamera } from '@element-plus/icons-vue';
import { listCameras, type Camera } from '@/api/camera';
import { getToken } from '@/utils/auth';
import FrameCanvas, { type BboxEvent } from './components/FrameCanvas.vue';

// ─── State ────────────────────────────────────────────────────────────────

const cameraList = ref<Camera[]>([]);
const wsConnected = ref(false);
const frameDataByCameraId = ref<Record<string, string>>({});
const bboxesByCameraId = ref<Record<string, BboxEvent[]>>({});
const lastFrameTimeByCameraId = ref<Record<string, number>>({});
const pendingBBoxQueue = ref<BboxEvent[]>([]);

const fullscreenVisible = ref(false);
const fullscreenCamera = ref<Camera | null>(null);

// ─── Connection Refs ──────────────────────────────────────────────────────

let ws: WebSocket | null = null;
let sseSource: EventSource | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempt = 0;
let sseClearTimer: ReturnType<typeof setInterval> | null = null;
let staleCheckTimer: ReturnType<typeof setInterval> | null = null;
let lastSseTimeByCameraId: Record<string, number> = {};

const MAX_RECONNECT_DELAY = 30000;
const INITIAL_RECONNECT_DELAY = 1000;
const STALE_FRAME_THRESHOLD = 5000;
const SSE_CLEAR_THRESHOLD = 5000;

// ─── URL Construction ─────────────────────────────────────────────────────

function getWsUrl(): string {
  const token = getToken();
  const base = location.origin.replace(/^http/, 'ws');
  return `${base}/dev-api/ws/live?token=${token}`;
}

function getSseUrl(): string {
  const token = getToken();
  return `${location.origin}/dev-api/sse/live?token=${token}`;
}

// ─── WebSocket ────────────────────────────────────────────────────────────

function connectWs(): void {
  const url = getWsUrl();
  ws = new WebSocket(url);

  ws.onopen = () => {
    wsConnected.value = true;
    reconnectAttempt = 0;
  };

  ws.onmessage = (event: MessageEvent) => {
    try {
      const msg = JSON.parse(event.data) as {
        camera_id: string;
        frame_data: string;
        building?: string;
        frame_sequence?: number;
        timestamp?: string;
      };
      if (!msg.camera_id || typeof msg.frame_data !== 'string') return;

      frameDataByCameraId.value[msg.camera_id] = msg.frame_data;
      lastFrameTimeByCameraId.value[msg.camera_id] = Date.now();
      processPendingQueue();
    } catch {
      // Ignore malformed messages
    }
  };

  ws.onclose = () => {
    wsConnected.value = false;
    scheduleReconnect();
  };

  ws.onerror = () => {
    ws?.close();
  };
}

function scheduleReconnect(): void {
  const delay = Math.min(
    INITIAL_RECONNECT_DELAY * Math.pow(2, reconnectAttempt),
    MAX_RECONNECT_DELAY,
  );
  reconnectAttempt++;
  reconnectTimer = setTimeout(() => {
    connectWs();
  }, delay);
}

function disconnectWs(): void {
  if (ws) {
    ws.onclose = null;
    ws.onerror = null;
    ws.onmessage = null;
    ws.onopen = null;
    ws.close();
    ws = null;
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  wsConnected.value = false;
}

// ─── SSE ──────────────────────────────────────────────────────────────────

function connectSse(): void {
  const url = getSseUrl();
  sseSource = new EventSource(url);

  sseSource.addEventListener('recognition', (event: MessageEvent) => {
    try {
      const data = JSON.parse(event.data) as BboxEvent;
      if (!data.camera_id) return;

      const cameraId = data.camera_id;

      if (frameDataByCameraId.value[cameraId]) {
        bboxesByCameraId.value[cameraId] = [data];
      } else {
        pendingBBoxQueue.value.push(data);
      }

      lastSseTimeByCameraId[cameraId] = Date.now();
    } catch {
      // Ignore malformed SSE events
    }
  });

  sseSource.onerror = () => {
    // EventSource auto-reconnects; nothing extra needed
  };
}

function disconnectSse(): void {
  if (sseSource) {
    sseSource.close();
    sseSource = null;
  }
}

// ─── Pending Queue ────────────────────────────────────────────────────────

let processingQueue = false;

function processPendingQueue(): void {
  if (processingQueue) return;
  processingQueue = true;

  const stillPending: BboxEvent[] = [];
  for (const bbox of pendingBBoxQueue.value) {
    if (frameDataByCameraId.value[bbox.camera_id]) {
      const existing = bboxesByCameraId.value[bbox.camera_id];
      if (existing) {
        existing.push(bbox);
      } else {
        bboxesByCameraId.value[bbox.camera_id] = [bbox];
      }
    } else {
      stillPending.push(bbox);
    }
  }
  pendingBBoxQueue.value = stillPending;
  processingQueue = false;
}

// ─── Timers ───────────────────────────────────────────────────────────────

function startIntervals(): void {
  // Check for stale frames every 1s (triggers reactive status text)
  staleCheckTimer = setInterval(() => {
    // Force reactivity by touching lastFrameTimeByCameraId
    // The computed status uses Date.now() comparison directly
  }, 1000);

  // Clear bboxes for cameras that haven't received SSE in 5s
  sseClearTimer = setInterval(() => {
    const now = Date.now();
    for (const cameraId of Object.keys(lastSseTimeByCameraId)) {
      if (now - lastSseTimeByCameraId[cameraId] > SSE_CLEAR_THRESHOLD) {
        bboxesByCameraId.value[cameraId] = [];
      }
    }
  }, 1000);
}

function stopIntervals(): void {
  if (staleCheckTimer) {
    clearInterval(staleCheckTimer);
    staleCheckTimer = null;
  }
  if (sseClearTimer) {
    clearInterval(sseClearTimer);
    sseClearTimer = null;
  }
}

// ─── Status Helpers ───────────────────────────────────────────────────────

function statusClass(cameraId: string): string {
  if (!wsConnected.value) return 'status-disconnected';
  const lastTime = lastFrameTimeByCameraId.value[cameraId];
  if (!lastTime) return 'status-waiting';
  if (Date.now() - lastTime > STALE_FRAME_THRESHOLD) return 'status-offline';
  return 'status-ok';
}

function statusText(cameraId: string): string {
  if (!wsConnected.value) return '连线中...';
  const lastTime = lastFrameTimeByCameraId.value[cameraId];
  if (!lastTime) return '等待画面...';
  const elapsed = Date.now() - lastTime;
  if (elapsed > STALE_FRAME_THRESHOLD) return '摄像头离线';
  const seconds = Math.floor(elapsed / 1000);
  return `${seconds}秒前更新`;
}

// ─── Fullscreen ───────────────────────────────────────────────────────────

function openFullscreen(cam: Camera): void {
  fullscreenCamera.value = cam;
  fullscreenVisible.value = true;
}

// ─── Camera Loading ───────────────────────────────────────────────────────

function loadCameras(): void {
  listCameras()
    .then((res: { data?: Camera[] }) => {
      cameraList.value = res.data ?? [];
    })
    .catch(() => {
      cameraList.value = [];
    });
}

function handleRefresh(): void {
  loadCameras();
}

// ─── Lifecycle ────────────────────────────────────────────────────────────

onMounted(() => {
  loadCameras();
  connectWs();
  connectSse();
  startIntervals();
});

onUnmounted(() => {
  disconnectWs();
  disconnectSse();
  stopIntervals();
});
</script>

<style scoped lang="scss">
$page-bg: #f0f2f5;
$card-bg: #ffffff;
$primary-color: #1890ff;
$text-primary: #262626;
$text-secondary: #8c8c8c;
$border-color: #e8e8e8;
$success-color: #52c41a;
$warning-color: #faad14;
$danger-color: #ff4d4f;

.live-page {
  min-height: 100%;
  background: $page-bg;
  padding: 16px;
}

/* ─── Header ─── */

.live-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: $card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  padding: 14px 20px;
  margin-bottom: 16px;
}

.live-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.live-header-icon {
  color: $primary-color;
}

.live-header-title {
  font-size: 16px;
  font-weight: 600;
  color: $text-primary;
}

.conn-tag {
  font-variant-numeric: tabular-nums;
}

.live-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.camera-count {
  font-size: 13px;
  color: $text-secondary;
}

/* ─── 2×2 Camera Grid ─── */

.camera-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

/* ─── Camera Panel ─── */

.camera-panel {
  background: $card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  cursor: pointer;
  transition:
    box-shadow 0.2s ease,
    transform 0.15s ease;

  &:hover {
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
    transform: translateY(-1px);
  }
}

.camera-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #f5f5f5;
}

.camera-panel-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.camera-panel-name {
  font-size: 14px;
  font-weight: 600;
  color: $text-primary;
}

.camera-panel-building {
  font-size: 12px;
  color: $text-secondary;
  background: #f5f5f5;
  padding: 1px 8px;
  border-radius: 4px;
}

.camera-panel-body {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #1a1a1a;
}

/* ─── Status Indicators ─── */

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;

  &::before {
    content: '';
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }
}

.status-ok {
  color: $success-color;

  &::before {
    background: $success-color;
    box-shadow: 0 0 4px rgba($success-color, 0.5);
  }
}

.status-disconnected {
  color: $danger-color;

  &::before {
    background: $danger-color;
  }
}

.status-waiting {
  color: $warning-color;

  &::before {
    background: $warning-color;
  }
}

.status-offline {
  color: $text-secondary;

  &::before {
    background: $text-secondary;
  }
}

/* ─── Empty State ─── */

.empty-state {
  display: flex;
  justify-content: center;
  padding: 80px 0;
  background: $card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

/* ─── Fullscreen Dialog ─── */

.fullscreen-dialog {
  :deep(.el-dialog__header) {
    padding: 14px 20px;
    border-bottom: 1px solid #f0f0f0;
    margin: 0;
  }

  :deep(.el-dialog__title) {
    font-size: 16px;
    font-weight: 600;
    color: $text-primary;
  }

  :deep(.el-dialog__body) {
    padding: 0;
    height: calc(100vh - 54px);
    background: #0d0d0d;
  }
}

.fullscreen-body {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;

  .frame-canvas-wrapper {
    max-width: 100%;
    max-height: 100%;
  }
}

/* ─── Responsive ─── */

@media (max-width: 768px) {
  .live-page {
    padding: 8px;
  }

  .camera-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .live-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .live-header-right {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
