<script setup lang="ts">
import type { EChartsOption, ECharts } from 'echarts';
import { debounce } from 'lodash-es';
import VueTypes from '@/utils/propTypes';
import {
  computed,
  PropType,
  ref,
  unref,
  watch,
  onMounted,
  onBeforeUnmount,
  onActivated,
} from 'vue';
import { isString } from '@/utils/is';
import { useDesign } from '@/hooks/web/useDesign';

const { getPrefixCls, variables } = useDesign();

const prefixCls = getPrefixCls('echart');

const props = defineProps({
  options: {
    type: Object as PropType<EChartsOption>,
    required: true,
  },
  width: VueTypes.oneOfType([Number, String]).def(''),
  height: VueTypes.oneOfType([Number, String]).def('500px'),
});

const theme = computed(() => {
  const echartTheme: boolean | string = 'auto';

  return echartTheme;
});

const options = computed(() => {
  return Object.assign(props.options, {
    darkMode: unref(theme),
  });
});

const elRef = ref<ElRef>();

let echartInstance: ECharts | null = null;

const contentEl = ref<Element>();

const styles = computed(() => {
  const width = isString(props.width) ? props.width : `${props.width}px`;
  const height = isString(props.height) ? props.height : `${props.height}px`;

  return {
    width,
    height,
  };
});

const initChart = async () => {
  if (unref(elRef) && props.options) {
    const echarts = (await import('@/plugins/echarts')).default;
    await import('echarts-wordcloud');
    echartInstance = echarts.init(unref(elRef) as HTMLElement);
    echartInstance?.setOption(unref(options));
  }
};

watch(
  () => options.value,
  options => {
    if (echartInstance) {
      echartInstance?.setOption(options);
    }
  },
  {
    deep: true,
  },
);

const resizeHandler = debounce(() => {
  if (echartInstance) {
    echartInstance.resize();
  }
}, 100);

const contentResizeHandler = async (e: TransitionEvent) => {
  if (e.propertyName === 'width') {
    resizeHandler();
  }
};

onMounted(() => {
  initChart();

  window.addEventListener('resize', resizeHandler);

  contentEl.value = document.getElementsByClassName(`${variables.namespace}-layout-content`)[0];
  unref(contentEl) &&
    (unref(contentEl) as Element).addEventListener('transitionend', contentResizeHandler);
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeHandler);
  unref(contentEl) &&
    (unref(contentEl) as Element).removeEventListener('transitionend', contentResizeHandler);
});

onActivated(() => {
  if (echartInstance) {
    echartInstance.resize();
  }
});
</script>

<template>
  <div ref="elRef" :class="[$attrs.class, prefixCls]" :style="styles"></div>
</template>
