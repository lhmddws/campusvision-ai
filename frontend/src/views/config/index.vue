<template>
  <div class="cv-config">
    <!-- Left sidebar: group navigation -->
    <aside class="cv-config__sidebar">
      <div class="cv-config__sidebar-header">
        <el-icon :size="18"><Setting /></el-icon>
        <span>配置分组</span>
      </div>
      <nav class="cv-config__nav">
        <button
          class="cv-config__nav-item"
          :class="{ 'cv-config__nav-item--active': activeGroup === '' }"
          @click="handleGroupSelect('')"
        >
          <el-icon><Collection /></el-icon>
          <span>全部配置</span>
          <span class="cv-config__nav-count">{{ configs.length }}</span>
        </button>
        <button
          v-for="group in groups"
          :key="group"
          class="cv-config__nav-item"
          :class="{ 'cv-config__nav-item--active': activeGroup === group }"
          @click="handleGroupSelect(group)"
        >
          <el-icon><Folder /></el-icon>
          <span>{{ group }}</span>
          <span class="cv-config__nav-count">{{ groupCount(group) }}</span>
        </button>
      </nav>
    </aside>

    <!-- Right panel: config items -->
    <main class="cv-config__main">
      <!-- Toolbar -->
      <div class="cv-config__toolbar">
        <div class="cv-config__toolbar-left">
          <h2 class="cv-config__title">
            {{ activeGroup || '全部配置' }}
          </h2>
          <transition name="cv-fade">
            <span v-if="dirtyCount > 0" class="cv-config__dirty-badge">
              {{ dirtyCount }} 项未保存
            </span>
          </transition>
          <span class="cv-config__item-count">{{ filteredConfigs.length }} 项</span>
        </div>
        <div class="cv-config__toolbar-right">
          <el-input
            v-model="searchQuery"
            placeholder="搜索配置键名…"
            clearable
            prefix-icon="Search"
            class="cv-config__search"
            size="default"
          />
          <el-button
            type="primary"
            :disabled="dirtyCount === 0"
            :loading="saveLoading"
            @click="handleBatchSave"
          >
            <el-icon><Check /></el-icon>
            保存修改
          </el-button>
          <el-button type="warning" plain @click="handleBatchReset">
            <el-icon><RefreshLeft /></el-icon>
            全部重置
          </el-button>
        </div>
      </div>

      <!-- Config items list -->
      <div v-loading="loading" class="cv-config__content">
        <TransitionGroup
          v-if="filteredConfigs.length > 0"
          name="cv-stagger"
          tag="div"
          class="cv-config__items"
        >
          <div
            v-for="(cfg, index) in filteredConfigs"
            :key="cfg.config_key"
            class="cv-config__item"
            :class="{ 'cv-config__item--dirty': isDirty(cfg.config_key) }"
            :style="{ '--i': index }"
          >
            <!-- Header: Chinese name + type badge -->
            <div class="cv-config__item-header">
              <span class="cv-config__item-label">{{ cfg.description || cfg.config_key }}</span>
              <span
                class="cv-config__type-badge"
                :class="`cv-config__type-badge--${cfg.config_type || 'string'}`"
              >
                {{ cfg.config_type || 'string' }}
              </span>
            </div>

            <!-- Config key (shows as muted subtitle) -->
            <code class="cv-config__item-key">{{ cfg.config_key }}</code>

            <!-- Value editor -->
            <div class="cv-config__item-editor">
              <!-- boolean / bool → el-switch -->
              <el-switch
                v-if="cfg.config_type === 'boolean' || cfg.config_type === 'bool'"
                v-model="editValues[cfg.config_key]"
                active-text="开启"
                inactive-text="关闭"
              />

              <!-- select → el-select (when config_options is set, takes priority over int/float/string) -->
              <el-select
                v-else-if="cfg.config_options"
                v-model="editValues[cfg.config_key]"
                placeholder="请选择"
                class="cv-config__select"
              >
                <el-option
                  v-for="opt in parseOptions(cfg.config_options)"
                  :key="opt"
                  :label="opt"
                  :value="opt"
                />
              </el-select>

              <!-- int / float / number → el-input-number -->
              <el-input-number
                v-else-if="cfg.config_type === 'int' || cfg.config_type === 'float' || cfg.config_type === 'number'"
                v-model="editValues[cfg.config_key]"
                :controls="true"
                class="cv-config__number-input"
              />

              <!-- json → el-input textarea with format -->
              <div v-else-if="cfg.config_type === 'json'" class="cv-config__json-wrap">
                <el-input
                  v-model="editValues[cfg.config_key]"
                  type="textarea"
                  :rows="4"
                  placeholder="请输入 JSON 内容"
                  class="cv-config__json-input"
                />
                <el-button
                  text
                  type="primary"
                  size="small"
                  class="cv-config__json-format"
                  @click="handleFormatJSON(cfg.config_key)"
                >
                  格式化
                </el-button>
              </div>

              <!-- string / default → el-input -->
              <el-input
                v-else
                v-model="editValues[cfg.config_key]"
                placeholder="请输入配置值"
              />
            </div>

            <!-- Default value + reset -->
            <div
              v-if="cfg.default_value !== null && cfg.default_value !== undefined"
              class="cv-config__item-default"
            >
              <span class="cv-config__default-label">默认值</span>
              <code class="cv-config__default-value">{{ cfg.default_value }}</code>
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
        </TransitionGroup>

        <!-- Empty state -->
        <el-empty v-else description="暂无配置项" />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Collection, Folder, Check, RefreshLeft, Setting, Search } from '@element-plus/icons-vue';
