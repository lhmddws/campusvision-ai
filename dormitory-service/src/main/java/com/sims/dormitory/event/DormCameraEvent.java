package com.sims.dormitory.event;

import org.springframework.context.ApplicationEvent;

public class DormCameraEvent extends ApplicationEvent {

    public enum EventType {
        REGISTERED,
        UPDATED,
        DELETED,
        STATUS_CHANGED
    }

    private final String cameraId;
    private final EventType eventType;
    private final String building;
    private final String status;

    public DormCameraEvent(Object source, String cameraId, EventType eventType, String building, String status) {
        super(source);
        this.cameraId = cameraId;
        this.eventType = eventType;
        this.building = building;
        this.status = status;
    }

    public DormCameraEvent(Object source, String cameraId, EventType eventType) {
        this(source, cameraId, eventType, null, null);
    }

    public String getCameraId() { return cameraId; }
    public EventType getEventType() { return eventType; }
    public String getBuilding() { return building; }
    public String getStatus() { return status; }

    @Override
    public String toString() {
        return "DormCameraEvent{" +
            "cameraId='" + cameraId + '\'' +
            ", eventType=" + eventType +
            ", building='" + building + '\'' +
            ", status='" + status + '\'' +
            '}';
    }
}
