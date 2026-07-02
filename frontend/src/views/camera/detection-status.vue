<template>
  <div class="cv-detection-page">
    <!-- KPI Cards -->
    <div class="kpi-grid">
      <div class="kpi-card">
        <div class="kpi-icon-circle kpi-icon-blue">
          <el-icon :size="28"><VideoCamera /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ statusCounts.total }}</div>
          <div class="kpi-label">摄像头总数</div>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon-circle kpi-icon-green">
          <el-icon :size="28"><CircleCheck /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ statusCounts.online }}</div>
          <div class="kpi-label">在线</div>
        </div>
        <div class="kpi-trend trend-up">
          <span>{{ statusPercent('online') }}%</span>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon-circle kpi-icon-gray">
          <el-icon :size="28"><CircleClose /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ statusCounts.offline }}</div>
          <div class="kpi-label">离线</div>
        </div>
        <div class="kpi-trend trend-neutral">
          <span>{{ statusPercent('offline') }}%</span>
        </div>
      </div>
      <div class="kpi-card">
        <div class="kpi-icon-circle kpi-icon-red">
          <el-icon :size="28"><Warning /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ statusCounts.error }}</div>
          <div class="kpi-label">异常</div>
        </div>
        <div class="kpi-trend" :class="statusCounts.error > 0 ? 'trend-down' : 'trend-up'">
          <span>{{ statusPercent('error') }}%</span>
        </div>
      </div>
    </div>

    <!-- Filter Bar -->
    <div class="cv-filter-bar">
      <div class="cv-filter-bar__left">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索名称 / ID / 楼栋"
          clearable
          prefix-icon="Search"
          class="cv-filter-bar__input"
          @clear="handleFilter"
          @keyup.enter="handleFilter"
        />
        <el-select
          v-model="filterBuilding"
          placeholder="全部楼栋"
          clearable
          class="cv-filter-bar__select"
          @change="handleFilter"
        >
          <el-option v-for="b in buildingOptions" :key="b" :label="b" :value="b" />
        </el-select>
        <el-select
          v-model="filterStatus"
          placeholder="全部状态"
          clearable
          class="cv-filter-bar__select"
          @change="handleFilter"
        >
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="异常" value="error" />
          <el-option label="未知" value="unknown" />
        </el-select>
      </div>
      <div class="cv-filter-bar__right">
        <div class="cv-auto-refresh">
          <span class="cv-auto-refresh__label">自动刷新</span>
          <el-switch v-model="autoRefresh" active-color="#1890ff" @change="toggleAutoRefresh" />
        </div>
        <el-button icon="Refresh" :loading="loading" @click="fetchData">刷新</el-button>
      </div>
    </div>

    <!-- Table Card -->
    <div class="cv-table-card">
      <el-table
        v-loading="loading"
        :data="filteredList"
        class="cv-table"
        :header-cell-style="{ background: '#FAFAFA', color: '#262626', fontWeight: 600 }"
        row-class-name="cv-table__row"
      >
        <el-table-column label="摄像头" min-width="180">
          <template #default="{ row }">
            <div class="cv-camera-name">
              <span class="cv-camera-name__text">{{ row.name }}</span>
              <span class="cv-camera-name__id">{{ row.camera_id }}</span>
            </div>
            <el-tag size="small" effect="plain" class="cv-building-tag">{{ row.building }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="RTSP地址" min-width="200">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--mono">{{ sanitizeRtsp(row.rtsp_url) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="方向" min-width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="directionTagType(row.direction)" effect="plain" size="small">
              {{ directionLabel(row.direction) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="检测状态" min-width="110" align="center">
          <template #default="{ row }">
            <span :class="['cv-status', `cv-status--${row.status || 'unknown'}`]">
              <span class="cv-status__dot"></span>
              {{ statusLabel(row.status) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="帧率" min-width="80" align="center">
          <template #default="{ row }">
            <span
              :class="['cv-cell-text', { 'cv-fps-low': row.fps_current != null && row.fps_current < 1 }]"
            >
              {{ row.fps_current != null ? row.fps_current.toFixed(1) : '-' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="总帧数" min-width="100" align="right">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--mono">{{ formatNumber(row.total_frames) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后事件" min-width="160" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--secondary">{{ row.last_event_time || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后心跳" min-width="160" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--secondary">{{ row.last_heartbeat || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="健康检查" min-width="160" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--secondary">{{ row.last_health_check || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="启用" min-width="80" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              active-color="#1890ff"
              @change="(val: boolean) => handleToggleEnabled(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="160" align="center" fixed="right">
          <template #default="{ row }">
            <div class="cv-actions">
              <el-button
                text
                type="primary"
                size="small"
                :loading="healthCheckLoading[row.camera_id]"
                @click="handleHealthCheck(row)"
              >
                健康检查
              </el-button>
              <el-button text type="primary" size="small" @click="openDetail(row)">
                详情
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty
        v-if="!loading && filteredList.length === 0"
        description="暂无摄像头数据"
        class="cv-empty"
      />
    </div>

    <!-- Detail Dialog -->
    <el-dialog
      v-model="detailVisible"
      :title="detailCamera ? `${detailCamera.name} - 检测详情` : '检测详情'"
      width="640px"
      append-to-body
      class="cv-dialog"
    >
      <template v-if="detailCamera">
        <div class="detail-section">
          <div class="detail-section__title">基本信息</div>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="摄像头ID">{{ detailCamera.camera_id }}</el-descriptions-item>
            <el-descriptions-item label="名称">{{ detailCamera.name }}</el-descriptions-item>
            <el-descriptions-item label="楼栋">{{ detailCamera.building }}</el-descriptions-item>
            <el-descriptions-item label="方向">{{ directionLabel(detailCamera.direction) }}</el-descriptions-item>
            <el-descriptions-item label="RTSP地址" :span="2">
              <span class="cv-cell-text cv-cell-text--mono">{{ sanitizeRtsp(detailCamera.rtsp_url) }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <span :class="['cv-status', `cv-status--${detailCamera.status || 'unknown'}`]">
                <span class="cv-status__dot"></span>
                {{ statusLabel(detailCamera.status) }}
              </span>
            </el-descriptions-item>
            <el-descriptions-item label="启用">{{ detailCamera.enabled ? '是' : '否' }}</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="detail-section">
          <div class="detail-section__title">实时检测指标</div>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="当前帧率">
              <span :class="{ 'cv-fps-low': detailCamera.fps_current != null && detailCamera.fps_current < 1 }">
                {{ detailCamera.fps_current != null ? detailCamera.fps_current.toFixed(1) + ' FPS' : '-' }}
              </span>
            </el-descriptions-item>
            <el-descriptions-item label="总帧数">{{ formatNumber(detailCamera.total_frames) }}</el-descriptions-item>
            <el-descriptions-item label="最后事件">{{ detailCamera.last_event_time || '-' }}</el-descriptions-item>
            <el-descriptions-item label="最后心跳">{{ detailCamera.last_heartbeat || '-' }}</el-descriptions-item>
            <el-descriptions-item label="最后健康检查">{{ detailCamera.last_health_check || '-' }}</el-descriptions-item>
            <el-descriptions-item label="备注">{{ detailCamera.remark || '-' }}</el-descriptions-item>
          </el-descriptions>
        </div>

        <div class="detail-section">
          <div class="detail-section__title">
            网关实时状态
            <el-button
              text
              type="primary"
              size="small"
              :loading="gatewayLoading"
              @click="fetchGatewayHealth"
            >
              刷新
            </el-button>
          </div>
          <template v-if="gatewayHealth">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="连接状态">
                <el-tag :type="gatewayHealth.connected ? 'success' : 'danger'" size="small">
                  {{ gatewayHealth.connected ? '已连接' : '未连接' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="运行时间">{{ gatewayHealth.uptime || '-' }}</el-descriptions-item>
              <el-descriptions-item label="当前帧率">{{ gatewayHealth.fps != null ? gatewayHealth.fps.toFixed(1) + ' FPS' : '-' }}</el-descriptions-item>
              <el-descriptions-item label="总帧数">{{ formatNumber(gatewayHealth.total_frames) }}</el-descriptions-item>
            </el-descriptions>
          </template>
          <el-empty v-else-if="!gatewayLoading" description="暂无网关数据" :image-size="48" />
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import {
  listCameras,
  updateCamera,
  healthCheck as healthCheckApi,
  getGatewayCameraHealth,
} from '@/api/camera';
import type { Camera } from '@/api/camera';

// ==================== Data ====================
const cameraList = ref<Camera[]>([]);
const loading = ref(false);

const searchKeyword = ref('');
const filterBuilding = ref('');
const filterStatus = ref('');

const autoRefresh = ref(true);
let refreshTimer: ReturnType<typeof setInterval> | null = null;

const buildingOptions = computed(() => {
  const set = new Set<string>();
  for (const c of cameraList.value) set.add(c.building);
  return Array.from(set).sort();
});

const filteredList = computed(() => {
  let list = cameraList.value;
  const kw = searchKeyword.value.toLowerCase();
  if (kw) {
    list = list.filter(
      c =>
        c.name.toLowerCase().includes(kw) ||
        c.camera_id.toLowerCase().includes(kw) ||
        c.building.toLowerCase().includes(kw),
    );
  }
  if (filterBuilding.value) {
    list = list.filter(c => c.building === filterBuilding.value);
  }
  if (filterStatus.value) {
    list = list.filter(c => (c.status || 'unknown') === filterStatus.value);
  }
  return list;
});

const statusCounts = computed(() => {
  const counts = { total: cameraList.value.length, online: 0, offline: 0, error: 0, unknown: 0 };
  for (const c of cameraList.value) {
    const s = (c.status || 'unknown') as keyof typeof counts;
    if (s in counts) counts[s]++;
    else counts.unknown++;
  }
  return counts;
});

function statusPercent(key: string): string {
  const t = statusCounts.value.total;
  if (t === 0) return '0';
  const v = (statusCounts.value as Record<string, number>)[key] ?? 0;
  return ((v / t) * 100).toFixed(1);
}

// ==================== Fetch ====================
function fetchData() {
  loading.value = true;
  listCameras()
    .then((res: any) => {
      cameraList.value = res.data || [];
    })
    .catch(() => {
      cameraList.value = [];
    })
    .finally(() => {
      loading.value = false;
    });
}

function handleFilter() {
  // client-side filter via computed, no re-fetch needed
}

// ==================== Auto-refresh ====================
function startAutoRefresh() {
  stopAutoRefresh();
  refreshTimer = setInterval(fetchData, 10000);
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

function toggleAutoRefresh(val: boolean | string | number) {
  if (val) startAutoRefresh();
  else stopAutoRefresh();
}

// ==================== Enable/disable ====================
function handleToggleEnabled(row: Camera, val: boolean) {
  updateCamera(row.camera_id, { enabled: val } as Partial<Camera>)
    .then(() => {
      row.enabled = val;
      ElMessage.success(val ? '已启用' : '已禁用');
    })
    .catch(() => {
      ElMessage.error('操作失败');
    });
}

// ==================== Health check ====================
const healthCheckLoading = reactive<Record<string, boolean>>({});

function handleHealthCheck(row: Camera) {
  healthCheckLoading[row.camera_id] = true;
  healthCheckApi(row.camera_id)
    .then(() => {
      ElMessage.success('健康检查已触发');
      fetchData();
    })
    .catch(() => {
      ElMessage.error('健康检查失败');
    })
    .finally(() => {
      healthCheckLoading[row.camera_id] = false;
    });
}

// ==================== Detail dialog ====================
const detailVisible = ref(false);
const detailCamera = ref<Camera | null>(null);
const gatewayHealth = ref<any>(null);
const gatewayLoading = ref(false);

function openDetail(row: Camera) {
  detailCamera.value = row;
  gatewayHealth.value = null;
  detailVisible.value = true;
  fetchGatewayHealth();
}

function fetchGatewayHealth() {
  if (!detailCamera.value) return;
  gatewayLoading.value = true;
  getGatewayCameraHealth(detailCamera.value.camera_id)
    .then((res: any) => {
      gatewayHealth.value = res.data || res;
    })
    .catch(() => {
      gatewayHealth.value = null;
    })
    .finally(() => {
      gatewayLoading.value = false;
    });
}

// ==================== Helpers ====================
const STATUS_MAP: Record<string, string> = {
  online: '在线',
  offline: '离线',
  error: '异常',
  unknown: '未知',
};

const DIRECTION_MAP: Record<string, { label: string; type: string }> = {
  entry: { label: '入口', type: '' },
  exit: { label: '出口', type: 'success' },
  both: { label: '双向', type: 'warning' },
};

function statusLabel(status: string | undefined): string {
  return STATUS_MAP[status || 'unknown'] ?? status ?? '未知';
}

function directionLabel(direction: string): string {
  return DIRECTION_MAP[direction]?.label ?? direction;
}

function directionTagType(direction: string): string {
  return DIRECTION_MAP[direction]?.type ?? '';
}

function sanitizeRtsp(url: string): string {
  if (!url) return '-';
  return url.replace(/:[^:@]+@/, ':***@');
}

function formatNumber(val: number | null | undefined): string {
  if (val == null) return '-';
  return val.toLocaleString('en-US');
}

// ==================== Lifecycle ====================
onMounted(() => {
  fetchData();
  if (autoRefresh.value) startAutoRefresh();
});

onBeforeUnmount(() => {
  stopAutoRefresh();
});
</script>

<style lang="scss" scoped>
@import '@/assets/styles/variables.module.scss';

$cv-primary: $primary-color;
$cv-success: $success-color;
$cv-warning: $warning-color;
$cv-danger: $danger-color;
$cv-page-bg: $page-bg;
$cv-card-bg: $card-bg;
$cv-text-primary: $text-primary;
$cv-text-secondary: $text-secondary;

.cv-detection-page {
  padding: 20px;
  min-height: 100%;
  background: $cv-page-bg;
}

// ─── KPI Grid ───
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.kpi-card {
  background: $cv-card-bg;
  border-radius: 8px;
  padding: 20px 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  transition: transform 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  }
}

.kpi-icon-circle {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-icon-blue { background: rgba($cv-primary, 0.1); color: $cv-primary; }
.kpi-icon-green { background: rgba($cv-success, 0.1); color: $cv-success; }
.kpi-icon-gray { background: rgba(#bfbfbf, 0.15); color: #8c8c8c; }
.kpi-icon-red { background: rgba($cv-danger, 0.1); color: $cv-danger; }

.kpi-body { flex: 1; min-width: 0; }

.kpi-value {
  font-size: 28px;
  font-weight: 700;
  color: $cv-text-primary;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.kpi-label {
  font-size: 13px;
  color: $cv-text-secondary;
  margin-top: 4px;
}

.kpi-trend {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  padding: 4px 8px;
  border-radius: 4px;
  white-space: nowrap;
}

.trend-up { color: $cv-success; background: rgba($cv-success, 0.08); }
.trend-down { color: $cv-danger; background: rgba($cv-danger, 0.08); }
.trend-neutral { color: $cv-text-secondary; background: rgba(140, 140, 140, 0.06); }

// ─── Filter Bar ───
.cv-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: $cv-card-bg;
  border-radius: 8px;
  margin-bottom: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);

  &__left { display: flex; align-items: center; gap: 12px; }
  &__right { display: flex; align-items: center; gap: 8px; }
  &__input { width: 240px; }
  &__select { width: 140px; }
}

.cv-auto-refresh {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-right: 8px;

  &__label { font-size: 13px; color: $cv-text-secondary; white-space: nowrap; }
}

// ─── Table Card ───
.cv-table-card {
  background: $cv-card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.cv-table {
  width: 100%;

  :deep(.cv-table__row) {
    transition: background-color 0.2s ease;
    &:hover > td { background-color: #e6f7ff !important; }
  }

  :deep(.el-table__inner-wrapper::before) { display: none; }
  :deep(th.el-table__cell) { border-bottom: 1px solid #f0f0f0; font-size: 13px; letter-spacing: 0.02em; }
  :deep(td.el-table__cell) { border-bottom: 1px solid #f0f0f0; }
}

// ─── Camera Name Cell ───
.cv-camera-name {
  display: flex;
  flex-direction: column;
  gap: 2px;

  &__text { font-weight: 500; color: $cv-text-primary; font-size: 14px; line-height: 1.4; }
  &__id { font-size: 12px; color: $cv-text-secondary; font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace; }
}

.cv-building-tag { margin-top: 4px; font-size: 11px; }

// ─── Cell Text ───
.cv-cell-text {
  color: $cv-text-primary;
  font-size: 13px;

  &--mono { font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace; font-size: 12px; color: $cv-text-secondary; }
  &--secondary { color: $cv-text-secondary; font-size: 12px; }
}

.cv-fps-low { color: $cv-danger !important; font-weight: 600; }

// ─── Status Indicator ───
.cv-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;

  &__dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  &--online {
    color: $cv-success;
    .cv-status__dot { background-color: $cv-success; box-shadow: 0 0 0 3px rgba($cv-success, 0.2); }
  }

  &--offline {
    color: $cv-text-secondary;
    .cv-status__dot { background-color: #bfbfbf; box-shadow: 0 0 0 3px rgba(#bfbfbf, 0.15); }
  }

  &--error {
    color: $cv-danger;
    .cv-status__dot { background-color: $cv-danger; box-shadow: 0 0 0 3px rgba($cv-danger, 0.2); }
  }

  &--unknown {
    color: #bfbfbf;
    .cv-status__dot { background-color: #d9d9d9; box-shadow: 0 0 0 3px rgba(#d9d9d9, 0.15); }
  }
}

// ─── Actions ───
.cv-actions { display: flex; align-items: center; justify-content: center; gap: 4px; }

// ─── Empty ───
.cv-empty { padding: 48px 0; }

// ─── Dialog ───
.cv-dialog {
  :deep(.el-dialog__header) { border-bottom: 1px solid #f0f0f0; padding-bottom: 16px; }
  :deep(.el-dialog__body) { padding: 24px 20px; }
}

.detail-section {
  margin-bottom: 20px;

  &:last-child { margin-bottom: 0; }

  &__title {
    font-size: 14px;
    font-weight: 600;
    color: $cv-text-primary;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

// ─── Responsive ───
@media (max-width: 1200px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 768px) {
  .cv-detection-page { padding: 12px; }

  .kpi-grid { grid-template-columns: 1fr; gap: 12px; }
  .kpi-card { padding: 16px; }

  .cv-filter-bar {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;

    &__left { flex-wrap: wrap; }
    &__input { width: 100%; }
    &__select { width: 100%; }
    &__right { justify-content: flex-end; }
  }
}
</style>
