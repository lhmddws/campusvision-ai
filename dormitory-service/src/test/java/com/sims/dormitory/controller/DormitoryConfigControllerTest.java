package com.sims.dormitory.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.common.exception.GlobalExceptionHandler;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.model.entity.DormConfig;
import com.sims.dormitory.service.DormitoryConfigService;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

import java.util.List;
import java.util.Set;

import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.doThrow;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("DormitoryConfigController API Tests")
class DormitoryConfigControllerTest {

    private MockMvc mockMvc;

    @Mock
    private DormitoryConfigService configService;

    @BeforeEach
    void setUp() {
        DormitoryConfigController controller = new DormitoryConfigController(configService);
        mockMvc = MockMvcBuilders.standaloneSetup(controller)
                .setControllerAdvice(new GlobalExceptionHandler())
                .build();
    }

    @Nested
    @DisplayName("GET /api/configs")
    class ListConfigs {

        @Test
        @DisplayName("returns all configs")
        void shouldListAll() throws Exception {
            DormConfig config = new DormConfig();
            config.setId(1L);
            config.setConfigKey("test.key");
            config.setConfigValue("test.value");

            when(configService.getAllConfigs(isNull())).thenReturn(List.of(config));

            mockMvc.perform(get("/api/configs"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200))
                    .andExpect(jsonPath("$.data[0].configKey").value("test.key"))
                    .andExpect(jsonPath("$.data[0].configValue").value("test.value"));
        }

        @Test
        @DisplayName("filters by group parameter")
        void shouldFilterByGroup() throws Exception {
            when(configService.getAllConfigs("nightly")).thenReturn(List.of());

            mockMvc.perform(get("/api/configs").param("group", "nightly"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200));
        }
    }

    @Nested
    @DisplayName("GET /api/configs/{key}")
    class GetConfig {

        @Test
        @DisplayName("returns config by key")
        void shouldReturnConfig() throws Exception {
            DormConfig config = new DormConfig();
            config.setId(1L);
            config.setConfigKey("test.key");
            config.setConfigValue("test.value");

            when(configService.getConfigByKey("test.key")).thenReturn(config);

            mockMvc.perform(get("/api/configs/test.key"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200))
                    .andExpect(jsonPath("$.data.configKey").value("test.key"));
        }

        @Test
        @DisplayName("returns error when key not found")
        void shouldReturnErrorWhenNotFound() throws Exception {
            when(configService.getConfigByKey("nonexistent"))
                    .thenThrow(new BusinessException(ErrorCode.NOT_FOUND));

            mockMvc.perform(get("/api/configs/nonexistent"))
                    .andExpect(status().isBadRequest())
                    .andExpect(jsonPath("$.code").value(404));
        }
    }

    @Nested
    @DisplayName("PUT /api/configs/{key}")
    class UpdateConfig {

        @Test
        @DisplayName("updates config value")
        void shouldUpdateConfig() throws Exception {
            mockMvc.perform(put("/api/configs/test.key")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("{\"value\": \"new_value\"}"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200));
        }

        @Test
        @DisplayName("returns error when value field missing")
        void shouldReturnErrorWhenValueMissing() throws Exception {
            mockMvc.perform(put("/api/configs/test.key")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("{}"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(400));
        }

        @Test
        @DisplayName("returns error when key not found")
        void shouldReturnErrorWhenNotFound() throws Exception {
            doThrow(new BusinessException(ErrorCode.NOT_FOUND))
                    .when(configService).updateConfig("nonexistent", "value");

            mockMvc.perform(put("/api/configs/nonexistent")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("{\"value\": \"value\"}"))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("PUT /api/configs/batch")
    class BatchUpdate {

        @Test
        @DisplayName("batch updates configs")
        void shouldBatchUpdate() throws Exception {
            mockMvc.perform(put("/api/configs/batch")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("[{\"key\": \"k1\", \"value\": \"v1\"}, {\"key\": \"k2\", \"value\": \"v2\"}]"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200));
        }

        @Test
        @DisplayName("returns error for empty body")
        void shouldReturnErrorForEmpty() throws Exception {
            mockMvc.perform(put("/api/configs/batch")
                            .contentType(MediaType.APPLICATION_JSON)
                            .content("[]"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(400));
        }
    }

    @Nested
    @DisplayName("POST /api/configs/{key}/reset")
    class ResetConfig {

        @Test
        @DisplayName("resets config to default")
        void shouldResetConfig() throws Exception {
            DormConfig config = new DormConfig();
            config.setId(1L);
            config.setConfigKey("test.key");
            config.setConfigValue("default_val");

            when(configService.resetConfig("test.key")).thenReturn(config);

            mockMvc.perform(post("/api/configs/test.key/reset"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200))
                    .andExpect(jsonPath("$.data.configValue").value("default_val"));
        }

        @Test
        @DisplayName("returns error when key not found")
        void shouldReturnErrorWhenNotFound() throws Exception {
            when(configService.resetConfig("nonexistent"))
                    .thenThrow(new BusinessException(ErrorCode.NOT_FOUND));

            mockMvc.perform(post("/api/configs/nonexistent/reset"))
                    .andExpect(status().isBadRequest());
        }
    }

    @Nested
    @DisplayName("GET /api/configs/groups")
    class ListGroups {

        @Test
        @DisplayName("returns group names")
        void shouldReturnGroups() throws Exception {
            when(configService.getGroups()).thenReturn(Set.of("nightly", "alert"));

            mockMvc.perform(get("/api/configs/groups"))
                    .andExpect(status().isOk())
                    .andExpect(jsonPath("$.code").value(200))
                    .andExpect(jsonPath("$.data.length()").value(2));
        }
    }
}

