package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormEventLog;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface DormEventLogMapper extends BaseMapper<DormEventLog> {
}
