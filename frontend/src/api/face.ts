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

/** 查询人脸记录列表 */
export function listFaces(page = 1, size = 20) {
  return request({
    url: '/sims/dorm/faces',
    method: 'get',
    params: { page, size },
  });
}

/** 新增人脸记录 */
export function addFace(data: { student_id: string; name: string; room_number?: string }) {
  return request({
    url: '/sims/dorm/faces',
    method: 'post',
    data,
  });
}

/** 修改人脸记录 */
export function updateFace(id: number, data: { name?: string; room_number?: string }) {
  return request({
    url: `/sims/dorm/faces/${id}`,
    method: 'put',
    data,
  });
}

/** 删除人脸记录 */
export function deleteFace(id: number) {
  return request({
    url: `/sims/dorm/faces/${id}`,
    method: 'delete',
  });
}

/** 批量导入人脸记录 */
export function batchImportFaces(
  students: { student_id: string; name: string; room_number?: string }[],
) {
  return request({
    url: '/sims/dorm/faces/batch-import',
    method: 'post',
    data: { students },
  });
}
