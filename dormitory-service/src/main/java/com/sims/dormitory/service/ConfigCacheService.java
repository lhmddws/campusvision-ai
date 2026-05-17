package com.sims.dormitory.service;

import com.sims.dormitory.model.entity.DormConfig;
import com.sims.dormitory.repository.DormConfigMapper;
import jakarta.annotation.PostConstruct;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.Collections;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;

@Component
public class ConfigCacheService {

    private static final Logger log = LoggerFactory.getLogger(ConfigCacheService.class);

    private final DormConfigMapper configMapper;

    private volatile Map<String, DormConfig> cache = Collections.emptyMap();

    public ConfigCacheService(DormConfigMapper configMapper) {
        this.configMapper = configMapper;
    }

    @PostConstruct
    public void init() {
        log.info("Initializing config cache from database...");
        reload();
    }

    public void reload() {
        try {
            java.util.List<DormConfig> configs = configMapper.selectList(null);
            Map<String, DormConfig> newCache = new ConcurrentHashMap<>();
            for (DormConfig config : configs) {
                newCache.put(config.getConfigKey(), config);
            }
            this.cache = newCache;
            log.info("Config cache loaded with {} entries", newCache.size());
        } catch (Exception e) {
            log.warn("Failed to reload config cache from DB, using stale cache ({} entries)", cache.size(), e);
        }
    }

    public void reload(String key) {
        try {
            DormConfig config = configMapper.findByKey(key);
            Map<String, DormConfig> newCache = new ConcurrentHashMap<>(cache);
            if (config != null) {
                newCache.put(key, config);
            } else {
                newCache.remove(key);
            }
            this.cache = newCache;
        } catch (Exception e) {
            log.warn("Failed to reload config key '{}' from DB, keeping stale value", key, e);
        }
    }

    public String get(String key) {
        DormConfig config = cache.get(key);
        return config != null ? config.getConfigValue() : null;
    }

    public String get(String key, String defaultValue) {
        String value = get(key);
        return value != null ? value : defaultValue;
    }

    public int getInt(String key, int defaultValue) {
        DormConfig config = cache.get(key);
        if (config == null || config.getConfigValue() == null) {
            return defaultValue;
        }
        try {
            return Integer.parseInt(config.getConfigValue().trim());
        } catch (NumberFormatException e) {
            log.warn("Config '{}' value '{}' is not a valid int, using default {}", key, config.getConfigValue(), defaultValue);
            return defaultValue;
        }
    }

    public boolean getBool(String key, boolean defaultValue) {
        DormConfig config = cache.get(key);
        if (config == null || config.getConfigValue() == null) {
            return defaultValue;
        }
        String val = config.getConfigValue().trim().toLowerCase();
        if ("true".equals(val) || "1".equals(val) || "yes".equals(val)) {
            return true;
        } else if ("false".equals(val) || "0".equals(val) || "no".equals(val)) {
            return false;
        }
        return defaultValue;
    }

    public double getDouble(String key, double defaultValue) {
        DormConfig config = cache.get(key);
        if (config == null || config.getConfigValue() == null) {
            return defaultValue;
        }
        try {
            return Double.parseDouble(config.getConfigValue().trim());
        } catch (NumberFormatException e) {
            log.warn("Config '{}' value '{}' is not a valid double, using default {}", key, config.getConfigValue(), defaultValue);
            return defaultValue;
        }
    }

    public Map<String, String> getAll() {
        return cache.entrySet().stream()
                .collect(Collectors.toMap(Map.Entry::getKey, e -> e.getValue().getConfigValue()));
    }

    public Map<String, String> getByGroup(String group) {
        return cache.values().stream()
                .filter(c -> group.equals(c.getGroupName()))
                .collect(Collectors.toMap(DormConfig::getConfigKey, DormConfig::getConfigValue));
    }

    public Set<String> getGroups() {
        return cache.values().stream()
                .map(DormConfig::getGroupName)
                .filter(g -> g != null && !g.isEmpty())
                .collect(Collectors.toSet());
    }

}
