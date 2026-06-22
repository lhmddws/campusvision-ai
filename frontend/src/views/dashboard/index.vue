<template>
  <div class="dashboard-container">
    <!-- Row 1: KPI Cards -->
    <div class="kpi-grid">
      <div class="kpi-card kpi-camera" v-loading="camerasLoading">
        <div class="kpi-icon-circle kpi-icon-blue">
          <el-icon :size="28"><VideoCamera /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">
            {{ cameraStatus.online }}<span class="kpi-unit"> / {{ cameraStatus.total }}</span>
          </div>
          <div class="kpi-label">在线摄像头</div>
        </div>
        <div class="kpi-trend trend-up">
          <el-icon :size="12"><Top /></el-icon>
          <span
            >{{
              cameraStatus.total > 0
                ? ((cameraStatus.online / cameraStatus.total) * 100).toFixed(1)
                : 0
            }}%</span
          >
        </div>
      </div>

      <div class="kpi-card kpi-event" v-loading="eventsLoading">
        <div class="kpi-icon-circle kpi-icon-green">
          <el-icon :size="28"><Sort /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ eventsTotal }}</div>
          <div class="kpi-label">今日进出</div>
        </div>
        <div class="kpi-trend trend-neutral">
          <span>今日</span>
        </div>
      </div>

      <div class="kpi-card kpi-alert" v-loading="alertsLoading">
        <div class="kpi-icon-circle kpi-icon-orange">
          <el-icon :size="28"><Bell /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ alertStats.unread ?? 0 }}</div>
          <div class="kpi-label">告警未处理</div>
        </div>
        <div class="kpi-trend" :class="(alertStats.unread ?? 0) > 0 ? 'trend-down' : 'trend-up'">
          <el-icon :size="12"
            ><Top v-if="(alertStats.unread ?? 0) === 0" /><Bottom v-else
          /></el-icon>
          <span
            >{{
              alertStats.total > 0
                ? (((alertStats.unread ?? 0) / alertStats.total) * 100).toFixed(1)
                : 0
            }}%</span
          >
        </div>
      </div>

      <div class="kpi-card kpi-attendance" v-loading="attendanceLoading">
        <div class="kpi-icon-circle kpi-icon-blue">
          <el-icon :size="28"><UserFilled /></el-icon>
        </div>
        <div class="kpi-body">
          <div class="kpi-value">{{ formatRate(attendanceStats.rate) }}</div>
          <div class="kpi-label">出勤率</div>
        </div>
        <div class="kpi-trend trend-up">
          <span>{{ attendanceStats.present ?? 0 }} / {{ attendanceStats.total ?? 0 }}</span>
        </div>
      </div>
    </div>

    <!-- Row 2: Charts -->
    <div class="charts-grid">
      <div class="chart-card">
        <div class="card-header">
          <span class="card-title">进出趋势</span>
        </div>
        <div ref="trendChartRef" class="chart-container"></div>
      </div>
      <div class="chart-card">
        <div class="card-header">
          <span class="card-title">告警分布</span>
        </div>
        <div ref="alertChartRef" class="chart-container"></div>
      </div>
    </div>

    <!-- Row 3: Activity List -->
    <div class="activity-card">
      <div class="card-header">
        <span class="card-title">实时活动</span>
        <span class="card-header-extra">共 {{ eventsTotal }} 条</span>
      </div>
      <div class="activity-list" v-loading="eventsLoading">
        <div v-if="events.length === 0" class="activity-empty">
          <el-empty description="暂无事件数据" :image-size="60" />
        </div>
        <div v-for="(event, index) in events" :key="event.id ?? index" class="activity-item">
          <div class="activity-dot" :class="eventDotClass(event.event_type)"></div>
          <div class="activity-info">
            <div class="activity-type">{{ formatEventType(event.event_type) }}</div>
            <div class="activity-meta">
              {{ event.building ?? event.building_name ?? '—' }} ·
              {{ event.camera ?? event.camera_name ?? '—' }}
            </div>
          </div>
          <div class="activity-student">{{ event.student ?? event.student_name ?? '—' }}</div>
          <div class="activity-time">{{ formatTime(event.event_time ?? event.created_at) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount, watch, nextTick } from 'vue';
import { VideoCamera, Bell, UserFilled, Top, Bottom, Sort } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import {
  getCamerasStatus,
  getAlertStats,
  getAttendanceStats,
  getRecentEvents,
} from '@/api/dashboard';
import echarts from '@/plugins/echarts';

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

// ─── Chart State ───

const trendChartRef = ref<HTMLElement>();
const alertChartRef = ref<HTMLElement>();
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let trendChart: any = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let alertChart: any = null;
let resizeHandler: (() => void) | null = null;

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
    ElMessage.error('获取摄像头状态数据失败');
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
    ElMessage.error('获取告警统计数据失败');
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
    ElMessage.error('获取出勤统计数据失败');
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
    ElMessage.error('获取最近事件数据失败');
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

function eventDotClass(type?: string): string {
  switch (type) {
    case 'entry':
      return 'dot-entry';
    case 'exit':
      return 'dot-exit';
    case 'stranger':
    case 'late_return':
    case 'absent':
    case 'abnormal':
      return 'dot-alert';
    default:
      return 'dot-default';
  }
}

// ─── ECharts ───

const trendBarColors: Record<string, string> = {
  entry: '#52C41A',
  exit: '#1890FF',
  stranger: '#FF4D4F',
  late_return: '#FAAD14',
  absent: '#8C8C8C',
  abnormal: '#FF4D4F',
};

function initTrendChart() {
  if (!trendChartRef.value) return;
  try {
    trendChart = echarts.init(trendChartRef.value);
  } catch (e) {
    console.warn('Failed to init trend chart:', e);
  }
}

function initAlertChart() {
  if (!alertChartRef.value) return;
  try {
    alertChart = echarts.init(alertChartRef.value);
  } catch (e) {
    console.warn('Failed to init alert chart:', e);
  }
}

function updateTrendChart() {
  if (!trendChart) return;

  const byType = alertStats.by_type ?? {};
  const labels: string[] = [];
  const values: number[] = [];
  const colors: string[] = [];

  // Render in known order
  const knownTypes = ['entry', 'exit', 'stranger', 'late_return', 'absent', 'abnormal'];
  for (const key of knownTypes) {
    if (byType[key] !== undefined) {
      labels.push(eventTypeMap[key] ?? key);
      values.push(byType[key]);
      colors.push(trendBarColors[key] ?? '#8C8C8C');
    }
  }
  // Append unknown types
  for (const [key, val] of Object.entries(byType)) {
    if (!knownTypes.includes(key)) {
      labels.push(key);
      values.push(val);
      colors.push('#8C8C8C');
    }
  }

  trendChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        top: '12%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: labels,
        axisLabel: { color: '#8C8C8C', fontSize: 12 },
        axisLine: { lineStyle: { color: '#E8E8E8' } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: '#8C8C8C', fontSize: 12 },
        splitLine: { lineStyle: { color: '#F0F0F0', type: 'dashed' } },
        axisLine: { show: false },
        axisTick: { show: false },
      },
      series: [
        {
          type: 'bar',
          data: values.map((v, i) => ({
            value: v,
            itemStyle: { color: colors[i], borderRadius: [4, 4, 0, 0] },
          })),
          barWidth: '45%',
          emphasis: {
            itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0,0,0,0.1)' },
          },
        },
      ],
    },
    true,
  );
}

