package com.sims.dormitory.service;

import com.sims.dormitory.model.dto.AttendanceStatsDTO;
import com.sims.dormitory.model.dto.DailySummaryDTO;
import com.sims.dormitory.model.entity.DormAttendanceRecord;

import java.time.LocalDate;
import java.util.List;

public interface DormitoryRecordService {

    void handleAttendance(DormAttendanceRecord record);

    AttendanceStatsDTO getAttendanceStats(Long buildingId, LocalDate startDate, LocalDate endDate);

    List<DailySummaryDTO> getDailySummary(Long buildingId, LocalDate startDate, LocalDate endDate);
}
