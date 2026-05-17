package com.sims.dormitory.model.query;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
public class EventQuery {
    private Long buildingId;
    private String cameraId;
    private String eventType;
    private String studentId;
    private LocalDateTime startTime;
    private LocalDateTime endTime;
    private Integer page = 1;
    private Integer size = 20;
}
