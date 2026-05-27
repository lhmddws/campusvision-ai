<template>
  <div class="app-container">
    <!-- 筛选栏 -->
    <el-form
      ref="queryRef"
      :model="queryParams"
      :inline="true"
      label-width="80px"
    >
      <el-form-item label="楼栋" prop="building">
        <el-input
          v-model="queryParams.building"
          placeholder="请输入楼栋"
          clearable
          style="width: 180px"
          @keyup.enter="handleQuery"
        />
      </el-form-item>
      <el-form-item label="摄像头" prop="camera_id">
        <el-input
          v-model="queryParams.camera_id"
          placeholder="请输入摄像头ID"
          clearable
          style="width: 180px"
          @keyup.enter="handleQuery"
        />
      </el-form-item>
      <el-form-item label="事件类型" prop="event_type">
        <el-select
          v-model="queryParams.event_type"
          placeholder="全部"
          clearable
          style="width: 180px"
        >
          <el-option label="进入" value="entry" />
          <el-option label="离开" value="exit" />
        </el-select>
      </el-form-item>
      <el-form-item label="学生ID" prop="student_id">
        <el-input
          v-model="queryParams.student_id"
          placeholder="请输入学生ID"
          clearable
          style="width: 180px"
          @keyup.enter="handleQuery"
        />
      </el-form-item>
      <el-form-item label="时间范围">
        <el-date-picker
          v-model="dateRange"
          type="datetimerange"
          range-separator="-"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          value-format="YYYY-MM-DDTHH:mm:ss[Z]"
          style="width: 380px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="Search" @click="handleQuery">搜索</el-button>
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </el-form-item>
    </el-form>

    <!-- 事件表格 -->
    <el-table v-loading="loading" :data="eventList" empty-text="暂无数据">
      <el-table-column label="时间" align="center" prop="timestamp" width="180">
        <template #default="scope">
          <span>{{ formatTime(scope.row.timestamp) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="楼栋" align="center" prop="building" :show-overflow-tooltip="true" />
      <el-table-column label="摄像头" align="center" prop="camera_id" :show-overflow-tooltip="true" width="140">
        <template #default="scope">
          <span>{{ scope.row.camera_id || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="事件类型" align="center" prop="event_type" width="100">
        <template #default="scope">
          <el-tag :type="scope.row.event_type === 'entry' ? 'success' : 'warning'">
            {{ scope.row.event_type === 'entry' ? '进入' : '离开' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="学生ID" align="center" prop="student_id" :show-overflow-tooltip="true" width="140">
        <template #default="scope">
          <span v-if="scope.row.is_stranger">
            <el-tag type="danger" size="small">陌生人</el-tag>
          </span>
          <span v-else>{{ scope.row.student_id || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="置信度" align="center" prop="confidence" width="100">
        <template #default="scope">
          <span v-if="scope.row.confidence != null" :style="{ color: confidenceColor(scope.row.confidence) }">
            {{ (scope.row.confidence * 100).toFixed(1) }}%
          </span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="快照" align="center" prop="snapshot_path" width="100">
        <template #default="scope">
          <el-image
            v-if="scope.row.snapshot_path"
            :src="scope.row.snapshot_path"
            :preview-src-list="[scope.row.snapshot_path]"
            fit="cover"
            style="width: 60px; height: 60px"
            preview-teleported
          />
          <span v-else>-</span>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <pagination
      v-show="total > 0"
      v-model:page="queryParams.page"
      v-model:limit="queryParams.size"
      :total="total"
      :page-sizes="[10, 20, 50, 100]"
      @pagination="getList"
    />
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