package com.sims.dormitory.model.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class EventDTO {
    private Long id;
    private String cameraId;
    private Long buildingId;
    private String eventType;
    private String studentId;
    private Boolean isStranger;
    private Double confidence;
    private String snapshotPath;
    private LocalDateTime timestamp;
    private LocalDateTime createdAt;
}