function updateAlertChart() {
  if (!alertChart) return;

  const severity = alertStats.by_severity ?? {};
  const data = [
    { value: severity.critical ?? 0, name: '严重', itemStyle: { color: '#FF4D4F' } },
    { value: severity.high ?? 0, name: '高危', itemStyle: { color: '#FAAD14' } },
    { value: severity.medium ?? 0, name: '中等', itemStyle: { color: '#1890FF' } },
    { value: severity.low ?? 0, name: '低级', itemStyle: { color: '#8C8C8C' } },
  ];

  alertChart.setOption(
    {
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
        textStyle: { color: '#8C8C8C', fontSize: 12 },
        itemWidth: 12,
        itemHeight: 12,
        itemGap: 16,
      },
      series: [
        {
          type: 'pie',
          radius: ['45%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          label: { show: false },
          emphasis: {
            label: { show: true, fontSize: 14, fontWeight: 'bold' },
          },
          labelLine: { show: false },
          data,
        },
      ],
    },
    true,
  );
}

function updateCharts() {
  updateTrendChart();
  updateAlertChart();
}

// ─── Lifecycle ───

onMounted(async () => {
  fetchAll();
  refreshTimer = setInterval(fetchAll, 30000);

  await nextTick();
  initTrendChart();
  initAlertChart();
  updateCharts();

  resizeHandler = () => {
    trendChart?.resize();
    alertChart?.resize();
  };
  window.addEventListener('resize', resizeHandler);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
  trendChart?.dispose();
  alertChart?.dispose();
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler);
  }
});

