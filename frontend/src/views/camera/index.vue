<template>
  <div class="app-container">
    <!-- 搜索栏 -->
    <el-form v-show="showSearch" :inline="true" :model="queryParams">
      <el-form-item label="楼栋" prop="building">
        <el-select
          v-model="queryParams.building"
          placeholder="全部楼栋"
          clearable
          style="width: 200px"
          @change="handleQuery"
        >
          <el-option
            v-for="b in buildingOptions"
            :key="b"
            :label="b"
            :value="b"
          />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="Search" @click="handleQuery">搜索</el-button>
        <el-button icon="Refresh" @click="resetQuery">重置</el-button>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="Plus" @click="handleAdd">新增摄像头</el-button>
      </el-form-item>
    </el-form>

    <!-- 摄像头列表 -->
    <el-table v-loading="loading" :data="cameraList" stripe>
      <el-table-column label="摄像头ID" align="center" prop="camera_id" min-width="120" />
      <el-table-column label="名称" align="center" prop="name" min-width="120" />
      <el-table-column label="楼栋" align="center" prop="building" min-width="100" />
      <el-table-column label="方向" align="center" prop="direction" min-width="80">
        <template #default="{ row }">
          <el-tag :type="directionTagType(row.direction)" effect="plain">
            {{ directionLabel(row.direction) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="分辨率" align="center" prop="resolution" min-width="100" />
      <el-table-column label="状态" align="center" prop="status" min-width="80">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)" effect="dark">
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="帧率" align="center" prop="fps_current" min-width="80">
        <template #default="{ row }">
          <span>{{ row.fps_current != null ? row.fps_current : '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="最后心跳" align="center" prop="last_heartbeat" min-width="160">
        <template #default="{ row }">
          <span>{{ row.last_heartbeat || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="启用" align="center" prop="enabled" min-width="80">
        <template #default="{ row }">
          <el-switch
            :model-value="row.enabled"
            @change="(val: boolean) => handleToggleEnabled(row, val)"
          />
        </template>
      </el-table-column>
      <el-table-column label="操作" align="center" class-name="small-padding fixed-width" min-width="180">
        <template #default="{ row }">
          <el-button text type="primary" @click="handleEdit(row)">编辑</el-button>
          <el-button
            text
            type="primary"
            :loading="healthCheckLoading[row.camera_id]"
            @click="handleHealthCheck(row)"
          >
            健康检查
          </el-button>
          <el-button text type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 空状态 -->
    <el-empty v-if="!loading && cameraList.length === 0" description="暂无摄像头数据" />

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px" append-to-body>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="摄像头ID" prop="camera_id">
          <el-input
            v-model="form.camera_id"
            placeholder="请输入摄像头ID"
            :disabled="isEdit"
          />
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
          <el-input
            v-model="form.remark"
            type="textarea"
            :rows="3"
            placeholder="请输入备注"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="submitForm">确 定</el-button>
          <el-button @click="cancelForm">取 消</el-button>
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
  getList();
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
  ElMessageBox.confirm(
    `是否确认删除摄像头「${row.name}」(${row.camera_id})？`,
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
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

<style scoped>
.app-container {
  padding: 20px;
}
</style>