import router from './router';
import { ElMessage } from 'element-plus';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { getToken } from '@/utils/auth';
import { isRelogin } from '@/utils/request';
import useUserStore from '@/store/modules/user';
import useSettingsStore from '@/store/modules/settings';

NProgress.configure({ showSpinner: false });

const whiteList = ['/login'];

router.beforeEach((to, from, next) => {
    NProgress.start();
    if (getToken()) {
        to.meta.title && useSettingsStore().setTitle(to.meta.title);
        if (to.path === '/login') {
            next({ path: '/' });
            NProgress.done();
        } else {
            if (useUserStore().roles.length === 0) {
                isRelogin.show = true;
                useUserStore()
                    .getInfo()
                    .then(() => {
                        isRelogin.show = false;
                        next({ ...to, replace: true });
                    })
                    .catch(err => {
                        useUserStore()
                            .logOut()
                            .then(() => {
                                ElMessage.error(err);
                                next({ path: '/' });
                            });
                    });
            } else {
                next();
            }
        }
    } else {
        if (whiteList.indexOf(to.path) !== -1) {
            next();
        } else {
            next(`/login?redirect=${to.fullPath}`);
            NProgress.done();
        }
    }
});

router.afterEach(() => {
    NProgress.done();
});
