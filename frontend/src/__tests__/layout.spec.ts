import { describe, it, expect } from 'vitest';
import { readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

// Resolve to frontend root — __dirname in vitest is the test file's directory
const __filename = fileURLToPath(import.meta.url);
const __dirname_path = dirname(__filename);
const frontendRoot = resolve(__dirname_path, '..');

describe('Layout Components — Theme Variables', () => {
  it('variables.module.scss defines dark sidebar theme colors', () => {
    const scss = readFileSync(resolve(frontendRoot, 'assets/styles/variables.module.scss'), 'utf8');

    // Dark sidebar background
    expect(scss).toContain('$base-menu-background: #001529');
    expect(scss).toContain('$base-menu-color: rgba(255, 255, 255, 0.8)');
    expect(scss).toContain('$base-logo-title-color: #ffffff');
    expect(scss).toContain('$base-menu-bg-active: rgba(255, 255, 255, 0.08)');

    // CampusVision theme colors
    expect(scss).toContain('$primary-color: #1890ff');
    expect(scss).toContain('$sidebar-bg: #001529');
    expect(scss).toContain('$sidebar-active: #1890ff');
    expect(scss).toContain('$page-bg: #f0f2f5');
    expect(scss).toContain('$card-bg: #ffffff');

    // Sidebar width
    expect(scss).toContain('$base-sidebar-width: 220px');
  });

  it('sidebar.scss uses dark theme hover states', () => {
    const scss = readFileSync(resolve(frontendRoot, 'assets/styles/sidebar.scss'), 'utf8');

    // Dark sidebar hover colors
    expect(scss).toContain('rgba(255, 255, 255, 0.08)');

    // Active state uses $sidebar-active (#1890FF)
    expect(scss).toContain('$sidebar-active');

    // White text color
    expect(scss).toContain('rgba(255, 255, 255, 0.8)');

    // Collapsed width is 64px
    expect(scss).toContain('width: 64px');
  });

  it('logo.vue uses #001529 background and white text', () => {
    const vue = readFileSync(resolve(frontendRoot, 'layout/components/sidebar/logo.vue'), 'utf8');

    expect(vue).toContain("backgroundColor: '#001529'");
    expect(vue).toContain('color: #ffffff');
  });

  it('Navbar.vue has white background and notification bell', () => {
    const vue = readFileSync(resolve(frontendRoot, 'layout/components/navbar.vue'), 'utf8');

    expect(vue).toContain('background: #ffffff');
    expect(vue).toContain('notification-bell');
    expect(vue).toContain('Bell');
    expect(vue).toContain('avatar-container');
  });

  it('TagsView index.vue has blue active state with border-bottom', () => {
    const vue = readFileSync(resolve(frontendRoot, 'layout/components/TagsView/index.vue'), 'utf8');

    expect(vue).toContain('#1890ff');
    expect(vue).toContain('border-bottom: 2px solid #1890ff');
    expect(vue).toContain('background: #ffffff');
  });

  it('layout index.vue uses 64px collapsed sidebar width', () => {
    const vue = readFileSync(resolve(frontendRoot, 'layout/index.vue'), 'utf8');

    expect(vue).toContain('calc(100% - 64px)');
  });

  it('AppMain.vue has page background color', () => {
    const vue = readFileSync(resolve(frontendRoot, 'layout/components/appMain.vue'), 'utf8');

    expect(vue).toContain('#f0f2f5');
  });

  it('index.scss has Element Plus CSS custom property overrides', () => {
    const scss = readFileSync(resolve(frontendRoot, 'assets/styles/index.scss'), 'utf8');

    expect(scss).toContain('--el-color-primary: #1890ff');
    expect(scss).toContain('--el-color-primary-light-3: #40a9ff');
    expect(scss).toContain('--el-bg-color: #f0f2f5');
  });
});