import { listConfigs, getConfigGroups, batchUpdateConfigs, resetConfig } from '@/api/config';

/** 配置项接口 */
interface Config {
  id: number;
  config_key: string;
  config_value: string;
  config_type: string | null;
  description: string | null;
  default_value: string | null;
  config_options: string | null;
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
/** 搜索关键词 */
const searchQuery = ref('');

/** 按分组 + 搜索关键词过滤配置列表 */
const filteredConfigs = computed(() => {
  let list = configs.value;

  // 按分组过滤
  if (activeGroup.value) {
    list = list.filter(c => c.group_name === activeGroup.value);
  }

  // 按关键词搜索 (key + description)
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    list = list.filter(
      c =>
        c.config_key.toLowerCase().includes(q) ||
        (c.description && c.description.toLowerCase().includes(q)),
    );
  }

  return list;
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

/** 将原始值转为编辑器所需类型 */
function castValue(cfg: Config): string | number | boolean {
  const raw = cfg.config_value ?? '';
  if (cfg.config_type === 'boolean') return raw === 'true';
  if (cfg.config_type === 'number') return Number(raw) || 0;
  return raw;
}

/** 解析 config_options JSON 字符串为选项数组 */
function parseOptions(raw: string | null): string[] {
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

/** 计算分组内配置数量 */
function groupCount(group: string): number {
  return configs.value.filter(c => c.group_name === group).length;
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

/** 分组切换 — 仅前端过滤，不重新请求服务端 */
function handleGroupSelect(index: string) {
  activeGroup.value = index;
}

/** 批量保存 */
async function handleBatchSave() {
  if (dirtyCount.value === 0) return;

  try {
    await ElMessageBox.confirm(`确定保存 ${dirtyCount.value} 项修改？`, '确认保存', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    });
  } catch {
    return; // 用户取消
  }

  const items = dirtyKeys.value.map(key => ({
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
    const serverMsg = e.response?.data?.message || e.response?.data?.msg || '';
    ElMessage.error('保存失败：' + (serverMsg || e.message || '未知错误'));
  } finally {
    saveLoading.value = false;
  }
}

/** 全部重置确认 */
async function handleBatchReset() {
  try {
    await ElMessageBox.confirm('确定将所有配置项重置为默认值？此操作不可撤销。', '确认重置', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    });
  } catch {
    return;
  }

  loading.value = true;
  try {
    // 逐个重置
    const promises = configs.value.map(cfg => resetConfig(cfg.config_key));
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

/** 格式化 JSON 配置值 */
function handleFormatJSON(key: string) {
  const raw = editValues.value[key];
  if (typeof raw !== 'string') return;
  try {
    const parsed = JSON.parse(raw);
    editValues.value[key] = JSON.stringify(parsed, null, 2);
  } catch {
    ElMessage.warning('JSON 格式无效，无法格式化');
  }
}

/** 单个重置 */
async function handleResetSingle(key: string) {
  try {
    await ElMessageBox.confirm(`确定将 ${key} 重置为默认值？`, '确认重置', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    });
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
    const idx = configs.value.findIndex(c => c.config_key === key);
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
@use '@/assets/styles/variables.module.scss' as vars;

// Local alias for $--color-info (Sass private member, not accessible via vars.*)
$color-info: #909399;

/* ─── Layout ────────────────────────────────────────────── */
.cv-config {
  display: flex;
  min-height: calc(100vh - 84px);
  background: vars.$page-bg;
  border-radius: 8px;
  overflow: hidden;
}

/* ─── Sidebar ───────────────────────────────────────────── */
.cv-config__sidebar {
  width: 240px;
  min-width: 240px;
  background: vars.$card-bg;
  border-right: 1px solid rgba($color-info, 0.15);
  display: flex;
  flex-direction: column;
}

.cv-config__sidebar-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 20px 20px 12px;
  font-size: 15px;
  font-weight: 600;
  color: vars.$text-primary;
  letter-spacing: 0.02em;
}

.cv-config__nav {
  display: flex;
  flex-direction: column;
  padding: 4px 8px 16px;
  gap: 2px;
  overflow-y: auto;
  flex: 1;
}

.cv-config__nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: vars.$text-secondary;
  transition:
    background 0.15s ease,
    color 0.15s ease,
    transform 0.12s ease;
  text-align: left;
  font-family: inherit;

  &:hover {
    background: rgba(vars.$primary-color, 0.06);
    color: vars.$text-primary;
  }

  &:active {
    transform: scale(0.97);
  }

  &--active {
    background: rgba(vars.$primary-color, 0.1);
    color: vars.$primary-color;
    font-weight: 500;

    &::before {
      content: '';
      position: absolute;
      left: -8px;
      top: 50%;
      transform: translateY(-50%);
      width: 3px;
      height: 20px;
      border-radius: 3px;
      background: vars.$primary-color;
    }

    .cv-config__nav-count {
      background: vars.$primary-color;
      color: #fff;
    }
  }
}

.cv-config__nav-count {
  margin-left: auto;
  font-size: 11px;
  font-weight: 600;
  padding: 1px 8px;
  border-radius: 10px;
  background: rgba(vars.$text-secondary, 0.12);
  color: vars.$text-secondary;
  transition: all 0.2s ease;
}

/* ─── Main panel ────────────────────────────────────────── */
.cv-config__main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 24px 28px;
}

/* ─── Toolbar ───────────────────────────────────────────── */
.cv-config__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.cv-config__toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cv-config__toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.el-button:active) {
    transform: scale(0.96);
  }
}

.cv-config__search {
  width: 220px;
}

.cv-config__item-count {
  font-size: 12px;
  color: vars.$text-secondary;
  padding: 2px 8px;
  border-left: 1px solid rgba($color-info, 0.2);
  white-space: nowrap;
}

.cv-config__title {
  font-size: 20px;
  font-weight: 700;
  color: vars.$text-primary;
  margin: 0;
  letter-spacing: -0.01em;
}

.cv-config__dirty-badge {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 12px;
  background: rgba(vars.$warning-color, 0.12);
  color: vars.$warning-color;
  animation: cv-pulse 1.8s ease-in-out infinite;
}

@keyframes cv-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.65;
  }
}

