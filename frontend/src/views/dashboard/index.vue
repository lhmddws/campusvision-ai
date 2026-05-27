<template>
  <div class="app-container dashboard-container">
    <!-- Row 1: KPI Cards -->
    <el-row :gutter="16" class="kpi-row">
      <el-col :xs="12" :sm="12" :md="6" :lg="6">
        <el-card v-loading="camerasLoading" shadow="hover" class="kpi-card kpi-camera">
          <div class="kpi-content">
            <div class="kpi-icon">
              <el-icon :size="32"><VideoCamera /></el-icon>
            </div>
            <div class="kpi-info">
              <div class="kpi-value">{{ cameraStatus.online }}<span class="kpi-unit"> / {{ cameraStatus.total }}</span></div>
              <div class="kpi-label">摄像头在线</div>
            </div>
          </div>
          <div class="kpi-footer">
            <el-tag :type="cameraStatus.error > 0 ? 'danger' : 'success'" size="small" effect="plain">
              {{ cameraStatus.error > 0 ? `${cameraStatus.error} 异常` : '全部正常' }}
            </el-tag>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="12" :sm="12" :md="6" :lg="6">
        <el-card v-loading="alertsLoading" shadow="hover" class="kpi-card kpi-alert">
          <div class="kpi-content">
            <div class="kpi-icon">
              <el-icon :size="32"><Bell /></el-icon>
            </div>
            <div class="kpi-info">
              <div class="kpi-value">{{ alertStats.today ?? 0 }}</div>
              <div class="kpi-label">今日告警</div>
            </div>
          </div>
          <div class="kpi-footer">
            <el-tag :type="(alertStats.unread ?? 0) > 0 ? 'warning' : 'info'" size="small" effect="plain">
              {{ alertStats.unread ?? 0 }} 未读
            </el-tag>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="12" :sm="12" :md="6" :lg="6">
        <el-card v-loading="attendanceLoading" shadow="hover" class="kpi-card kpi-attendance">
          <div class="kpi-content">
            <div class="kpi-icon">
              <el-icon :size="32"><UserFilled /></el-icon>
            </div>
            <div class="kpi-info">
              <div class="kpi-value">{{ formatRate(attendanceStats.rate) }}</div>
              <div class="kpi-label">出勤率</div>
            </div>
          </div>
          <div class="kpi-footer">
            <span class="kpi-sub">在寝 {{ attendanceStats.present ?? 0 }} / 总计 {{ attendanceStats.total ?? 0 }}</span>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="12" :sm="12" :md="6" :lg="6">
        <el-card v-loading="attendanceLoading" shadow="hover" class="kpi-card kpi-student">
          <div class="kpi-content">
            <div class="kpi-icon">
              <el-icon :size="32"><Avatar /></el-icon>
            </div>
            <div class="kpi-info">
              <div class="kpi-value">{{ attendanceStats.present ?? 0 }}</div>
              <div class="kpi-label">在线学生</div>
            </div>
          </div>
          <div class="kpi-footer">
            <span class="kpi-sub">迟到 {{ attendanceStats.late ?? 0 }} · 陌生人 {{ attendanceStats.stranger ?? 0 }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Row 2: Camera Status + Alert Summary -->
    <el-row :gutter="16" class="detail-row">
      <el-col :xs="24" :sm="24" :md="16" :lg="16">
        <el-card v-loading="camerasLoading" shadow="hover" class="detail-card">
          <template #header>
            <div class="card-header">
              <span>摄像头状态</span>
              <el-tag size="small" effect="plain" type="info">
                在线 {{ cameraStatus.online }} / 离线 {{ cameraStatus.offline }} / 异常 {{ cameraStatus.error }}
              </el-tag>
            </div>
          </template>
          <div v-if="cameraStatus.cameras && cameraStatus.cameras.length > 0" class="camera-grid">
            <div
              v-for="cam in cameraStatus.cameras"
              :key="cam.camera_id ?? cam.id"
              class="camera-item"
            >
              <el-card shadow="never" class="camera-card" :body-style="{ padding: '12px' }">
                <div class="camera-status-dot" :class="statusDotClass(cam.status)"></div>
                <div class="camera-info">
                  <div class="camera-name" :title="cam.name ?? cam.camera_name">
                    {{ cam.name ?? cam.camera_name ?? '—' }}
                  </div>
                  <div class="camera-meta">{{ cam.building ?? cam.building_name ?? '—' }}</div>
                  <div class="camera-fps" v-if="cam.fps !== undefined && cam.fps !== null">
                    {{ cam.fps }} FPS
                  </div>
                </div>
                <el-tag :type="statusTagType(cam.status)" size="small" effect="dark" class="camera-tag">
                  {{ statusLabel(cam.status) }}
                </el-tag>
              </el-card>
            </div>
          </div>
          <el-empty v-else description="暂无摄像头数据" :image-size="60" />
        </el-card>
      </el-col>

      <el-col :xs="24" :sm="24" :md="8" :lg="8">
        <el-card v-loading="alertsLoading" shadow="hover" class="detail-card">
          <template #header>
            <div class="card-header">
              <span>告警概览</span>
              <el-tag size="small" effect="plain" type="danger">
                共 {{ alertStats.total ?? 0 }}
              </el-tag>
            </div>
          </template>
          <div class="alert-summary">
            <div class="alert-item alert-critical">
              <div class="alert-level">
                <el-tag type="danger" effect="dark" size="large" round>严重</el-tag>
              </div>
              <div class="alert-count">{{ alertStats.by_severity?.critical ?? 0 }}</div>
            </div>
            <div class="alert-item alert-high">
              <div class="alert-level">
                <el-tag type="warning" effect="dark" size="large" round>高危</el-tag>
              </div>
              <div class="alert-count">{{ alertStats.by_severity?.high ?? 0 }}</div>
            </div>
            <div class="alert-item alert-medium">
              <div class="alert-level">
                <el-tag type="primary" effect="dark" size="large" round>中等</el-tag>
              </div>
              <div class="alert-count">{{ alertStats.by_severity?.medium ?? 0 }}</div>
            </div>
            <div class="alert-item alert-low">
              <div class="alert-level">
                <el-tag type="info" effect="dark" size="large" round>低级</el-tag>
              </div>
              <div class="alert-count">{{ alertStats.by_severity?.low ?? 0 }}</div>
            </div>
          </div>
          <el-divider />
          <div class="alert-type-breakdown">
            <div class="alert-type-title">按类型统计</div>
            <div v-for="(count, type) in alertStats.by_type" :key="type" class="alert-type-row">
              <span class="alert-type-label">{{ formatEventType(type as string) }}</span>
              <span class="alert-type-count">{{ count }}</span>
            </div>
            <el-empty v-if="!alertStats.by_type || Object.keys(alertStats.by_type).length === 0" description="暂无数据" :image-size="40" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Row 3: Recent Events Table -->
    <el-row :gutter="16" class="events-row">
      <el-col :span="24">
        <el-card v-loading="eventsLoading" shadow="hover" class="detail-card">
          <template #header>
            <div class="card-header">
              <span>最近事件</span>
              <el-tag size="small" effect="plain" type="info">
                共 {{ eventsTotal }} 条
              </el-tag>
            </div>
          </template>
          <el-table :data="events" stripe style="width: 100%" empty-text="暂无事件数据">
            <el-table-column prop="event_time" label="时间" min-width="170">
              <template #default="{ row }">
                {{ formatTime(row.event_time ?? row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column prop="building" label="楼栋" min-width="100">
              <template #default="{ row }">
                {{ row.building ?? row.building_name ?? '—' }}
              </template>
            </el-table-column>
            <el-table-column prop="camera" label="摄像头" min-width="120">
              <template #default="{ row }">
                {{ row.camera ?? row.camera_name ?? '—' }}
              </template>
            </el-table-column>
            <el-table-column prop="event_type" label="事件类型" min-width="110">
              <template #default="{ row }">
                <el-tag :type="eventTypeTagType(row.event_type)" size="small" effect="plain">
                  {{ formatEventType(row.event_type) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="student" label="学生" min-width="100">
              <template #default="{ row }">
                {{ row.student ?? row.student_name ?? '—' }}
              </template>
            </el-table-column>
            <el-table-column prop="confidence" label="置信度" min-width="90">
              <template #default="{ row }">
                <span v-if="row.confidence !== undefined && row.confidence !== null">
                  {{ (row.confidence * 100).toFixed(1) }}%
                </span>
                <span v-else>—</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue';
import { VideoCamera, Bell, UserFilled, Avatar } from '@element-plus/icons-vue';
import { getCamerasStatus, getAlertStats, getAttendanceStats, getRecentEvents } from '@/api/dashboard';

// ─── TypeScript Interfaces ───

interface CameraItem {
  camera_id?: number;
  id?: number;
  name?: string;
  camera_name?: string;
  building?: string;
  building_name?: string;
  status?: string;
  fps?: number;
}

interface CamerasStatusData {
  total: number;
  online: number;
  offline: number;
  error: number;
  cameras: CameraItem[];
}

interface AlertStatsData {
  total: number;
  unread: number;
  today: number;
  by_type: Record<string, number>;
  by_severity: {
    critical?: number;
    high?: number;
    medium?: number;
    low?: number;
  };
}

interface AttendanceStatsData {
  total: number;
  present: number;
  absent: number;
  late: number;
  stranger: number;
  rate: number;
}

interface EventItem {
  id?: number;
  event_time?: string;
  created_at?: string;
  building?: string;
  building_name?: string;
  camera?: string;
  camera_name?: string;
  event_type?: string;
  student?: string;
  student_name?: string;
  confidence?: number;
  [key: string]: unknown;
}

// ─── Reactive State ───

const camerasLoading = ref(false);
const alertsLoading = ref(false);
const attendanceLoading = ref(false);
const eventsLoading = ref(false);

const cameraStatus = reactive<CamerasStatusData>({
  total: 0,
  online: 0,
  offline: 0,
  error: 0,
  cameras: [],
});

const alertStats = reactive<AlertStatsData>({
  total: 0,
  unread: 0,
  today: 0,
  by_type: {},
  by_severity: { critical: 0, high: 0, medium: 0, low: 0 },
});

const attendanceStats = reactive<AttendanceStatsData>({
  total: 0,
  present: 0,
  absent: 0,
  late: 0,
  stranger: 0,
  rate: 0,
});

const events = ref<EventItem[]>([]);
const eventsTotal = ref(0);

let refreshTimer: ReturnType<typeof setInterval> | null = null;

// ─── Data Fetching ───

async function fetchCamerasStatus() {
  camerasLoading.value = true;
  try {
    const res = await getCamerasStatus();
    const data = res.data as CamerasStatusData;
    if (data) {
      Object.assign(cameraStatus, {
        total: data.total ?? 0,
        online: data.online ?? 0,
        offline: data.offline ?? 0,
        error: data.error ?? 0,
        cameras: data.cameras ?? [],
      });
    }
  } catch {
    // error handled by request interceptor
  } finally {
    camerasLoading.value = false;
  }
}

async function fetchAlertStats() {
  alertsLoading.value = true;
  try {
    const res = await getAlertStats();
    const data = res.data as AlertStatsData;
    if (data) {
      Object.assign(alertStats, {
        total: data.total ?? 0,
        unread: data.unread ?? 0,
        today: data.today ?? 0,
        by_type: data.by_type ?? {},
        by_severity: {
          critical: data.by_severity?.critical ?? 0,
          high: data.by_severity?.high ?? 0,
          medium: data.by_severity?.medium ?? 0,
          low: data.by_severity?.low ?? 0,
        },
      });
    }
  } catch {
    // error handled by request interceptor
  } finally {
    alertsLoading.value = false;
  }
}

async function fetchAttendanceStats() {
  attendanceLoading.value = true;
  try {
    const res = await getAttendanceStats();
    const data = res.data as AttendanceStatsData;
    if (data) {
      Object.assign(attendanceStats, {
        total: data.total ?? 0,
        present: data.present ?? 0,
        absent: data.absent ?? 0,
        late: data.late ?? 0,
        stranger: data.stranger ?? 0,
        rate: data.rate ?? 0,
      });
    }
  } catch {
    // error handled by request interceptor
  } finally {
    attendanceLoading.value = false;
  }
}

async function fetchRecentEvents() {
  eventsLoading.value = true;
  try {
    const res = await getRecentEvents(1, 5);
    const data = res.data as { items: EventItem[]; total: number };
    if (data) {
      events.value = data.items ?? [];
      eventsTotal.value = data.total ?? 0;
    }
  } catch {
    // error handled by request interceptor
  } finally {
    eventsLoading.value = false;
  }
}

function fetchAll() {
  fetchCamerasStatus();
  fetchAlertStats();
  fetchAttendanceStats();
  fetchRecentEvents();
}

// ─── Formatters ───

function formatRate(rate: number): string {
  if (rate === undefined || rate === null) return '—';
  // rate may be 0-1 or 0-100
  const pct = rate > 1 ? rate : rate * 100;
  return pct.toFixed(1) + '%';
}

function formatTime(dateStr?: string): string {
  if (!dateStr) return '—';
  try {
    const d = new Date(dateStr);
    if (Number.isNaN(d.getTime())) return dateStr;
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    });
  } catch {
    return dateStr;
  }
}

const eventTypeMap: Record<string, string> = {
  entry: '进入',
  exit: '离开',
  stranger: '陌生人',
  late_return: '晚归',
  absent: '缺勤',
  abnormal: '异常行为',
};

function formatEventType(type?: string): string {
  if (!type) return '—';
  return eventTypeMap[type] ?? type;
}

function eventTypeTagType(type?: string): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (!type) return 'info';
  const map: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
    entry: 'success',
    exit: '',
    stranger: 'danger',
    late_return: 'warning',
    absent: 'warning',
    abnormal: 'danger',
  };
  return map[type] ?? 'info';
}

function statusDotClass(status?: string): string {
  switch (status) {
    case 'online':
      return 'dot-online';
    case 'offline':
      return 'dot-offline';
    case 'error':
      return 'dot-error';
    default:
      return 'dot-offline';
  }
}

function statusTagType(status?: string): '' | 'success' | 'danger' | 'warning' | 'info' {
  switch (status) {
    case 'online':
      return 'success';
    case 'offline':
      return 'danger';
    case 'error':
      return 'warning';
    default:
      return 'info';
  }
}

function statusLabel(status?: string): string {
  switch (status) {
    case 'online':
      return '在线';
    case 'offline':
      return '离线';
    case 'error':
      return '异常';
    default:
      return '未知';
  }
}

// ─── Lifecycle ───

onMounted(() => {
  fetchAll();
  refreshTimer = setInterval(fetchAll, 30000);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<style scoped lang="scss">
.dashboard-container {
  padding: 16px;
}

/* ─── KPI Cards ─── */

.kpi-row {
  margin-bottom: 16px;
}

.kpi-card {
  border-radius: 8px;
  transition: transform 0.2s ease;

  &:hover {
    transform: translateY(-2px);
  }
}

.kpi-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.kpi-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  flex-shrink: 0;
}

.kpi-camera .kpi-icon {
  background: rgba(2, 167, 151, 0.1);
  color: #02a797;
}

.kpi-alert .kpi-icon {
  background: rgba(245, 108, 108, 0.1);
  color: #f56c6c;
}

.kpi-attendance .kpi-icon {
  background: rgba(103, 194, 58, 0.1);
  color: #67c23a;
}

.kpi-student .kpi-icon {
  background: rgba(64, 158, 255, 0.1);
  color: #409eff;
}

.kpi-info {
  flex: 1;
  min-width: 0;
}

.kpi-value {
  font-size: 28px;
  font-weight: 600;
  color: #1f2329;
  line-height: 1.2;

  .kpi-unit {
    font-size: 14px;
    font-weight: 400;
    color: #a5a7a9;
  }
}

.kpi-label {
  font-size: 13px;
  color: #63656a;
  margin-top: 4px;
}

.kpi-footer {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.kpi-sub {
  font-size: 12px;
  color: #a5a7a9;
}

/* ─── Detail Cards ─── */

.detail-row {
  margin-bottom: 16px;
}

.detail-card {
  border-radius: 8px;
  margin-bottom: 0;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-weight: 500;
  font-size: 15px;
  color: #1f2329;
}

/* ─── Camera Grid ─── */

.camera-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}

.camera-card {
  position: relative;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  transition: border-color 0.2s ease;

  &:hover {
    border-color: #02a797;
  }
}

.camera-item {
  position: relative;
}

.camera-status-dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  z-index: 1;
}

.dot-online {
  background-color: #67c23a;
  box-shadow: 0 0 4px rgba(103, 194, 58, 0.6);
}

.dot-offline {
  background-color: #f56c6c;
  box-shadow: 0 0 4px rgba(245, 108, 108, 0.6);
}

.dot-error {
  background-color: #e6a23c;
  box-shadow: 0 0 4px rgba(230, 162, 60, 0.6);
}

.camera-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-right: 48px;
}

.camera-name {
  font-size: 14px;
  font-weight: 500;
  color: #1f2329;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.camera-meta {
  font-size: 12px;
  color: #a5a7a9;
}

.camera-fps {
  font-size: 12px;
  color: #63656a;
  font-variant-numeric: tabular-nums;
}

.camera-tag {
  position: absolute;
  bottom: 12px;
  right: 12px;
}

/* ─── Alert Summary ─── */

.alert-summary {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.alert-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-radius: 6px;
}

.alert-critical {
  background: rgba(245, 108, 108, 0.06);
}

.alert-high {
  background: rgba(230, 162, 60, 0.06);
}

.alert-medium {
  background: rgba(64, 158, 255, 0.06);
}

.alert-low {
  background: rgba(144, 147, 153, 0.06);
}

.alert-count {
  font-size: 20px;
  font-weight: 600;
  color: #1f2329;
  font-variant-numeric: tabular-nums;
}

.alert-type-breakdown {
  margin-top: 4px;
}

.alert-type-title {
  font-size: 13px;
  font-weight: 500;
  color: #63656a;
  margin-bottom: 8px;
}

.alert-type-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px solid #f5f5f5;

  &:last-child {
    border-bottom: none;
  }
}

.alert-type-label {
  font-size: 13px;
  color: #63656a;
}

.alert-type-count {
  font-size: 14px;
  font-weight: 500;
  color: #1f2329;
  font-variant-numeric: tabular-nums;
}

/* ─── Events Row ─── */

.events-row {
  margin-bottom: 16px;
}

/* ─── Responsive ─── */

@media (max-width: 768px) {
  .kpi-card {
    margin-bottom: 12px;
  }

  .camera-grid {
    grid-template-columns: 1fr;
  }
}
</style>