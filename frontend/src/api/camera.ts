import request from '@/utils/request';

/** 摄像头实体 */
export interface Camera {
  id: number;
  camera_id: string;
  name: string;
  building: string;
  rtsp_url: string;
  direction: string;
  status: string;
  fps_current: number | null;
  total_frames: number | null;
  last_heartbeat: string | null;
  last_event_time: string | null;
  enabled: boolean;
  config_json: string | null;
  remark: string | null;
  created_at: string;
  updated_at: string;
}

/** 查询摄像头列表 */
export function listCameras(building?: string) {
  return request({ url: '/sims/dorm/cameras', method: 'get', params: { building } });
}

/** 查询单个摄像头详情 */
export function getCamera(id: string) {
  return request({ url: `/sims/dorm/cameras/${id}`, method: 'get' });
}

/** 新增摄像头 */
export function addCamera(data: Partial<Camera>) {
  return request({ url: '/sims/dorm/cameras', method: 'post', data });
}

/** 修改摄像头 */
export function updateCamera(id: string, data: Partial<Camera>) {
  return request({ url: `/sims/dorm/cameras/${id}`, method: 'put', data });
}

/** 删除摄像头 */
export function deleteCamera(id: string) {
  return request({ url: `/sims/dorm/cameras/${id}`, method: 'delete' });
}

/** 查询摄像头状态 */
export function getCameraStatus(id: string) {
  return request({ url: `/sims/dorm/cameras/${id}/status`, method: 'get' });
}

/** 触发健康检查 */
export function healthCheck(id: string) {
  return request({ url: `/sims/dorm/cameras/${id}/health-check`, method: 'post' });
}

/** 查询摄像头快照列表 */
export function getCameraSnapshots(id: string, page: number, size: number) {
  return request({ url: `/sims/dorm/cameras/${id}/snapshots`, method: 'get', params: { page, size } });
}