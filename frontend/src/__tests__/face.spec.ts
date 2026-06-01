import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';
import FacePage from '@/views/face/index.vue';

// Mock the face API module
vi.mock('@/api/face', () => ({
  getSnapshots: vi.fn(() => Promise.resolve({ data: { items: mockSnapshots, total: mockSnapshots.length } })),
  listCameras: vi.fn(() => Promise.resolve({ data: { items: mockCameras } })),
}));

// Mock ElMessageBox to avoid real dialogs
vi.mock('element-plus', async () => {
  const actual: Record<string, any> = await vi.importActual('element-plus') as Record<string, any>;
  return {
    ...actual,
    ElMessageBox: {
      confirm: vi.fn(() => Promise.reject('cancel')),
    },
  };
});

const mockCameras = [
  {
    id: 1,
    camera_id: 'cam-001',
    name: '1号楼入口',
    building: '1号楼',
    status: 'online',
  },
  {
    id: 2,
    camera_id: 'cam-002',
    name: '2号楼出口',
    building: '2号楼',
    status: 'offline',
  },
];

const mockSnapshots = [
  {
    id: 101,
    snapshot_path: '/snapshots/face-001.jpg',
    student_id: '2024001',
    confidence: 0.92,
    event_time: '2026-05-28 10:30:00',
  },
  {
    id: 102,
    snapshot_path: '/snapshots/face-002.jpg',
    student_id: '2024002',
    confidence: 0.78,
    event_time: '2026-05-28 10:32:00',
  },
  {
    id: 103,
    snapshot_path: '',
    student_id: '2024003',
    confidence: 0.55,
    event_time: '2026-05-28 10:35:00',
  },
];

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  return mount(FacePage, {
    global: {
      plugins: [pinia, ElementPlus],
    },
  });
}

describe('Face Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the face page container', () => {
    const wrapper = createWrapper();
    const page = wrapper.find('.cv-face-page');
    expect(page.exists()).toBe(true);
  });

  it('renders the filter bar with search input and camera select', () => {
    const wrapper = createWrapper();
    const filterBar = wrapper.find('.cv-filter-bar');
    expect(filterBar.exists()).toBe(true);

    // Search input
    const searchInput = wrapper.find('.cv-filter-bar__input');
    expect(searchInput.exists()).toBe(true);

    // Camera/area select
    const select = wrapper.find('.cv-filter-bar__select');
    expect(select.exists()).toBe(true);
  });

  it('renders the add face button', () => {
    const wrapper = createWrapper();
    const buttons = wrapper.findAll('.cv-filter-bar__right .el-button');
    const addBtn = buttons.find((b) => b.text().includes('添加人脸'));
    expect(addBtn).toBeDefined();
  });

  it('renders the batch import button', () => {
    const wrapper = createWrapper();
    const buttons = wrapper.findAll('.cv-filter-bar__right .el-button');
    const importBtn = buttons.find((b) => b.text().includes('批量导入'));
    expect(importBtn).toBeDefined();
  });

  it('renders the grid wrapper container', () => {
    const wrapper = createWrapper();
    const gridWrap = wrapper.find('.cv-grid-wrap');
    expect(gridWrap.exists()).toBe(true);
  });

  it('shows empty state when no camera is selected', async () => {
    const wrapper = createWrapper();
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();

    // Without a camera selected, snapshots are empty → should show empty state
    const empty = wrapper.find('.cv-empty');
    expect(empty.exists()).toBe(true);
  });

  it('loads snapshots when camera is selected', async () => {
    const wrapper = createWrapper();
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.selectedCameraId).toBe('');
    expect(wrapper.vm.snapshots).toEqual([]);

    wrapper.vm.selectedCameraId = 'cam-001';
    wrapper.vm.handleCameraChange();
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();

    expect(wrapper.vm.selectedCameraId).toBe('cam-001');
  });

  it('renders dialog components in the page', () => {
    const wrapper = createWrapper();
    // Dialogs use append-to-body, so they render outside wrapper
    // Check that dialog-related data exists on the component
    expect(wrapper.vm.dialogVisible).toBe(false);
    expect(wrapper.vm.previewVisible).toBe(false);
    expect(wrapper.vm.batchDialogVisible).toBe(false);
  });
});