package com.sims.dormitory.controller;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.response.ApiResponse;
import com.sims.dormitory.model.dto.AttendanceStatsDTO;
import com.sims.dormitory.model.dto.DailySummaryDTO;
import com.sims.dormitory.model.dto.EventQueryDTO;
import com.sims.dormitory.model.entity.DormAttendanceRecord;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.service.DormitoryRecordService;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDate;
import java.util.List;

@RestController
@RequestMapping("/sims/dorm/records")
public class DormitoryRecordController {

    private final DormitoryRecordService recordService;

    public DormitoryRecordController(DormitoryRecordService recordService) {
        this.recordService = recordService;
    }

    @PostMapping("/attendance")
    public ApiResponse<Void> handleAttendance(@RequestBody DormAttendanceRecord record) {
        recordService.handleAttendance(record);
        return ApiResponse.success(null);
    }

    @GetMapping("/attendance/stats")
    public ApiResponse<AttendanceStatsDTO> getAttendanceStats(
            @RequestParam(required = false) Long buildingId,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate startDate,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate endDate) {
        AttendanceStatsDTO stats = recordService.getAttendanceStats(buildingId, startDate, endDate);
        return ApiResponse.success(stats);
    }

    @GetMapping("/attendance/daily-summary")
    public ApiResponse<List<DailySummaryDTO>> getDailySummary(
            @RequestParam(required = false) Long buildingId,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate startDate,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) LocalDate endDate) {
        List<DailySummaryDTO> summaries = recordService.getDailySummary(buildingId, startDate, endDate);
        return ApiResponse.success(summaries);
    }

    @GetMapping("/events")
    public ApiResponse<Page<DormEventLog>> getEvents(EventQueryDTO query) {
        // TODO: implement event log query with pagination
        return ApiResponse.success(new Page<>());
    }
}
