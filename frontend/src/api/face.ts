import request from '@/utils/request';

/** 人脸匹配 — 将 512 维嵌入向量与已注册人脸进行比对 */
export function faceMatch(embedding: number[]) {
  return request({
    url: '/api/face/match',
    method: 'post',
    data: { embedding },
  });
}

/** 人脸嵌入 — 从图片计算嵌入向量（当前为桩实现，返回 null） */
export function faceEmbed(imagePath: string) {
  return request({
    url: '/api/face/embed',
    method: 'post',
    data: { image_path: imagePath },
  });
}

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