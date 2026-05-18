package com.sims.dormitory.model.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Getter;
import lombok.Setter;
import lombok.ToString;

import java.time.LocalDateTime;

@Getter
@Setter
@ToString
@TableName("dorm_camera")
public class DormCamera {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String cameraId;
    private String building;
    private String type;            // default "RTSP"
    private String name;
    private String rtspUrl;
    private String protocol;        // default "rtsp"
    private String host;
    private Integer port;           // default 554
    private String path;
    private String username;
    private String passwordEnc;
    private String nonce;
    private String keyId;           // default "v1"
    private String direction;
    private String resolution;
    private String status;
    private Double fpsCurrent;
    private Long totalFrames;
    private LocalDateTime lastHeartbeat;
    private LocalDateTime lastEventTime;
    private Boolean enabled;
    private String configJson;
    private String remark;
    private LocalDateTime lastHealthCheck;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;

    /**
     * Build RTSP URL from component fields or fall back to rtspUrl.
     * @param decryptedPassword the decrypted password (null if using backward compat)
     * @return full RTSP URL string
     */
    public String buildRtspUrl(String decryptedPassword) {
        if (this.passwordEnc != null && decryptedPassword != null) {
            String userInfo = this.username != null && !this.username.isEmpty()
                ? this.username + ":" + decryptedPassword
                : null;
            try {
                java.net.URI uri = new java.net.URI(
                    this.protocol != null ? this.protocol : "rtsp",
                    userInfo,
                    this.host,
                    this.port != null ? this.port : 554,
                    this.path,
                    null,
                    null
                );
                return uri.toString();
            } catch (Exception e) {
                throw new RuntimeException("Failed to build RTSP URL from components", e);
            }
        }
        return this.rtspUrl;
    }
}
