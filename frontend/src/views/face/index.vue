<template>
  <div class="cv-face-page">
    <!-- Filter Bar -->
    <div class="cv-filter-bar">
      <div class="cv-filter-bar__left">
        <el-select
          v-model="selectedCameraId"
          placeholder="选择区域"
          clearable
          class="cv-filter-bar__select"
          @change="handleCameraChange"
        >
          <el-option
            v-for="cam in cameraList"
            :key="cam.id"
            :label="cam.name || cam.camera_id"
            :value="cam.camera_id"
          />
        </el-select>
        <el-input
          v-model="searchKeyword"
          placeholder="搜索姓名 / 学号"
          clearable
          prefix-icon="Search"
          class="cv-filter-bar__input"
          @clear="handleQuery"
          @keyup.enter="handleQuery"
        />
      </div>
      <div class="cv-filter-bar__right">
        <el-button type="primary" icon="Plus" @click="handleAdd">添加人脸</el-button>
        <el-button icon="Upload" @click="handleBatchImport">批量导入</el-button>
      </div>
    </div>

    <!-- Photo Card Grid -->
    <div v-loading="snapshotsLoading" class="cv-grid-wrap">
      <div v-if="filteredSnapshots.length > 0" class="cv-face-grid">
        <div
          v-for="snap in paginatedSnapshots"
          :key="snap.id"
          class="cv-face-card"
          @click="handlePreview(snap)"
        >
          <div class="cv-face-card__photo">
            <el-image
              v-if="snap.snapshot_path"
              :src="snap.snapshot_path"
              fit="cover"
              class="cv-face-card__img"
            >
              <template #error>
                <div class="cv-face-card__placeholder">
                  <el-icon :size="36"><Picture /></el-icon>
                </div>
              </template>
            </el-image>
            <div v-else class="cv-face-card__placeholder">
              <el-icon :size="36"><Picture /></el-icon>
            </div>
            <!-- Hover overlay with action icons -->
            <div class="cv-face-card__overlay">
              <el-button
                circle
                size="small"
                icon="Edit"
                type="primary"
                @click.stop="handleEdit(snap)"
              />
              <el-button
                circle
                size="small"
                icon="Delete"
                type="danger"
                @click.stop="handleDelete(snap)"
              />
            </div>
            <!-- Confidence badge -->
            <el-tag
              v-if="snap.confidence"
              :type="confidenceTagType(snap.confidence)"
              size="small"
              class="cv-face-card__badge"
            >
              {{ (snap.confidence * 100).toFixed(0) }}%
            </el-tag>
          </div>
          <div class="cv-face-card__info">
            <div class="cv-face-card__name">
              {{ snap.student_id || '未知' }}
            </div>
            <div v-if="snap.event_time" class="cv-face-card__meta">
              {{ snap.event_time }}
            </div>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <el-empty
        v-else-if="!snapshotsLoading"
        description="暂无人脸快照数据"
        :image-size="120"
        class="cv-empty"
      />
    </div>

    <!-- Pagination -->
    <div v-if="filteredSnapshots.length > 0" class="cv-pagination">
      <el-pagination
        v-model:current-page="snapshotsPage"
        v-model:page-size="snapshotsSize"
        :total="filteredSnapshots.length"
        :page-sizes="[12, 20, 40, 60]"
        layout="total, sizes, prev, pager, next"
        background
      />
    </div>

    <!-- Photo Preview Dialog -->
    <el-dialog
      v-model="previewVisible"
      title="快照预览"
      width="640px"
      append-to-body
      class="cv-dialog cv-dialog--preview"
      :close-on-click-modal="true"
    >
      <div v-if="previewSnap" class="cv-preview">
        <el-image
          v-if="previewSnap.snapshot_path"
          :src="previewSnap.snapshot_path"
          fit="contain"
          class="cv-preview__img"
        >
          <template #error>
            <div class="cv-face-card__placeholder" style="height: 360px;">
              <el-icon :size="64"><Picture /></el-icon>
            </div>
          </template>
        </el-image>
        <div class="cv-preview__details">
          <div v-if="previewSnap.student_id" class="cv-preview__row">
            <span class="cv-preview__label">学号</span>
            <span class="cv-preview__value">{{ previewSnap.student_id }}</span>
          </div>
          <div v-if="previewSnap.confidence" class="cv-preview__row">
            <span class="cv-preview__label">置信度</span>
            <el-tag :type="confidenceTagType(previewSnap.confidence)" size="small">
              {{ (previewSnap.confidence * 100).toFixed(1) }}%
            </el-tag>
          </div>
          <div v-if="previewSnap.event_time" class="cv-preview__row">
            <span class="cv-preview__label">时间</span>
            <span class="cv-preview__value">{{ previewSnap.event_time }}</span>
          </div>
        </div>
      </div>
    </el-dialog>

    <!-- Add/Edit Face Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="520px"
      append-to-body
      class="cv-dialog"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="学号" prop="student_id">
          <el-input v-model="form.student_id" placeholder="请输入学号" />
        </el-form-item>
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入姓名" />
        </el-form-item>
        <el-form-item label="房间号" prop="room_number">
          <el-input v-model="form.room_number" placeholder="请输入房间号" />
        </el-form-item>
        <el-form-item label="人脸照片" prop="photo">
          <el-upload
            class="cv-face-upload"
            :auto-upload="false"
            :limit="1"
            accept="image/*"
            :on-change="handlePhotoChange"
            :on-remove="handlePhotoRemove"
            list-type="picture-card"
          >
            <el-icon><Plus /></el-icon>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="cv-dialog-footer">
          <el-button @click="cancelForm">取 消</el-button>
          <el-button type="primary" @click="submitForm">确 定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- Batch Import Dialog -->
    <el-dialog
      v-model="batchDialogVisible"
      title="批量导入"
      width="520px"
      append-to-body
      class="cv-dialog"
    >
      <el-upload
        class="cv-batch-upload"
        drag
        :auto-upload="false"
        accept=".csv,.xlsx"
        :on-change="handleBatchFileChange"
        :limit="1"
      >
        <el-icon :size="48" class="cv-batch-upload__icon"><Upload /></el-icon>
        <div class="cv-batch-upload__text">将文件拖到此处，或<em>点击上传</em></div>
        <template #tip>
          <div class="cv-batch-upload__tip">支持 .csv / .xlsx 格式</div>
        </template>
      </el-upload>
      <template #footer>
        <div class="cv-dialog-footer">
          <el-button @click="batchDialogVisible = false">取 消</el-button>
          <el-button type="primary" :disabled="!batchFile" @click="submitBatchImport">开始导入</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { Picture, Plus, Upload } from '@element-plus/icons-vue';
