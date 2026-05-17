package com.sims.dormitory.service.impl;

import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.model.dto.ConfigUpdateDTO;
import com.sims.dormitory.model.entity.DormConfig;
import com.sims.dormitory.repository.DormConfigMapper;
import com.sims.dormitory.service.ConfigCacheService;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;
import java.util.Set;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("DormitoryConfigServiceImpl Unit Tests")
class DormitoryConfigServiceImplTest {

    @Mock
    private DormConfigMapper configMapper;

    @Mock
    private ConfigCacheService configCacheService;

    @InjectMocks
    private DormitoryConfigServiceImpl configService;

    // ──────────────────────────────────────────────
    // getAllConfigs
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getAllConfigs()")
    class GetAllConfigs {

        @Test
        @DisplayName("returns all configs")
        void shouldReturnAllConfigs() {
            DormConfig c1 = buildConfig(1L, "nightly_report_time", "23:00", "Nightly report generation time");
            DormConfig c2 = buildConfig(2L, "curfew_time", "22:00", "Curfew time");
            DormConfig c3 = buildConfig(3L, "absent_threshold_minutes", "30", "Absent threshold in minutes");

            when(configMapper.selectList(null)).thenReturn(List.of(c1, c2, c3));

            List<DormConfig> result = configService.getAllConfigs();

            assertNotNull(result);
            assertEquals(3, result.size());
            assertEquals("nightly_report_time", result.get(0).getConfigKey());
            assertEquals("22:00", result.get(1).getConfigValue());
            verify(configMapper).selectList(null);
        }

        @Test
        @DisplayName("returns empty list when no configs exist")
        void shouldReturnEmptyWhenNoConfigs() {
            when(configMapper.selectList(null)).thenReturn(List.of());

            List<DormConfig> result = configService.getAllConfigs();

            assertNotNull(result);
            assertTrue(result.isEmpty());
        }
    }

    // ──────────────────────────────────────────────
    // getAllConfigs(group)
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getAllConfigs(group)")
    class GetAllConfigsByGroup {

        @Test
        @DisplayName("filters by group name")
        void shouldFilterByGroup() {
            DormConfig c1 = buildConfig(1L, "nightly_report.trigger_time", "23:00", null);
            c1.setGroupName("nightly");
            when(configMapper.selectList(any())).thenReturn(List.of(c1));

            List<DormConfig> result = configService.getAllConfigs("nightly");

            assertEquals(1, result.size());
            verify(configMapper).selectList(any());
        }

        @Test
        @DisplayName("returns all when group is null")
        void shouldReturnAllWhenNull() {
            when(configMapper.selectList(null)).thenReturn(List.of());

            List<DormConfig> result = configService.getAllConfigs(null);

            assertNotNull(result);
            verify(configMapper).selectList(null);
        }

        @Test
        @DisplayName("returns all when group is blank")
        void shouldReturnAllWhenBlank() {
            when(configMapper.selectList(null)).thenReturn(List.of());

            List<DormConfig> result = configService.getAllConfigs("  ");

            assertNotNull(result);
            verify(configMapper).selectList(null);
        }
    }

    // ──────────────────────────────────────────────
    // updateConfig
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("updateConfig()")
    class UpdateConfig {

        @Test
        @DisplayName("updates existing config and reloads cache")
        void shouldUpdateConfig() {
            DormConfig existing = new DormConfig();
            existing.setId(1L);
            existing.setConfigKey("nightly_report_time");
            existing.setConfigValue("23:00");
            existing.setUpdatedAt(LocalDateTime.now().minusDays(1));

            when(configMapper.findByKey("nightly_report_time")).thenReturn(existing);

            configService.updateConfig("nightly_report_time", "22:30");

            verify(configMapper).updateById(any(DormConfig.class));
            verify(configCacheService).reload("nightly_report_time");
        }

