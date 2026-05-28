<template>
  <div class="login-container">
    <!-- Left Panel: Brand & Visual (60%) -->
    <div class="login-brand">
      <div class="brand-pattern brand-pattern--1"></div>
      <div class="brand-pattern brand-pattern--2"></div>
      <div class="brand-pattern brand-pattern--3"></div>
      <div class="brand-content">
        <div class="brand-logo">
          <img :src="LogoImg" alt="CampusVision AI" />
        </div>
        <h1 class="brand-title">{{ title }}</h1>
        <p class="brand-tagline">{{ subtitle }}</p>
        <div class="brand-features">
          <div class="brand-feature">
            <svg class="brand-feature__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
            <span>智能安防监控</span>
          </div>
          <div class="brand-feature">
            <svg class="brand-feature__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
              <circle cx="9" cy="7" r="4" />
              <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
              <path d="M16 3.13a4 4 0 0 1 0 7.75" />
            </svg>
            <span>人脸识别考勤</span>
          </div>
          <div class="brand-feature">
            <svg class="brand-feature__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
            </svg>
            <span>实时异常预警</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Right Panel: Login Form (40%) -->
    <div class="login-form-panel">
      <div class="login-card">
        <div class="login-card__header">
          <h2 class="login-card__title">欢迎登录</h2>
          <p class="login-card__subtitle">请输入您的账号信息</p>
        </div>

        <el-form ref="loginRef" :model="loginForm" :rules="loginRules" class="login-form">
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              type="text"
              size="large"
              auto-complete="off"
              placeholder="请输入账号"
            >
              <template #prefix>
                <svg-icon icon-class="user" class="el-input__icon input-icon" />
              </template>
            </el-input>
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              size="large"
              auto-complete="new-password"
              placeholder="请输入密码"
              show-password
              @keyup.enter="handleLogin"
            >
              <template #prefix>
                <svg-icon icon-class="password" class="el-input__icon input-icon" />
              </template>
            </el-input>
          </el-form-item>

          <div class="login-options">
            <el-checkbox v-model="loginForm.rememberMe">记住密码</el-checkbox>
          </div>

          <el-form-item>
            <el-button
              :loading="loading"
              size="large"
              type="primary"
              class="login-btn"
              @click.prevent="handleLogin"
            >
              <span v-if="!loading">登 录</span>
              <span v-else>登 录 中...</span>
            </el-button>
          </el-form-item>
        </el-form>

        <div class="login-footer">
          <span>Copyright &copy; {{ new Date().getFullYear() }} CampusVision AI</span>
        </div>
      </div>
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
@import '@/assets/styles/variables.module.scss';

// ── Design Tokens ──────────────────────────────────────────
$brand-gradient-start: #001529;
$brand-gradient-end: #1890FF;
$brand-accent: #40a9ff;
$form-bg: $card-bg;
$form-shadow: 0 20px 60px rgba(0, 0, 0, 0.08);
$input-height: 44px;
$transition-smooth: 0.3s cubic-bezier(0.4, 0, 0.2, 1);

// ── Layout ─────────────────────────────────────────────────
.login-container {
  display: flex;
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background-color: $page-bg;
}

// ── Left Brand Panel (60%) ────────────────────────────────
.login-brand {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 60%;
  height: 100%;
  overflow: hidden;
  background: linear-gradient(135deg, $brand-gradient-start 0%, darken($brand-gradient-end, 15%) 50%, $brand-gradient-end 100%);

  // Decorative geometric patterns
  .brand-pattern {
    position: absolute;
    border-radius: 50%;
    opacity: 0.08;

    &--1 {
      width: 600px;
      height: 600px;
      top: -120px;
      left: -160px;
      background: radial-gradient(circle, rgba(255, 255, 255, 0.3) 0%, transparent 70%);
      animation: float-pattern 18s ease-in-out infinite alternate;
    }

    &--2 {
      width: 400px;
      height: 400px;
      bottom: -80px;
      right: -60px;
      background: radial-gradient(circle, rgba(64, 169, 255, 0.4) 0%, transparent 70%);
      animation: float-pattern 14s ease-in-out infinite alternate-reverse;
    }

    &--3 {
      width: 250px;
      height: 250px;
      top: 50%;
      left: 65%;
      background: radial-gradient(circle, rgba(255, 255, 255, 0.15) 0%, transparent 70%);
      animation: float-pattern 22s ease-in-out infinite alternate;
    }
  }

  // Subtle grid overlay
  &::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
    background-size: 60px 60px;
    pointer-events: none;
  }
}

.brand-content {
  position: relative;
  z-index: 2;
  text-align: center;
  color: #ffffff;
  padding: 0 48px;
}

