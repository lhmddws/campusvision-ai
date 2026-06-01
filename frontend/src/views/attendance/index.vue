<template>
  <div class="attendance-page">
    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-bar__left">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          class="filter-bar__date"
          @change="handleQuery"
        />
        <el-select
          v-model="selectedBuilding"
          placeholder="全部区域"
          clearable
          class="filter-bar__select"
          @change="handleQuery"
        >
          <el-option
            v-for="b in buildings"
            :key="b.value"
            :label="b.label"
            :value="b.value"
          />
        </el-select>
      </div>
      <div class="filter-bar__right">
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </div>
    </div>

    <!-- KPI Cards -->
    <div class="kpi-grid" v-loading="statsLoading">
      <div class="kpi-card kpi-card--blue">
        <div class="kpi-card__icon">
          <el-icon :size="28"><User /></el-icon>
        </div>
        <div class="kpi-card__body">
          <div class="kpi-card__value">{{ stats.total }}</div>
          <div class="kpi-card__label">应到人数</div>
        </div>
      </div>
      <div class="kpi-card kpi-card--green">
        <div class="kpi-card__icon">
          <el-icon :size="28"><CircleCheck /></el-icon>
        </div>
        <div class="kpi-card__body">
          <div class="kpi-card__value">{{ stats.present }}</div>
          <div class="kpi-card__label">实到人数</div>
        </div>
      </div>
      <div class="kpi-card kpi-card--primary">
        <div class="kpi-card__icon">
          <el-icon :size="28"><TrendCharts /></el-icon>
        </div>
        <div class="kpi-card__body">
          <div class="kpi-card__value">{{ ratePercent }}%</div>
          <div class="kpi-card__label">出勤率</div>
        </div>
        <div class="kpi-card__trend" :class="rateTrendClass">
          <el-icon :size="12"><Top v-if="stats.rate >= 0.9" /><Bottom v-else /></el-icon>
          <span>{{ stats.rate >= 0.9 ? '良好' : '偏低' }}</span>
        </div>
      </div>
    </div>

    <!-- Trend Chart -->
    <div class="chart-card">
      <div class="card-header">
        <span class="card-title">出勤趋势</span>
        <span class="card-header-extra">近7天</span>
      </div>
      <div ref="trendChartRef" class="chart-container"></div>
      <div v-if="dailySummary.length === 0 && !chartLoading" class="chart-empty">
        <el-empty description="暂无趋势数据" :image-size="80" />
      </div>
    </div>

    <!-- Daily Summary Table -->
    <div class="table-card">
      <div class="card-header">
        <span class="card-title">考勤明细</span>
      </div>
      <el-table
        v-loading="tableLoading"
        :data="dailySummary"
        style="width: 100%"
        :default-sort="{ prop: 'date', order: 'descending' }"
        empty-text="暂无考勤数据"
      >
        <el-table-column prop="date" label="日期" sortable align="center" min-width="120" />
        <el-table-column prop="building_name" label="区域" align="center" min-width="120">
          <template #default="{ row }">
            {{ row.building_name || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="出勤率" align="center" min-width="160">
          <template #default="{ row }">
            <div class="rate-cell">
              <el-progress
                :percentage="Math.round((row.checkin_rate ?? 0) * 100)"
                :stroke-width="14"
                :color="getRateColor(row.checkin_rate ?? 0)"
              />
              <span class="rate-cell__text">{{ ((row.checkin_rate ?? 0) * 100).toFixed(1) }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" align="center" min-width="100">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.checkin_rate ?? 0)" size="small" effect="light">
              {{ statusLabel(row.checkin_rate ?? 0) }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue';
import { User, CircleCheck, TrendCharts, Top, Bottom } from '@element-plus/icons-vue';
import { getAttendanceStats, getDailySummary } from '@/api/attendance';
import type { AttendanceStats, DailySummary as DailySummaryType } from '@/api/attendance';
import echarts from '@/plugins/echarts';

/** Building options */
const buildings = [
  { label: 'A栋', value: 'A' },
  { label: 'B栋', value: 'B' },
  { label: 'C栋', value: 'C' },
  { label: 'D栋', value: 'D' },
];

/** Default date range: last 7 days */
function getDefaultDateRange(): [string, string] {
  const end = new Date();
  const start = new Date();
  start.setDate(start.getDate() - 6);
  const fmt = (d: Date) => {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, '0');
    const day = String(d.getDate()).padStart(2, '0');
    return `${y}-${m}-${day}`;
  };
  return [fmt(start), fmt(end)];
}

const dateRange = ref<[string, string]>(getDefaultDateRange());
const selectedBuilding = ref('');

const stats = ref<AttendanceStats>({
  total: 0,
  present: 0,
  absent: 0,
  late: 0,
  stranger: 0,
  rate: 0,
});

const dailySummary = ref<DailySummaryType[]>([]);
const statsLoading = ref(false);
const tableLoading = ref(false);
const chartLoading = ref(false);

/** Attendance rate as percentage (integer) */
const ratePercent = computed(() => Math.round(stats.value.rate * 100));

/** Rate trend indicator class */
const rateTrendClass = computed(() => {
  return stats.value.rate >= 0.9 ? 'trend-up' : 'trend-down';
});

/** Rate color thresholds */
function getRateColor(rate: number): string {
  if (rate >= 0.9) return '#52C41A';
  if (rate >= 0.7) return '#FAAD14';
  return '#FF4D4F';
}

/** Status tag type for el-tag */
function statusTagType(rate: number): '' | 'success' | 'warning' | 'danger' | 'info' {
  if (rate >= 0.9) return 'success';
  if (rate >= 0.7) return 'warning';
  return 'danger';
}

/** Status label text */
function statusLabel(rate: number): string {
  if (rate >= 0.9) return '正常';
  if (rate >= 0.7) return '迟到';
  return '缺勤';
}

/** Build query params */
function buildParams() {
  const params: { building_id?: number; start_date?: string; end_date?: string } = {};
  if (selectedBuilding.value) {
    // Map building letter to ID (A=1, B=2, C=3, D=4)
    const buildingMap: Record<string, number> = { A: 1, B: 2, C: 3, D: 4 };
    params.building_id = buildingMap[selectedBuilding.value];
  }
  if (dateRange.value && dateRange.value.length === 2) {
    params.start_date = dateRange.value[0];
    params.end_date = dateRange.value[1];
  }
  return params;
}

/** Fetch attendance stats */
function fetchStats() {
  statsLoading.value = true;
  getAttendanceStats(buildParams())
    .then((res: any) => {
      const data = res.data ?? res;
      stats.value = {
        total: data.total ?? 0,
        present: data.present ?? 0,
        absent: data.absent ?? 0,
        late: data.late ?? 0,
        stranger: data.stranger ?? 0,
        rate: data.rate ?? 0,
      };
    })
    .catch(() => {
      // Keep default zeros on error
    })
    .finally(() => {
      statsLoading.value = false;
    });
}

/** Fetch daily summary */
function fetchDailySummary() {
  tableLoading.value = true;
  chartLoading.value = true;
  getDailySummary(buildParams())
    .then((res: any) => {
      const data = res.data ?? res;
      dailySummary.value = Array.isArray(data) ? data : [];
    })
    .catch(() => {
      dailySummary.value = [];
    })
    .finally(() => {
      tableLoading.value = false;
      chartLoading.value = false;
    });
}

/** Search */
function handleQuery() {
  fetchStats();
  fetchDailySummary();
}

/** Reset */
function resetQuery() {
  selectedBuilding.value = '';
  dateRange.value = getDefaultDateRange();
  handleQuery();
}

// ─── ECharts ───

const trendChartRef = ref<HTMLElement>();
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let trendChart: any = null;
let resizeHandler: (() => void) | null = null;

function initTrendChart() {
  if (!trendChartRef.value) return;
  try {
    trendChart = echarts.init(trendChartRef.value);
  } catch (e) {
    console.warn('Failed to init trend chart:', e);
  }
}

function updateTrendChart() {
  if (!trendChart || dailySummary.value.length === 0) return;

  const sorted = [...dailySummary.value].sort((a, b) => a.date.localeCompare(b.date));
  const dates = sorted.map((item) => item.date);
  const rates = sorted.map((item) => Math.round((item.checkin_rate ?? 0) * 100));

  trendChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          const p = Array.isArray(params) ? params[0] : params;
          return `${p.axisValue}<br/>出勤率: ${p.value}%`;
        },
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
        data: dates,
        boundaryGap: false,
        axisLabel: { color: '#8C8C8C', fontSize: 12 },
        axisLine: { lineStyle: { color: '#E8E8E8' } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: {
          color: '#8C8C8C',
          fontSize: 12,
          formatter: '{value}%',
        },
        splitLine: { lineStyle: { color: '#F0F0F0', type: 'dashed' } },
        axisLine: { show: false },
        axisTick: { show: false },
      },
      series: [
        {
          type: 'line',
          data: rates,
          smooth: true,
          symbol: 'circle',
          symbolSize: 8,
          lineStyle: { color: '#1890FF', width: 3 },
          itemStyle: { color: '#1890FF', borderWidth: 2, borderColor: '#fff' },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(24,144,255,0.25)' },
                { offset: 1, color: 'rgba(24,144,255,0.02)' },
              ],
            },
          },
        },
      ],
    },
    true
  );
}

