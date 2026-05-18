package com.sims.dormitory.model.dto;

import jakarta.validation.constraints.NotBlank;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

@Getter
@Setter
@ToString
public class CameraCreateDTO {

    @NotBlank(message = "cameraId不能为空")
    private String cameraId;

    @NotBlank(message = "building不能为空")
    private String building;

    @NotBlank(message = "名称不能为空")
    private String name;

    @NotBlank(message = "RTSP地址不能为空")
    private String rtspUrl;

    private String direction;
    private String resolution;
    private String remark;

    private String type;
    private String protocol;
    private String host;
    private Integer port;
    private String path;
    private String username;
}
