package com.sims.dormitory.model.dto;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
public class CameraUpdateDTO {
    private String name;
    private String building;
    private String rtspUrl;
    private String direction;
    private String resolution;
    private Boolean enabled;
    private String status;
    private String remark;
}
