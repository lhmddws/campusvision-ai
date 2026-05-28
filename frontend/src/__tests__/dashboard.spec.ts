import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';
import ElementPlus from 'element-plus';

// ─── Mock API ───
vi.mock('@/api/dashboard', () => ({
  getCamerasStatus: vi.fn(() =>
    Promise.resolve({
      data: { total: 15, online: 12, offline: 2, error: 1, cameras: [] },
    })
  ),
  getAlertStats: vi.fn(() =>
    Promise.resolve({
      data: {
        total: 50,
        unread: 8,
        today: 12,
        by_type: { entry: 30, exit: 15, stranger: 3, late_return: 2 },
        by_severity: { critical: 2, high: 5, medium: 8, low: 35 },
      },
    })
  ),
  getAttendanceStats: vi.fn(() =>
    Promise.resolve({
      data: { total: 200, present: 180, absent: 15, late: 5, stranger: 3, rate: 0.9 },
    })
  ),
  getRecentEvents: vi.fn(() =>
    Promise.resolve({
      data: {
        items: [
          { id: 1, event_type: 'entry', building: 'A栋', camera: '前门', student: '张三', event_time: '2026-05-28T10:00:00' },
          { id: 2, event_type: 'exit', building: 'B栋', camera: '后门', student: '李四', event_time: '2026-05-28T09:30:00' },
        ],
        total: 45,
      },
    })
  ),
}));

// ─── Mock ECharts ───
const mockChartInstance = {
  setOption: vi.fn(),
  resize: vi.fn(),
  dispose: vi.fn(),
};

vi.mock('@/plugins/echarts', () => ({
  default: {
    init: vi.fn(() => mockChartInstance),
  },
}));

import Dashboard from '@/views/dashboard/index.vue';

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div>Home</div>' } }],
  });

  return mount(Dashboard, {
    global: {
      plugins: [pinia, router, ElementPlus],
    },
  });
}

describe('Dashboard Page', () => {
  let wrapper: VueWrapper<any>;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount();
    }
  });

  it('renders 4 KPI metric cards', () => {
    wrapper = createWrapper();
    const kpiCards = wrapper.findAll('.kpi-card');
    expect(kpiCards.length).toBe(4);
  });

  it('renders KPI card labels: 在线摄像头, 今日进出, 告警未处理, 出勤率', () => {
    wrapper = createWrapper();
    const labels = wrapper.findAll('.kpi-label');
    expect(labels.length).toBe(4);
    expect(labels[0].text()).toBe('在线摄像头');
    expect(labels[1].text()).toBe('今日进出');
    expect(labels[2].text()).toBe('告警未处理');
    expect(labels[3].text()).toBe('出勤率');
  });

  it('renders KPI icon circles with color classes', () => {
    wrapper = createWrapper();
    // Card 1 (在线摄像头) and Card 4 (出勤率) use blue
    const blueIcons = wrapper.findAll('.kpi-icon-blue');
    expect(blueIcons.length).toBe(2);
    // Card 2 (今日进出) uses green
    expect(wrapper.find('.kpi-icon-green').exists()).toBe(true);
    // Card 3 (告警未处理) uses orange
    expect(wrapper.find('.kpi-icon-orange').exists()).toBe(true);
  });

  it('renders trend indicators on all KPI cards', () => {
    wrapper = createWrapper();
    const trends = wrapper.findAll('.kpi-trend');
    expect(trends.length).toBe(4);
  });

  it('renders 2 chart containers (trend + alert)', () => {
    wrapper = createWrapper();
    const chartContainers = wrapper.findAll('.chart-container');
    expect(chartContainers.length).toBe(2);
  });

  it('renders chart cards with correct titles', () => {
    wrapper = createWrapper();
    const chartTitles = wrapper.findAll('.chart-card .card-title');
    expect(chartTitles.length).toBe(2);
    expect(chartTitles[0].text()).toBe('进出趋势');
    expect(chartTitles[1].text()).toBe('告警分布');
  });

  it('renders activity section with title', () => {
    wrapper = createWrapper();
    const activityCard = wrapper.find('.activity-card');
    expect(activityCard.exists()).toBe(true);
    expect(activityCard.find('.card-title').text()).toBe('实时活动');
  });

  it('initializes echarts on mount', async () => {
    wrapper = createWrapper();
    // Wait for async onMounted (nextTick + chart init)
    await vi.dynamicImportSettled();
    const echartsModule = await import('@/plugins/echarts');
    expect(echartsModule.default.init).toHaveBeenCalled();
  });
});