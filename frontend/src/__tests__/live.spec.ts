import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';

/** Bounding box matching live/components/FrameCanvas.vue export */
interface BBox {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  name: string | null;
  confidence: number;
  frame_sequence: number;
}

// ── Module Mocks ──
// Data is defined INSIDE the factory to avoid hoisting TDZ issues

vi.mock('@/api/camera', () => {
  const cameras = [
    {
      id: 1,
      camera_id: 'cam-A',
      name: 'A栋入口',
      building: 'A',
      rtsp_url: 'rtsp://192.168.1.10:554/stream',
      direction: 'entry',
      status: 'online',
      fps_current: 5,
      total_frames: 1000,
      last_heartbeat: null,
      last_event_time: null,
      enabled: true,
      config_json: null,
      remark: '',
      created_at: '2026-01-01',
      updated_at: '2026-05-28',
    },
    {
      id: 2,
      camera_id: 'cam-B',
      name: 'B栋出口',
      building: 'B',
      rtsp_url: 'rtsp://192.168.1.11:554/stream',
      direction: 'exit',
      status: 'online',
      fps_current: 3,
      total_frames: 500,
      last_heartbeat: null,
      last_event_time: null,
      enabled: true,
      config_json: null,
      remark: '',
      created_at: '2026-01-02',
      updated_at: '2026-05-28',
    },
    {
      id: 3,
      camera_id: 'cam-C',
      name: 'C栋大厅',
      building: 'C',
      rtsp_url: 'rtsp://192.168.1.12:554/stream',
      direction: 'both',
      status: 'online',
      fps_current: 8,
      total_frames: 2000,
      last_heartbeat: null,
      last_event_time: null,
      enabled: true,
      config_json: null,
      remark: '',
      created_at: '2026-01-03',
      updated_at: '2026-05-28',
    },
    {
      id: 4,
      camera_id: 'cam-D',
      name: 'D栋通道',
      building: 'D',
      rtsp_url: 'rtsp://192.168.1.13:554/stream',
      direction: 'entry',
      status: 'online',
      fps_current: null,
      total_frames: 800,
      last_heartbeat: null,
      last_event_time: null,
      enabled: true,
      config_json: null,
      remark: '',
      created_at: '2026-01-04',
      updated_at: '2026-05-28',
    },
  ];
  return {
    listCameras: vi.fn(() => Promise.resolve({ data: cameras })),
  };
});

vi.mock('@/utils/auth', () => ({
  getToken: vi.fn(() => 'test-token'),
}));

// ── Imports (after mocks) ──

import LivePage from '@/views/live/index.vue';
import FrameCanvas from '@/views/live/components/FrameCanvas.vue';

// ── Helpers ──

function flushPromises(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 50));
}

/** Create a properly sized mount target for canvas tests */
function createSizedContainer(): HTMLDivElement {
  const el = document.createElement('div');
  el.style.width = '640px';
  el.style.height = '360px';
  el.style.position = 'absolute';
  el.style.left = '0';
  el.style.top = '0';
  document.body.appendChild(el);
  return el;
}

const mockBboxes: BBox[] = [
  { x1: 100, y1: 200, x2: 300, y2: 400, name: '张三', confidence: 0.95, frame_sequence: 1 },
  { x1: 400, y1: 250, x2: 500, y2: 380, name: null, confidence: 0.45, frame_sequence: 2 },
];

// Create a minimal canvas 2D context mock with all methods used by FrameCanvas
function createMockContext() {
  return {
    save: vi.fn(),
    restore: vi.fn(),
    clearRect: vi.fn(),
    drawImage: vi.fn(),
    strokeRect: vi.fn(),
    fillText: vi.fn(),
    measureText: vi.fn(() => ({ width: 80 } as TextMetrics)),
    beginPath: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    closePath: vi.fn(),
    fill: vi.fn(),
    fillStyle: '#000',
    strokeStyle: '#000',
    lineWidth: 1,
    font: '12px sans-serif',
    textBaseline: 'alphabetic' as CanvasTextBaseline,
  } as unknown as CanvasRenderingContext2D;
}

// ── Tests ──

