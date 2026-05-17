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
@TableName("dorm_attendance_record")
public class DormAttendanceRecord {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long buildingId;
    private Long roomId;
    private Long studentId;
    private LocalDate date;
    private String status;
    private LocalDateTime checkTime;
    private String remark;
    private LocalDateTime createdAt;
}
