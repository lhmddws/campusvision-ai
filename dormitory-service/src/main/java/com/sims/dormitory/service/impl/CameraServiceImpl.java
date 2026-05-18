package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.client.CameraPushClient;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.event.DormCameraEvent;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.dto.CameraUpdateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormCameraLog;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.repository.DormCameraLogMapper;
import com.sims.dormitory.repository.DormCameraMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
import com.sims.dormitory.service.CameraService;
import com.sims.dormitory.util.CryptoService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
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
    private final DormCameraLogMapper cameraLogMapper;
    private final CameraPushClient pushClient;
    private final CryptoService cryptoService;
    private final ApplicationEventPublisher eventPublisher;

    public CameraServiceImpl(DormCameraMapper cameraMapper, DormEventLogMapper eventLogMapper,
                              DormCameraLogMapper cameraLogMapper,
                              CameraPushClient pushClient, CryptoService cryptoService,
                              ApplicationEventPublisher eventPublisher) {
        this.cameraMapper = cameraMapper;
        this.eventLogMapper = eventLogMapper;
        this.cameraLogMapper = cameraLogMapper;
        this.pushClient = pushClient;
        this.cryptoService = cryptoService;
        this.eventPublisher = eventPublisher;
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
        camera.setBuilding(dto.getBuilding());
        camera.setName(dto.getName());
        camera.setRtspUrl(dto.getRtspUrl());
        camera.setDirection(dto.getDirection());
        camera.setResolution(dto.getResolution());
        camera.setRemark(dto.getRemark());
        camera.setStatus("unknown");
        camera.setEnabled(true);
        camera.setCreatedAt(LocalDateTime.now());
        camera.setUpdatedAt(LocalDateTime.now());
        cameraMapper.insert(camera);
        log.info("Camera registered: cameraId={}, building={}", dto.getCameraId(), dto.getBuilding());
        insertLog(camera, null, "unknown", "Camera registered");

        if (dto.getRtspUrl() != null && !dto.getRtspUrl().isEmpty()) {
            try {
                URI uri = new URI(dto.getRtspUrl());
                camera.setProtocol(uri.getScheme());
                camera.setHost(uri.getHost());
                if (uri.getPort() > 0) camera.setPort(uri.getPort());
                camera.setPath(uri.getRawPath());
                if (uri.getUserInfo() != null) {
                    String[] parts = uri.getUserInfo().split(":", 2);
                    camera.setUsername(parts[0]);
                    if (parts.length > 1) {
                        CryptoService.EncryptedPassword ep = cryptoService.encryptPassword(parts[1]);
                        camera.setPasswordEnc(ep.ciphertext());
                        camera.setNonce(ep.nonce());
                    }
                }
            } catch (Exception e) {
                log.warn("Failed to parse RTSP URL for encryption: {}", e.getMessage());
            }
        }

        eventPublisher.publishEvent(new DormCameraEvent(this, camera.getCameraId(), DormCameraEvent.EventType.REGISTERED, camera.getBuilding(), camera.getStatus()));

        try {
            pushClient.notifyRegister(camera);
        } catch (Exception e) {
            log.warn("Push notification failed for register: {}", e.getMessage());
        }

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

        eventPublisher.publishEvent(new DormCameraEvent(this, cameraId, DormCameraEvent.EventType.UPDATED, existing.getBuilding(), existing.getStatus()));

        try {
            pushClient.notifyUpdate(cameraId, existing);
        } catch (Exception e) {
            log.warn("Push notification failed for update: {}", e.getMessage());
        }
    }

    @Override
    public Map<String, Object> getCameraStatus(String building) {
        // TODO: implement camera status aggregation
        List<DormCamera> cameras;
        if (building != null && !building.isEmpty()) {
            cameras = cameraMapper.selectList(
                    new LambdaQueryWrapper<DormCamera>().eq(DormCamera::getBuilding, building));
        } else {
            cameras = cameraMapper.selectList(null);
        }

        List<Map<String, Object>> buildingList = cameras.stream().map(c -> {
            Map<String, Object> item = new HashMap<>();
            item.put("building", c.getBuilding());
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
    public List<DormCamera> getCameras(String building) {
        if (building != null && !building.isEmpty()) {
            return cameraMapper.findByBuilding(building);
        }
        return cameraMapper.selectList(null);
    }

    @Override
    public void healthCheck(String cameraId) {
        DormCamera camera = getByCameraId(cameraId);
        String oldStatus = camera.getStatus();
        String gatewayUrl = "http://localhost:8080/health";

        try {
            HttpClient client = HttpClient.newHttpClient();
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(gatewayUrl))
                    .timeout(Duration.ofSeconds(5))
                    .GET()
                    .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
            if (response.statusCode() == 200) {
                camera.setStatus("ONLINE");
                camera.setLastHeartbeat(LocalDateTime.now());
                log.info("Health check OK for camera: {}", cameraId);
            } else {
                camera.setStatus("OFFLINE");
                log.warn("Health check FAILED for camera: {} - HTTP {}", cameraId, response.statusCode());
            }
        } catch (Exception e) {
            camera.setStatus("OFFLINE");
            log.warn("Health check FAILED for camera: {} - {}", cameraId, e.getMessage());
        }

        String newStatus = camera.getStatus();
        if (!java.util.Objects.equals(oldStatus, newStatus)) {
            String reason = "ONLINE".equals(newStatus) ? "Health check" : "Health check failed";
            insertLog(camera, oldStatus, newStatus, reason);
        }

        camera.setUpdatedAt(LocalDateTime.now());
        cameraMapper.updateById(camera);
    }

    @Override
    @Transactional
    public void deleteCamera(String cameraId) {
        DormCamera camera = getByCameraId(cameraId);
        insertLog(camera, camera.getStatus(), "DELETED", "Camera deleted");
        cameraMapper.deleteById(camera.getId());
        log.info("Camera deleted: cameraId={}", cameraId);

        eventPublisher.publishEvent(new DormCameraEvent(this, cameraId, DormCameraEvent.EventType.DELETED, camera.getBuilding(), camera.getStatus()));

        try {
            pushClient.notifyDelete(cameraId);
        } catch (Exception e) {
            log.warn("Push notification failed for delete: {}", e.getMessage());
        }
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
            camera.setLastEventTime(
                    LocalDateTime.ofEpochSecond(timestampMs / 1000, 0, java.time.ZoneOffset.ofHours(8)));
        } else {
            camera.setLastEventTime(LocalDateTime.now());
        }
        cameraMapper.updateById(camera);
        log.debug("Updated last event time for camera: {} -> {}", cameraId, camera.getLastEventTime());
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

    private void insertLog(DormCamera camera, String statusFrom, String statusTo, String reason) {
        try {
            DormCameraLog logEntry = new DormCameraLog();
            logEntry.setCameraId(camera.getCameraId());
            logEntry.setBuilding(camera.getBuilding());
            logEntry.setStatusFrom(statusFrom);
            logEntry.setStatusTo(statusTo);
            logEntry.setReason(reason);
            logEntry.setFpsAtTime(null);
            logEntry.setCreatedAt(LocalDateTime.now());
            cameraLogMapper.insert(logEntry);
        } catch (Exception e) {
            log.warn("Failed to insert camera log for cameraId={}: {}", camera.getCameraId(), e.getMessage());
        }
    }
}