// ─── Lifecycle ───

onMounted(async () => {
  handleQuery();

  await nextTick();
  initTrendChart();

  resizeHandler = () => {
    trendChart?.resize();
  };
  window.addEventListener('resize', resizeHandler);
});

onBeforeUnmount(() => {
  trendChart?.dispose();
  if (resizeHandler) {
    window.removeEventListener('resize', resizeHandler);
  }
});

watch(dailySummary, () => {
  updateTrendChart();
}, { deep: true });
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

.attendance-page {
  padding: 20px;
  background: $cv-page-bg;
  min-height: 100%;
}

// ─── Filter Bar ───

.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  background: $cv-card-bg;
  border-radius: 8px;
  padding: 16px 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.filter-bar__left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-bar__right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-bar__date {
  width: 280px;
}

.filter-bar__select {
  width: 140px;
}

// ─── KPI Cards ───

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
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
  position: relative;
  overflow: hidden;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  }

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 4px;
    height: 100%;
  }
}

.kpi-card--blue::before {
  background: $cv-primary;
}

.kpi-card--green::before {
  background: $cv-success;
}

.kpi-card--primary::before {
  background: $cv-primary;
}

.kpi-card__icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-card--blue .kpi-card__icon {
  background: rgba(24, 144, 255, 0.1);
  color: $cv-primary;
}

