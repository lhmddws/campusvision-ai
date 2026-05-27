<template>
  <div class="app-container config-page">
    <el-row :gutter="16">
      <!-- 左侧分组导航 -->
      <el-col :span="5">
        <el-card shadow="never" class="group-card">
          <template #header>
            <span class="group-card__title">配置分组</span>
          </template>
          <el-menu
            :default-active="activeGroup"
            class="group-menu"
            @select="handleGroupSelect"
          >
            <el-menu-item index="">
              <el-icon><Collection /></el-icon>
              <span>全部</span>
            </el-menu-item>
            <el-menu-item
              v-for="group in groups"
              :key="group"
              :index="group"
            >
              <el-icon><Folder /></el-icon>
              <span>{{ group }}</span>
            </el-menu-item>
          </el-menu>
        </el-card>
      </el-col>

      <!-- 右侧配置列表 -->
      <el-col :span="19">
        <!-- 工具栏 -->
        <div class="toolbar">
          <div class="toolbar__left">
            <el-tag v-if="dirtyCount > 0" type="warning" effect="dark" class="dirty-tag">
              {{ dirtyCount }} 项未保存
            </el-tag>
          </div>
          <div class="toolbar__right">
            <el-button
              type="primary"
              :disabled="dirtyCount === 0"
              :loading="saveLoading"
              @click="handleBatchSave"
            >
              保存修改
            </el-button>
            <el-button type="warning" plain @click="handleBatchReset">
              全部重置
            </el-button>
          </div>
        </div>

        <!-- 配置卡片列表 -->
        <div v-loading="loading" class="config-list">
          <template v-if="filteredConfigs.length > 0">
            <el-card
              v-for="cfg in filteredConfigs"
              :key="cfg.config_key"
              shadow="hover"
              class="config-card"
              :class="{ 'config-card--dirty': isDirty(cfg.config_key) }"
            >
              <!-- 卡片头部：配置键 + 类型标签 -->
              <template #header>
                <div class="config-card__header">
                  <code class="config-card__key">{{ cfg.config_key }}</code>
                  <el-tag
                    v-if="cfg.config_type"
                    size="small"
                    :type="typeTagColor(cfg.config_type)"
                    effect="plain"
                    class="config-card__type"
                  >
                    {{ cfg.config_type }}
                  </el-tag>
                </div>
              </template>

              <!-- 卡片内容 -->
              <div class="config-card__body">
                <!-- 描述 -->
                <p v-if="cfg.description" class="config-card__desc">
                  {{ cfg.description }}
                </p>

                <!-- 值编辑器 -->
                <div class="config-card__editor">
                  <label class="config-card__label">当前值</label>

                  <!-- boolean → el-switch -->
                  <el-switch
                    v-if="cfg.config_type === 'boolean'"
                    v-model="editValues[cfg.config_key]"
                    active-text="true"
                    inactive-text="false"
                    @change="markDirty(cfg.config_key)"
                  />

                  <!-- number → el-input-number -->
                  <el-input-number
                    v-else-if="cfg.config_type === 'number'"
                    v-model="editValues[cfg.config_key]"
                    :controls="true"
                    class="config-card__input-number"
                    @change="markDirty(cfg.config_key)"
                  />

                  <!-- json → el-input textarea -->
                  <el-input
                    v-else-if="cfg.config_type === 'json'"
                    v-model="editValues[cfg.config_key]"
                    type="textarea"
                    :rows="4"
                    placeholder="请输入 JSON 内容"
                    @input="markDirty(cfg.config_key)"
                  />

                  <!-- string / default → el-input -->
                  <el-input
                    v-else
                    v-model="editValues[cfg.config_key]"
                    placeholder="请输入配置值"
                    @input="markDirty(cfg.config_key)"
                  />
                </div>

                <!-- 默认值 + 重置按钮 -->
                <div v-if="cfg.default_value !== null && cfg.default_value !== undefined" class="config-card__default">
                  <span class="config-card__default-label">默认值：</span>
                  <code class="config-card__default-value">{{ cfg.default_value }}</code>
                  <el-button
                    text
                    type="primary"
                    size="small"
                    @click="handleResetSingle(cfg.config_key)"
                  >
                    重置
                  </el-button>
                </div>
              </div>
            </el-card>
          </template>

          <!-- 空状态 -->
          <el-empty v-else description="暂无配置项" />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Collection, Folder } from '@element-plus/icons-vue';