watch(
  alertStats,
  () => {
    updateCharts();
  },
  { deep: true },
);
</script>

<style scoped lang="scss">
@import '@/assets/styles/variables.module.scss';

// ─── Theme Aliases ───
$cv-primary: $primary-color;
$cv-success: $success-color;
$cv-warning: $warning-color;
$cv-danger: $danger-color;
$cv-page-bg: $page-bg;
$cv-card-bg: $card-bg;
$cv-text-primary: $text-primary;
$cv-text-secondary: $text-secondary;

.dashboard-container {
  padding: 20px;
  background: $cv-page-bg;
  min-height: 100%;
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
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
  position: relative;

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

.kpi-icon-blue {
  background: rgba(24, 144, 255, 0.1);
  color: $cv-primary;
}

.kpi-icon-green {
  background: rgba(82, 196, 26, 0.1);
  color: $cv-success;
}

.kpi-icon-orange {
  background: rgba(250, 173, 20, 0.1);
  color: $cv-warning;
}

.kpi-body {
  flex: 1;
  min-width: 0;
}

.kpi-value {
  font-size: 28px;
  font-weight: 700;
  color: $cv-text-primary;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;

  .kpi-unit {
    font-size: 14px;
    font-weight: 400;
    color: $cv-text-secondary;
  }
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

.trend-up {
  color: $cv-success;
  background: rgba(82, 196, 26, 0.08);
}

.trend-down {
  color: $cv-danger;
  background: rgba(255, 77, 79, 0.08);
}

.trend-neutral {
  color: $cv-text-secondary;
  background: rgba(140, 140, 140, 0.06);
}

// ─── Charts Grid ───

.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

.chart-card {
  background: $cv-card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.chart-container {
  width: 100%;
  height: 320px;
  min-height: 0;
}

// ─── Activity Card ───

.activity-card {
  background: $cv-card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid #f0f0f0;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: $cv-text-primary;
}

.card-header-extra {
  font-size: 12px;
  color: $cv-text-secondary;
}

// ─── Activity List ───

.activity-list {
  padding: 0;
}

.activity-item {
  display: flex;
  align-items: center;
  padding: 14px 24px;
  border-bottom: 1px solid #f5f5f5;
  transition: background-color 0.15s ease;

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background-color: #fafafa;
  }
}

.activity-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-right: 14px;
}

.dot-entry {
  background: $cv-success;
  box-shadow: 0 0 6px rgba(82, 196, 26, 0.4);
}

.dot-exit {
  background: $cv-warning;
  box-shadow: 0 0 6px rgba(250, 173, 20, 0.4);
}

.dot-alert {
  background: $cv-danger;
  box-shadow: 0 0 6px rgba(255, 77, 79, 0.4);
}

.dot-default {
  background: $cv-text-secondary;
  box-shadow: 0 0 6px rgba(140, 140, 140, 0.3);
}

.activity-info {
  flex: 1;
  min-width: 0;
}

.activity-type {
  font-size: 14px;
  font-weight: 500;
  color: $cv-text-primary;
}

.activity-meta {
  font-size: 12px;
  color: $cv-text-secondary;
  margin-top: 2px;
}

.activity-student {
  font-size: 13px;
  color: $cv-text-primary;
  margin-left: 16px;
  white-space: nowrap;
}

.activity-time {
  font-size: 12px;
  color: $cv-text-secondary;
  margin-left: 16px;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.activity-empty {
  padding: 40px 0;
  text-align: center;
}

// ─── Responsive ───

@media (max-width: 1200px) {
  .kpi-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .dashboard-container {
    padding: 12px;
  }

  .kpi-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .kpi-card {
    padding: 16px;
  }

  .charts-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .chart-container {
    height: 260px;
  }
}
</style>
