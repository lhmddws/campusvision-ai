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
    private Long buildingId;
    private String name;
    private String rtspUrl;
    private String status;
    private LocalDateTime lastHealthCheck;
    private Boolean enabled;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
