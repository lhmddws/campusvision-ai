package com.sims.dormitory.service.impl;

import com.sims.dormitory.model.dto.AttendanceStatsDTO;
import com.sims.dormitory.model.dto.DailySummaryDTO;
import com.sims.dormitory.model.entity.DormAttendanceRecord;
import com.sims.dormitory.repository.DormAttendanceRecordMapper;
import com.sims.dormitory.service.DormitoryRecordService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.time.LocalDate;
import java.util.List;

@Service
public class DormitoryRecordServiceImpl implements DormitoryRecordService {

    private static final Logger log = LoggerFactory.getLogger(DormitoryRecordServiceImpl.class);

    private final DormAttendanceRecordMapper attendanceRecordMapper;

    public DormitoryRecordServiceImpl(DormAttendanceRecordMapper attendanceRecordMapper) {
        this.attendanceRecordMapper = attendanceRecordMapper;
    }

    @Override
    public void handleAttendance(DormAttendanceRecord record) {
        // TODO: implement attendance handling logic
        // 1. Validate record data
        // 2. Insert or update attendance record
        // 3. Update student status if needed
        // 4. Trigger alerts for absences
        log.info("Handling attendance for studentId={}, date={}, status={}",
                record.getStudentId(), record.getDate(), record.getStatus());
        attendanceRecordMapper.insert(record);
    }

    @Override
    public AttendanceStatsDTO getAttendanceStats(Long buildingId, LocalDate startDate, LocalDate endDate) {
        // TODO: implement attendance statistics
        // 1. Query attendance records by building and date range
        // 2. Aggregate by status (present, absent, late, stranger)
        // 3. Calculate attendance rate
        log.info("Getting attendance stats for buildingId={}, from={}, to={}",
                buildingId, startDate, endDate);
        return new AttendanceStatsDTO(0, 0, 0, 0, 0, 0.0);
    }

    @Override
    public List<DailySummaryDTO> getDailySummary(Long buildingId, LocalDate startDate, LocalDate endDate) {
        // TODO: implement daily summary
        // 1. Query nightly reports for date range
        // 2. Calculate check-in rate per day
        // 3. Return daily summary list
        log.info("Getting daily summary for buildingId={}, from={}, to={}",
                buildingId, startDate, endDate);
        return List.of();
    }
}
