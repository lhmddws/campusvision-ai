package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.dto.CameraUpdateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.repository.DormCameraMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
import com.sims.dormitory.service.CameraService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

@Service
public class CameraServiceImpl implements CameraService {

    private static final Logger log = LoggerFactory.getLogger(CameraServiceImpl.class);

    private final DormCameraMapper cameraMapper;
    private final DormEventLogMapper eventLogMapper;

    public CameraServiceImpl(DormCameraMapper cameraMapper, DormEventLogMapper eventLogMapper) {
        this.cameraMapper = cameraMapper;
        this.eventLogMapper = eventLogMapper;
    }

    @Override
    @Transactional
    public DormCamera registerCamera(CameraCreateDTO dto) {
        // TODO: implement camera registration with limit check
        long count = cameraMapper.selectCount(null);
        if (count >= 50) {
            throw new BusinessException(ErrorCode.CAMERA_LIMIT_EXCEEDED);
        }
        DormCamera camera = new DormCamera();
        camera.setCameraId(dto.getCameraId());
        camera.setBuildingId(dto.getBuildingId());
        camera.setName(dto.getName());
        camera.setRtspUrl(dto.getRtspUrl());
        camera.setStatus("unknown");
        camera.setEnabled(true);
        camera.setCreatedAt(LocalDateTime.now());
        camera.setUpdatedAt(LocalDateTime.now());
        cameraMapper.insert(camera);
        log.info("Camera registered: cameraId={}, buildingId={}", dto.getCameraId(), dto.getBuildingId());
        return camera;
    }

    @Override
    @Transactional
    public void updateCamera(String cameraId, CameraUpdateDTO dto) {
        // TODO: implement camera update
        DormCamera existing = getByCameraId(cameraId);
        if (dto.getName() != null) existing.setName(dto.getName());
        if (dto.getRtspUrl() != null) existing.setRtspUrl(dto.getRtspUrl());
        if (dto.getEnabled() != null) existing.setEnabled(dto.getEnabled());
        if (dto.getStatus() != null) existing.setStatus(dto.getStatus());
        existing.setUpdatedAt(LocalDateTime.now());
        cameraMapper.updateById(existing);
        log.info("Camera updated: cameraId={}", cameraId);
    }

    @Override
    public Map<String, Object> getCameraStatus(String building) {
        // TODO: implement camera status aggregation
        List<DormCamera> cameras;
        if (building != null) {
            cameras = cameraMapper.selectList(
                    new LambdaQueryWrapper<DormCamera>().eq(DormCamera::getBuildingId, building));
        } else {
            cameras = cameraMapper.selectList(null);
        }

        List<Map<String, Object>> buildingList = cameras.stream().map(c -> {
            Map<String, Object> item = new HashMap<>();
            item.put("buildingId", c.getBuildingId());
            item.put("cameraId", c.getCameraId());
            item.put("status", c.getStatus());
            item.put("lastHealthCheck", c.getLastHealthCheck());
            return item;
        }).collect(Collectors.toList());

        long online = cameras.stream().filter(c -> "ONLINE".equals(c.getStatus())).count();
        long offline = cameras.stream().filter(c -> "OFFLINE".equals(c.getStatus())).count();
        long idle = cameras.stream().filter(c -> "IDLE".equals(c.getStatus())).count();

        Map<String, Object> result = new HashMap<>();
        result.put("cameras", buildingList);
        result.put("summary", Map.of("total", cameras.size(), "online", online, "offline", offline, "idle", idle));
        return result;
    }

    @Override
    public DormCamera getByCameraId(String cameraId) {
        DormCamera camera = cameraMapper.findByCameraId(cameraId);
        if (camera == null) {
            throw new BusinessException(ErrorCode.NOT_FOUND);
        }
        return camera;
    }

    @Override
    public List<DormCamera> getCameras(Long buildingId) {
        if (buildingId != null) {
            return cameraMapper.findByBuildingId(buildingId);
        }
        return cameraMapper.selectList(null);
    }

    @Override
    public void healthCheck(String cameraId) {
        // TODO: implement health check against stream-gateway
        // 1. HTTP GET stream-gateway health endpoint
        // 2. Update camera status and lastHeartbeat
        // 3. Log status changes
        log.info("Health check for camera: {}", cameraId);
    }

    @Override
    public List<DormCamera> listEnabledCameras() {
        return cameraMapper.findEnabledCameras();
    }

    @Override
    public List<DormCamera> listOnlineCameras() {
        return cameraMapper.selectList(
                new LambdaQueryWrapper<DormCamera>()
                        .eq(DormCamera::getStatus, "ONLINE")
                        .eq(DormCamera::getEnabled, true));
    }

    @Override
    @Transactional
    public void updateLastEventTime(String cameraId, Long timestampMs) {
        DormCamera camera = getByCameraId(cameraId);
        if (timestampMs != null) {
            camera.setUpdatedAt(
                    LocalDateTime.ofEpochSecond(timestampMs / 1000, 0, java.time.ZoneOffset.ofHours(8)));
        } else {
            camera.setUpdatedAt(LocalDateTime.now());
        }
        cameraMapper.updateById(camera);
        log.debug("Updated last event time for camera: {} -> {}", cameraId, camera.getUpdatedAt());
    }

    @Override
    public Page<DormEventLog> querySnapshots(String cameraId, LocalDateTime startTime,
                                             LocalDateTime endTime, int page, int size) {
        Page<DormEventLog> eventPage = new Page<>(page, size);
        LambdaQueryWrapper<DormEventLog> wrapper = new LambdaQueryWrapper<>();
        wrapper.eq(DormEventLog::getCameraId, cameraId);
        if (startTime != null) {
            wrapper.ge(DormEventLog::getTimestamp, startTime);
        }
        if (endTime != null) {
            wrapper.le(DormEventLog::getTimestamp, endTime);
        }
        wrapper.orderByDesc(DormEventLog::getTimestamp);
        return eventLogMapper.selectPage(eventPage, wrapper);
    }
}
