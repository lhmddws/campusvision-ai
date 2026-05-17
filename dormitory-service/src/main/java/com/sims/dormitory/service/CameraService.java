package com.sims.dormitory.service;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.dto.CameraUpdateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormEventLog;

import java.util.List;
import java.util.Map;

public interface CameraService {

    DormCamera registerCamera(CameraCreateDTO dto);

    void updateCamera(String cameraId, CameraUpdateDTO dto);

    Map<String, Object> getCameraStatus(String building);

    DormCamera getByCameraId(String cameraId);

    List<DormCamera> getCameras(String building);

    void healthCheck(String cameraId);

    List<DormCamera> listEnabledCameras();

    List<DormCamera> listOnlineCameras();

    void updateLastEventTime(String cameraId, Long timestampMs);

    void deleteCamera(String cameraId);

    Page<DormEventLog> querySnapshots(String cameraId, java.time.LocalDateTime startTime,
                                      java.time.LocalDateTime endTime, int page, int size);
}
