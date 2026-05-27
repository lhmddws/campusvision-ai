<template>
  <div v-if="!item.hidden">
    <template
      v-if="
        hasOneShowingChild(item.children, item) &&
        (!onlyOneChild.children || onlyOneChild.noShowingChildren) &&
        !item.alwaysShow
      "
    >
      <app-link v-if="onlyOneChild.meta" :to="resolvePath(onlyOneChild.path, onlyOneChild.query)">
        <el-menu-item
          :index="resolvePath(onlyOneChild.path)"
          :class="{ 'sub-menu-title-noDropdown': !isNest }"
          class="select-none outer-most"
          :style="{ color: variables.menuSubColor }"
        >
          <svg-icon v-if="!isNest" :icon-class="onlyOneChild.meta.icon || (item.meta && item.meta.icon)" />
          <template #title>
            <span
              class="menu-title"
              :style="{ opacity: isCollapse ? 0 : 1 }"
              :title="hasTitle(onlyOneChild.meta.title)"
            >
              {{ onlyOneChild.meta.title }}
            </span>
          </template>
        </el-menu-item>
      </app-link>
    </template>
    <el-sub-menu
      v-else
      ref="subMenu"
      :index="resolvePath(item.path)"
      popper-append-to-body
      popper-class="popper-menu"
      :popper-offset="14"
      :class="isCollapse ? 'menu-active' : ''"
    >
      <template v-if="item.meta" #title>
        <svg-icon :icon-class="item.meta && item.meta.icon" />
        <span v-if="!isCollapse" class="menu-title" :title="hasTitle(item.meta.title)">
          {{ item.meta.title }}
        </span>
      </template>

      <sidebar-item
        v-for="child in item.children"
        :key="child.path"
        :is-nest="true"
        :item="child"
        :base-path="resolvePath(child.path)"
        class="nest-menu"
      />
    </el-sub-menu>
  </div>
</template>

<script setup lang="ts" name="SidebarItem">
import { isExternal } from '@/utils/validate';
import AppLink from './Link.vue';
import { getNormalPath } from '@/utils/ruoyi';
// import subMenu from 'element-plus/es/components/menu/src/sub-menu';
// import item from 'element-plus/es/components/space/src/item';
import variables from '@/assets/styles/variables.module.scss';
import useSettingsStore from '@/store/modules/settings';
import { ref, computed } from 'vue';

const props = defineProps({
  // route object
  item: {
    type: Object,
    required: true,
  },
  isNest: {
    type: Boolean,
    default: false,
  },
  basePath: {
    type: String,
    default: '',
  },
  isCollapse: {
    type: Boolean,
    default: false,
  },
});

const onlyOneChild = ref<any>({});
const sideTheme = computed(() => settingsStore.sideTheme);
const settingsStore = useSettingsStore();

function hasOneShowingChild(children: any[] = [], parent: any) {
  if (!children) {
    children = [];
  }
  const showingChildren = children.filter(item => {
    if (item.hidden) {
      return false;
    } else {
      // Temp set(will be used if only has one showing child)
      onlyOneChild.value = item;
      return true;
    }
  });

  // When there is only one child router, the child router is displayed by default
  if (showingChildren.length === 1) {
    return true;
  }

  // Show parent if there are no child router to display
  if (showingChildren.length === 0) {
    onlyOneChild.value = { ...parent, path: '', noShowingChildren: true };
    return true;
  }

  return false;
}

function resolvePath(routePath: any, routeQuery?: any) {
  if (isExternal(routePath)) {
    return routePath;
  }
  if (isExternal(props.basePath)) {
    return props.basePath;
  }
  if (routeQuery) {
    let query = JSON.parse(routeQuery);
    return { path: getNormalPath(props.basePath + '/' + routePath), query: query };
  }
  return getNormalPath(props.basePath + '/' + routePath);
}

function hasTitle(title: any) {
  if (title.length > 5) {
    return title;
  } else {
    return '';
  }
}
</script>
<style lang="scss" scoped>
:deep(.el-menu-item) {
  &.is-active {
    background: rgba(0, 0, 0, 0.05);
    border-radius: 4px;
  }
}

// :deep(.el-sub-menu) {
//   width: 208px;
//   margin: 0 auto;
//   border-radius: 4px;
//   overflow: hidden;
// }
</style>
