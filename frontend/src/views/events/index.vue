<template>
  <div class="events-page">
    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-bar-inner">
        <el-form
          ref="queryRef"
          :model="queryParams"
          :inline="true"
          class="filter-form"
        >
          <el-form-item label="日期范围" prop="dateRange">
            <el-date-picker
              v-model="dateRange"
              type="datetimerange"
              range-separator="—"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              value-format="YYYY-MM-DDTHH:mm:ss[Z]"
              class="filter-date-picker"
            />
          </el-form-item>
          <el-form-item label="区域" prop="building">
            <el-select
              v-model="queryParams.building"
              placeholder="全部楼栋"
              clearable
              class="filter-select"
            >
              <el-option label="A栋" value="A" />
              <el-option label="B栋" value="B" />
              <el-option label="C栋" value="C" />
              <el-option label="D栋" value="D" />
            </el-select>
          </el-form-item>
          <el-form-item label="事件类型" prop="event_type">
            <el-select
              v-model="queryParams.event_type"
              placeholder="全部类型"
              clearable
              class="filter-select"
            >
              <el-option label="进入" value="entry" />
              <el-option label="离开" value="exit" />
            </el-select>
          </el-form-item>
          <el-form-item label="姓名" prop="student_id">
            <el-input
              v-model="queryParams.student_id"
              placeholder="搜索学生ID"
              clearable
              class="filter-input"
              @keyup.enter="handleQuery"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleQuery" class="filter-btn-primary">
              <el-icon><Search /></el-icon>
              <span>查询</span>
            </el-button>
            <el-button @click="resetQuery" class="filter-btn-reset">
              <el-icon><Refresh /></el-icon>
              <span>重置</span>
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- Events Table -->
    <div class="events-table-card">
      <div class="table-header">
        <div class="table-title">
          <el-icon :size="18" class="table-title-icon"><List /></el-icon>
          <span>事件记录</span>
        </div>
        <div class="table-meta">
          <el-tag type="info" effect="plain" size="small" class="total-tag">
            共 {{ total }} 条
          </el-tag>
        </div>
      </div>

      <el-table
        v-loading="loading"
        :data="eventList"
        empty-text="暂无事件数据"
        class="events-table"
        :header-cell-style="{ background: '#FAFAFA', color: '#262626', fontWeight: 500 }"
        :row-style="{ height: '64px' }"
        :cell-style="{ padding: '8px 0' }"
      >
        <el-table-column label="时间" align="center" prop="timestamp" min-width="170">
          <template #default="scope">
            <span class="cell-time">{{ formatTime(scope.row.timestamp) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="姓名" align="center" prop="student_id" min-width="120">
          <template #default="scope">
            <span v-if="scope.row.is_stranger" class="stranger-badge">
              <el-tag type="danger" size="small" effect="dark" round>陌生人</el-tag>
            </span>
            <span v-else class="cell-name">{{ scope.row.student_id || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="区域" align="center" prop="building" min-width="100">
          <template #default="scope">
            <span class="cell-building">{{ scope.row.building || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="事件类型" align="center" prop="event_type" min-width="100">
          <template #default="scope">
            <el-tag
              :type="scope.row.event_type === 'entry' ? 'success' : 'warning'"
              size="small"
              effect="light"
              class="direction-tag"
            >
              <el-icon v-if="scope.row.event_type === 'entry'" class="tag-icon"><Right /></el-icon>
              <el-icon v-else class="tag-icon"><Back /></el-icon>
              {{ scope.row.event_type === 'entry' ? '进入' : '离开' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="置信度" align="center" prop="confidence" min-width="90">
          <template #default="scope">
            <span v-if="scope.row.confidence != null" class="cell-confidence" :style="{ color: confidenceColor(scope.row.confidence) }">
              {{ (scope.row.confidence * 100).toFixed(1) }}%
            </span>
            <span v-else class="cell-dash">—</span>
          </template>
        </el-table-column>

        <el-table-column label="识别方式" align="center" min-width="100">
          <template #default="scope">
            <el-tag
              v-if="scope.row.is_stranger"
              type="danger"
              size="small"
              effect="plain"
            >
              陌生人
            </el-tag>
            <el-tag
              v-else-if="scope.row.confidence != null && scope.row.confidence >= 0.8"
              type="success"
              size="small"
              effect="plain"
            >
              人脸识别
            </el-tag>
            <el-tag
              v-else
              type="info"
              size="small"
              effect="plain"
            >
              待确认
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="抓拍照片" align="center" prop="snapshot_path" min-width="100">
          <template #default="scope">
            <el-popover
              v-if="scope.row.snapshot_path"
              placement="right"
              :width="280"
              trigger="hover"
              :show-after="200"
            >
              <template #reference>
                <div class="photo-thumb-wrapper">
                  <el-image
                    :src="scope.row.snapshot_path"
                    fit="cover"
                    class="photo-thumb"
                    loading="lazy"
                  >
                    <template #error>
                      <div class="photo-thumb-fallback">
                        <el-icon :size="16"><Picture /></el-icon>
                      </div>
                    </template>
                  </el-image>
                </div>
              </template>
              <div class="photo-preview">
                <el-image
                  :src="scope.row.snapshot_path"
                  fit="contain"
                  class="photo-preview-img"
                >
                  <template #error>
                    <div class="photo-preview-fallback">图片加载失败</div>
                  </template>
                </el-image>
              </div>
            </el-popover>
            <span v-else class="cell-dash">—</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-wrapper">
        <pagination
          v-show="total > 0"
          v-model:page="queryParams.page"
          v-model:limit="queryParams.size"
          :total="total"
          :page-sizes="[10, 20, 50, 100]"
          @pagination="getList"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getEvents, type EventLog } from '@/api/events';
import type { FormInstance } from 'element-plus';
import { onMounted, ref } from 'vue';
import { parseTime } from '@/utils/ruoyi';

/** 查询参数 */
const queryParams = ref({
  page: 1,
  size: 20,
  building: '',
  camera_id: '',
  event_type: '',
  student_id: '',
});

/** 日期范围 */
const dateRange = ref<[string, string] | null>(null);

/** 表格数据 */
const eventList = ref<EventLog[]>([]);
const loading = ref(false);
const total = ref(0);

const queryRef = ref<FormInstance>();

/** 格式化时间 */
function formatTime(time: string | null): string {
  if (!time) return '-';
  return parseTime(time) as string || '-';
}

/** 置信度颜色 */
function confidenceColor(confidence: number): string {
  if (confidence >= 0.8) return '#67c23a';
  if (confidence >= 0.6) return '#e6a23c';
  return '#f56c6c';
}

/** 查询列表 */
function getList() {
  loading.value = true;
  const params: Record<string, unknown> = {
    page: queryParams.value.page,
    size: queryParams.value.size,
  };
  if (queryParams.value.building) params.building = queryParams.value.building;
  if (queryParams.value.camera_id) params.camera_id = queryParams.value.camera_id;
  if (queryParams.value.event_type) params.event_type = queryParams.value.event_type;
  if (queryParams.value.student_id) params.student_id = queryParams.value.student_id;
  if (dateRange.value && dateRange.value.length === 2) {
    params.start_time = dateRange.value[0];
    params.end_time = dateRange.value[1];
  }

  getEvents(params as any)
    .then((res: any) => {
      eventList.value = res.data?.items ?? [];
      total.value = res.data?.total ?? 0;
    })
    .finally(() => {
      loading.value = false;
    });
}

/** 搜索 */
function handleQuery() {
  queryParams.value.page = 1;
  getList();
}

/** 重置 */
function resetQuery() {
  dateRange.value = null;
  queryRef.value?.resetFields();
  queryParams.value = {
    page: 1,
    size: 20,
    building: '',
    camera_id: '',
    event_type: '',
    student_id: '',
  };
  getList();
}

onMounted(() => {
  getList();
});
</script>

<style scoped lang="scss">
$primary-color: #1890FF;
$card-bg: #FFFFFF;
$page-bg: #F0F2F5;
$text-primary: #262626;
$text-secondary: #8C8C8C;
$success-color: #52C41A;
$warning-color: #FAAD14;
$danger-color: #FF4D4F;
$border-color: #E8E8E8;
$hover-bg: #E6F7FF;

.events-page {
  min-height: 100%;
  background: $page-bg;
  padding: 16px;
}

/* ─── Filter Bar ─── */

.filter-bar {
  background: $card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  margin-bottom: 16px;
  overflow: hidden;
}

.filter-bar-inner {
  padding: 16px 20px 4px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0;

  :deep(.el-form-item) {
    margin-bottom: 12px;
    margin-right: 16px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    color: $text-secondary;
    font-weight: 400;
    padding-right: 8px;
  }

  :deep(.el-form-item__content) {
    align-items: center;
  }
}

.filter-date-picker {
  width: 360px;

  :deep(.el-range-separator) {
    color: $text-secondary;
  }

  :deep(.el-input__wrapper) {
    border-radius: 6px;
  }
}

.filter-select {
  width: 140px;

  :deep(.el-input__wrapper) {
    border-radius: 6px;
  }
}

.filter-input {
  width: 180px;

  :deep(.el-input__wrapper) {
    border-radius: 6px;
  }
}

.filter-btn-primary {
  border-radius: 6px;
  font-weight: 500;
  padding: 8px 20px;
  background: $primary-color;
  border-color: $primary-color;

  &:hover,
  &:focus {
    background: lighten($primary-color, 10%);
    border-color: lighten($primary-color, 10%);
  }
}

.filter-btn-reset {
  border-radius: 6px;
  font-weight: 500;
  padding: 8px 20px;
  color: $text-secondary;
  border-color: $border-color;

  &:hover,
  &:focus {
    color: $primary-color;
    border-color: $primary-color;
  }
}

/* ─── Table Card ─── */

.events-table-card {
  background: $card-bg;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 12px;
  border-bottom: 1px solid #F5F5F5;
}

.table-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: $text-primary;
}

.table-title-icon {
  color: $primary-color;
}

.total-tag {
  font-variant-numeric: tabular-nums;
}

/* ─── Table Styles ─── */

.events-table {
  width: 100%;

  :deep(.el-table__header th) {
    font-size: 13px;
    color: $text-secondary;
    font-weight: 500;
    background: #FAFAFA !important;
  }

  :deep(.el-table__row) {
    transition: background-color 0.15s ease;

    &:hover > td {
      background: $hover-bg !important;
    }
  }

  :deep(.el-table__body td) {
    border-bottom: 1px solid #F5F5F5;
  }

  :deep(.el-table__empty-block) {
    min-height: 200px;
  }
}

/* ─── Cell Styles ─── */

.cell-time {
  font-size: 13px;
  color: $text-primary;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.cell-name {
  font-size: 13px;
  color: $text-primary;
  font-weight: 500;
}

.cell-building {
  font-size: 13px;
  color: $text-primary;
}

.cell-confidence {
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.cell-dash {
  color: #D9D9D9;
  font-size: 13px;
}

/* ─── Direction Tags ─── */

.direction-tag {
  font-weight: 500;
  border-radius: 4px;

  .tag-icon {
    margin-right: 2px;
    font-size: 12px;
  }
}

/* ─── Stranger Badge ─── */

.stranger-badge {
  display: inline-flex;
  align-items: center;
}

/* ─── Photo Thumbnails ─── */

.photo-thumb-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #F0F0F0;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: $primary-color;
    box-shadow: 0 2px 8px rgba($primary-color, 0.2);
  }
}

.photo-thumb {
  width: 48px;
  height: 48px;
  display: block;
}

.photo-thumb-fallback {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #F5F5F5;
  color: #BFBFBF;
  border-radius: 6px;
}

.photo-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
}

.photo-preview-img {
  max-width: 260px;
  max-height: 260px;
  border-radius: 4px;
}

.photo-preview-fallback {
  width: 260px;
  height: 180px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #F5F5F5;
  color: #8C8C8C;
  font-size: 14px;
  border-radius: 4px;
}

/* ─── Pagination ─── */

.pagination-wrapper {
  padding: 16px 20px;
  display: flex;
  justify-content: flex-end;
  background: $card-bg;
  border-top: 1px solid #F5F5F5;

  :deep(.el-pagination) {
    padding: 0;
  }

  :deep(.pagination-container) {
    background: transparent;
    padding: 0;
  }
}

/* ─── Responsive ─── */

@media (max-width: 768px) {
  .events-page {
    padding: 8px;
  }

  .filter-bar-inner {
    padding: 12px 12px 4px;
  }

  .filter-form {
    :deep(.el-form-item) {
      margin-right: 8px;
      margin-bottom: 8px;
    }
  }

  .filter-date-picker {
    width: 100%;
  }

  .filter-select {
    width: 120px;
  }

  .filter-input {
    width: 140px;
  }

  .table-header {
    padding: 12px 12px 8px;
  }
}
</style>