.brand-logo {
  display: flex;
  justify-content: center;
  margin-bottom: 32px;

  img {
    width: 88px;
    height: 88px;
    filter: drop-shadow(0 4px 12px rgba(0, 0, 0, 0.2));
    animation: logo-entrance 0.8s cubic-bezier(0.34, 1.56, 0.64, 1) both;
  }
}

.brand-title {
  font-size: 40px;
  font-weight: 700;
  letter-spacing: 2px;
  line-height: 1.3;
  margin: 0 0 12px;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  animation: title-entrance 0.6s 0.2s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.brand-tagline {
  font-size: 18px;
  font-weight: 400;
  opacity: 0.85;
  margin: 0 0 48px;
  letter-spacing: 4px;
  animation: title-entrance 0.6s 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) both;
}

.brand-features {
  display: flex;
  flex-direction: column;
  gap: 16px;
  align-items: flex-start;
  max-width: 280px;
  margin: 0 auto;
}

.brand-feature {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 15px;
  font-weight: 400;
  opacity: 0.9;
  animation: feature-entrance 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) both;

  &:nth-child(1) { animation-delay: 0.5s; }
  &:nth-child(2) { animation-delay: 0.65s; }
  &:nth-child(3) { animation-delay: 0.8s; }

  &__icon {
    width: 22px;
    height: 22px;
    flex-shrink: 0;
    opacity: 0.8;
  }
}

// ── Right Form Panel (40%) ────────────────────────────────
.login-form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40%;
  height: 100%;
  background-color: $page-bg;
  padding: 24px;
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: $form-bg;
  border-radius: 12px;
  box-shadow: $form-shadow;
  padding: 48px 40px 32px;
  animation: card-entrance 0.6s 0.15s cubic-bezier(0.34, 1.56, 0.64, 1) both;

  &__header {
    margin-bottom: 36px;
    text-align: center;
  }

  &__title {
    font-size: 26px;
    font-weight: 700;
    color: $text-primary;
    margin: 0 0 8px;
  }

  &__subtitle {
    font-size: 14px;
    color: $text-secondary;
    margin: 0;
  }
}

// ── Form Styles ────────────────────────────────────────────
.login-form {
  :deep(.el-form-item) {
    margin-bottom: 22px;
  }

  :deep(.el-input__wrapper) {
    height: $input-height;
    border-radius: 8px;
    transition: box-shadow $transition-smooth;
  }

  :deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 2px rgba($primary-color, 0.2);
  }

  :deep(.el-input__inner) {
    font-size: 15px;

    &:autofill {
      box-shadow: 0 0 0 1000px #fff inset;
    }

    &:-webkit-autofill {
      box-shadow: 0 0 0 1000px #fff inset;
    }
  }

  .input-icon {
    width: 18px;
    height: 18px;
    color: $text-secondary;
  }
}

.login-options {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;

  :deep(.el-checkbox__label) {
    color: $text-secondary;
    font-size: 14px;
  }
}

.login-btn {
  width: 100%;
  height: $input-height !important;
  border-radius: 8px !important;
  font-size: 16px !important;
  font-weight: 600 !important;
  letter-spacing: 2px;
  transition: all $transition-smooth;

  &:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 16px rgba($primary-color, 0.4);
  }

  &:active {
    transform: translateY(0);
  }
}

// ── Footer ─────────────────────────────────────────────────
.login-footer {
  margin-top: 24px;
  text-align: center;
  font-size: 12px;
  color: $text-secondary;
  letter-spacing: 0.5px;
}

// ── Animations ─────────────────────────────────────────────
@keyframes float-pattern {
  0% {
    transform: translate(0, 0) scale(1);
  }
  50% {
    transform: translate(30px, -20px) scale(1.05);
  }
  100% {
    transform: translate(-15px, 15px) scale(0.98);
  }
}

@keyframes logo-entrance {
  from {
    opacity: 0;
    transform: scale(0.6) translateY(20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

@keyframes title-entrance {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes feature-entrance {
  from {
    opacity: 0;
    transform: translateX(-20px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes card-entrance {
  from {
    opacity: 0;
    transform: translateY(24px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

// ── Responsive ─────────────────────────────────────────────
@media screen and (max-width: 1024px) {
  .login-container {
    flex-direction: column;
  }

  .login-brand {
    width: 100%;
    height: 40%;
    min-height: 260px;
  }

  .brand-title {
    font-size: 28px;
  }

  .brand-tagline {
    font-size: 15px;
    margin-bottom: 0;
  }

  .brand-features {
    display: none;
  }

  .login-form-panel {
    width: 100%;
    height: 60%;
    padding: 16px;
  }

  .login-card {
    padding: 32px 24px 24px;
    box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.06);
    border-radius: 16px 16px 0 0;
  }
}

@media screen and (max-width: 480px) {
  .login-card {
    padding: 24px 20px 20px;
  }

  .brand-logo img {
    width: 64px;
    height: 64px;
  }

  .brand-title {
    font-size: 22px;
  }
}
</style>