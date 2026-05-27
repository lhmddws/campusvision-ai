<script setup lang="ts">
import { ElDialog, ElScrollbar } from 'element-plus';
import VueTypes from '@/utils/propTypes';
import { computed, useAttrs, ref, unref, useSlots, watch, nextTick, onMounted } from 'vue';
import { isNumber } from '@/utils/is';

const slots = useSlots();

const props = defineProps({
  modelValue: VueTypes.bool.def(false),
  title: VueTypes.string.def('Dialog'),
  fullscreen: VueTypes.bool.def(false),
  width: VueTypes.oneOfType([String, Number]).def('400px'),
  maxHeight: VueTypes.oneOfType([String, Number]).def('400px')
});

const getBindValue = computed(() => {
  const delArr: string[] = ['fullscreen', 'title', 'maxHeight'];
  const attrs = useAttrs();
  const obj = { ...attrs, ...props };
  for (const key in obj) {
    if (delArr.indexOf(key) !== -1) {
      delete obj[key];
    }
  }
  return obj;
});

const isFullscreen = ref(false);

const elDialogRef = ref();

const toggleFull = () => {
  isFullscreen.value = !unref(isFullscreen);
};

const dialogHeight = ref(isNumber(props.maxHeight) ? `${props.maxHeight}px` : props.maxHeight);

watch(
  () => isFullscreen.value,
  async (val: boolean) => {
    await nextTick();
    if (val) {
      const windowHeight = document.documentElement.offsetHeight;
      dialogHeight.value = `${windowHeight - 55 - 60 - (slots.footer ? 63 : 0)}px`;
    } else {
      dialogHeight.value = isNumber(props.maxHeight) ? `${props.maxHeight}px` : props.maxHeight;
    }
  },
  {
    immediate: true
  }
);

</script>

<template>
  <ElDialog
    v-bind="getBindValue"
    :fullscreen="isFullscreen"
    destroy-on-close
    lock-scroll
    draggable
    :width="props.width"
    :close-on-click-modal="false"
    append-to-body
  >
    <template #header>
      <div class="flex justify-between">
        <slot name="title">
          {{ title }}
        </slot>
        <Icon
          v-if="fullscreen"
          class="mr-6 cursor-pointer is-hover z-10 mt-1"
          :icon="isFullscreen ? 'zmdi:fullscreen-exit' : 'zmdi:fullscreen'"
          color="var(--el-color-info)"
          @click="toggleFull"
        />
      </div>
    </template>

    <!-- <ElScrollbar max-height="calc(80vh - 100px)"> -->
      <slot></slot>
    <!-- </ElScrollbar> -->

    <template v-if="slots.footer" #footer>
      <slot name="footer"></slot>
    </template>
  </ElDialog>
</template>

<style lang="scss">
.#{elNamespace}-dialog__header {
  margin-right: 0 !important;
  border-bottom: 1px solid var(--tags-view-border-color);
}

.#{elNamespace}-dialog__footer {
  border-top: 1px solid var(--tags-view-border-color);
}

.is-hover {
  &:hover {
    color: var(--el-color-primary) !important;
  }
}

.dark {
  .#{elNamespace}-dialog__header {
    border-bottom: 1px solid var(--el-border-color);
  }

  .#{elNamespace}-dialog__footer {
    border-top: 1px solid var(--el-border-color);
  }
}
</style>