import { getSnapshots, listCameras } from '@/api/face';

// ── 类型定义 ──────────────────────────────────────────

interface Camera {
  id: number;
  camera_id: string;
  name: string;
  building: string;
  status: string;
}

interface Snapshot {
  id: number;
  snapshot_path: string;
  student_id: string;
  confidence: number;
  event_time: string;
}

// ── 摄像头列表 ────────────────────────────────────────

const cameraList = ref<Camera[]>([]);
const selectedCameraId = ref('');

async function fetchCameras() {
  try {
    const res: any = await listCameras();
    cameraList.value = res.data?.items ?? res.data ?? [];
  } catch {
    cameraList.value = [];
  }
}

function handleCameraChange() {
  snapshotsPage.value = 1;
  fetchSnapshots();
}

// ── 快照列表 ───────────────────────────────────────────

const snapshots = ref<Snapshot[]>([]);
const snapshotsLoading = ref(false);
const snapshotsPage = ref(1);
const snapshotsSize = ref(12);
const snapshotsTotal = ref(0);

async function fetchSnapshots() {
  if (!selectedCameraId.value) {
    snapshots.value = [];
    snapshotsTotal.value = 0;
    return;
  }
  snapshotsLoading.value = true;
  try {
    const res: any = await getSnapshots(
      selectedCameraId.value,
      snapshotsPage.value,
      snapshotsSize.value,
    );
    const data = res.data;
    snapshots.value = data?.items ?? [];
    snapshotsTotal.value = data?.total ?? 0;
  } catch {
    snapshots.value = [];
    snapshotsTotal.value = 0;
  } finally {
    snapshotsLoading.value = false;
  }
}