describe('FrameCanvas', () => {
  const originalGetContext = HTMLCanvasElement.prototype.getContext;
  const originalGetBoundingClientRect = HTMLElement.prototype.getBoundingClientRect;

  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      disconnect: vi.fn(),
    })));
    vi.stubGlobal('createImageBitmap', vi.fn().mockResolvedValue({ close: vi.fn() }));
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      blob: vi.fn().mockResolvedValue(new Blob()),
    }));

    HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue(createMockContext()) as typeof HTMLCanvasElement.prototype.getContext;

    // jsdom does not compute CSS layout, so getBoundingClientRect always
    // returns 0 width. Mock it so renderFrame's containerWidth check passes.
    HTMLElement.prototype.getBoundingClientRect = vi.fn(() => ({
      x: 0, y: 0, width: 640, height: 360,
      top: 0, right: 640, bottom: 360, left: 0,
    })) as typeof HTMLElement.prototype.getBoundingClientRect;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    HTMLCanvasElement.prototype.getContext = originalGetContext;
    HTMLElement.prototype.getBoundingClientRect = originalGetBoundingClientRect;
  });

  it('renders empty state with "等待画面..." when frameData is empty', () => {
    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '' },
      global: { plugins: [ElementPlus] },
    });
    expect(wrapper.text()).toContain('等待画面...');
    const overlay = wrapper.find('.frame-canvas__overlay');
    expect(overlay.exists()).toBe(true);
  });

  it('renders offline state with "摄像头离线" when offline prop is true', () => {
    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '', offline: true },
      global: { plugins: [ElementPlus] },
    });
    expect(wrapper.text()).toContain('摄像头离线');
    const overlay = wrapper.find('.frame-canvas__overlay--dark');
    expect(overlay.exists()).toBe(true);
  });

  it('renders canvas element when frameData is provided', () => {
    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '/9j/4AAQSkZJRgABAQAAAQABAAD' },
      global: { plugins: [ElementPlus] },
    });
    expect(wrapper.find('canvas').exists()).toBe(true);
    expect(wrapper.find('.frame-canvas__overlay').exists()).toBe(false);
  });

  it('calls drawImage when frameData is provided', async () => {
    // renderFrame is triggered by watcher when frameData changes.
    // Start with empty to avoid initial no-change case, then set the prop.
    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '' },
      attachTo: createSizedContainer(),
      global: { plugins: [ElementPlus] },
    });

    await wrapper.setProps({ frameData: '/9j/4AAQSkZJRgABAQAAAQABAAD' });
    await wrapper.vm.$nextTick();
    await flushPromises();

    const canvas = wrapper.find('canvas').element as HTMLCanvasElement;
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const ctx = canvas.getContext('2d')!;
    expect(ctx.drawImage).toHaveBeenCalled();
  });

  it('draws bbox rectangles on canvas when bboxes are provided', async () => {
    const wrapper = mount(FrameCanvas, {
      props: {
        cameraId: 'cam-001',
        frameData: '',
        bboxes: [],
      },
      attachTo: createSizedContainer(),
      global: { plugins: [ElementPlus] },
    });

    // Set frameData + bboxes together to trigger watcher once
    await wrapper.setProps({ frameData: '/9j/4AAQSkZJRgABAQAAAQABAAD', bboxes: mockBboxes });
    await wrapper.vm.$nextTick();
    await flushPromises();

    const canvas = wrapper.find('canvas').element as HTMLCanvasElement;
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const ctx = canvas.getContext('2d')!;
    expect(ctx.strokeRect).toHaveBeenCalledTimes(2);

    const calls = (ctx.strokeRect as ReturnType<typeof vi.fn>).mock.calls;
    for (const call of calls) {
      expect(call.length).toBe(4);
    }
  });

  it('draws bbox labels with fillText', async () => {
    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '', bboxes: [] },
      attachTo: createSizedContainer(),
      global: { plugins: [ElementPlus] },
    });

    await wrapper.setProps({ frameData: '/9j/4AAQSkZJRgABAQAAAQABAAD', bboxes: mockBboxes });
    await wrapper.vm.$nextTick();
    await flushPromises();

    const canvas = wrapper.find('canvas').element as HTMLCanvasElement;
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const ctx = canvas.getContext('2d')!;
    expect(ctx.fillText).toHaveBeenCalledTimes(2);

    const firstCall = (ctx.fillText as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(firstCall[0]).toContain('张三');
  });

  it('does not call strokeRect when bboxes prop becomes empty after initial render', async () => {
    const initialBboxes: BBox[] = [
      { x1: 100, y1: 200, x2: 300, y2: 400, name: '测试', confidence: 0.9, frame_sequence: 1 },
    ];

    const wrapper = mount(FrameCanvas, {
      props: { cameraId: 'cam-001', frameData: '' },
      attachTo: createSizedContainer(),
      global: { plugins: [ElementPlus] },
    });

    // Trigger initial render with bboxes
    await wrapper.setProps({ frameData: '/9j/4AAQSkZJRgABAQAAAQABAAD', bboxes: initialBboxes });
    await wrapper.vm.$nextTick();
    await flushPromises();

    const canvas = wrapper.find('canvas').element as HTMLCanvasElement;
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const ctx = canvas.getContext('2d')!;

    // Reset call counts for the re-render check
    vi.clearAllMocks();

    await wrapper.setProps({ bboxes: [] });
    await wrapper.vm.$nextTick();
    await flushPromises();

    expect(ctx.strokeRect).not.toHaveBeenCalled();
  });
});

