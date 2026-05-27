import { defineConfig, loadEnv } from 'vite';
import createVitePlugins from './vite/plugins';
import path from 'path';

const root = process.cwd();
function pathResolve(dir: string) {
  return path.resolve(root, '.', dir);
}

export default defineConfig(({ mode, command }) => {
  const env = loadEnv(mode, process.cwd());
  const { VITE_APP_ENV } = env;
  return {
    plugins: createVitePlugins(env, command === 'build'),
    // 部署生产环境和开发环境下的URL。
    // 默认情况下，vite 会假设你的应用是被部署在一个域名的根路径上
    // 例如 https://www.ruoyi.vip/。如果应用被部署在一个子路径上，你就需要用这个选项指定这个子路径。例如，如果你的应用被部署在 https://www.ruoyi.vip/admin/，则设置 baseUrl 为 /admin/。
    base: VITE_APP_ENV === 'production' ? '/' : '/',
    server: {
      port: 3000,
      host: true,
      open: true,
      hmr:true,
      proxy: {
        '/dev-api': {
          target: 'http://localhost:8083',
          changeOrigin: true,
          rewrite: p => p.replace(/^\/dev-api/, ''),
        },
      },
    },
    resolve: {
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.scss', '.css'],
      alias: [
        {
          find: 'vue-i18n',
          replacement: 'vue-i18n/dist/vue-i18n.cjs.js'
        },
        {
          find: /\@\//,
          replacement: `${pathResolve('src')}/`
        },
        {
          find: /\~\//,
          replacement: `/`
        },
      ]
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "./src/assets/styles/element-ui.scss" as *;`,
          javascriptEnabled: true
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('element-plus/theme')) {
              return 'ele';
            }
          },
        },
      },
    },
  };
});
