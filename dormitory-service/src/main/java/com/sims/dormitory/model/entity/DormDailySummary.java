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
@TableName("dorm_daily_summary")
public class DormDailySummary {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long buildingId;
    private LocalDate summaryDate;
    private Double checkinRate;
    private String details;
    private LocalDateTime createdAt;
}
