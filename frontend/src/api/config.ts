import request from '@/utils/request';

/** 查询配置列表（可按分组筛选） */
export function listConfigs(group?: string) {
  return request({
    url: '/api/configs',
    method: 'get',
    params: { group },
  });
}

/** 查询所有配置分组 */
export function getConfigGroups() {
  return request({
    url: '/api/configs/groups',
    method: 'get',
  });
}

/** 查询单个配置项 */
export function getConfig(key: string) {
  return request({
    url: `/api/configs/${key}`,
    method: 'get',
  });
}

/** 更新单个配置项 */
export function updateConfig(key: string, value: string) {
  return request({
    url: `/api/configs/${key}`,
    method: 'put',
    data: { value },
  });
}

/** 批量更新配置项 */
export function batchUpdateConfigs(items: { key: string; value: string }[]) {
  return request({
    url: '/api/configs/batch',
    method: 'put',
    data: items,
  });
}

/** 重置配置项为默认值 */
export function resetConfig(key: string) {
  return request({
    url: `/api/configs/${key}/reset`,
    method: 'post',
  });
}