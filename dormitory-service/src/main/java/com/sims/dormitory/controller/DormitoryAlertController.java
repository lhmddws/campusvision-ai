package com.sims.dormitory.controller;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.response.ApiResponse;
import com.sims.dormitory.model.dto.AlertDTO;
import com.sims.dormitory.service.DormitoryAlertService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/sims/dorm/alerts")
public class DormitoryAlertController {

    private final DormitoryAlertService alertService;

    public DormitoryAlertController(DormitoryAlertService alertService) {
        this.alertService = alertService;
    }

    @GetMapping
    public ApiResponse<Page<AlertDTO>> getAlerts(
            @RequestParam(required = false) Long buildingId,
            @RequestParam(required = false) String alertType,
            @RequestParam(required = false) Boolean acknowledged,
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "20") int size) {
        Page<AlertDTO> result = alertService.getAlerts(buildingId, alertType, acknowledged, page, size);
        return ApiResponse.success(result);
    }

    @PostMapping("/{id}/acknowledge")
    public ApiResponse<Map<String, Object>> acknowledgeAlert(
            @PathVariable Long id,
            @RequestParam(required = false, defaultValue = "system") String acknowledgedBy) {
        alertService.acknowledgeAlert(id, acknowledgedBy);
        Map<String, Object> result = new HashMap<>();
        result.put("id", id);
        result.put("acknowledged", true);
        return ApiResponse.success(result);
    }

    @GetMapping("/stats")
    public ApiResponse<Map<String, Object>> getAlertStats(
            @RequestParam(required = false) Long buildingId) {
        long total = alertService.getAlertCount(buildingId, null);
        long unresolved = alertService.getAlertCount(buildingId, false);
        Map<String, Object> stats = new HashMap<>();
        stats.put("total", total);
        stats.put("unresolved", unresolved);
        return ApiResponse.success(stats);
    }
}
