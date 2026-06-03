import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';

// Mock the alerts API to avoid network calls
vi.mock('@/api/alerts', () => ({
  getAlerts: vi.fn(() =>
    Promise.resolve({
      data: {
        items: [
          {
            id: 1,
            alert_id: 'ALT-001',
            alert_type: 'stranger',
            building: 'A',
            student_id: null,
            severity: 'high',
            description: '陌生人进入A栋大厅',
            face_snapshot_url: null,
            is_read: false,
            is_resolved: false,
            occurred_at: '2026-05-28T08:30:00Z',
            created_at: '2026-05-28T08:30:00Z',
          },
          {
            id: 2,
            alert_id: 'ALT-002',
            alert_type: 'late_return',
            building: 'B',
            student_id: 'STU001',
            severity: 'medium',
            description: '学生晚归',
            face_snapshot_url: null,
            is_read: true,
            is_resolved: true,
            occurred_at: '2026-05-28T22:15:00Z',
            created_at: '2026-05-28T22:15:00Z',
          },
          {
            id: 3,
            alert_id: 'ALT-003',
            alert_type: 'absence',
            building: 'C',
            student_id: 'STU002',
            severity: 'low',
            description: '学生缺勤',
            face_snapshot_url: null,
            is_read: false,
            is_resolved: false,
            occurred_at: '2026-05-28T09:00:00Z',
            created_at: '2026-05-28T09:00:00Z',
          },
        ],
        total: 3,
        page: 1,
        size: 20,
      },
    }),
  ),
  acknowledgeAlert: vi.fn(() => Promise.resolve({ data: { success: true } })),
  getAlertStats: vi.fn(() =>
    Promise.resolve({
      data: {
        total: 10,
        unread: 4,
        today: 2,
        by_type: { stranger: 3, late_return: 5, absence: 2 },
        by_severity: { high: 2, medium: 5, low: 3 },
      },
    }),
  ),
}));

// Mock parseTime utility
vi.mock('@/utils/ruoyi', () => ({
  parseTime: vi.fn((time: string | null) => {
    if (!time) return '-';
    return time.replace('T', ' ').replace('Z', '');
  }),
}));

import Alerts from '@/views/alerts/index.vue';

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  return mount(Alerts, {
    global: {
      plugins: [pinia, ElementPlus],
      stubs: {
        pagination: {
          template: '<div class="pagination-stub" />',
          props: ['total', 'page', 'limit', 'pageSizes'],
        },
      },
    },
  });
}

describe('Alerts Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the stats row with 4 stat cards', () => {
    const wrapper = createWrapper();

    const statsRow = wrapper.find('.stats-row');
    expect(statsRow.exists()).toBe(true);

    const statCards = wrapper.findAll('.stat-card');
    expect(statCards.length).toBe(4);

    // Check stat card variants
    expect(wrapper.find('.stat-card--total').exists()).toBe(true);
    expect(wrapper.find('.stat-card--unread').exists()).toBe(true);
    expect(wrapper.find('.stat-card--today').exists()).toBe(true);
    expect(wrapper.find('.stat-card--resolved').exists()).toBe(true);
  });

  it('renders the filter bar with type, severity, status selects and search input', () => {
    const wrapper = createWrapper();

    const filterBar = wrapper.find('.filter-bar');
    expect(filterBar.exists()).toBe(true);

    // 3 selects: alert_type, severity, acknowledged
    const selects = wrapper.findAllComponents({ name: 'ElSelect' });
    expect(selects.length).toBeGreaterThanOrEqual(3);

    // Search input for building
    const inputs = wrapper.findAllComponents({ name: 'ElInput' });
    expect(inputs.length).toBeGreaterThanOrEqual(1);

    // Query and Reset buttons
    const primaryBtn = wrapper.find('.filter-btn-primary');
    expect(primaryBtn.exists()).toBe(true);

    const resetBtn = wrapper.find('.filter-btn-reset');
    expect(resetBtn.exists()).toBe(true);

    // Batch acknowledge button
    const batchBtn = wrapper.find('.filter-btn-batch');
    expect(batchBtn.exists()).toBe(true);
  });

  it('renders severity filter options (HIGH/MEDIUM/LOW)', () => {
    const wrapper = createWrapper();

    const options = wrapper.findAllComponents({ name: 'ElOption' });
    const severityValues = ['high', 'medium', 'low'];
    for (const val of severityValues) {
      const opt = options.find(o => o.props('value') === val);
      expect(opt).toBeTruthy();
    }
  });

  it('renders the alerts table card with title and total tag', () => {
    const wrapper = createWrapper();

    const tableCard = wrapper.find('.alerts-table-card');
    expect(tableCard.exists()).toBe(true);

    const tableTitle = wrapper.find('.table-title');
    expect(tableTitle.exists()).toBe(true);
    expect(tableTitle.text()).toContain('告警列表');

    const totalTag = wrapper.find('.total-tag');
    expect(totalTag.exists()).toBe(true);
  });

  it('renders severity badges with correct tag types', async () => {
    const wrapper = createWrapper();

    // Wait for API mock to resolve
    await vi.dynamicImportSettled?.();
    await wrapper.vm.$nextTick();

    // Find all el-tag components
    const tags = wrapper.findAllComponents({ name: 'ElTag' });

    // Should have severity badges (high=danger, medium=warning, low=info)
    const severityTags = tags.filter(
      t =>
        t.props('type') === 'danger' || t.props('type') === 'warning' || t.props('type') === 'info',
    );
    expect(severityTags.length).toBeGreaterThan(0);
  });

  it('renders status tags (待处理=warning, 已确认=success)', async () => {
    const wrapper = createWrapper();

    await vi.dynamicImportSettled?.();
    await wrapper.vm.$nextTick();

    const tags = wrapper.findAllComponents({ name: 'ElTag' });

    // Should have success tags (已确认) and warning tags (待处理)
    const successTags = tags.filter(t => t.props('type') === 'success');
    const warningTags = tags.filter(t => t.props('type') === 'warning');
    expect(successTags.length + warningTags.length).toBeGreaterThan(0);
  });

  it('renders acknowledge and detail action buttons', async () => {
    const wrapper = createWrapper();

    await vi.dynamicImportSettled?.();
    await wrapper.vm.$nextTick();

    // Find all el-button components
    const buttons = wrapper.findAllComponents({ name: 'ElButton' });

    // Should have acknowledge (确认) and detail (查看详情) buttons in action column
    const actionButtons = buttons.filter(
      b => b.text().includes('确认') || b.text().includes('查看详情'),
    );
    expect(actionButtons.length).toBeGreaterThan(0);
  });

  it('renders pagination component', () => {
    const wrapper = createWrapper();

    const pagination = wrapper.find('.pagination-stub');
    expect(pagination.exists()).toBe(true);
  });
});