        @Test
        @DisplayName("updates config with different value types")
        void shouldUpdateWithDifferentValues() {
            DormConfig existing = buildConfig(2L, "absent_threshold_minutes", "30", null);
            when(configMapper.findByKey("absent_threshold_minutes")).thenReturn(existing);

            configService.updateConfig("absent_threshold_minutes", "45");

            verify(configMapper).updateById(any(DormConfig.class));
            verify(configCacheService).reload("absent_threshold_minutes");
        }

        @Test
        @DisplayName("throws NOT_FOUND when config key does not exist")
        void shouldThrowWhenKeyNotFound() {
            when(configMapper.findByKey("nonexistent_key")).thenReturn(null);

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> configService.updateConfig("nonexistent_key", "value"));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
            verify(configMapper, never()).updateById(any(DormConfig.class));
            verify(configCacheService, never()).reload(anyString());
        }

        @Test
        @DisplayName("throws NOT_FOUND for null key")
        void shouldThrowWhenKeyIsNull() {
            when(configMapper.findByKey(null)).thenReturn(null);

            assertThrows(BusinessException.class,
                    () -> configService.updateConfig(null, "value"));
        }

        @Test
        @DisplayName("updates with empty string value")
        void shouldUpdateWithEmptyValue() {
            DormConfig existing = buildConfig(3L, "some_key", "old", null);
            when(configMapper.findByKey("some_key")).thenReturn(existing);

            configService.updateConfig("some_key", "");

            verify(configMapper).updateById(any(DormConfig.class));
            verify(configCacheService).reload("some_key");
        }
    }

    // ──────────────────────────────────────────────
    // getConfigByKey
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getConfigByKey()")
    class GetConfigByKey {

        @Test
        @DisplayName("returns config when key exists")
        void shouldReturnConfig() {
            DormConfig expected = buildConfig(1L, "nightly_report_time", "23:00", "Schedule");
            when(configMapper.findByKey("nightly_report_time")).thenReturn(expected);

            DormConfig result = configService.getConfigByKey("nightly_report_time");

            assertNotNull(result);
            assertEquals("nightly_report_time", result.getConfigKey());
            assertEquals("23:00", result.getConfigValue());
            assertEquals("Schedule", result.getDescription());
        }

        @Test
        @DisplayName("throws NOT_FOUND when key does not exist")
        void shouldThrowWhenNotFound() {
            when(configMapper.findByKey("ghost_key")).thenReturn(null);

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> configService.getConfigByKey("ghost_key"));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
        }

        @Test
        @DisplayName("throws NOT_FOUND for null key")
        void shouldThrowWhenKeyIsNull() {
            when(configMapper.findByKey(null)).thenReturn(null);

            assertThrows(BusinessException.class,
                    () -> configService.getConfigByKey(null));
        }
    }

    // ──────────────────────────────────────────────
    // resetConfig
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("resetConfig()")
    class ResetConfig {

        @Test
        @DisplayName("resets config to default value and reloads cache")
        void shouldResetToDefault() {
            DormConfig resetCfg = buildConfig(1L, "test.key", "default_val", null);
            resetCfg.setDefaultValue("default_val");

            when(configMapper.resetByKey("test.key")).thenReturn(1);
            when(configMapper.findByKey("test.key")).thenReturn(resetCfg);

            DormConfig result = configService.resetConfig("test.key");

            assertEquals("default_val", result.getConfigValue());
            verify(configMapper).resetByKey("test.key");
            verify(configCacheService).reload("test.key");
        }

        @Test
        @DisplayName("throws NOT_FOUND when key does not exist")
        void shouldThrowWhenNotFound() {
            when(configMapper.resetByKey("ghost")).thenReturn(0);

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> configService.resetConfig("ghost"));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
        }
    }

    // ──────────────────────────────────────────────
    // batchUpdate
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("batchUpdate()")
    class BatchUpdate {

        @Test
        @DisplayName("updates multiple configs")
        void shouldUpdateMultiple() {
            DormConfig c1 = buildConfig(1L, "key1", "old1", null);
            DormConfig c2 = buildConfig(2L, "key2", "old2", null);
            when(configMapper.findByKey("key1")).thenReturn(c1);
            when(configMapper.findByKey("key2")).thenReturn(c2);

            ConfigUpdateDTO upd1 = new ConfigUpdateDTO();
            upd1.setConfigKey("key1");
            upd1.setConfigValue("new1");
            ConfigUpdateDTO upd2 = new ConfigUpdateDTO();
            upd2.setConfigKey("key2");
            upd2.setConfigValue("new2");

            configService.batchUpdate(List.of(upd1, upd2));

            verify(configMapper, times(2)).updateById(any(DormConfig.class));
            verify(configCacheService).reload("key1");
            verify(configCacheService).reload("key2");
        }

        @Test
        @DisplayName("does nothing for empty list")
        void shouldDoNothingForEmpty() {
            configService.batchUpdate(List.of());
            verify(configMapper, never()).updateById(any(DormConfig.class));
        }

        @Test
        @DisplayName("does nothing for null")
        void shouldDoNothingForNull() {
            configService.batchUpdate(null);
            verify(configMapper, never()).updateById(any(DormConfig.class));
        }
    }

    // ──────────────────────────────────────────────
    // getConfigMap
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getConfigMap()")
    class GetConfigMap {

        @Test
        @DisplayName("returns configs for specific group")
        void shouldReturnByGroup() {
            DormConfig c1 = buildConfig(1L, "nightly_report.trigger_time", "23:00", null);
            c1.setGroupName("nightly");
            DormConfig c2 = buildConfig(2L, "late_return.threshold", "22:00", null);
            c2.setGroupName("nightly");

            when(configMapper.findByGroup("nightly")).thenReturn(List.of(c1, c2));

            Map<String, String> result = configService.getConfigMap("nightly");

            assertEquals(2, result.size());
            assertEquals("23:00", result.get("nightly_report.trigger_time"));
            assertEquals("22:00", result.get("late_return.threshold"));
        }

        @Test
        @DisplayName("returns empty map when group has no configs")
        void shouldReturnEmptyForUnknownGroup() {
            when(configMapper.findByGroup("unknown")).thenReturn(List.of());

            Map<String, String> result = configService.getConfigMap("unknown");

            assertTrue(result.isEmpty());
        }

        @Test
        @DisplayName("returns all configs when group is null")
        void shouldReturnAllWhenNull() {
            DormConfig c1 = buildConfig(1L, "key_a", "val_a", null);
            when(configMapper.selectList(null)).thenReturn(List.of(c1));

            Map<String, String> result = configService.getConfigMap(null);

            assertEquals(1, result.size());
        }
    }

    // ──────────────────────────────────────────────
    // getGroups
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getGroups()")
    class GetGroups {

        @Test
        @DisplayName("returns distinct group names")
        void shouldReturnGroups() {
            when(configMapper.findDistinctGroups()).thenReturn(Set.of("nightly", "alert", "sync"));

            Set<String> result = configService.getGroups();

            assertEquals(3, result.size());
            assertTrue(result.contains("nightly"));
        }

        @Test
        @DisplayName("returns empty set when no groups exist")
        void shouldReturnEmptyWhenNoGroups() {
            when(configMapper.findDistinctGroups()).thenReturn(Set.of());

            Set<String> result = configService.getGroups();

            assertTrue(result.isEmpty());
        }
    }

    // ──────────────────────────────────────────────
    // Helpers
    // ──────────────────────────────────────────────

    private static DormConfig buildConfig(Long id, String key, String value, String description) {
        DormConfig config = new DormConfig();
        config.setId(id);
        config.setConfigKey(key);
        config.setConfigValue(value);
        config.setDescription(description);
        config.setUpdatedAt(LocalDateTime.now());
        return config;
    }
}
