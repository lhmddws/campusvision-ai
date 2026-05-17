package com.sims.dormitory.model.dto;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class AttendanceStatsDTO {
    private long total;
    private long present;
    private long absent;
    private long late;
    private long stranger;
    private double rate;
}
