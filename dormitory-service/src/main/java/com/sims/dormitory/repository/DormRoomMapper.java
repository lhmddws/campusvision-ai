package com.sims.dormitory.repository;

import com.baomidou.mybatisplus.core.mapper.BaseMapper;
import com.sims.dormitory.model.entity.DormRoom;
import org.apache.ibatis.annotations.Mapper;

@Mapper
public interface DormRoomMapper extends BaseMapper<DormRoom> {
}