function handleSizeChange() {
  snapshotsPage.value = 1;
  fetchSnapshots();
}

// ── 搜索过滤 ───────────────────────────────────────────

const searchKeyword = ref('');

const filteredSnapshots = computed(() => {
  let list = snapshots.value;
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase();
    list = list.filter(
      (s) =>
        (s.student_id && s.student_id.toLowerCase().includes(kw)),
    );
  }
  return list;
});

const paginatedSnapshots = computed(() => {
  const start = (snapshotsPage.value - 1) * snapshotsSize.value;
  return filteredSnapshots.value.slice(start, start + snapshotsSize.value);
});

function handleQuery() {
  snapshotsPage.value = 1;
  fetchSnapshots();
}

// ── 工具函数 ───────────────────────────────────────────

function confidenceTagType(confidence: number): 'success' | 'warning' | 'danger' {
  if (confidence >= 0.85) return 'success';
  if (confidence >= 0.65) return 'warning';
  return 'danger';
}

// ── 照片预览 ───────────────────────────────────────────

const previewVisible = ref(false);
const previewSnap = ref<Snapshot | null>(null);

function handlePreview(snap: Snapshot) {
  previewSnap.value = snap;
  previewVisible.value = true;
}

// ── 新增/编辑对话框 ────────────────────────────────────

const dialogVisible = ref(false);
const dialogTitle = ref('');
const isEdit = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
  student_id: '',
  name: '',
  room_number: '',
  photo: null as File | null,
});

const rules = reactive<FormRules>({
  student_id: [{ required: true, message: '请输入学号', trigger: 'blur' }],
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
});

function resetForm() {
  form.student_id = '';
  form.name = '';
  form.room_number = '';
  form.photo = null;
  formRef.value?.resetFields();
}

function handleAdd() {
  resetForm();
  isEdit.value = false;
  dialogTitle.value = '添加人脸';
  dialogVisible.value = true;
}

function handleEdit(snap: Snapshot) {
  resetForm();
  isEdit.value = true;
  dialogTitle.value = '编辑人脸';
  form.student_id = snap.student_id || '';
  form.name = '';
  form.room_number = '';
  dialogVisible.value = true;
}

function handlePhotoChange(file: any) {
  form.photo = file.raw;
}

function handlePhotoRemove() {
  form.photo = null;
}

function submitForm() {
  formRef.value?.validate((valid: boolean) => {
    if (!valid) return;
    // TODO: Call face CRUD API when available
    ElMessage.success(isEdit.value ? '修改成功' : '添加成功');
    dialogVisible.value = false;
    fetchSnapshots();
  });
}

function cancelForm() {
  dialogVisible.value = false;
  resetForm();
}

// ── 删除 ───────────────────────────────────────────────

function handleDelete(snap: Snapshot) {
  ElMessageBox.confirm(
    `是否确认删除学号「${snap.student_id || snap.id}」的人脸记录？`,
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    },
  )
    .then(() => {
      // TODO: Call face delete API when available
      ElMessage.success('删除成功');
      fetchSnapshots();
    })
    .catch((err: any) => {
      if (err !== 'cancel' && err !== 'close') {
        ElMessage.error('删除失败');
      }
    });
}

// ── 批量导入 ───────────────────────────────────────────

const batchDialogVisible = ref(false);
const batchFile = ref<File | null>(null);

function handleBatchImport() {
  batchFile.value = null;
  batchDialogVisible.value = true;
}

function handleBatchFileChange(file: any) {
  batchFile.value = file.raw;
}

function submitBatchImport() {
  if (!batchFile.value) return;
  // TODO: Call batch import API when available
  ElMessage.success('批量导入已提交');
  batchDialogVisible.value = false;
  fetchSnapshots();
}

// ── 初始化 ─────────────────────────────────────────────

onMounted(() => {
  fetchCameras();
});
</script>

