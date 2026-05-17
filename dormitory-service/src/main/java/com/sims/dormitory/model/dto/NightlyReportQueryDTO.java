package com.sims.dormitory.model.dto;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDate;

@Getter
@Setter
@ToString
public class NightlyReportQueryDTO {
    private Long buildingId;
    private LocalDate startDate;
    private LocalDate endDate;
    private Integer page = 1;
    private Integer size = 20;
}
