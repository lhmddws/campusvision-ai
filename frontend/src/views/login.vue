<template>
    <div class="login">
        <div class="circle-box">
            <div class="circle circle1"></div>
            <div class="circle circle2"></div>
            <div class="circle circle3"></div>
        </div>
        <div class="backbox">
            <div class="back-left"></div>
            <div class="back-right"></div>
        </div>

        <div class="login-box">
            <div class="system-infos">
                <div class="system-logo">
                    <img :src="LogoImg" alt="" />
                </div>
                <div class="system-title">{{ title }}</div>
                <div class="system-subtitle">{{ subtitle }}</div>
            </div>
            <el-form ref="loginRef" :model="loginForm" :rules="loginRules" class="login-form">
                <h3 class="title">欢迎登录</h3>
                <el-form-item prop="username" class="username">
                    <el-input
 v-model="loginForm.username" type="text" size="large" auto-complete="off" placeholder="账号"
                        style="font-size: 16px">
                        <template #prefix><svg-icon icon-class="user" class="el-input__icon input-icon" /></template>
                    </el-input>
                </el-form-item>
                <el-form-item prop="password" class="password">
                    <el-input
 v-model="loginForm.password" type="password" size="large" auto-complete="new-password"
                        placeholder="密码" style="font-size: 16px; letter-spacing: 4px" clearable show-password
                        @keyup.enter="handleLogin">
                        <template #prefix><svg-icon icon-class="password" class="el-input__icon input-icon" /></template>
                    </el-input>
                </el-form-item>
                <el-checkbox v-model="loginForm.rememberMe" style="margin: 0px 0px 40px 0px">记住密码</el-checkbox>
                <el-form-item style="width: 100%">
                    <el-button
 :loading="loading" size="large" type="primary" style="width: 100%; height: 44px !important"
                        @click.prevent="handleLogin">
                        <span v-if="!loading">登 录</span>
                        <span v-else>登 录 中...</span>
                    </el-button>
                </el-form-item>
            </el-form>
        </div>
    </div>
</template>

<script setup lang="ts">
import Cookies from 'js-cookie';
import { encrypt, decrypt } from '@/utils/jsencrypt';
import useUserStore from '@/store/modules/user';
import { useRouter } from 'vue-router';
import { FormInstance } from 'element-plus';
import { ref } from 'vue';
import LogoImg from '@/assets/logo/logo.png';

const userStore = useUserStore();
const router = useRouter();
const loginForm = ref<any>({
  username: '',
  password: '',
  rememberMe: false,
});

const loginRules = {
  username: [{ required: true, trigger: 'blur', message: '请输入您的账号' }],
  password: [{ required: true, trigger: 'blur', message: '请输入您的密码' }],
};

const loading = ref(false);
const title = ref(import.meta.env.VITE_APP_TITLE);
const subtitle = ref(import.meta.env.VITE_APP_SUBTITLE);
const redirect = ref(undefined);
const loginRef = ref<FormInstance>();

function handleLogin() {
  loginRef.value?.validate(valid => {
    if (valid) {
      loading.value = true;
      if (loginForm.value.rememberMe) {
        Cookies.set('username', loginForm.value.username, { expires: 30 });
        const enPwd = encrypt(loginForm.value.password);
        if (enPwd) {
          Cookies.set('password', enPwd, { expires: 30 });
        }
        Cookies.set('rememberMe', String(loginForm.value.rememberMe), { expires: 30 });
      } else {
        Cookies.remove('username');
        Cookies.remove('password');
        Cookies.remove('rememberMe');
      }
      userStore
        .login(loginForm.value)
        .then(() => {
          router.push({ path: redirect.value || '/' });
        })
        .catch(() => {
          loading.value = false;
        });
    }
  });
}

function getCookie() {
  const username = Cookies.get('username');
  const password = Cookies.get('password');
  const rememberMe = Cookies.get('rememberMe');
  loginForm.value = {
    username: username === undefined ? loginForm.value.username : username,
    password: password === undefined ? loginForm.value.password : decrypt(password) || '',
    rememberMe: rememberMe === undefined ? false : Boolean(rememberMe),
  };
}

getCookie();
</script>

