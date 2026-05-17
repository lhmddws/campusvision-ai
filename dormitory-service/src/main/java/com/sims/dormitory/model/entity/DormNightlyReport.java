package com.sims.dormitory.model.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDate;
import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@TableName("dorm_nightly_report")
public class DormNightlyReport {
    @TableId(type = IdType.AUTO)
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
