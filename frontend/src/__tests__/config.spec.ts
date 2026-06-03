import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import ElementPlus from 'element-plus';
import ConfigPage from '@/views/config/index.vue';

// ─── Mock data ────────────────────────────────────────────
const mockConfigs = [
  {
    id: 1,
    config_key: 'fps_day',
    config_value: '5',
    config_type: 'number',
    description: '白天帧率',
    default_value: '5',
    group_name: '通用设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 2,
    config_key: 'fps_night',
    config_value: '1',
    config_type: 'number',
    description: '夜间帧率',
    default_value: '1',
    group_name: '通用设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 3,
    config_key: 'motion_threshold',
    config_value: '0.05',
    config_type: 'number',
    description: '运动检测阈值',
    default_value: '0.05',
    group_name: '识别设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 4,
    config_key: 'behavior_enabled',
    config_value: 'false',
    config_type: 'boolean',
    description: '是否启用行为分析',
    default_value: 'false',
    group_name: '识别设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 5,
    config_key: 'alert_notify_enabled',
    config_value: 'true',
    config_type: 'boolean',
    description: '告警通知开关',
    default_value: 'true',
    group_name: '告警设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 6,
    config_key: 'snapshot_path',
    config_value: '',
    config_type: 'string',
    description: '快照存储路径',
    default_value: null,
    group_name: '通用设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 7,
    config_key: 'kafka_brokers',
    config_value: 'localhost:9092',
    config_type: 'string',
    description: 'Kafka 代理地址',
    default_value: 'localhost:9092',
    group_name: '通用设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
  {
    id: 8,
    config_key: 'redis_config',
    config_value: '{"host":"127.0.0.1","port":6379,"db":0}',
    config_type: 'json',
    description: 'Redis 连接配置',
    default_value: '{"host":"127.0.0.1","port":6379,"db":0}',
    group_name: '通用设置',
    created_at: '2026-01-01',
    updated_at: '2026-05-28',
  },
];

const mockGroups = ['通用设置', '告警设置', '识别设置'];

// ─── Mock API ─────────────────────────────────────────────
vi.mock('@/api/config', () => ({
  listConfigs: vi.fn(() => Promise.resolve({ data: mockConfigs })),
  getConfigGroups: vi.fn(() => Promise.resolve({ data: mockGroups })),
  batchUpdateConfigs: vi.fn(() => Promise.resolve({ data: {} })),
  resetConfig: vi.fn((key: string) => {
    const cfg = mockConfigs.find(c => c.config_key === key);
    return Promise.resolve({ data: cfg });
  }),
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

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  return mount(ConfigPage, {
    global: {
      plugins: [pinia, ElementPlus],
    },
  });
}

describe('Config Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the config page container', () => {
    const wrapper = createWrapper();
    expect(wrapper.find('.cv-config').exists()).toBe(true);
  });

  it('renders the sidebar with group navigation', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const sidebar = wrapper.find('.cv-config__sidebar');
    expect(sidebar.exists()).toBe(true);

    const navItems = wrapper.findAll('.cv-config__nav-item');
    // "全部配置" + mockGroups.length
    expect(navItems.length).toBe(mockGroups.length + 1);
  });

  it('renders the main panel with toolbar', () => {
    const wrapper = createWrapper();
    const main = wrapper.find('.cv-config__main');
    expect(main.exists()).toBe(true);

    const toolbar = wrapper.find('.cv-config__toolbar');
    expect(toolbar.exists()).toBe(true);
  });

  it('renders config items for all configs by default', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const items = wrapper.findAll('.cv-config__item');
    expect(items.length).toBe(mockConfigs.length);
  });

  it('renders boolean configs as el-switch', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const switches = wrapper.findAllComponents({ name: 'ElSwitch' });
    // behavior_enabled and alert_notify_enabled
    expect(switches.length).toBe(2);
  });

  it('renders number configs as el-input-number', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const numberInputs = wrapper.findAllComponents({ name: 'ElInputNumber' });
    // fps_day, fps_night, motion_threshold
    expect(numberInputs.length).toBe(3);
  });

  it('renders json configs as el-input textarea', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    // Find textareas — el-input with type="textarea" renders as textarea
    const textareas = wrapper.findAll('textarea');
    expect(textareas.length).toBeGreaterThanOrEqual(1);
  });

  it('renders string configs as el-input', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    // snapshot_path and kafka_brokers are string type inputs
    const allInputs = wrapper.findAll('.cv-config__item-editor input[type="text"]');
    expect(allInputs.length).toBeGreaterThanOrEqual(2);
  });

  it('displays type badges with correct classes', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const badges = wrapper.findAll('.cv-config__type-badge');
    expect(badges.length).toBe(mockConfigs.length);

    // Check specific type badge classes
    const booleanBadge = badges.filter(b => b.classes().includes('cv-config__type-badge--boolean'));
    expect(booleanBadge.length).toBe(2); // behavior_enabled, alert_notify_enabled

    const numberBadge = badges.filter(b => b.classes().includes('cv-config__type-badge--number'));
    expect(numberBadge.length).toBe(3); // fps_day, fps_night, motion_threshold

    const jsonBadge = badges.filter(b => b.classes().includes('cv-config__type-badge--json'));
    expect(jsonBadge.length).toBe(1); // redis_config

    const stringBadge = badges.filter(b => b.classes().includes('cv-config__type-badge--string'));
    expect(stringBadge.length).toBe(2); // snapshot_path, kafka_brokers
  });

  it('shows default value row when default_value exists', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const defaultRows = wrapper.findAll('.cv-config__item-default');
    // All configs except snapshot_path (default_value: null)
    expect(defaultRows.length).toBe(mockConfigs.length - 1);
  });

  it('renders group count badges in sidebar', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const counts = wrapper.findAll('.cv-config__nav-count');
    // "全部配置" count + group counts
    expect(counts.length).toBe(mockGroups.length + 1);

    // "全部配置" shows total count
    expect(counts[0].text()).toBe(String(mockConfigs.length));
  });

  it('renders save and reset buttons in toolbar', () => {
    const wrapper = createWrapper();
    const buttons = wrapper.findAll('.cv-config__toolbar-right .el-button');
    expect(buttons.length).toBe(2); // save + reset
  });

  it('renders config key as code element', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const keys = wrapper.findAll('.cv-config__item-key');
    expect(keys.length).toBe(mockConfigs.length);

    // Check first key text
    expect(keys[0].text()).toBe('fps_day');
  });

  it('renders description text for configs with description', async () => {
    const wrapper = createWrapper();
    await flushPromises();

    const descs = wrapper.findAll('.cv-config__item-desc');
    // All mock configs have descriptions
    expect(descs.length).toBe(mockConfigs.length);
  });
});