describe('LivePage', () => {
  const originalGetContext = HTMLCanvasElement.prototype.getContext;

  beforeEach(() => {
    vi.stubGlobal('WebSocket', vi.fn().mockImplementation(() => ({
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
      send: vi.fn(),
      readyState: 0,
    })));
    vi.stubGlobal('EventSource', vi.fn().mockImplementation(() => ({
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      close: vi.fn(),
    })));
    vi.stubGlobal('ResizeObserver', vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      disconnect: vi.fn(),
    })));
    vi.stubGlobal('createImageBitmap', vi.fn().mockResolvedValue({ close: vi.fn() }));
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      blob: vi.fn().mockResolvedValue(new Blob()),
    }));

    HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue(createMockContext()) as typeof HTMLCanvasElement.prototype.getContext;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    HTMLCanvasElement.prototype.getContext = originalGetContext;
  });

  function createLivePageWrapper(): VueWrapper<any> {
    const pinia = createPinia();
    setActivePinia(pinia);
    return mount(LivePage, {
      global: {
        plugins: [pinia, ElementPlus],
      },
    });
  }

  it('mounts without errors', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    expect(wrapper.exists()).toBe(true);
    expect(wrapper.find('.live-page').exists()).toBe(true);
  });

  it('renders camera grid with camera panels after API resolves', async () => {
    const wrapper = createLivePageWrapper();
    // Mount triggers onMounted → loadCameras() → listCameras()
    // Wait for mount + async promise resolution + reactivity flush
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const grid = wrapper.find('.camera-grid');
    expect(grid.exists()).toBe(true);

    const panels = wrapper.findAll('.camera-panel');
    expect(panels.length).toBe(4);
  });

  it('displays correct camera names for all 4 cameras', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const names = wrapper.findAll('.camera-panel-name');
    expect(names.length).toBe(4);
    expect(names[0].text()).toBe('A栋入口');
    expect(names[1].text()).toBe('B栋出口');
    expect(names[2].text()).toBe('C栋大厅');
    expect(names[3].text()).toBe('D栋通道');
  });

  it('shows camera count in the header', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const count = wrapper.find('.camera-count');
    expect(count.text()).toContain('4');
  });

  it('renders connection status tag', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const connTag = wrapper.find('.conn-tag');
    expect(connTag.exists()).toBe(true);
    expect(connTag.text()).toBe('连接断开');
  });

  it('renders refresh button', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const refreshBtn = wrapper.find('.live-header-right .el-button');
    expect(refreshBtn.exists()).toBe(true);
    expect(refreshBtn.text()).toContain('刷新');
  });

  it('shows building tags for each camera', async () => {
    const wrapper = createLivePageWrapper();
    await wrapper.vm.$nextTick();
    await flushPromises();
    await wrapper.vm.$nextTick();

    const buildingTags = wrapper.findAll('.camera-panel-building');
    expect(buildingTags.length).toBe(4);
    expect(buildingTags[0].text()).toBe('A栋');
    expect(buildingTags[1].text()).toBe('B栋');
    expect(buildingTags[2].text()).toBe('C栋');
    expect(buildingTags[3].text()).toBe('D栋');
  });
});
