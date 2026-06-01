import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';
import ElementPlus from 'element-plus';

vi.mock('@/api/attendance', () => ({
  getInspectionList: vi.fn(() =>
    Promise.resolve({
      data: [
        {
          building: 'A',
          room: 'A-101',
          total_students: 4,
          present_count: 3,
          unknown_count: 1,
          students: [
            { student_id: '2024001', student_name: '张三', status: '在寝' },
            { student_id: '2024002', student_name: '李四', status: '在寝' },
            { student_id: '2024003', student_name: '王五', status: '在寝' },
            { student_id: '2024004', student_name: '赵六', status: '未归' },
          ],
        },
        {
          building: 'A',
          room: 'A-102',
          total_students: 2,
          present_count: 2,
          unknown_count: 0,
          students: [
            { student_id: '2024005', student_name: '钱七', status: '在寝' },
            { student_id: '2024006', student_name: '孙八', status: '在寝' },
          ],
        },
      ],
    })
  ),
}));

import Inspection from '@/views/attendance/inspection.vue';

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div>Home</div>' } }],
  });

  return mount(Inspection, {
    global: {
      plugins: [pinia, router, ElementPlus],
    },
  });
}

describe('Inspection Page', () => {
  let wrapper: VueWrapper<any>;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount();
    }
  });

  it('renders building filter select', () => {
    wrapper = createWrapper();
    const select = wrapper.find('.building-select');
    expect(select.exists()).toBe(true);
  });

  it('renders export buttons', () => {
    wrapper = createWrapper();
    const buttons = wrapper.findAll('button');
    const buttonTexts = buttons.map((b) => b.text());
    expect(buttonTexts.some((t) => t.includes('导出 Excel'))).toBe(true);
    expect(buttonTexts.some((t) => t.includes('导出 PDF'))).toBe(true);
  });

  it('renders table after data loads', async () => {
    wrapper = createWrapper();
    await vi.dynamicImportSettled();
    const table = wrapper.find('.el-table');
    expect(table.exists()).toBe(true);
  });

  it('shows room data in table rows', async () => {
    wrapper = createWrapper();
    await vi.dynamicImportSettled();
    const cells = wrapper.findAll('.el-table__body-wrapper .cell');
    const allText = cells.map((c) => c.text()).join(' ');
    expect(allText).toContain('A-101');
    expect(allText).toContain('A-102');
  });

  it('displays status tags for rooms', async () => {
    wrapper = createWrapper();
    await vi.dynamicImportSettled();
    const tags = wrapper.findAll('.el-tag');
    expect(tags.length).toBeGreaterThanOrEqual(2);
  });
});