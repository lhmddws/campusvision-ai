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
@TableName("dorm_event_log")
public class DormEventLog {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String cameraId;
    private Long buildingId;
    private String eventType;
    private String studentId;
    private Boolean isStranger;
    private Double confidence;
    private String snapshotPath;
    private LocalDateTime timestamp;
    private LocalDateTime createdAt;
}
