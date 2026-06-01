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

/** 查寝房间 DTO */
export interface InspectionStudent {
  student_id: string;
  student_name: string;
  status: string;
}

export interface InspectionRoom {
  building: string;
  room: string;
  total_students: number;
  present_count: number;
  unknown_count: number;
  students: InspectionStudent[];
}

/** 获取查寝名单 */
export function getInspectionList(building?: string) {
  const params: Record<string, string> = {};
  if (building) {
    params.building = building;
  }
  return request({ url: '/sims/dorm/records/attendance/inspection-list', method: 'get', params });
}