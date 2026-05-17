package com.sims.dormitory.controller;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.response.ApiResponse;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.dto.CameraUpdateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.service.CameraService;
import jakarta.validation.Valid;
import org.springframework.format.annotation.DateTimeFormat;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/sims/dorm/cameras")
public class CameraController {

    private final CameraService cameraService;

    public CameraController(CameraService cameraService) {
        this.cameraService = cameraService;
    }

    @PostMapping
    public ApiResponse<DormCamera> registerCamera(@RequestBody @Valid CameraCreateDTO dto) {
        DormCamera camera = cameraService.registerCamera(dto);
        return ApiResponse.success(camera);
    }

    @PutMapping("/{id}")
    public ApiResponse<Map<String, Object>> updateCamera(
            @PathVariable String id,
            @RequestBody CameraUpdateDTO dto) {
        cameraService.updateCamera(id, dto);
        Map<String, Object> result = new HashMap<>();
        result.put("cameraId", id);
        result.put("updated", true);
        return ApiResponse.success(result);
    }

    @GetMapping
    public ApiResponse<List<DormCamera>> getCameras(
            @RequestParam(required = false) Long buildingId) {
        List<DormCamera> cameras = cameraService.getCameras(buildingId);
        return ApiResponse.success(cameras);
    }

    @GetMapping("/{id}")
    public ApiResponse<DormCamera> getCamera(@PathVariable String id) {
        DormCamera camera = cameraService.getByCameraId(id);
        return ApiResponse.success(camera);
    }

    @GetMapping("/{id}/status")
    public ApiResponse<Map<String, Object>> getCameraStatus(@PathVariable String id) {
        DormCamera camera = cameraService.getByCameraId(id);
        Map<String, Object> status = new HashMap<>();
        status.put("cameraId", camera.getCameraId());
        status.put("status", camera.getStatus());
        status.put("lastHealthCheck", camera.getLastHealthCheck());
        status.put("enabled", camera.getEnabled());
        return ApiResponse.success(status);
    }
}
