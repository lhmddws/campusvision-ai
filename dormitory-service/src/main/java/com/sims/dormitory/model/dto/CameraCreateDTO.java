package com.sims.dormitory.model.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
public class CameraCreateDTO {

    @NotBlank(message = "cameraId不能为空")
    private String cameraId;

    @NotNull(message = "buildingId不能为空")
    private Long buildingId;

    @NotBlank(message = "名称不能为空")
    private String name;

    @NotBlank(message = "RTSP地址不能为空")
    private String rtspUrl;
}
