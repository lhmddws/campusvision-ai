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
@TableName("dorm_room")
public class DormRoom {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long buildingId;
    private String roomNumber;
    private Integer floor;
    private Integer capacity;
    private Integer currentCount;
    private String genderType;
    private Boolean enabled;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
