<template>
  <div class="cv-camera-page">
    <!-- Filter Bar -->
    <div class="cv-filter-bar">
      <div class="cv-filter-bar__left">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索摄像头名称 / ID"
          clearable
          prefix-icon="Search"
          class="cv-filter-bar__input"
          @clear="handleQuery"
          @keyup.enter="handleQuery"
        />
        <el-select
          v-model="queryParams.building"
          placeholder="全部区域"
          clearable
          class="cv-filter-bar__select"
          @change="handleQuery"
        >
          <el-option v-for="b in buildingOptions" :key="b" :label="b" :value="b" />
        </el-select>
        <el-select
          v-model="statusFilter"
          placeholder="全部状态"
          clearable
          class="cv-filter-bar__select"
          @change="handleQuery"
        >
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="异常" value="error" />
        </el-select>
      </div>
      <div class="cv-filter-bar__right">
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
        <el-button type="primary" icon="Plus" @click="handleAdd">添加摄像头</el-button>
      </div>
    </div>

    <!-- Table Card -->
    <div class="cv-table-card">
      <el-table
        v-loading="loading"
        :data="filteredCameraList"
        class="cv-table"
        :header-cell-style="{ background: '#FAFAFA', color: '#262626', fontWeight: 600 }"
        row-class-name="cv-table__row"
      >
        <el-table-column label="名称" prop="name" min-width="140">
          <template #default="{ row }">
            <div class="cv-camera-name">
              <span class="cv-camera-name__text">{{ row.name }}</span>
              <span class="cv-camera-name__id">{{ row.camera_id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="区域" prop="building" min-width="100" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text">{{ row.building }}</span>
          </template>
        </el-table-column>
        <el-table-column label="IP地址" min-width="140" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--mono">{{ extractIp(row.rtsp_url) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" prop="status" min-width="100" align="center">
          <template #default="{ row }">
            <span :class="['cv-status', `cv-status--${row.status}`]">
              <span class="cv-status__dot"></span>
              {{ statusLabel(row.status) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="方向" prop="direction" min-width="80" align="center">
          <template #default="{ row }">
            <el-tag
              :type="directionTagType(row.direction)"
              effect="plain"
              size="small"
              class="cv-direction-tag"
            >
              {{ directionLabel(row.direction) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="帧率" prop="fps_current" min-width="70" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text">{{ row.fps_current != null ? row.fps_current : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="最后在线时间" min-width="160" align="center">
          <template #default="{ row }">
            <span class="cv-cell-text cv-cell-text--secondary">{{
              row.last_heartbeat || '-'
            }}</span>
          </template>
        </el-table-column>
        <el-table-column label="启用" prop="enabled" min-width="80" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enabled"
              :active-color="$primaryColor"
              @change="(val: boolean) => handleToggleEnabled(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="200" align="center" fixed="right">
          <template #default="{ row }">
            <div class="cv-actions">
              <el-button text type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button
                text
                type="primary"
                size="small"
                :loading="healthCheckLoading[row.camera_id]"
                @click="handleHealthCheck(row)"
              >
                健康检查
              </el-button>
              <el-button text type="danger" size="small" @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- Empty State -->
      <el-empty
        v-if="!loading && filteredCameraList.length === 0"
        description="暂无摄像头数据"
        class="cv-empty"
      />

      <!-- Pagination -->
      <div class="cv-pagination">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredCameraList.length"
          layout="total, sizes, prev, pager, next, jumper"
          background
        />
      </div>
    </div>

    <!-- 新增/编辑对话框 (preserved) -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="560px"
      append-to-body
      class="cv-dialog"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="摄像头ID" prop="camera_id">
          <el-input v-model="form.camera_id" placeholder="请输入摄像头ID" :disabled="isEdit" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入摄像头名称" />
        </el-form-item>
        <el-form-item label="楼栋" prop="building">
          <el-input v-model="form.building" placeholder="请输入所属楼栋" />
        </el-form-item>
        <el-form-item label="RTSP地址" prop="rtsp_url">
          <el-input v-model="form.rtsp_url" placeholder="请输入RTSP流地址" />
        </el-form-item>
        <el-form-item label="方向" prop="direction">
          <el-select v-model="form.direction" placeholder="请选择方向">
            <el-option label="入口 (entry)" value="entry" />
            <el-option label="出口 (exit)" value="exit" />
            <el-option label="双向 (both)" value="both" />
          </el-select>
        </el-form-item>
        <el-form-item label="分辨率" prop="resolution">
          <el-input v-model="form.resolution" placeholder="例如 1920x1080" />
        </el-form-item>
        <el-form-item label="备注" prop="remark">
          <el-input v-model="form.remark" type="textarea" :rows="3" placeholder="请输入备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="cv-dialog-footer">
          <el-button @click="cancelForm">取 消</el-button>
          <el-button type="primary" @click="submitForm">确 定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import {
  listCameras,
  addCamera,
  updateCamera,
  deleteCamera,
  healthCheck as healthCheckApi,
} from '@/api/camera';
import type { Camera } from '@/api/camera';

// ==================== 列表数据 ====================
const cameraList = ref<Camera[]>([]);
const loading = ref(false);
const showSearch = ref(true);

const queryParams = reactive({
  building: undefined as string | undefined,
});

/** 楼栋选项（从列表数据中提取去重） */
const buildingOptions = computed(() => {
  const set = new Set<string>();
  for (const c of cameraList.value) {
    set.add(c.building);
  }
  return Array.from(set).sort();
});

/** 查询摄像头列表 */
function getList() {
  loading.value = true;
  listCameras(queryParams.building)
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

/** 搜索 */
function handleQuery() {
  getList();
}

/** 重置搜索 */
function resetQuery() {
  queryParams.building = undefined;
  statusFilter.value = '';
  searchKeyword.value = '';
  getList();
}

// ==================== 前端过滤 & 分页 ====================
const searchKeyword = ref('');
const statusFilter = ref('');

const $primaryColor = '#1890FF';

/** 前端过滤列表 */
const filteredCameraList = computed(() => {
  let list = cameraList.value;
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase();
    list = list.filter(
      c => c.name.toLowerCase().includes(kw) || c.camera_id.toLowerCase().includes(kw),
    );
  }
  if (statusFilter.value) {
    list = list.filter(c => c.status === statusFilter.value);
  }
  return list;
});

/** 分页 */
const pagination = reactive({
  current: 1,
  size: 20,
});

/** 从 RTSP URL 提取 IP */
function extractIp(rtspUrl: string): string {
  if (!rtspUrl) return '-';
  const match = rtspUrl.match(/@([^:/]+)/);
  return match ? match[1] : rtspUrl.replace(/^rtsp:\/\//, '').split(/[:/]/)[0] || '-';
}

// ==================== 状态/方向标签 ====================
const STATUS_MAP: Record<string, { label: string; type: string }> = {
  online: { label: '在线', type: 'success' },
  offline: { label: '离线', type: 'info' },
  error: { label: '异常', type: 'danger' },
};

const DIRECTION_MAP: Record<string, { label: string; type: string }> = {
  entry: { label: '入口', type: '' },
  exit: { label: '出口', type: 'success' },
  both: { label: '双向', type: 'warning' },
};

function statusTagType(status: string): string {
  return STATUS_MAP[status]?.type ?? 'info';
}

function statusLabel(status: string): string {
  return STATUS_MAP[status]?.label ?? status;
}

function directionTagType(direction: string): string {
  return DIRECTION_MAP[direction]?.type ?? '';
}

function directionLabel(direction: string): string {
  return DIRECTION_MAP[direction]?.label ?? direction;
}

// ==================== 启用/禁用切换 ====================
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

// ==================== 健康检查 ====================
const healthCheckLoading = reactive<Record<string, boolean>>({});

function handleHealthCheck(row: Camera) {
  healthCheckLoading[row.camera_id] = true;
  healthCheckApi(row.camera_id)
    .then(() => {
      ElMessage.success('健康检查已触发，请稍后查看状态');
      // 刷新列表以获取最新状态
      getList();
    })
    .catch(() => {
      ElMessage.error('健康检查失败');
    })
    .finally(() => {
      healthCheckLoading[row.camera_id] = false;
    });
}

// ==================== 删除 ====================
function handleDelete(row: Camera) {
  ElMessageBox.confirm(`是否确认删除摄像头「${row.name}」(${row.camera_id})？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      return deleteCamera(row.camera_id);
    })
    .then(() => {
      ElMessage.success('删除成功');
      getList();
    })
    .catch((err: any) => {
      if (err !== 'cancel' && err !== 'close') {
        ElMessage.error('删除失败');
      }
    });
}

// ==================== 新增/编辑对话框 ====================
const dialogVisible = ref(false);
const dialogTitle = ref('');
const isEdit = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
  camera_id: '',
  name: '',
  building: '',
  rtsp_url: '',
  direction: 'entry',
  resolution: '',
  remark: '',
});

const rules = reactive<FormRules>({
  camera_id: [{ required: true, message: '请输入摄像头ID', trigger: 'blur' }],
  name: [{ required: true, message: '请输入摄像头名称', trigger: 'blur' }],
  building: [{ required: true, message: '请输入所属楼栋', trigger: 'blur' }],
  rtsp_url: [{ required: true, message: '请输入RTSP流地址', trigger: 'blur' }],
});

/** 重置表单 */
function resetForm() {
  form.camera_id = '';
  form.name = '';
  form.building = '';
  form.rtsp_url = '';
  form.direction = 'entry';
  form.resolution = '';
  form.remark = '';
  formRef.value?.resetFields();
}

/** 新增 */
function handleAdd() {
  resetForm();
  isEdit.value = false;
  dialogTitle.value = '新增摄像头';
  dialogVisible.value = true;
}

/** 编辑 */
function handleEdit(row: Camera) {
  resetForm();
  isEdit.value = true;
  dialogTitle.value = '编辑摄像头';
  form.camera_id = row.camera_id;
  form.name = row.name;
  form.building = row.building;
  form.rtsp_url = row.rtsp_url;
  form.direction = row.direction;
  form.resolution = row.resolution;
  form.remark = row.remark || '';
  dialogVisible.value = true;
}

/** 提交表单 */
function submitForm() {
  formRef.value?.validate((valid: boolean) => {
    if (!valid) return;

    if (isEdit.value) {
      updateCamera(form.camera_id, {
        name: form.name,
        building: form.building,
        rtsp_url: form.rtsp_url,
        direction: form.direction,
        resolution: form.resolution,
        remark: form.remark,
      } as Partial<Camera>)
        .then(() => {
          ElMessage.success('修改成功');
          dialogVisible.value = false;
          getList();
        })
        .catch(() => {
          ElMessage.error('修改失败');
        });
    } else {
      addCamera({
        camera_id: form.camera_id,
        name: form.name,
        building: form.building,
        rtsp_url: form.rtsp_url,
        direction: form.direction,
        resolution: form.resolution,
        remark: form.remark,
      } as Partial<Camera>)
        .then(() => {
          ElMessage.success('新增成功');
          dialogVisible.value = false;
          getList();
        })
        .catch(() => {
          ElMessage.error('新增失败');
        });
    }
  });
}

/** 取消对话框 */
function cancelForm() {
  dialogVisible.value = false;
  resetForm();
}

// ==================== 初始化 ====================
onMounted(() => {
  getList();
});
</script>

<style lang="scss" scoped>
// Theme variables
$primary-color: #1890ff;
$success-color: #52c41a;
$danger-color: #ff4d4f;
$warning-color: #faad14;
$page-bg: #f0f2f5;
$card-bg: #ffffff;
$text-primary: #262626;
$text-secondary: #8c8c8c;
$border-color: #f0f0f0;

.cv-camera-page {
  padding: 20px;
  min-height: 100%;
  background: $page-bg;
}

// ─── Filter Bar ───
.cv-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: $card-bg;
  border-radius: 8px;
  margin-bottom: 16px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);

  &__left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__input {
    width: 240px;
  }

  &__select {
    width: 140px;
  }
}

// ─── Table Card ───
.cv-table-card {
  background: $card-bg;
  border-radius: 8px;
  padding: 0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.cv-table {
  width: 100%;

  // Row hover
  :deep(.cv-table__row) {
    transition: background-color 0.2s ease;

    &:hover > td {
      background-color: #e6f7ff !important;
    }
  }

  // Remove inner borders for clean look
  :deep(.el-table__inner-wrapper::before) {
    display: none;
  }

  :deep(th.el-table__cell) {
    border-bottom: 1px solid $border-color;
    font-size: 13px;
    letter-spacing: 0.02em;
  }

  :deep(td.el-table__cell) {
    border-bottom: 1px solid $border-color;
  }
}

// ─── Camera Name Cell ───
.cv-camera-name {
  display: flex;
  flex-direction: column;
  gap: 2px;

  &__text {
    font-weight: 500;
    color: $text-primary;
    font-size: 14px;
    line-height: 1.4;
  }

  &__id {
    font-size: 12px;
    color: $text-secondary;
    font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  }
}

// ─── Cell Text ───
.cv-cell-text {
  color: $text-primary;
  font-size: 13px;

  &--mono {
    font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
    font-size: 12px;
    color: $text-secondary;
  }

  &--secondary {
    color: $text-secondary;
    font-size: 12px;
  }
}

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
    color: $success-color;

    .cv-status__dot {
      background-color: $success-color;
      box-shadow: 0 0 0 3px rgba($success-color, 0.2);
    }
  }

  &--offline {
    color: $text-secondary;

    .cv-status__dot {
      background-color: #bfbfbf;
      box-shadow: 0 0 0 3px rgba(#bfbfbf, 0.15);
    }
  }

  &--error {
    color: $danger-color;

    .cv-status__dot {
      background-color: $danger-color;
      box-shadow: 0 0 0 3px rgba($danger-color, 0.2);
    }
  }
}

// ─── Direction Tag ───
.cv-direction-tag {
  font-size: 12px;
}

// ─── Actions ───
.cv-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

// ─── Empty State ───
.cv-empty {
  padding: 48px 0;
}

// ─── Pagination ───
.cv-pagination {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid $border-color;
}

// ─── Dialog ───
.cv-dialog {
  :deep(.el-dialog__header) {
    border-bottom: 1px solid $border-color;
    padding-bottom: 16px;
  }

  :deep(.el-dialog__body) {
    padding: 24px 20px;
  }
}

.cv-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

// ─── Responsive ───
@media (max-width: 768px) {
  .cv-filter-bar {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;

    &__left {
      flex-wrap: wrap;
    }

    &__input {
      width: 100%;
    }

    &__select {
      width: 100%;
    }

    &__right {
      justify-content: flex-end;
    }
  }
}
</style>
