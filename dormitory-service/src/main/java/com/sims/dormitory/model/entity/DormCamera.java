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
@TableName("dorm_camera")
public class DormCamera {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String cameraId;
    private String building;
    private String name;
    private String rtspUrl;
    private String direction;
    private String resolution;
    private String status;
    private Double fpsCurrent;
    private Long totalFrames;
    private LocalDateTime lastHeartbeat;
    private LocalDateTime lastEventTime;
    private Boolean enabled;
    private String configJson;
    private String remark;
    private LocalDateTime lastHealthCheck;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
