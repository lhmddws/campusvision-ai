package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.model.dto.NightlyReportDTO;
import com.sims.dormitory.model.entity.DormNightlyReport;
import com.sims.dormitory.repository.DormNightlyReportMapper;
import com.sims.dormitory.service.DormitoryReportService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.List;
import java.util.stream.Collectors;

@Service
public class DormitoryReportServiceImpl implements DormitoryReportService {

    private static final Logger log = LoggerFactory.getLogger(DormitoryReportServiceImpl.class);

    private final DormNightlyReportMapper nightlyReportMapper;

    public DormitoryReportServiceImpl(DormNightlyReportMapper nightlyReportMapper) {
        this.nightlyReportMapper = nightlyReportMapper;
    }

    @Override
    @Transactional
    public DormNightlyReport generateNightlyReport(Long buildingId, LocalDate date) {
        // TODO: implement nightly report generation
        // 1. Get all students for building
        // 2. Query attendance records for the date
        // 3. Determine present/absent/late/stranger counts
        // 4. Save or update report record
        log.info("Generating nightly report for buildingId={}, date={}", buildingId, date);

        DormNightlyReport report = new DormNightlyReport();
        report.setBuildingId(buildingId);
        report.setReportDate(date);
        report.setTotalStudents(0);
        report.setPresentCount(0);
        report.setAbsentCount(0);
        report.setLateCount(0);
        report.setStrangerCount(0);
        report.setGeneratedAt(LocalDateTime.now());
        nightlyReportMapper.insert(report);
        return report;
    }

    @Override
    public NightlyReportDTO getNightlyReport(Long buildingId, LocalDate date) {
        DormNightlyReport report = nightlyReportMapper.findByBuildingAndDate(buildingId, date);
        if (report == null) {
            return null;
        }
        return toDTO(report);
    }

    @Override
    public Page<NightlyReportDTO> getReportHistory(Long buildingId, LocalDate startDate,
                                                   LocalDate endDate, int page, int size) {
        Page<DormNightlyReport> reportPage = new Page<>(page, size);
        LambdaQueryWrapper<DormNightlyReport> wrapper = new LambdaQueryWrapper<>();
        if (buildingId != null) {
            wrapper.eq(DormNightlyReport::getBuildingId, buildingId);
        }
        if (startDate != null) {
            wrapper.ge(DormNightlyReport::getReportDate, startDate);
        }
        if (endDate != null) {
            wrapper.le(DormNightlyReport::getReportDate, endDate);
        }
        wrapper.orderByDesc(DormNightlyReport::getReportDate);

        Page<DormNightlyReport> result = nightlyReportMapper.selectPage(reportPage, wrapper);

        List<NightlyReportDTO> dtoList = result.getRecords().stream()
                .map(this::toDTO)
                .collect(Collectors.toList());

        Page<NightlyReportDTO> dtoPage = new Page<>(result.getCurrent(), result.getSize(), result.getTotal());
        dtoPage.setRecords(dtoList);
        return dtoPage;
    }

    private NightlyReportDTO toDTO(DormNightlyReport report) {
        return new NightlyReportDTO(
                report.getId(),
                report.getBuildingId(),
                report.getReportDate(),
                report.getTotalStudents(),
                report.getPresentCount(),
                report.getAbsentCount(),
                report.getLateCount(),
                report.getStrangerCount(),
                report.getGeneratedAt()
        );
    }
}
