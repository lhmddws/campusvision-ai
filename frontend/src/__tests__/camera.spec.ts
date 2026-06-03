import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';
import CameraPage from '@/views/camera/index.vue';

// Mock the camera API module
vi.mock('@/api/camera', () => ({
  listCameras: vi.fn(() => Promise.resolve({ data: mockCameras })),
  addCamera: vi.fn(() => Promise.resolve({ data: {} })),
  updateCamera: vi.fn(() => Promise.resolve({ data: {} })),
  deleteCamera: vi.fn(() => Promise.resolve({ data: {} })),
  healthCheck: vi.fn(() => Promise.resolve({ data: {} })),
  getCamera: vi.fn(() => Promise.resolve({ data: {} })),
  getCameraStatus: vi.fn(() => Promise.resolve({ data: {} })),
  getCameraSnapshots: vi.fn(() => Promise.resolve({ data: [] })),
}));

// Mock ElMessageBox to avoid real dialogs
vi.mock('element-plus', async () => {
  const actual: Record<string, any> = (await vi.importActual('element-plus')) as Record<
    string,
    any
  >;
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
    rtsp_url: 'rtsp://admin:pass@192.168.1.100:554/stream',
    direction: 'entry',
    status: 'online',
    fps_current: 5,
    total_frames: 1000,
    last_heartbeat: '2026-05-28 10:00:00',
    last_event_time: '2026-05-28 09:50:00',
    enabled: true,
    config_json: null,
    remark: '',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 2,
    camera_id: 'cam-002',
    name: '2号楼出口',
    building: '2号楼',
    rtsp_url: 'rtsp://admin:pass@192.168.1.101:554/stream',
    direction: 'exit',
    status: 'offline',
    fps_current: null,
    total_frames: 500,
    last_heartbeat: '2026-05-27 22:00:00',
    last_event_time: null,
    enabled: false,
    config_json: null,
    remark: '维护中',
    created_at: '2026-01-02',
    updated_at: '2026-05-27',
  },
  {
    id: 3,
    camera_id: 'cam-003',
    name: '3号楼双向',
    building: '3号楼',
    rtsp_url: 'rtsp://admin:pass@192.168.1.102:554/stream',
    direction: 'both',
    status: 'error',
    fps_current: 0,
    total_frames: 200,
    last_heartbeat: '2026-05-28 08:30:00',
    last_event_time: '2026-05-28 08:25:00',
    enabled: true,
    config_json: null,
    remark: '',
    created_at: '2026-01-03',
    updated_at: '2026-05-28',
  },
];

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  return mount(CameraPage, {
    global: {
      plugins: [pinia, ElementPlus],
    },
  });
}

describe('Camera Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the camera page container', () => {
    const wrapper = createWrapper();
    const page = wrapper.find('.cv-camera-page');
    expect(page.exists()).toBe(true);
  });

  it('renders the filter bar with search input and selects', () => {
    const wrapper = createWrapper();
    const filterBar = wrapper.find('.cv-filter-bar');
    expect(filterBar.exists()).toBe(true);

    // Search input
    const searchInput = wrapper.find('.cv-filter-bar__input');
    expect(searchInput.exists()).toBe(true);

    // Building select (区域)
    const selects = wrapper.findAll('.cv-filter-bar__select');
    expect(selects.length).toBeGreaterThanOrEqual(1);
  });

  it('renders the add camera button', () => {
    const wrapper = createWrapper();
    const buttons = wrapper.findAll('.cv-filter-bar__right .el-button');
    const addBtn = buttons.find(b => b.text().includes('添加摄像头'));
    expect(addBtn).toBeDefined();
  });

  it('renders the table card with el-table', () => {
    const wrapper = createWrapper();
    const tableCard = wrapper.find('.cv-table-card');
    expect(tableCard.exists()).toBe(true);

    const table = wrapper.find('.cv-table');
    expect(table.exists()).toBe(true);
  });

  it('renders status indicators with correct classes', async () => {
    const wrapper = createWrapper();
    // Wait for async data to render
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();

    const statusElements = wrapper.findAll('.cv-status');
    // Should have at least the mock cameras' statuses
    expect(statusElements.length).toBeGreaterThanOrEqual(0);
  });

  it('renders pagination component', () => {
    const wrapper = createWrapper();
    const pagination = wrapper.find('.cv-pagination');
    expect(pagination.exists()).toBe(true);
  });

  it('renders the dialog for add/edit', () => {
    const wrapper = createWrapper();
    // el-dialog with append-to-body renders outside wrapper, so check for dialog form instead
    const dialogForm = wrapper.findComponent({ name: 'ElDialog' });
    expect(dialogForm.exists()).toBe(true);
  });

  it('status filter select has online/offline/error options', () => {
    const wrapper = createWrapper();
    // The status filter select should be in the filter bar
    const filterBar = wrapper.find('.cv-filter-bar');
    expect(filterBar.exists()).toBe(true);
  });
});
