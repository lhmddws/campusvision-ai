import request from '@/utils/request';

/** 摄像头状态概览 */
export function getCamerasStatus(building?: string) {
  return request({
    url: '/sims/dorm/cameras/status',
    method: 'get',
    params: { building },
  });
}

/** 告警统计 */
export function getAlertStats(building?: string) {
  return request({
    url: '/sims/dorm/alerts/stats',
    method: 'get',
    params: { building },
  });
}

/** 出勤统计 */
export function getAttendanceStats(params?: {
  building_id?: number;
  start_date?: string;
  end_date?: string;
}) {
  return request({
    url: '/sims/dorm/records/attendance/stats',
    method: 'get',
    params,
  });
}

/** 最近事件列表 */
export function getRecentEvents(page = 1, size = 5) {
  return request({
    url: '/sims/dorm/records/events',
    method: 'get',
    params: { page, size },
  });
}