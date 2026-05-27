<script setup lang="ts">
import { ElCard, ElTooltip } from 'element-plus';
import VueTypes from '@/utils/propTypes';
import { useDesign } from '@/hooks/web/useDesign';

const { getPrefixCls } = useDesign();

const prefixCls = getPrefixCls('content-wrap');

defineProps({
  title: VueTypes.string.def(''),
  message: VueTypes.string.def('')
});
</script>

<template>
  <ElCard :class="[prefixCls]" shadow="never">
    <template v-if="title" #header>
      <div class="flex items-center">
        <span class="text-4 font-bold">{{ title }}</span>
        <ElTooltip v-if="message" effect="dark" placement="right">
          <template #content>
            <div class="max-w-200">{{ message }}</div>
          </template>
          <Icon class="ml-1" icon="bi:question-circle-fill" :size="14" />
        </ElTooltip>
      </div>
    </template>
      <slot></slot>
  </ElCard>
</template>
