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
public class AlertDTO {
    private Long id;
    private Long buildingId;
    private String alertType;
    private String message;
    private String details;
    private Boolean acknowledged;
    private String acknowledgedBy;
    private LocalDateTime acknowledgedAt;
    private LocalDateTime createdAt;
}