import {
  listConfigs,
  getConfigGroups,
  batchUpdateConfigs,
  resetConfig,
} from '@/api/config';

/** 配置项接口 */
interface Config {
  id: number;
  config_key: string;
  config_value: string;
  config_type: string | null;
  description: string | null;
  default_value: string | null;
  group_name: string | null;
  created_at: string;
  updated_at: string;
}

// ─── 状态 ───────────────────────────────────────────────
const loading = ref(false);
const saveLoading = ref(false);
const configs = ref<Config[]>([]);
const groups = ref<string[]>([]);
const activeGroup = ref('');

/** 当前编辑值：key → value (string | number | boolean) */
const editValues = ref<Record<string, string | number | boolean>>({});

/** 原始值快照，用于脏检测 */
const originalValues = ref<Record<string, string | number | boolean>>({});

// ─── 计算属性 ────────────────────────────────────────────
/** 按当前分组过滤配置列表 */
const filteredConfigs = computed(() => {
  if (!activeGroup.value) return configs.value;
  return configs.value.filter((c) => c.group_name === activeGroup.value);
});

/** 脏配置键列表 */
const dirtyKeys = computed(() => {
  const keys: string[] = [];
  for (const key of Object.keys(editValues.value)) {
    if (editValues.value[key] !== originalValues.value[key]) {
      keys.push(key);
    }
  }
  return keys;
});

/** 脏配置数量 */
const dirtyCount = computed(() => dirtyKeys.value.length);

// ─── 方法 ────────────────────────────────────────────────
/** 判断指定配置是否被修改 */
function isDirty(key: string): boolean {
  return editValues.value[key] !== originalValues.value[key];
}

/** 标记配置为脏（由编辑器 change/input 事件触发） */
function markDirty(_key: string) {
  // 脏检测基于 editValues vs originalValues 的 computed，
  // 此函数仅作为事件回调占位，无需额外操作
}

/** 类型标签颜色 */
function typeTagColor(type: string | null): '' | 'success' | 'warning' | 'danger' | 'info' {
  const map: Record<string, '' | 'success' | 'warning' | 'danger' | 'info'> = {
    string: '',
    number: 'success',
    boolean: 'warning',
    json: 'danger',
  };
  return map[type ?? 'string'] ?? 'info';
}

/** 将原始值转为编辑器所需类型 */
function castValue(cfg: Config): string | number | boolean {
  const raw = cfg.config_value ?? '';
  if (cfg.config_type === 'boolean') return raw === 'true';
  if (cfg.config_type === 'number') return Number(raw) || 0;
  return raw;
}

/** 加载配置列表 */
async function loadConfigs(group?: string) {
  loading.value = true;
  try {
    const res: any = await listConfigs(group);
    const list: Config[] = res.data ?? res ?? [];
    configs.value = list;
    // 初始化编辑值和原始值
    const ev: Record<string, string | number | boolean> = {};
    const ov: Record<string, string | number | boolean> = {};
    for (const cfg of list) {
      const val = castValue(cfg);
      ev[cfg.config_key] = val;
      ov[cfg.config_key] = val;
    }
    editValues.value = ev;
    originalValues.value = ov;
  } catch (e: any) {
    ElMessage.error('加载配置失败：' + (e.message ?? '未知错误'));
  } finally {
    loading.value = false;
  }
}

/** 加载分组列表 */
async function loadGroups() {
  try {
    const res: any = await getConfigGroups();
    groups.value = res.data ?? res ?? [];
  } catch (e: any) {
    ElMessage.error('加载分组失败：' + (e.message ?? '未知错误'));
  }
}

/** 分组切换 */
function handleGroupSelect(index: string) {
  activeGroup.value = index;
  // 切换分组时重新加载该分组的配置
  loadConfigs(index || undefined);
}

