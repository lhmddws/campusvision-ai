import request from '@/utils/request';

/** 告警列表查询参数 */
export interface AlertListParams {
  page?: number;
  size?: number;
  building?: string;
  alert_type?: string;
  acknowledged?: string;
}

/** 告警实体 */
export interface Alert {
  id: number;
  alert_id: string;
  alert_type: string;
  building: string | null;
  student_id: string | null;
  severity: string;
  description: string | null;
  face_snapshot_url: string | null;
  is_read: boolean;
  is_resolved: boolean;
  occurred_at: string;
  created_at: string;
}

/** 告警分页响应 */
export interface AlertListResult {
  items: Alert[];
  total: number;
  page: number;
  size: number;
}

/** 告警统计 */
export interface AlertStats {
  total: number;
  unread: number;
  today: number;
  by_type: Record<string, number>;
  by_severity: Record<string, number>;
}

/** 查询告警列表 */
export function getAlerts(params: AlertListParams) {
  return request({
    url: '/sims/dorm/alerts',
    method: 'get',
    params,
  });
}

/** 确认告警 */
export function acknowledgeAlert(id: number) {
  return request({
    url: `/sims/dorm/alerts/${id}/acknowledge`,
    method: 'post',
  });
}

/** 查询告警统计 */
export function getAlertStats(building?: string) {
  return request({
    url: '/sims/dorm/alerts/stats',
    method: 'get',
    params: { building },
  });
}