<template>
  <div class="app-container attendance-page">
    <!-- 筛选栏 -->
    <el-form ref="queryRef" :model="queryParams" :inline="true" class="filter-bar">
      <el-form-item label="楼栋ID" prop="building_id">
        <el-input-number
          v-model="queryParams.building_id"
          :min="1"
          :controls="false"
          placeholder="请输入楼栋ID"
          style="width: 160px"
        />
      </el-form-item>
      <el-form-item label="日期范围" prop="dateRange">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 280px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="Search" @click="handleQuery">搜索</el-button>
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row" v-loading="statsLoading">
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-card--blue">
          <div class="stat-card__inner">
            <span class="stat-card__label">总人数</span>
            <span class="stat-card__value">{{ stats.total }}</span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-card--green">
          <div class="stat-card__inner">
            <span class="stat-card__label">在寝</span>
            <span class="stat-card__value">{{ stats.present }}</span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-card--red">
          <div class="stat-card__inner">
            <span class="stat-card__label">未归</span>
            <span class="stat-card__value">{{ stats.absent }}</span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="6">
        <el-card shadow="hover" class="stat-card stat-card--orange">
          <div class="stat-card__inner">
            <span class="stat-card__label">晚归</span>
            <span class="stat-card__value">{{ stats.late }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 出勤率进度条 -->
    <el-card shadow="never" class="rate-card">
      <div class="rate-card__header">
        <span class="rate-card__title">出勤率</span>
        <span class="rate-card__percent">{{ ratePercent }}</span>
      </div>
      <el-progress
        :percentage="ratePercent"
        :stroke-width="20"
        :color="rateColor"
        :text-inside="true"
        :format="() => ''"
      />
    </el-card>

    <!-- 每日汇总表格 -->
    <el-card shadow="never" class="table-card">
      <template #header>
        <span class="table-card__title">每日汇总</span>
      </template>
      <el-table
        v-loading="tableLoading"
        :data="dailySummary"
        style="width: 100%"
        :default-sort="{ prop: 'date', order: 'descending' }"
        empty-text="暂无考勤数据"
      >
        <el-table-column prop="date" label="日期" sortable align="center" min-width="120" />
        <el-table-column prop="building_name" label="楼栋名称" align="center" min-width="140" />
        <el-table-column label="签到率" align="center" min-width="200">
          <template #default="{ row }">
            <div class="rate-cell">
              <el-progress
                :percentage="Math.round(row.checkin_rate * 100)"
                :stroke-width="14"
                :color="getRateColor(row.checkin_rate)"
              />
              <span class="rate-cell__text">{{ (row.checkin_rate * 100).toFixed(1) }}%</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { getAttendanceStats, getDailySummary } from '@/api/attendance';
import type { AttendanceStats, DailySummary as DailySummaryType } from '@/api/attendance';
import type { FormInstance } from 'element-plus';

/** 默认日期范围：最近7天 */
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

const queryRef = ref<FormInstance>();
const dateRange = ref<[string, string]>(getDefaultDateRange());

const queryParams = reactive({
  building_id: undefined as number | undefined,
  start_date: '' as string,
  end_date: '' as string,
});

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

/** 出勤率百分比（整数） */
const ratePercent = computed(() => Math.round(stats.value.rate * 100));

/** 出勤率进度条颜色 */
const rateColor = computed(() => {
  const r = stats.value.rate;
  if (r >= 0.9) return '#30b08f';
  if (r >= 0.7) return '#fec171';
  return '#c03639';
});

/** 根据签到率返回颜色 */
function getRateColor(rate: number): string {
  if (rate >= 0.9) return '#30b08f';
  if (rate >= 0.7) return '#fec171';
  return '#c03639';
}

/** 构建查询参数 */
function buildParams() {
  const params: { building_id?: number; start_date?: string; end_date?: string } = {};
  if (queryParams.building_id !== undefined && queryParams.building_id !== null) {
    params.building_id = queryParams.building_id;
  }
  if (dateRange.value && dateRange.value.length === 2) {
    params.start_date = dateRange.value[0];
    params.end_date = dateRange.value[1];
  }
  return params;
}

/** 查询考勤统计 */
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
    .finally(() => {
      statsLoading.value = false;
    });
}

/** 查询每日汇总 */
function fetchDailySummary() {
  tableLoading.value = true;
  getDailySummary(buildParams())
    .then((res: any) => {
      const data = res.data ?? res;
      dailySummary.value = Array.isArray(data) ? data : [];
    })
    .finally(() => {
      tableLoading.value = false;
    });
}

/** 搜索 */
function handleQuery() {
  fetchStats();
  fetchDailySummary();
}

/** 重置 */
function resetQuery() {
  queryParams.building_id = undefined;
  dateRange.value = getDefaultDateRange();
  queryRef.value?.resetFields();
  handleQuery();
}

onMounted(() => {
  handleQuery();
});
</script>

<style scoped lang="scss">
.attendance-page {
  .filter-bar {
    margin-bottom: 16px;
  }

  /* ---- 统计卡片 ---- */
  .stats-row {
    margin-bottom: 16px;
  }

  .stat-card {
    border-radius: 8px;
    border: none;
    transition: transform 0.2s ease, box-shadow 0.2s ease;

    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    }

    :deep(.el-card__body) {
      padding: 20px;
    }
  }

  .stat-card__inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .stat-card__label {
    font-size: 14px;
    color: #63656a;
  }

  .stat-card__value {
    font-size: 32px;
    font-weight: 700;
    line-height: 1.2;
  }

  /* 卡片颜色主题 */
  .stat-card--blue {
    border-top: 3px solid #3a71a8;
    .stat-card__value {
      color: #3a71a8;
    }
  }

  .stat-card--green {
    border-top: 3px solid #30b08f;
    .stat-card__value {
      color: #30b08f;
    }
  }

  .stat-card--red {
    border-top: 3px solid #c03639;
    .stat-card__value {
      color: #c03639;
    }
  }

  .stat-card--orange {
    border-top: 3px solid #ff9d2b;
    .stat-card__value {
      color: #ff9d2b;
    }
  }

  /* ---- 出勤率卡片 ---- */
  .rate-card {
    margin-bottom: 16px;
    border-radius: 8px;

    :deep(.el-card__body) {
      padding: 16px 20px;
    }
  }

  .rate-card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .rate-card__title {
    font-size: 16px;
    font-weight: 600;
    color: #1f2329;
  }

  .rate-card__percent {
    font-size: 24px;
    font-weight: 700;
    color: #1f2329;
  }

  /* ---- 每日汇总表格 ---- */
  .table-card {
    border-radius: 8px;

    :deep(.el-card__body) {
      padding: 0;
    }
  }

  .table-card__title {
    font-size: 16px;
    font-weight: 600;
    color: #1f2329;
  }

  .rate-cell {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .rate-cell__text {
    font-size: 13px;
    font-weight: 500;
    color: #1f2329;
    white-space: nowrap;
  }
}
</style>