<style lang="scss" scoped>
// ── Theme Variables ──────────────────────────────────────
$primary-color: #1890FF;
$success-color: #52C41A;
$danger-color: #FF4D4F;
$warning-color: #FAAD14;
$page-bg: #F0F2F5;
$card-bg: #FFFFFF;
$text-primary: #262626;
$text-secondary: #8C8C8C;
$border-color: #F0F0F0;
$shadow-sm: 0 1px 4px rgba(0, 0, 0, 0.04);
$shadow-md: 0 4px 12px rgba(0, 0, 0, 0.08);
$shadow-hover: 0 8px 24px rgba(0, 0, 0, 0.12);
$radius-sm: 8px;
$radius-md: 12px;

// ── Page Container ──────────────────────────────────────
.cv-face-page {
  padding: 20px;
  min-height: 100%;
  background: $page-bg;
}

// ── Filter Bar ──────────────────────────────────────────
.cv-filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: $card-bg;
  border-radius: $radius-sm;
  margin-bottom: 16px;
  box-shadow: $shadow-sm;

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
    width: 180px;
  }
}

// ── Grid Wrapper ────────────────────────────────────────
.cv-grid-wrap {
  min-height: 200px;
}

// ── Face Card Grid ──────────────────────────────────────
.cv-face-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}

// ── Face Card ───────────────────────────────────────────
.cv-face-card {
  background: $card-bg;
  border-radius: $radius-md;
  overflow: hidden;
  box-shadow: $shadow-sm;
  cursor: pointer;
  transition: transform 0.25s ease, box-shadow 0.25s ease;

  &:hover {
    transform: translateY(-4px);
    box-shadow: $shadow-hover;
  }

  &__photo {
    position: relative;
    width: 100%;
    height: 220px;
    overflow: hidden;
    background: #F5F5F5;
  }

  &__img {
    width: 100%;
    height: 100%;
  }

  &__placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    color: #D9D9D9;
    background: linear-gradient(135deg, #FAFAFA 0%, #F0F0F0 100%);
  }

  &__overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    background: rgba(0, 0, 0, 0.45);
    opacity: 0;
    transition: opacity 0.2s ease;
  }

  &:hover &__overlay {
    opacity: 1;
  }

  &__badge {
    position: absolute;
    top: 8px;
    right: 8px;
    backdrop-filter: blur(4px);
    background: rgba(255, 255, 255, 0.85);
  }

  &__info {
    padding: 12px 14px 14px;
  }

  &__name {
    font-size: 15px;
    font-weight: 600;
    color: $text-primary;
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__meta {
    font-size: 12px;
    color: $text-secondary;
    margin-top: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

// ── Empty State ─────────────────────────────────────────
.cv-empty {
  padding: 48px 0;
}

// ── Pagination ───────────────────────────────────────────
.cv-pagination {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
}

// ── Preview Dialog ──────────────────────────────────────
.cv-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;

  &__img {
    width: 100%;
    max-height: 480px;
    border-radius: $radius-sm;
    overflow: hidden;
  }

  &__details {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 0 4px;
  }

  &__row {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &__label {
    font-size: 13px;
    color: $text-secondary;
  }

  &__value {
    font-size: 14px;
    font-weight: 500;
    color: $text-primary;
  }
}

// ── Dialog ──────────────────────────────────────────────
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

// ── Face Upload ──────────────────────────────────────────
.cv-face-upload {
  :deep(.el-upload) {
    border: 2px dashed $border-color;
    border-radius: $radius-sm;
    transition: border-color 0.2s ease;

    &:hover {
      border-color: $primary-color;
    }
  }
}

// ── Batch Upload ────────────────────────────────────────
.cv-batch-upload {
  &__icon {
    color: $primary-color;
    margin-bottom: 8px;
  }

  &__text {
    font-size: 14px;
    color: $text-secondary;

    em {
      color: $primary-color;
      font-style: normal;
    }
  }

  &__tip {
    font-size: 12px;
    color: $text-secondary;
    margin-top: 8px;
  }
}

// ── Responsive ──────────────────────────────────────────
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

  .cv-face-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 12px;
  }

  .cv-face-card__photo {
    height: 160px;
  }
}

@media (min-width: 1600px) {
  .cv-face-grid {
    grid-template-columns: repeat(5, 1fr);
  }
}
</style>