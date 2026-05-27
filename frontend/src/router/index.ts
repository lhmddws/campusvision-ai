import { createWebHistory, createRouter, RouteRecordRaw } from 'vue-router';
/* Layout */
import Layout from '@/layout/index.vue';

/**
 * Note: 路由配置项
 *
 * hidden: true                     // 当设置 true 的时候该路由不会再侧边栏出现 如401，login等页面，或者如一些编辑页面/edit/1
 * alwaysShow: true                 // 当你一个路由下面的 children 声明的路由大于1个时，自动会变成嵌套的模式--如组件页面
 *                                  // 只有一个时，会将那个子路由当做根路由显示在侧边栏--如引导页面
 *                                  // 若你想不管路由下面的 children 声明的个数都显示你的根路由
 *                                  // 你可以设置 alwaysShow: true，这样它就会忽略之前定义的规则，一直显示根路由
 * redirect: noRedirect             // 当设置 noRedirect 的时候该路由在面包屑导航中不可被点击
 * name:'router-name'               // 设定路由的名字，一定要填写不然使用<keep-alive>时会出现各种问题
 * query: '{"id": 1, "name": "ry"}' // 访问路由的默认传递参数
 * roles: ['admin', 'common']       // 访问路由的角色权限
 * permissions: ['a:a:a', 'b:b:b']  // 访问路由的菜单权限
 * meta : {
    noCache: true                   // 如果设置为true，则不会被 <keep-alive> 缓存(默认 false)
    title: 'title'                  // 设置该路由在侧边栏和面包屑中展示的名字
    icon: 'svg-name'                // 设置该路由的图标，对应路径src/assets/icons/svg
    breadcrumb: false               // 如果设置为false，则不会在breadcrumb面包屑中显示
    activeMenu: '/system/user'      // 当路由设置了该属性，则会高亮相对应的侧边栏。
  }
 */
declare module 'vue-router' {
  interface RouteMeta {
    hidden?: boolean;
    title?: string;
    icon?: string;
    elSvgIcon?: string;
    permissions?: string[];
  }
  interface _RouteRecordBase {
    hidden?: boolean;
    parentPath?: string;
    permissions?: string[];
  }
  interface _RouteLocationBase {
    title?: string;
  }
}

export const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/redirect',
    component: Layout,
    hidden: true,
    children: [
      {
        path: '/redirect/:path(.*)',
        component: () => import('@/views/redirect/index.vue'),
      },
    ],
  },
  {
    path: '/login',
    component: () => import('@/views/login.vue'),
    hidden: true,
  },
  {
    path: '/:pathMatch(.*)*',
    component: () => import('@/views/error/404.vue'),
    hidden: true,
  },
  {
    path: '/401',
    component: () => import('@/views/error/401.vue'),
    hidden: true,
  },
  {
    path: '',
    component: Layout,
    redirect: '/index',
    children: [
      {
        path: '/index',
        component: () => import('@/views/dashboard/index.vue'),
        name: 'Dashboard',
        meta: { title: '仪表盘', icon: 'dashboard', affix: true },
      },
    ],
  },
  {
    path: '/camera',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/camera/index.vue'),
        name: 'Camera',
        meta: { title: '摄像头管理', icon: 'monitor' },
      },
    ],
  },
  {
    path: '/events',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/events/index.vue'),
        name: 'Events',
        meta: { title: '进出事件', icon: 'log' },
      },
    ],
  },
  {
    path: '/attendance',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/attendance/index.vue'),
        name: 'Attendance',
        meta: { title: '考勤统计', icon: 'date' },
      },
    ],
  },
  {
    path: '/alerts',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/alerts/index.vue'),
        name: 'Alerts',
        meta: { title: '告警管理', icon: 'message' },
      },
    ],
  },
  {
    path: '/face',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/face/index.vue'),
        name: 'Face',
        meta: { title: '人脸录入', icon: 'peoples' },
      },
    ],
  },
  {
    path: '/config',
    component: Layout,
    children: [
      {
        path: 'index',
        component: () => import('@/views/config/index.vue'),
        name: 'Config',
        meta: { title: '系统配置', icon: 'system' },
      },
    ],
  },
];

export const dynamicRoutes: RouteRecordRaw[] = [];

const router = createRouter({
  history: createWebHistory(),
  routes: constantRoutes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition;
    } else {
      return { top: 0 };
    }
  },
});

export default router;