.kpi-card--green .kpi-card__icon {
  background: rgba(82, 196, 26, 0.1);
  color: $cv-success;
}

.kpi-card--primary .kpi-card__icon {
  background: rgba(24, 144, 255, 0.1);
  color: $cv-primary;
}

.kpi-card__body {
  flex: 1;
  min-width: 0;
}

.kpi-card__value {
  font-size: 28px;
  font-weight: 700;
  color: $cv-text-primary;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.kpi-card__label {
  font-size: 13px;
  color: $cv-text-secondary;
  margin-top: 4px;
}

.kpi-card__trend {
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

// ─── Chart Card ───

.chart-card {
  background: $cv-card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  margin-bottom: 16px;
  position: relative;
}

.chart-container {
  width: 100%;
  height: 320px;
  min-height: 0;
}

.chart-empty {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
}

// ─── Table Card ───

.table-card {
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

.rate-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rate-cell__text {
  font-size: 13px;
  font-weight: 500;
  color: $cv-text-primary;
  white-space: nowrap;
}

// ─── Responsive ───

@media (max-width: 768px) {
  .attendance-page {
    padding: 12px;
  }

  .kpi-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .filter-bar {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .filter-bar__left {
    flex-direction: column;
  }

  .filter-bar__date {
    width: 100%;
  }

  .filter-bar__select {
    width: 100%;
  }

  .chart-container {
    height: 260px;
  }
}
</style>