package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormNightlyReport;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;

import java.time.LocalDate;

@Mapper
public interface DormNightlyReportMapper extends BaseMapper<DormNightlyReport> {

    @Select("SELECT * FROM dorm_nightly_report WHERE building_id = #{buildingId} AND report_date = #{reportDate}")
    DormNightlyReport findByBuildingAndDate(@Param("buildingId") Long buildingId, @Param("reportDate") LocalDate reportDate);
}
