package com.sims.dormitory.controller;

import com.sims.dormitory.common.response.ApiResponse;
import com.sims.dormitory.model.dto.ConfigUpdateDTO;
import com.sims.dormitory.model.entity.DormConfig;
import com.sims.dormitory.service.DormitoryConfigService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@RestController
@RequestMapping("/api/configs")
public class DormitoryConfigController {

    private final DormitoryConfigService configService;

    public DormitoryConfigController(DormitoryConfigService configService) {
        this.configService = configService;
    }

    @GetMapping
    public ApiResponse<List<DormConfig>> listConfigs(
            @RequestParam(required = false) String group) {
        List<DormConfig> configs = configService.getAllConfigs(group);
        return ApiResponse.success(configs);
    }

    @GetMapping("/{key}")
    public ApiResponse<DormConfig> getConfig(@PathVariable String key) {
        DormConfig config = configService.getConfigByKey(key);
        return ApiResponse.success(config);
    }

    @PutMapping("/{key}")
    public ApiResponse<Void> updateConfig(
            @PathVariable String key,
            @RequestBody Map<String, String> body) {
        String value = body.get("value");
        if (value == null) {
            return ApiResponse.error(400, "Field 'value' is required");
        }
        configService.updateConfig(key, value);
        return ApiResponse.success(null);
    }

    @PutMapping("/batch")
    public ApiResponse<Void> batchUpdate(@RequestBody List<Map<String, String>> body) {
        if (body == null || body.isEmpty()) {
            return ApiResponse.error(400, "Request body must be a non-empty array");
        }
        List<ConfigUpdateDTO> updates = body.stream().map(m -> {
            ConfigUpdateDTO dto = new ConfigUpdateDTO();
            dto.setConfigKey(m.get("key"));
            dto.setConfigValue(m.get("value"));
            return dto;
        }).collect(Collectors.toList());
        configService.batchUpdate(updates);
        return ApiResponse.success(null);
    }

    @PostMapping("/{key}/reset")
    public ApiResponse<DormConfig> resetConfig(@PathVariable String key) {
        DormConfig config = configService.resetConfig(key);
        return ApiResponse.success(config);
    }

    @GetMapping("/groups")
    public ApiResponse<Set<String>> listGroups() {
        Set<String> groups = configService.getGroups();
        return ApiResponse.success(groups);
    }
}
