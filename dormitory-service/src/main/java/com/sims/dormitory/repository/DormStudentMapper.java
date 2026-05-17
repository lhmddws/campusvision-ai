package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormStudent;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface DormStudentMapper extends BaseMapper<DormStudent> {

    @Select("SELECT * FROM dorm_student WHERE building_id = #{buildingId} AND status = #{status}")
    List<DormStudent> findByBuildingAndStatus(@Param("buildingId") Long buildingId, @Param("status") String status);

    @Select("SELECT * FROM dorm_student WHERE room_id = #{roomId}")
    List<DormStudent> findByRoomId(@Param("roomId") Long roomId);

    @Select("SELECT * FROM dorm_student WHERE student_id = #{studentId}")
    DormStudent findByStudentId(@Param("studentId") String studentId);
}
