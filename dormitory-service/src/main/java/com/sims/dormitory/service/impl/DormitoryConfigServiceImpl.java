package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.model.dto.ConfigUpdateDTO;
import com.sims.dormitory.model.entity.DormConfig;
import com.sims.dormitory.repository.DormConfigMapper;
import com.sims.dormitory.service.ConfigCacheService;
import com.sims.dormitory.service.DormitoryConfigService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@Service
public class DormitoryConfigServiceImpl implements DormitoryConfigService {

    private static final Logger log = LoggerFactory.getLogger(DormitoryConfigServiceImpl.class);

    private final DormConfigMapper configMapper;
    private final ConfigCacheService configCacheService;

    public DormitoryConfigServiceImpl(DormConfigMapper configMapper, ConfigCacheService configCacheService) {
        this.configMapper = configMapper;
        this.configCacheService = configCacheService;
    }

    @Override
    public List<DormConfig> getAllConfigs() {
        return configMapper.selectList(null);
    }

    @Override
    public List<DormConfig> getAllConfigs(String group) {
        if (group == null || group.isBlank()) {
            return getAllConfigs();
        }
        return configMapper.selectList(
                new LambdaQueryWrapper<DormConfig>()
                        .eq(DormConfig::getGroupName, group));
    }

    @Override
    @Transactional
    public void updateConfig(String configKey, String configValue) {
        DormConfig config = configMapper.findByKey(configKey);
        if (config == null) {
            throw new BusinessException(ErrorCode.NOT_FOUND);
        }
        config.setConfigValue(configValue);
        config.setUpdatedAt(LocalDateTime.now());
        configMapper.updateById(config);
        configCacheService.reload(configKey);
        log.info("Config updated: key={}, value={}", configKey, configValue);
    }

    @Override
    @Transactional
    public DormConfig resetConfig(String configKey) {
        int affected = configMapper.resetByKey(configKey);
        if (affected == 0) {
            throw new BusinessException(ErrorCode.NOT_FOUND);
        }
        configCacheService.reload(configKey);
        DormConfig config = configMapper.findByKey(configKey);
        log.info("Config reset to default: key={}, value={}", configKey, config.getConfigValue());
        return config;
    }

    @Override
    @Transactional
    public void batchUpdate(List<ConfigUpdateDTO> updates) {
        if (updates == null || updates.isEmpty()) {
            return;
        }
        for (ConfigUpdateDTO update : updates) {
            updateConfig(update.getConfigKey(), update.getConfigValue());
        }
        log.info("Batch updated {} configs", updates.size());
    }

    @Override
    public DormConfig getConfigByKey(String configKey) {
        DormConfig config = configMapper.findByKey(configKey);
        if (config == null) {
            throw new BusinessException(ErrorCode.NOT_FOUND);
        }
        return config;
    }

    @Override
    public Map<String, String> getConfigMap(String group) {
        List<DormConfig> configs;
        if (group == null || group.isBlank()) {
            configs = configMapper.selectList(null);
        } else {
            configs = configMapper.findByGroup(group);
        }
        return configs.stream()
                .collect(Collectors.toMap(DormConfig::getConfigKey, DormConfig::getConfigValue));
    }

    @Override
    public Set<String> getGroups() {
        return configMapper.findDistinctGroups();
    }
}
