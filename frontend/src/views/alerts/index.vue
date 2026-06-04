<template>
  <div class="alerts-page">
    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card stat-card--total">
        <div class="stat-card__icon-wrap">
          <el-icon :size="28" color="#1890FF"><Warning /></el-icon>
        </div>
        <div class="stat-card__content">
          <div class="stat-card__value">{{ stats.total }}</div>
          <div class="stat-card__label">总告警</div>
        </div>
      </div>
      <div class="stat-card stat-card--unread">
        <div class="stat-card__icon-wrap">
          <el-icon :size="28" color="#FF4D4F"><Bell /></el-icon>
        </div>
        <div class="stat-card__content">
          <div class="stat-card__value">{{ stats.unread }}</div>
          <div class="stat-card__label">未读</div>
        </div>
      </div>
      <div class="stat-card stat-card--today">
        <div class="stat-card__icon-wrap">
          <el-icon :size="28" color="#FAAD14"><Timer /></el-icon>
        </div>
        <div class="stat-card__content">
          <div class="stat-card__value">{{ stats.today }}</div>
          <div class="stat-card__label">今日新增</div>
        </div>
      </div>
      <div class="stat-card stat-card--resolved">
        <div class="stat-card__icon-wrap">
          <el-icon :size="28" color="#52C41A"><CircleCheck /></el-icon>
        </div>
        <div class="stat-card__content">
          <div class="stat-card__value">{{ resolvedCount }}</div>
          <div class="stat-card__label">已处理</div>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-bar__left">
        <el-select
          v-model="queryParams.alert_type"
          placeholder="告警类型"
          clearable
          class="filter-select"
        >
          <el-option label="陌生人" value="stranger" />
          <el-option label="晚归" value="late_return" />
          <el-option label="缺勤" value="absence" />
          <el-option label="异常" value="abnormal" />
        </el-select>
        <el-select v-model="severityFilter" placeholder="级别" clearable class="filter-select">
          <el-option label="高危" value="high" />
          <el-option label="中等" value="medium" />
          <el-option label="低" value="low" />
        </el-select>
        <el-select
          v-model="queryParams.acknowledged"
          placeholder="确认状态"
          clearable
          class="filter-select"
        >
          <el-option label="未确认" value="false" />
          <el-option label="已确认" value="true" />
        </el-select>
        <el-input
          v-model="queryParams.building"
          placeholder="搜索楼栋"
          clearable
          prefix-icon="Search"
          class="filter-input"
          @keyup.enter="handleQuery"
        />
      </div>
      <div class="filter-bar__right">
        <el-button type="primary" icon="Search" class="filter-btn-primary" @click="handleQuery"
          >搜索</el-button
        >
        <el-button icon="Refresh" class="filter-btn-reset" @click="resetQuery">重置</el-button>
        <el-button
          type="warning"
          icon="Check"
          class="filter-btn-batch"
          @click="handleBatchAcknowledge"
          >全部确认</el-button
        >
      </div>
    </div>

    <!-- 告警列表 -->
    <div class="alerts-table-card">
      <div class="table-header">
        <span class="table-title">告警列表</span>
        <el-tag class="total-tag" effect="plain" round>{{ total }} 条记录</el-tag>
      </div>
      <el-table
        v-loading="loading"
        :data="filteredList"
        style="width: 100%"
        :header-cell-style="{ background: '#FAFAFA', color: '#262626', fontWeight: 600 }"
        :row-style="{ cursor: 'default' }"
      >
        <el-table-column label="级别" align="center" width="100">
          <template #default="{ row }">
            <el-tag
              :type="severityBadgeType(row.severity)"
              effect="dark"
              round
              class="severity-badge"
            >
              {{ severityLabel(row.severity) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" align="center" prop="occurred_at" width="170">
          <template #default="{ row }">
            <span class="time-text">{{ parseTime(row.occurred_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="来源" align="center" width="100">
          <template #default="{ row }">
            <span class="source-text">{{ row.building || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="描述" align="left" min-width="200" :show-overflow-tooltip="true">
          <template #default="{ row }">
            <el-tooltip
              v-if="row.description && row.description.length > 40"
              :content="row.description"
              placement="top"
            >
              <span class="desc-text">{{ row.description.slice(0, 40) + '...' }}</span>
            </el-tooltip>
            <span v-else class="desc-text">{{ row.description || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" align="center" width="100">
          <template #default="{ row }">
            <el-tag
              :type="row.is_resolved ? 'success' : 'warning'"
              effect="light"
              round
              class="status-tag"
            >
              {{ row.is_resolved ? '已确认' : '待处理' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" align="center" width="180" class-name="action-column">
          <template #default="{ row }">
            <el-button
              type="primary"
              text
              size="small"
              :disabled="row.is_resolved"
              @click="handleAcknowledge(row)"
            >
              确认
            </el-button>
            <el-button type="info" text size="small" @click="handleDetail(row)">
              查看详情
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <pagination
        v-show="total > 0"
        v-model:page="queryParams.page"
        v-model:limit="queryParams.size"
        :total="total"
        :page-sizes="[10, 20, 50]"
        @pagination="getList"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Warning, Bell, Timer, CircleCheck } from '@element-plus/icons-vue';
import { getAlerts, acknowledgeAlert, getAlertStats } from '@/api/alerts';
import type { Alert, AlertStats, AlertListResult } from '@/api/alerts';
import { parseTime } from '@/utils/ruoyi';

/** 告警类型映射 */
const ALERT_TYPE_MAP: Record<string, { label: string; type: string }> = {
  stranger: { label: '陌生人', type: 'danger' },
  late_return: { label: '晚归', type: 'warning' },
  absence: { label: '缺勤', type: 'info' },
  abnormal: { label: '异常', type: 'warning' },
};

/** 严重程度映射 */
const SEVERITY_MAP: Record<string, { label: string; type: string }> = {
  critical: { label: '严重', type: 'danger' },
  high: { label: '高危', type: 'warning' },
  medium: { label: '中等', type: 'primary' },
  low: { label: '低', type: 'info' },
};

/** 告警类型标签 */
function alertTypeTag(type: string) {
  return ALERT_TYPE_MAP[type] || { label: type, type: 'info' };
}

/** 严重程度标签 */
function severityTag(severity: string) {
  return SEVERITY_MAP[severity] || { label: severity, type: 'info' };
}

/** 告警列表数据 */
const alertList = ref<Alert[]>([]);
const loading = ref(true);
const total = ref(0);

/** 告警统计 */
const stats = ref<AlertStats>({
  total: 0,
  unread: 0,
  today: 0,
  by_type: {},
  by_severity: {},
});

/** 已处理数 = total - unread 近似，或从 by_severity 推导 */
const resolvedCount = computed(() => {
  return stats.value.total - stats.value.unread;
});

/** 查询参数 */
const queryParams = reactive({
  page: 1,
  size: 20,
  building: '',
  alert_type: '',
  acknowledged: '',
});

/** 级别筛选 (client-side) */
const severityFilter = ref('');

/** 级别 → el-tag type 映射 */
function severityBadgeType(severity: string): string {
  const map: Record<string, string> = {
    critical: 'danger',
    high: 'danger',
    medium: 'warning',
    low: 'info',
  };
  return map[severity] || 'info';
}

/** 级别 → 中文标签 */
function severityLabel(severity: string): string {
  const map: Record<string, string> = {
    critical: '严重',
    high: '高危',
    medium: '中等',
    low: '低',
  };
  return map[severity] || severity;
}

/** 客户端级别筛选 */
const filteredList = computed(() => {
  if (!severityFilter.value) return alertList.value;
  return alertList.value.filter(a => a.severity === severityFilter.value);
});

/** 获取告警列表 */
function getList() {
  loading.value = true;
  const params: Record<string, any> = {
    page: queryParams.page,
    size: queryParams.size,
  };
  if (queryParams.building) params.building = queryParams.building;
  if (queryParams.alert_type) params.alert_type = queryParams.alert_type;
  if (queryParams.acknowledged) params.acknowledged = queryParams.acknowledged;

  getAlerts(params)
    .then((res: any) => {
      const data = res.data as AlertListResult;
      alertList.value = data.items || [];
      total.value = data.total || 0;
    })
    .finally(() => {
      loading.value = false;
    });
}

/** 获取告警统计 */
function getStats() {
  const building = queryParams.building || undefined;
  getAlertStats(building).then((res: any) => {
    stats.value = res.data as AlertStats;
  });
}

/** 搜索按钮 */
function handleQuery() {
  queryParams.page = 1;
  getList();
  getStats();
}

/** 重置按钮 */
function resetQuery() {
  queryParams.building = '';
  queryParams.alert_type = '';
  queryParams.acknowledged = '';
  queryParams.page = 1;
  queryParams.size = 20;
  severityFilter.value = '';
  handleQuery();
}

/** 确认告警 */
function handleAcknowledge(row: Alert) {
  ElMessageBox.confirm(`是否确认告警 "${row.alert_id}"？`, '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      return acknowledgeAlert(row.id);
    })
    .then(() => {
      ElMessage.success('确认成功');
      getList();
      getStats();
    })
    .catch(() => {
      // 用户取消
    });
}

/** 批量确认 */
function handleBatchAcknowledge() {
  const unresolved = alertList.value.filter(a => !a.is_resolved);
  if (unresolved.length === 0) {
    ElMessage.info('没有待处理的告警');
    return;
  }
  ElMessageBox.confirm(`是否确认全部 ${unresolved.length} 条待处理告警？`, '批量确认', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(async () => {
      let successCount = 0;
      for (const alert of unresolved) {
        try {
          await acknowledgeAlert(alert.id);
          successCount++;
        } catch {
          // skip failed
        }
      }
      ElMessage.success(`成功确认 ${successCount} 条告警`);
      getList();
      getStats();
    })
    .catch(() => {
      // 用户取消
    });
}

/** 查看详情 */
function handleDetail(row: Alert) {
  // TODO: navigate to detail page or open drawer
  ElMessage.info(`查看告警详情: ${row.alert_id}`);
}

onMounted(() => {
  getList();
  getStats();
});
</script>

<style lang="scss" scoped>
@use '@/assets/styles/variables.module.scss' as vars;

$primary: vars.$primary-color;
$success: vars.$success-color;
$warning: vars.$warning-color;
$danger: vars.$danger-color;
$page-bg: vars.$page-bg;
$card-bg: vars.$card-bg;
$text-primary: vars.$text-primary;
$text-secondary: vars.$text-secondary;
$border-color: #f0f0f0;
$hover-bg: #e6f7ff;

.alerts-page {
  padding: 16px;
  background: $page-bg;
  min-height: 100vh;
}

/* ── Stats Row ── */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  background: $card-bg;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  transition:
    box-shadow 0.2s,
    transform 0.2s;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    transform: translateY(-2px);
  }

  &__icon-wrap {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  &--total .stat-card__icon-wrap {
    background: rgba(24, 144, 255, 0.1);
  }
  &--unread .stat-card__icon-wrap {
    background: rgba(255, 77, 79, 0.1);
  }
  &--today .stat-card__icon-wrap {
    background: rgba(250, 173, 20, 0.1);
  }
  &--resolved .stat-card__icon-wrap {
    background: rgba(82, 196, 26, 0.1);
  }

  &__content {
    flex: 1;
    min-width: 0;
  }

  &__value {
    font-size: 28px;
    font-weight: 700;
    color: $text-primary;
    line-height: 1.2;
  }

  &__label {
    font-size: 13px;
    color: $text-secondary;
    margin-top: 4px;
  }
}

/* ── Filter Bar ── */
.filter-bar {
  background: $card-bg;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  flex-wrap: wrap;

  &__left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.filter-select {
  width: 140px;
}

.filter-input {
  width: 180px;
}

.filter-btn-primary {
  background: $primary;
  border-color: $primary;
}

.filter-btn-reset {
  border-color: $border-color;
  color: $text-secondary;
}

.filter-btn-batch {
  background: $warning;
  border-color: $warning;
  color: #fff;
}

/* ── Table Card ── */
.alerts-table-card {
  background: $card-bg;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.table-title {
  font-size: 16px;
  font-weight: 600;
  color: $text-primary;
}

.total-tag {
  background: rgba(24, 144, 255, 0.08);
  color: $primary;
  border-color: rgba(24, 144, 255, 0.2);
}

/* ── Severity Badges ── */
.severity-badge {
  font-weight: 600;
  min-width: 52px;
  text-align: center;
}

/* ── Status Tags ── */
.status-tag {
  font-weight: 500;
  min-width: 60px;
  text-align: center;
}

/* ── Table Row Styles ── */
:deep(.el-table) {
  --el-table-border-color: #{$border-color};
  --el-table-header-bg-color: #fafafa;

  .el-table__row:hover > td {
    background: $hover-bg !important;
  }
}

.time-text {
  font-size: 13px;
  color: $text-secondary;
  font-variant-numeric: tabular-nums;
}

.source-text {
  font-weight: 500;
  color: $text-primary;
}

.desc-text {
  color: $text-primary;
  line-height: 1.5;
}

.action-column {
  .el-button {
    padding: 4px 8px;
  }
}

/* ── Responsive ── */
@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }

  .filter-bar {
    flex-direction: column;
    align-items: stretch;

    &__left,
    &__right {
      flex-wrap: wrap;
    }

    .filter-select,
    .filter-input {
      width: 100%;
    }
  }
}
</style>
