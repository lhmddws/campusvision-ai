package com.sims.dormitory.model.dto;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
public class AlertQueryDTO {
    private Long buildingId;
    private String alertType;
    private Boolean acknowledged;
    private LocalDateTime startDate;
    private LocalDateTime endDate;
    private Integer page = 1;
    private Integer size = 20;
}
