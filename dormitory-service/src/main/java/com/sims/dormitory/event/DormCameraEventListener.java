package com.sims.dormitory.event;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;

@Component
public class DormCameraEventListener {

    private static final Logger log = LoggerFactory.getLogger(DormCameraEventListener.class);

    @EventListener
    public void handleCameraEvent(DormCameraEvent event) {
        log.info("Camera event received: cameraId={}, eventType={}, building={}, status={}",
            event.getCameraId(), event.getEventType(), event.getBuilding(), event.getStatus());

        // V1: Logging only. Future: push notification, camera_log write, etc.
        // V1 is synchronous only — no async processing or retry.
    }
}
