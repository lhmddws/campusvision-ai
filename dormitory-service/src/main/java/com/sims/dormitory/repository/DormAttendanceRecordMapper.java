package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormAttendanceRecord;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.time.LocalDate;
import java.util.List;

@Mapper
public interface DormAttendanceRecordMapper extends BaseMapper<DormAttendanceRecord> {

    @Select("SELECT * FROM dorm_attendance_record WHERE building_id = #{buildingId} AND date = #{date}")
    List<DormAttendanceRecord> findByBuildingAndDate(@Param("buildingId") Long buildingId, @Param("date") LocalDate date);

    @Select("SELECT COUNT(*) FROM dorm_attendance_record WHERE building_id = #{buildingId} AND date = #{date} AND status = #{status}")
    long countByBuildingAndDateAndStatus(@Param("buildingId") Long buildingId, @Param("date") LocalDate date, @Param("status") String status);

    int insertBatch(@Param("list") List<DormAttendanceRecord> records);
}
