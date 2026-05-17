package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormCamera;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface DormCameraMapper extends BaseMapper<DormCamera> {

    @Select("SELECT * FROM dorm_camera WHERE building_id = #{buildingId}")
    List<DormCamera> findByBuildingId(@Param("buildingId") Long buildingId);

    @Select("SELECT * FROM dorm_camera WHERE camera_id = #{cameraId}")
    DormCamera findByCameraId(@Param("cameraId") String cameraId);

    @Select("SELECT * FROM dorm_camera WHERE enabled = true")
    List<DormCamera> findEnabledCameras();
}
