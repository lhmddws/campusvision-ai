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
@TableName("dorm_student")
public class DormStudent {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String studentId;
    private String name;
    private String gender;
    private Long buildingId;
    private Long roomId;
    private String bedNumber;
    private String status;
    private LocalDateTime lastEventTime;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
