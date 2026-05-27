import request from '@/utils/request';

/** 事件日志查询参数 */
export interface EventLogQuery {
  page?: number;
  size?: number;
  building?: string;
  camera_id?: string;
  event_type?: string;
  student_id?: string;
  start_time?: string;
  end_time?: string;
}

/** 事件日志实体 */
export interface EventLog {
  id: number;
  camera_id: string | null;
  building: string;
  event_type: string;
  student_id: string | null;
  is_stranger: boolean;
  confidence: number | null;
  snapshot_path: string | null;
  timestamp: string;
  created_at: string;
}

/** 分页响应 */
export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
}

/** 查询事件日志列表 */
export function getEvents(params: EventLogQuery) {
  return request({
    url: '/sims/dorm/records/events',
    method: 'get',
    params,
  });
}