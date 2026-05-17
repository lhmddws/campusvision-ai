package com.sims.dormitory.model.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDate;
import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class NightlyReportDTO {
    private Long id;
    private Long buildingId;
    private LocalDate reportDate;
    private Integer totalStudents;
    private Integer presentCount;
    private Integer absentCount;
    private Integer lateCount;
    private Integer strangerCount;
    private LocalDateTime generatedAt;
}
