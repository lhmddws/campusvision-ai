package com.sims.dormitory.model.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@TableName("dorm_alert")
public class DormAlert {
    @TableId(type = IdType.AUTO)
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
