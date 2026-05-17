package com.sims.dormitory.scheduler;

import com.sims.dormitory.model.entity.DormBuilding;
import com.sims.dormitory.repository.DormBuildingMapper;
import com.sims.dormitory.service.DormitoryReportService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.LocalDate;
import java.util.List;

@Component
public class NightlyReportTask {

    private static final Logger log = LoggerFactory.getLogger(NightlyReportTask.class);

    private final DormitoryReportService reportService;
    private final DormBuildingMapper buildingMapper;

    public NightlyReportTask(DormitoryReportService reportService, DormBuildingMapper buildingMapper) {
        this.reportService = reportService;
        this.buildingMapper = buildingMapper;
    }

    @Scheduled(cron = "0 0 23 * * ?")
    public void generateNightlyReports() {
        log.info("=== Starting nightly report generation ===");
        try {
            List<DormBuilding> buildings = buildingMapper.selectList(null);
            LocalDate today = LocalDate.now();
            for (DormBuilding building : buildings) {
                try {
                    reportService.generateNightlyReport(building.getId(), today);
                    log.info("Nightly report generated for building: {} (id={})", building.getName(), building.getId());
                } catch (Exception e) {
                    log.error("Failed to generate nightly report for building: {}", building.getId(), e);
                }
            }
            log.info("=== Nightly report generation completed ===");
        } catch (Exception e) {
            log.error("Nightly report generation failed", e);
        }
    }
}
