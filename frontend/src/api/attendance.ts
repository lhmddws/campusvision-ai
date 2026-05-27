import request from '@/utils/request';

/** 考勤统计概览 */
export interface AttendanceStats {
  total: number;
  present: number;
  absent: number;
  late: number;
  stranger: number;
  rate: number;
}

/** 每日考勤汇总 */
export interface DailySummary {
  date: string;
  building_name: string;
  checkin_rate: number;
}

/** 查询参数 */
export interface AttendanceQuery {
  building_id?: number;
  start_date?: string;
  end_date?: string;
}

/** 获取考勤统计概览 */
export function getAttendanceStats(params: AttendanceQuery) {
  return request({ url: '/sims/dorm/records/attendance/stats', method: 'get', params });
}

/** 获取每日考勤汇总 */
export function getDailySummary(params: AttendanceQuery) {
  return request({ url: '/sims/dorm/records/attendance/daily-summary', method: 'get', params });
}