<style lang="scss" scoped>
.login {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100%;
    background-color: var(--el-color-primary);

    input:-internal-autofill-selected {
        background-color: #ffffff !important;
    }

    .backbox {
        position: absolute;
        top: 0;
        left: 0;
        display: flex;
        width: 100%;
        height: 100%;

        .back-left {
            width: 70.3125%;
            height: 100%;
            background-color: #ffffff;
        }

        .back-right {
            width: 29.6875%;
            height: 100%;
            background: rgba(2, 167, 151, 0.05);
            backdrop-filter: blur(95px);
        }
    }

    .login-box {
        display: flex;
        align-items: center;
        justify-content: space-between;

        .system-infos {
            min-width: 1220px;
            z-index: 11;

            .system-logo {
                margin-top: 48px;
            }

            .system-title {
                margin-top: 16px;
                font-weight: 700;
                font-size: 60px;
                line-height: 90px;
                color: #1f2329;
            }

            .system-subtitle {
                font-weight: 400;
                font-size: 24px;
                line-height: 36px;
                color: #63656a;
            }

            .system-empower {
                display: flex;
                align-items: center;
                margin-top: 75px;
                font-weight: 400;
                font-size: 14px;
                line-height: 20px;
                color: #63656a;

                .empower-box {
                    margin-right: 8px;
                    padding: 7px 12px;
                    border-radius: 4px;
                    border: 1px solid #e4e4e5;

                    .clock-class {
                        color: #ff6060;
                        margin-right: 4px;
                    }

                    .empower-day {
                        margin-left: 8px;
                        color: #ff6060;
                    }
                }
            }
        }

        .login-form {
            padding: 64px 40px;
            position: absolute;
            width: 420px;
            height: 449px;
            right: 19%;
            background: rgba(250, 250, 250, 0.9);
            border: 1px solid #e4e4e5;
            backdrop-filter: blur(25px);
            border-radius: 8px;
            font-size: 16px;
            z-index: 20;

            :deep(.el-input__inner) {
                background: rgba(255, 255, 255, 0.1);
                color: #1f2329;

                &:autofill {
                    box-shadow: 0px 0px 1000px rgba(250, 250, 250, 0.9) inset;
                }

                &:-webkit-autofill {
                    box-shadow: 0px 0px 1000px rgba(250, 250, 250, 0.9) inset;
                }
            }
            :deep(.el-input__wrapper) {
                background: rgba(255, 255, 255, 0.4);
            }

            .title {
                font-weight: 700;
                font-size: 32px;
                line-height: 48px;
                color: #1f2329;
            }

            .username {
                margin-top: 40px;
                font-size: 16px;
            }

            .password {
                margin-top: 24px;


            }

            .el-input {
                height: 44px;

                input {
                    height: 44px;
                }
            }

            .input-icon {
                height: 20px;
                width: 20px;
                margin-left: 0px;
                color: #63656a;
            }
        }

        .login-tip {
            font-size: 13px;
            text-align: center;
            color: #bfbfbf;
        }

        .login-code {
            width: 33%;
            height: 40px;
            float: right;

            img {
                cursor: pointer;
                vertical-align: middle;
            }
        }
    }

    .circle-box {
        overflow: hidden;
        position: absolute;
        top: 0;
        right: 0;
        width: 500px;
        height: 100%;

        .circle {
            position: absolute;
            border-radius: 50%;
            box-sizing: border-box;
            animation: cc 3s infinite alternate ease-in-out;
        }

        .circle1 {
            width: 372px;
            height: 380px;
            top: 26px;
            background: #59afff;
            animation: bb 3s infinite linear;
        }

        .circle2 {
            width: 255px;
            height: 260px;
            top: 40%;
            right: 10%;
            background: #86ff7b;
            animation: bb 4s infinite linear;
        }

        .circle3 {
            width: 225px;
            height: 230px;
            top: 580px;
            background: #5dbdf3;
            animation: bb 2s infinite linear;
        }
    }

    @keyframes cc {
        to {
            transform: translateY(100px);
        }
    }

    @keyframes bb {
        0% {
            transform: translate(0px, 0px);
        }

        25% {
            transform: translate(20px, 30px);
        }

        50% {
            transform: translate(50px, 0px);
        }

        75% {
            transform: translate(-20px, -30px);
        }

        100% {
            transform: translate(0px, 0px);
        }
    }
}

.el-login-footer {
    height: 40px;
    line-height: 40px;
    position: fixed;
    bottom: 0;
    width: 100%;
    text-align: center;
    color: #fff;
    font-size: 12px;
    letter-spacing: 1px;
}

.login-code-img {
    height: 40px;
    padding-left: 12px;
}</style>
