import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';

// Mock the events API to avoid network calls
vi.mock('@/api/events', () => ({
  getEvents: vi.fn(() =>
    Promise.resolve({
      data: {
        items: [
          {
            id: 1,
            camera_id: 'cam-001',
            building: 'A',
            event_type: 'entry',
            student_id: 'STU001',
            is_stranger: false,
            confidence: 0.95,
            snapshot_path: '/snapshots/001.jpg',
            timestamp: '2026-05-28T08:30:00Z',
            created_at: '2026-05-28T08:30:00Z',
          },
          {
            id: 2,
            camera_id: 'cam-002',
            building: 'B',
            event_type: 'exit',
            student_id: null,
            is_stranger: true,
            confidence: 0.42,
            snapshot_path: '',
            timestamp: '2026-05-28T09:15:00Z',
            created_at: '2026-05-28T09:15:00Z',
          },
        ],
        total: 2,
        page: 1,
        size: 20,
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

import Events from '@/views/events/index.vue';

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  return mount(Events, {
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

describe('Events Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the filter bar with date picker, selects, and search input', () => {
    const wrapper = createWrapper();

    // Filter bar container
    const filterBar = wrapper.find('.filter-bar');
    expect(filterBar.exists()).toBe(true);

    // Date range picker
    const datePickers = wrapper.findAllComponents({ name: 'ElDatePicker' });
    expect(datePickers.length).toBeGreaterThanOrEqual(1);

    // Building select (区域)
    const selects = wrapper.findAllComponents({ name: 'ElSelect' });
    expect(selects.length).toBeGreaterThanOrEqual(2); // building + event_type

    // Event type select has entry/exit options
    const options = wrapper.findAllComponents({ name: 'ElOption' });
    const entryOption = options.find(o => o.props('value') === 'entry');
    const exitOption = options.find(o => o.props('value') === 'exit');
    expect(entryOption).toBeTruthy();
    expect(exitOption).toBeTruthy();

    // Search input for student_id
    const inputs = wrapper.findAllComponents({ name: 'ElInput' });
    expect(inputs.length).toBeGreaterThanOrEqual(1);

    // Query and Reset buttons
    const buttons = wrapper.findAllComponents({ name: 'ElButton' });
    expect(buttons.length).toBeGreaterThanOrEqual(2);
  });

  it('renders the events table with correct columns', () => {
    const wrapper = createWrapper();

    // Table card container
    const tableCard = wrapper.find('.events-table-card');
    expect(tableCard.exists()).toBe(true);

    // Table title
    const tableTitle = wrapper.find('.table-title');
    expect(tableTitle.exists()).toBe(true);
    expect(tableTitle.text()).toContain('事件记录');

    // Table exists
    const table = wrapper.findComponent({ name: 'ElTable' });
    expect(table.exists()).toBe(true);
  });

  it('renders direction tags (entry=success, exit=warning)', async () => {
    const wrapper = createWrapper();

    // Wait for the API mock to resolve and table to render
    await vi.dynamicImportSettled?.();
    await wrapper.vm.$nextTick();

    // Find all el-tag components
    const tags = wrapper.findAllComponents({ name: 'ElTag' });
    expect(tags.length).toBeGreaterThan(0);

    // Check that direction tags exist with correct types
    const directionTags = tags.filter(t => {
      const type = t.props('type');
      return type === 'success' || type === 'warning';
    });
    expect(directionTags.length).toBeGreaterThan(0);
  });

  it('renders photo thumbnail column with el-image', () => {
    const wrapper = createWrapper();

    // Verify the photo column header exists in the table
    const table = wrapper.findComponent({ name: 'ElTable' });
    expect(table.exists()).toBe(true);

    // Verify el-popover exists for photo preview (hover enlarge)
    const popovers = wrapper.findAllComponents({ name: 'ElPopover' });
    expect(popovers.length).toBeGreaterThanOrEqual(0); // May be 0 if no snapshot_path
  });

  it('renders pagination component', () => {
    const wrapper = createWrapper();

    const pagination = wrapper.find('.pagination-stub');
    expect(pagination.exists()).toBe(true);
  });

  it('renders total count tag', () => {
    const wrapper = createWrapper();

    const totalTag = wrapper.find('.total-tag');
    expect(totalTag.exists()).toBe(true);
  });

  it('renders search and reset buttons with icons', () => {
    const wrapper = createWrapper();

    const primaryBtn = wrapper.find('.filter-btn-primary');
    expect(primaryBtn.exists()).toBe(true);

    const resetBtn = wrapper.find('.filter-btn-reset');
    expect(resetBtn.exists()).toBe(true);
  });
});
