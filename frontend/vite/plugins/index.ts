import vue from '@vitejs/plugin-vue';
import VueJsx from '@vitejs/plugin-vue-jsx';
import { resolve } from 'path';

import createAutoImport from './auto-import';
import createComponents from './components';
import createSvgIcon from './svg-icon';
import createCompression from './compression';
import createSetupExtend from './setup-extend';
import { PluginOption } from 'vite';
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite';

export default function createVitePlugins(viteEnv: Record<string, string>, isBuild = false) {
  const vitePlugins: PluginOption[] = [vue()];
  vitePlugins.push(VueJsx());
  vitePlugins.push(createAutoImport());
  vitePlugins.push(createComponents());
  vitePlugins.push(createSetupExtend());
  vitePlugins.push(createSvgIcon(isBuild));
  vitePlugins.push(
    VueI18nPlugin({
      runtimeOnly: true,
      compositionOnly: true,
      include: [resolve(__dirname, 'src/locales/**')],
    })
  );
  isBuild && vitePlugins.push(...createCompression(viteEnv));
  return vitePlugins;
}
