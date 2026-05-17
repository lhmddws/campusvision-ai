package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormBuilding;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

@Mapper
public interface DormBuildingMapper extends BaseMapper<DormBuilding> {

    @Select("SELECT * FROM dorm_building WHERE code = #{code}")
    DormBuilding findByCode(@Param("code") String code);
}
