package com.sims.dormitory.model.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDate;

@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class DailySummaryDTO {
    private LocalDate date;
    private String buildingName;
    private Double checkinRate;
}
