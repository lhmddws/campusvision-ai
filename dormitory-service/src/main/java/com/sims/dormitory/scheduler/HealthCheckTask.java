package com.sims.dormitory.scheduler;

import com.sims.dormitory.service.CameraService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

@Component
public class HealthCheckTask {

    private static final Logger log = LoggerFactory.getLogger(HealthCheckTask.class);

    private final CameraService cameraService;

    public HealthCheckTask(CameraService cameraService) {
        this.cameraService = cameraService;
    }

    @Scheduled(fixedRate = 30000)
    public void checkCameras() {
        try {
            cameraService.listEnabledCameras().forEach(camera -> {
                try {
                    cameraService.healthCheck(camera.getCameraId());
                } catch (Exception e) {
                    log.warn("Health check failed for camera: {}", camera.getCameraId(), e);
                }
            });
        } catch (Exception e) {
            log.error("Camera health check task failed", e);
        }
    }
}
