package com.sims.dormitory.service;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.model.dto.NightlyReportDTO;
import com.sims.dormitory.model.entity.DormNightlyReport;

import java.time.LocalDate;

public interface DormitoryReportService {

    DormNightlyReport generateNightlyReport(Long buildingId, LocalDate date);

    NightlyReportDTO getNightlyReport(Long buildingId, LocalDate date);

    Page<NightlyReportDTO> getReportHistory(Long buildingId, LocalDate startDate, LocalDate endDate, int page, int size);
}
