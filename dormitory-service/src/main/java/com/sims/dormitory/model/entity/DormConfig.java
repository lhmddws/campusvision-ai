package com.sims.dormitory.model.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableField;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@TableName("dorm_config")
public class DormConfig {
    @TableId(type = IdType.AUTO)
    private Long id;

    private String configKey;

    private String configValue;

    @TableField("config_type")
    private String configType;

    private String description;

    @TableField("default_value")
    private String defaultValue;

    @TableField("group_name")
    private String groupName;

    @TableField("created_at")
    private LocalDateTime createdAt;

    private LocalDateTime updatedAt;
}
