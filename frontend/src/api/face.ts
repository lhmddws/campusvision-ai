import request from '@/utils/request';

/** 查询摄像头快照列表 */
export function getSnapshots(cameraId: string, page = 1, size = 20) {
  return request({
    url: `/sims/dorm/cameras/${cameraId}/snapshots`,
    method: 'get',
    params: { page, size },
  });
}

/** 获取摄像头列表（可按楼栋筛选） */
export function listCameras(building?: string) {
  return request({
    url: '/sims/dorm/cameras',
    method: 'get',
    params: building ? { building } : undefined,
  });
}