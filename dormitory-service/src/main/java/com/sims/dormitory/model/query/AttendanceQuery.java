package com.sims.dormitory.model.query;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDate;

@Getter
@Setter
@ToString
public class AttendanceQuery {
    private Long buildingId;
    private Long roomId;
    private LocalDate startDate;
    private LocalDate endDate;
    private String status;
    private Integer page = 1;
    private Integer size = 20;
}
