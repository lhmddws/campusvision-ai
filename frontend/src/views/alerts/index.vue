<template>
  <div class="app-container">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="mb-4">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__content">
            <div class="stat-card__label">总告警</div>
            <div class="stat-card__value">{{ stats.total }}</div>
          </div>
          <el-icon class="stat-card__icon" :size="40" color="#409EFF"><Warning /></el-icon>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__content">
            <div class="stat-card__label">未读</div>
            <div class="stat-card__value">
              <el-badge :value="stats.unread" :max="999" class="stat-badge">
                <span>{{ stats.unread }}</span>
              </el-badge>
            </div>
          </div>
          <el-icon class="stat-card__icon" :size="40" color="#F56C6C"><Bell /></el-icon>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__content">
            <div class="stat-card__label">今日新增</div>
            <div class="stat-card__value">{{ stats.today }}</div>
          </div>
          <el-icon class="stat-card__icon" :size="40" color="#E6A23C"><Timer /></el-icon>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card__content">
            <div class="stat-card__label">已处理</div>
            <div class="stat-card__value">{{ resolvedCount }}</div>
          </div>
          <el-icon class="stat-card__icon" :size="40" color="#67C23A"><CircleCheck /></el-icon>
        </el-card>
      </el-col>
    </el-row>

    <!-- 筛选栏 -->
    <el-form ref="queryRef" :model="queryParams" :inline="true" label-width="68px">
      <el-form-item label="楼栋" prop="building">
        <el-input
          v-model="queryParams.building"
          placeholder="请输入楼栋"
          clearable
          style="width: 200px"
          @keyup.enter="handleQuery"
        />
      </el-form-item>
      <el-form-item label="告警类型" prop="alert_type">
        <el-select
          v-model="queryParams.alert_type"
          placeholder="全部"
          clearable
          style="width: 200px"
        >
          <el-option label="陌生人" value="stranger" />
          <el-option label="晚归" value="late_return" />
          <el-option label="缺勤" value="absence" />
          <el-option label="异常" value="abnormal" />
        </el-select>
      </el-form-item>
      <el-form-item label="确认状态" prop="acknowledged">
        <el-select
          v-model="queryParams.acknowledged"
          placeholder="全部"
          clearable
          style="width: 200px"
        >
          <el-option label="未确认" value="false" />
          <el-option label="已确认" value="true" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="Search" @click="handleQuery">搜索</el-button>
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 告警列表 -->
    <el-table v-loading="loading" :data="alertList" style="width: 100%">
      <el-table-column label="告警ID" align="center" prop="alert_id" width="160" :show-overflow-tooltip="true" />
      <el-table-column label="类型" align="center" prop="alert_type" width="100">
        <template #default="{ row }">
          <el-tag :type="alertTypeTag(row.alert_type).type" effect="dark">
            {{ alertTypeTag(row.alert_type).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="楼栋" align="center" prop="building" width="120" :show-overflow-tooltip="true">
        <template #default="{ row }">
          {{ row.building || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="学生ID" align="center" prop="student_id" width="120" :show-overflow-tooltip="true">
        <template #default="{ row }">
          {{ row.student_id || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="严重程度" align="center" prop="severity" width="100">
        <template #default="{ row }">
          <el-tag :type="severityTag(row.severity).type" effect="plain">
            {{ severityTag(row.severity).label }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="描述" align="center" prop="description" min-width="200" :show-overflow-tooltip="true">
        <template #default="{ row }">
          <el-tooltip
            v-if="row.description && row.description.length > 30"
            :content="row.description"
            placement="top"
          >
            <span>{{ row.description.slice(0, 30) + '...' }}</span>
          </el-tooltip>
          <span v-else>{{ row.description || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" align="center" prop="is_resolved" width="90">
        <template #default="{ row }">
          <el-tag :type="row.is_resolved ? 'success' : 'info'" effect="light">
            {{ row.is_resolved ? '已处理' : '未处理' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="时间" align="center" prop="occurred_at" width="170">
        <template #default="{ row }">
          <span>{{ parseTime(row.occurred_at) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" align="center" width="100" class-name="small-padding fixed-width">
        <template #default="{ row }">
          <el-button
            type="primary"
            text
            :disabled="row.is_resolved"
            @click="handleAcknowledge(row)"
          >
            确认
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

onMounted(() => {
  getList();
  getStats();
});
</script>

<style scoped>
.stat-card {
  position: relative;
  overflow: hidden;
}

.stat-card :deep(.el-card__body) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px;
}

.stat-card__content {
  flex: 1;
}

.stat-card__label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-card__value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  line-height: 1;
}

.stat-card__icon {
  flex-shrink: 0;
  opacity: 0.85;
}

.stat-badge :deep(.el-badge__content) {
  top: -4px;
}
</style>