import { login, logout, getInfo } from '@/api/login';
import { getToken, setToken, removeToken } from '@/utils/auth';
import defAva from '@/assets/images/profile.jpg';
import { defineStore } from 'pinia';

const useUserStore = defineStore('user', {
  state: (): {
        token?: string;
        name: string;
        avatar: string;
        roles: any[];
        permissions: string[];
    } => ({
    token: getToken(),
    name: '',
    avatar: '',
    roles: [],
    permissions: [],
  }),
  actions: {
    login(userInfo: { username: string; password: string }) {
      const username = userInfo.username.trim();
      const password = userInfo.password;
      return new Promise((resolve, reject) => {
        login(username, password)
          .then((res: any) => {
            setToken(res.data.token);
            this.token = res.data.token;
            resolve(1);
          })
          .catch(error => {
            reject(error);
          });
      });
    },
    // 获取用户信息
    getInfo() {
      return new Promise((resolve, reject) => {
        getInfo()
          .then((res: any) => {
            const user = res.data.user;
            const avatar =
                            user.avatar === '' || user.avatar == null
                              ? defAva
                              : import.meta.env.VITE_APP_BASE_API + user.avatar;

            if (res.data.roles && res.data.roles.length > 0) {
              // 验证返回的roles是否是一个非空数组
              this.roles = res.data.roles;
              this.permissions = res.data.permissions;
            } else {
              this.roles = ['ROLE_DEFAULT'];
            }
            this.name = user.userName;
            this.avatar = avatar;
            resolve(res);
          })
          .catch(error => {
            reject(error);
          });
      });
    },
    // 退出系统
    logOut() {
      return new Promise((resolve, reject) => {
        logout()
          .then(() => {
            this.token = '';
            this.roles = [];
            this.permissions = [];
            removeToken();
            resolve(1);
          })
          .catch(error => {
            reject(error);
          });
      });
    },
  },
});

export default useUserStore;
