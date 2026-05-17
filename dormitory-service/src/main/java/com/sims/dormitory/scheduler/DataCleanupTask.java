package com.sims.dormitory.scheduler;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class DataCleanupTask {

    private static final Logger log = LoggerFactory.getLogger(DataCleanupTask.class);

    private final JdbcTemplate jdbcTemplate;

    public DataCleanupTask(JdbcTemplate jdbcTemplate) {
        this.jdbcTemplate = jdbcTemplate;
    }

    @Scheduled(cron = "0 0 3 * * ?")
    public void cleanupOldData() {
        log.info("=== Starting data cleanup ===");
        try {
            int deletedEvents = jdbcTemplate.update(
                    "DELETE FROM dorm_event_log WHERE created_at < NOW() - INTERVAL 90 DAY");
            log.info("Cleaned up {} old event logs", deletedEvents);

            int deletedStrangers = jdbcTemplate.update(
                    "DELETE FROM dorm_stranger_record WHERE created_at < NOW() - INTERVAL 30 DAY");
            log.info("Cleaned up {} old stranger records", deletedStrangers);

            int deletedAlerts = jdbcTemplate.update(
                    "DELETE FROM dorm_alert WHERE created_at < NOW() - INTERVAL 90 DAY");
            log.info("Cleaned up {} old alerts", deletedAlerts);

            log.info("=== Data cleanup completed ===");
        } catch (Exception e) {
            log.error("Data cleanup failed", e);
        }
    }
}
