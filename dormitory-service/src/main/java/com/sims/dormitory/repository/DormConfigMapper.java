package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormConfig;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Param;
import org.apache.ibatis.annotations.Select;
import org.apache.ibatis.annotations.Update;

import java.util.Set;

@Mapper
public interface DormConfigMapper extends BaseMapper<DormConfig> {

    @Select("SELECT * FROM dorm_config WHERE config_key = #{configKey}")
    DormConfig findByKey(@Param("configKey") String configKey);

    @Select("SELECT * FROM dorm_config WHERE group_name = #{group}")
    java.util.List<DormConfig> findByGroup(@Param("group") String group);

    @Select("SELECT DISTINCT group_name FROM dorm_config WHERE group_name IS NOT NULL AND group_name != ''")
    Set<String> findDistinctGroups();

    @Update("UPDATE dorm_config SET config_value = #{value}, updated_at = NOW() WHERE config_key = #{key}")
    int updateValueByKey(@Param("key") String key, @Param("value") String value);

    @Update("UPDATE dorm_config SET config_value = default_value, updated_at = NOW() WHERE config_key = #{key}")
    int resetByKey(@Param("key") String key);
}
