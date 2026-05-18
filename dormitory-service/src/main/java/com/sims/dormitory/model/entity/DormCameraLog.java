package com.sims.dormitory.model.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.math.BigDecimal;
import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@TableName("dorm_camera_log")
public class DormCameraLog {

    @TableId(type = IdType.AUTO)
    private Long id;

    private String cameraId;

    private String building;

    private String statusFrom;

    private String statusTo;

    private String reason;

    private BigDecimal fpsAtTime;

    private LocalDateTime createdAt;
}
