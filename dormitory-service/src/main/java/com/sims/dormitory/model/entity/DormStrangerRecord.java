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
@TableName("dorm_stranger_record")
public class DormStrangerRecord {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long buildingId;
    private String cameraId;
    private String snapshotPath;
    private Double confidence;
    private LocalDateTime detectedAt;
    private Boolean acknowledged;
    private LocalDateTime createdAt;
}
