package com.sims.dormitory.service;

import com.sims.dormitory.model.dto.ConfigUpdateDTO;
import com.sims.dormitory.model.entity.DormConfig;

import java.util.List;
import java.util.Map;
import java.util.Set;

public interface DormitoryConfigService {

    List<DormConfig> getAllConfigs();

    List<DormConfig> getAllConfigs(String group);

    DormConfig getConfigByKey(String configKey);

    void updateConfig(String configKey, String configValue);

    DormConfig resetConfig(String configKey);

    void batchUpdate(List<ConfigUpdateDTO> updates);

    Map<String, String> getConfigMap(String group);

    Set<String> getGroups();
}