/* ─── Content area ──────────────────────────────────────── */
.cv-config__content {
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  flex: 1;
}

.cv-config__items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ─── Config item card ──────────────────────────────────── */
.cv-config__item {
  background: vars.$card-bg;
  border-radius: 10px;
  padding: 18px 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 1px 2px rgba(0, 0, 0, 0.03);
  transition:
    box-shadow 0.25s ease,
    transform 0.25s ease;

  &:hover {
    box-shadow:
      0 4px 16px rgba(0, 0, 0, 0.07),
      0 2px 4px rgba(0, 0, 0, 0.04);
    transform: translateY(-1px);
  }

  &--dirty {
    box-shadow:
      0 1px 3px rgba(0, 0, 0, 0.05),
      0 1px 2px rgba(0, 0, 0, 0.03),
      inset 3px 0 0 vars.$warning-color;
    background: rgba(vars.$warning-color, 0.02);
  }
}

.cv-config__item-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.cv-config__item-label {
  font-size: 15px;
  font-weight: 600;
  color: vars.$text-primary;
  line-height: 1.4;
}

.cv-config__item-key {
  display: inline-block;
  font-family:
    'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  color: vars.$text-secondary;
  background: rgba(vars.$text-secondary, 0.08);
  padding: 1px 8px;
  border-radius: 4px;
  margin-top: 4px;
}

.cv-config__type-badge {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: 2px 8px;
  border-radius: 4px;

  &--string {
    background: rgba($color-info, 0.08);
    color: $color-info;
  }
  &--number {
    background: rgba(vars.$success-color, 0.1);
    color: vars.$success-color;
  }
  &--boolean {
    background: rgba(vars.$warning-color, 0.1);
    color: vars.$warning-color;
  }
  &--json {
    background: rgba(vars.$danger-color, 0.1);
    color: vars.$danger-color;
  }
}

.cv-config__item-desc {
  font-size: 13px;
  color: vars.$text-secondary;
  margin: 0 0 14px;
  line-height: 1.6;
  text-wrap: pretty;
}

.cv-config__item-editor {
  margin-bottom: 4px;
}

.cv-config__number-input {
  width: 100%;
}

.cv-config__select {
  width: 100%;
}

.cv-config__json-wrap {
  position: relative;
}

.cv-config__json-format {
  position: absolute;
  right: 8px;
  top: 4px;
  z-index: 1;
}

.cv-config__json-input {
  :deep(textarea) {
    font-family:
      'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', monospace;
    font-size: 13px;
    line-height: 1.5;
  }
}

/* ─── Default value row ──────────────────────────────────── */
.cv-config__item-default {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 12px;
  margin-top: 8px;
  border-top: 1px dashed rgba($color-info, 0.15);
}

.cv-config__default-label {
  font-size: 12px;
  font-weight: 500;
  color: vars.$text-secondary;
}

.cv-config__default-value {
  font-family:
    'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
  color: vars.$text-secondary;
  background: rgba($color-info, 0.06);
  padding: 2px 8px;
  border-radius: 4px;
}

/* ─── Transitions ───────────────────────────────────────── */
.cv-fade-enter-active,
.cv-fade-leave-active {
  transition: opacity 0.25s ease;
}
.cv-fade-enter-from,
.cv-fade-leave-to {
  opacity: 0;
}

/* Stagger entrance for config items */
.cv-stagger-enter-active {
  animation: cv-item-enter 0.3s ease-out both;
  animation-delay: calc(var(--i, 0) * 40ms);
}

.cv-stagger-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.cv-stagger-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@keyframes cv-item-enter {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
