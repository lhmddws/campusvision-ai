package com.sims.dormitory.model.dto;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
public class CameraUpdateDTO {
    private Long id;
    private String name;
    private String rtspUrl;
    private Boolean enabled;
    private String status;
}
