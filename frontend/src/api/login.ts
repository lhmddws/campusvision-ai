import request from '@/utils/request';

export function login(username: string, password: string) {
  return request({
    url: '/api/auth/login',
    headers: { isToken: false },
    method: 'post',
    data: { username, password },
  });
}

export function getInfo() {
  return request({
    url: '/api/auth/info',
    method: 'get',
  });
}

export function logout() {
  return request({
    url: '/api/auth/logout',
    method: 'post',
  });
}
