<template>
  <div class="app-container">
    <el-row :gutter="16">
      <!-- 左栏：快照库 -->
      <el-col :span="14">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>快照库</span>
              <el-select
                v-model="selectedCameraId"
                placeholder="选择摄像头"
                clearable
                style="width: 220px"
                @change="handleCameraChange"
              >
                <el-option
                  v-for="cam in cameraList"
                  :key="cam.id"
                  :label="cam.name || cam.camera_id"
                  :value="cam.camera_id"
                />
              </el-select>
            </div>
          </template>

          <div v-loading="snapshotsLoading">
            <!-- 快照网格 -->
            <div v-if="snapshots.length > 0" class="snapshot-grid">
              <el-card
                v-for="snap in snapshots"
                :key="snap.id"
                shadow="never"
                class="snapshot-card"
              >
                <div class="snapshot-img-wrap">
                  <el-image
                    v-if="snap.snapshot_path"
                    :src="snap.snapshot_path"
                    fit="cover"
                    class="snapshot-img"
                  >
                    <template #error>
                      <div class="image-placeholder">
                        <el-icon :size="32"><Picture /></el-icon>
                      </div>
                    </template>
                  </el-image>
                  <div v-else class="image-placeholder">
                    <el-icon :size="32"><Picture /></el-icon>
                  </div>
                </div>
                <div class="snapshot-info">
                  <div v-if="snap.student_id" class="info-row">
                    <span class="info-label">学号</span>
                    <span class="info-value">{{ snap.student_id }}</span>
                  </div>
                  <div v-if="snap.confidence" class="info-row">
                    <span class="info-label">置信度</span>
                    <el-tag
                      :type="confidenceTagType(snap.confidence)"
                      size="small"
                    >
                      {{ (snap.confidence * 100).toFixed(1) }}%
                    </el-tag>
                  </div>
                  <div v-if="snap.event_time" class="info-row">
                    <span class="info-label">时间</span>
                    <span class="info-value text-muted">{{ snap.event_time }}</span>
                  </div>
                </div>
              </el-card>
            </div>

            <!-- 空状态 -->
            <el-empty
              v-else-if="!snapshotsLoading"
              description="暂无快照数据"
              :image-size="120"
            />

            <!-- 分页 -->
            <div v-if="snapshotsTotal > 0" class="pagination-wrap">
              <el-pagination
                v-model:current-page="snapshotsPage"
                v-model:page-size="snapshotsSize"
                :total="snapshotsTotal"
                :page-sizes="[12, 20, 40]"
                layout="total, sizes, prev, pager, next"
                background
                @current-change="fetchSnapshots"
                @size-change="handleSizeChange"
              />
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 右栏：摄像头状态 -->
      <el-col :span="10">
        <!-- 摄像头状态卡片 -->
        <el-card v-if="cameraList.length > 0" shadow="hover" class="mt-4">
          <template #header>
            <span>摄像头状态</span>
          </template>
          <el-table :data="cameraList" size="small" max-height="240">
            <el-table-column prop="camera_id" label="ID" width="80" />
            <el-table-column prop="name" label="名称" :show-overflow-tooltip="true" />
            <el-table-column prop="building" label="楼栋" width="100" />
            <el-table-column prop="status" label="状态" width="80">
              <template #default="{ row }">
                <el-tag
                  :type="row.status === 'online' ? 'success' : 'danger'"
                  size="small"
                >
                  {{ row.status === 'online' ? '在线' : '离线' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { Picture } from '@element-plus/icons-vue';
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

// ── 工具函数 ───────────────────────────────────────────

function confidenceTagType(confidence: number): 'success' | 'warning' | 'danger' {
  if (confidence >= 0.85) return 'success';
  if (confidence >= 0.65) return 'warning';
  return 'danger';
}

// ── 初始化 ─────────────────────────────────────────────

onMounted(() => {
  fetchCameras();
});
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.snapshot-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.snapshot-card {
  transition: transform 0.2s ease;
}
.snapshot-card:hover {
  transform: translateY(-2px);
}

.snapshot-img-wrap {
  width: 100%;
  height: 140px;
  overflow: hidden;
  border-radius: 4px;
  background-color: var(--el-fill-color-light);
}

.snapshot-img {
  width: 100%;
  height: 100%;
}

.image-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  color: var(--el-text-color-placeholder);
  background-color: var(--el-fill-color-lighter);
}

.snapshot-info {
  padding: 8px 0 0;
  font-size: 13px;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.info-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.info-value {
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.text-muted {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.mt-4 {
  margin-top: 16px;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>