/** 批量保存 */
async function handleBatchSave() {
  if (dirtyCount.value === 0) return;

  try {
    await ElMessageBox.confirm(
      `确定保存 ${dirtyCount.value} 项修改？`,
      '确认保存',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
    );
  } catch {
    return; // 用户取消
  }

  const items = dirtyKeys.value.map((key) => ({
    key,
    value: String(editValues.value[key]),
  }));

  saveLoading.value = true;
  try {
    await batchUpdateConfigs(items);
    ElMessage.success('保存成功');
    // 更新原始值，清除脏状态
    for (const key of dirtyKeys.value) {
      originalValues.value[key] = editValues.value[key];
    }
  } catch (e: any) {
    ElMessage.error('保存失败：' + (e.message ?? '未知错误'));
  } finally {
    saveLoading.value = false;
  }
}

/** 全部重置确认 */
async function handleBatchReset() {
  try {
    await ElMessageBox.confirm(
      '确定将所有配置项重置为默认值？此操作不可撤销。',
      '确认重置',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
    );
  } catch {
    return;
  }

  loading.value = true;
  try {
    // 逐个重置
    const promises = configs.value.map((cfg) => resetConfig(cfg.config_key));
    await Promise.all(promises);
    ElMessage.success('全部重置成功');
    // 重新加载
    await loadConfigs(activeGroup.value || undefined);
  } catch (e: any) {
    ElMessage.error('重置失败：' + (e.message ?? '未知错误'));
  } finally {
    loading.value = false;
  }
}

/** 单个重置 */
async function handleResetSingle(key: string) {
  try {
    await ElMessageBox.confirm(
      `确定将 ${key} 重置为默认值？`,
      '确认重置',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' },
    );
  } catch {
    return;
  }

  try {
    const res: any = await resetConfig(key);
    const resetCfg: Config = res.data ?? res;
    // 更新本地值
    const val = castValue(resetCfg);
    editValues.value[key] = val;
    originalValues.value[key] = val;
    // 同步 configs 数组中的值
    const idx = configs.value.findIndex((c) => c.config_key === key);
    if (idx !== -1) {
      configs.value[idx] = resetCfg;
    }
    ElMessage.success('重置成功');
  } catch (e: any) {
    ElMessage.error('重置失败：' + (e.message ?? '未知错误'));
  }
}

// ─── 初始化 ──────────────────────────────────────────────
onMounted(() => {
  loadGroups();
  loadConfigs();
});
</script>

<style lang="scss" scoped>
/* ─── 配置页面布局 ─────────────────────────────────────── */
.config-page {
  min-height: calc(100vh - 84px);
}

/* ─── 分组卡片 ─────────────────────────────────────────── */
.group-card {
  :deep(.el-card__header) {
    padding: 12px 16px;
    background: var(--el-fill-color-light);
  }

  &__title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.group-menu {
  border-right: none;

  .el-menu-item {
    height: 40px;
    line-height: 40px;
    border-radius: 6px;
    margin-bottom: 2px;

    &.is-active {
      background-color: rgba(2, 167, 151, 0.08);
      color: #02a797;
      font-weight: 500;
    }

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }
}

/* ─── 工具栏 ───────────────────────────────────────────── */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding: 0 4px;

  &__left {
    display: flex;
    align-items: center;
  }

  &__right {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.dirty-tag {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

/* ─── 配置卡片 ─────────────────────────────────────────── */
.config-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.config-card {
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &--dirty {
    border-left: 3px solid #e6a23c;
  }

  &__header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__key {
    font-family: 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono',
      'Courier New', monospace;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    padding: 2px 8px;
    border-radius: 4px;
  }

  &__type {
    flex-shrink: 0;
  }

  &__body {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  &__desc {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }

  &__editor {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  &__label {
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-regular);
  }

  &__input-number {
    width: 100%;
  }

  &__default {
    display: flex;
    align-items: center;
    gap: 4px;
    padding-top: 8px;
    border-top: 1px dashed var(--el-border-color-lighter);
  }

  &__default-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  &__default-value {
    font-family: 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono',
      'Courier New', monospace;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    padding: 1px 6px;
    border-radius: 3px;
  }
}
</style>