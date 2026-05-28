import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, VueWrapper } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';
import ElementPlus from 'element-plus';

vi.mock('js-cookie', () => ({
  default: {
    get: vi.fn(() => undefined),
    set: vi.fn(),
    remove: vi.fn(),
  },
}));

vi.mock('@/utils/jsencrypt', () => ({
  encrypt: vi.fn((txt: string) => `enc_${txt}`),
  decrypt: vi.fn((txt: string) => txt.replace('enc_', '')),
}));

vi.mock('@/api/login', () => ({
  login: vi.fn(() => Promise.resolve({ data: { token: 'mock-token' } })),
  logout: vi.fn(() => Promise.resolve()),
  getInfo: vi.fn(() => Promise.resolve()),
  getCodeImg: vi.fn(),
  register: vi.fn(),
}));

vi.mock('@/utils/auth', () => ({
  getToken: vi.fn(() => ''),
  setToken: vi.fn(),
  removeToken: vi.fn(),
}));

import Login from '@/views/login.vue';

function createWrapper(): VueWrapper<any> {
  const pinia = createPinia();
  setActivePinia(pinia);

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div>Home</div>' } }],
  });

  return mount(Login, {
    global: {
      plugins: [pinia, router, ElementPlus],
      stubs: {
        'svg-icon': {
          template: '<span class="svg-icon-stub"><slot /></span>',
        },
      },
    },
  });
}

describe('Login Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the brand section with logo and title', () => {
    const wrapper = createWrapper();
    const brandSection = wrapper.find('.login-brand');
    expect(brandSection.exists()).toBe(true);
    const brandTitle = wrapper.find('.brand-title');
    expect(brandTitle.exists()).toBe(true);
    const brandLogo = wrapper.find('.brand-logo img');
    expect(brandLogo.exists()).toBe(true);
  });

  it('renders username and password inputs', () => {
    const wrapper = createWrapper();
    const inputs = wrapper.findAll('input');
    expect(inputs.length).toBeGreaterThanOrEqual(2);
  });

  it('renders the login button', () => {
    const wrapper = createWrapper();
    const loginBtn = wrapper.find('.login-btn');
    expect(loginBtn.exists()).toBe(true);
    expect(loginBtn.text()).toContain('登 录');
  });

  it('renders the remember-me checkbox', () => {
    const wrapper = createWrapper();
    const checkbox = wrapper.find('.el-checkbox');
    expect(checkbox.exists()).toBe(true);
    expect(checkbox.text()).toContain('记住密码');
  });

  it('renders the form panel on the right side', () => {
    const wrapper = createWrapper();
    const formPanel = wrapper.find('.login-form-panel');
    expect(formPanel.exists()).toBe(true);
    const loginCard = wrapper.find('.login-card');
    expect(loginCard.exists()).toBe(true);
    expect(loginCard.text()).toContain('欢迎登录');
  });

  it('renders the brand logo image', () => {
    const wrapper = createWrapper();
    const logoImg = wrapper.find('.brand-logo img');
    expect(logoImg.exists()).toBe(true);
  });

  it('renders copyright footer', () => {
    const wrapper = createWrapper();
    const footer = wrapper.find('.login-footer');
    expect(footer.exists()).toBe(true);
    expect(footer.text()).toContain('CampusVision AI');
  });
});
