package com.sims.dormitory.service.impl;

import com.sims.dormitory.client.CameraPushClient;
import com.sims.dormitory.event.DormCameraEvent;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormCameraLog;
import com.sims.dormitory.repository.DormCameraLogMapper;
import com.sims.dormitory.repository.DormCameraMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
import com.sims.dormitory.util.CryptoService;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.MockedStatic;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.context.ApplicationEventPublisher;

import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.LocalDateTime;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("CameraServiceImpl CameraLog Tests")
class CameraServiceImplLogTest {

    @Mock
    private DormCameraMapper cameraMapper;

    @Mock
    private DormEventLogMapper eventLogMapper;

    @Mock
    private DormCameraLogMapper cameraLogMapper;

    @Mock
    private CameraPushClient pushClient;

    @Mock
    private CryptoService cryptoService;

    @Mock
    private ApplicationEventPublisher eventPublisher;

    @InjectMocks
    private CameraServiceImpl cameraService;

    // ──────────────────────────────────────────────
    // registerCamera log tests
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("registerCamera() log")
    class RegisterCameraLog {

        @Test
        @DisplayName("creates log entry with status_to=unknown on registration")
        void shouldCreateLogOnRegister() {
            when(cameraMapper.selectCount(null)).thenReturn(10L);
            when(cameraMapper.insert(any(DormCamera.class))).thenReturn(1);

            CameraCreateDTO dto = new CameraCreateDTO();
            dto.setCameraId("CAM-001");
            dto.setBuilding("A");
            dto.setName("Main Entrance");
            dto.setRtspUrl("rtsp://192.168.1.100:554/stream1");

            cameraService.registerCamera(dto);

            ArgumentCaptor<DormCameraLog> logCaptor = ArgumentCaptor.forClass(DormCameraLog.class);
            verify(cameraLogMapper).insert(logCaptor.capture());
            DormCameraLog logEntry = logCaptor.getValue();
            assertNull(logEntry.getStatusFrom());
            assertEquals("unknown", logEntry.getStatusTo());
            assertEquals("Camera registered", logEntry.getReason());
            assertEquals("CAM-001", logEntry.getCameraId());
            assertEquals("A", logEntry.getBuilding());
            assertNull(logEntry.getFpsAtTime());
            assertNotNull(logEntry.getCreatedAt());
        }
    }

    // ──────────────────────────────────────────────
    // healthCheck log tests
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("healthCheck() log")
    class HealthCheckLog {

        @Test
        @DisplayName("creates log entry when status changes to ONLINE")
        void shouldCreateLogWhenOnline() throws Exception {
            DormCamera camera = new DormCamera();
            camera.setId(1L);
            camera.setCameraId("CAM-001");
            camera.setBuilding("A");
            camera.setStatus("OFFLINE");
            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(camera);

            HttpClient mockClient = mock(HttpClient.class);
            HttpResponse<String> mockResponse = mock(HttpResponse.class);
            when(mockResponse.statusCode()).thenReturn(200);
            when(mockClient.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
                    .thenReturn(mockResponse);

            try (MockedStatic<HttpClient> httpClientMock = mockStatic(HttpClient.class)) {
                httpClientMock.when(HttpClient::newHttpClient).thenReturn(mockClient);

                cameraService.healthCheck("CAM-001");

                ArgumentCaptor<DormCameraLog> logCaptor = ArgumentCaptor.forClass(DormCameraLog.class);
                verify(cameraLogMapper).insert(logCaptor.capture());
                DormCameraLog logEntry = logCaptor.getValue();
                assertEquals("OFFLINE", logEntry.getStatusFrom());
                assertEquals("ONLINE", logEntry.getStatusTo());
                assertEquals("Health check", logEntry.getReason());
                assertEquals("CAM-001", logEntry.getCameraId());
                assertEquals("A", logEntry.getBuilding());
                assertNull(logEntry.getFpsAtTime());
                assertNotNull(logEntry.getCreatedAt());
            }
        }

        @Test
        @DisplayName("does NOT create log entry when status unchanged")
        void shouldNotCreateLogWhenStatusUnchanged() throws Exception {
            DormCamera camera = new DormCamera();
            camera.setId(1L);
            camera.setCameraId("CAM-001");
            camera.setBuilding("A");
            camera.setStatus("OFFLINE");
            camera.setLastHeartbeat(LocalDateTime.now().minusHours(1));
            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(camera);

            HttpClient mockClient = mock(HttpClient.class);
            when(mockClient.send(any(HttpRequest.class), any(HttpResponse.BodyHandler.class)))
                    .thenThrow(new java.io.IOException("Connection refused"));

            try (MockedStatic<HttpClient> httpClientMock = mockStatic(HttpClient.class)) {
                httpClientMock.when(HttpClient::newHttpClient).thenReturn(mockClient);

                cameraService.healthCheck("CAM-001");

                verify(cameraLogMapper, never()).insert(any(DormCameraLog.class));
            }
        }
    }

    // ──────────────────────────────────────────────
    // deleteCamera log tests
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("deleteCamera() log")
    class DeleteCameraLog {

        @Test
        @DisplayName("creates log entry with status_to=DELETED on deletion")
        void shouldCreateLogOnDelete() {
            DormCamera camera = new DormCamera();
            camera.setId(1L);
            camera.setCameraId("CAM-001");
            camera.setBuilding("A");
            camera.setStatus("ONLINE");
            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(camera);

            cameraService.deleteCamera("CAM-001");

            ArgumentCaptor<DormCameraLog> logCaptor = ArgumentCaptor.forClass(DormCameraLog.class);
            verify(cameraLogMapper).insert(logCaptor.capture());
            DormCameraLog logEntry = logCaptor.getValue();
            assertEquals("ONLINE", logEntry.getStatusFrom());
            assertEquals("DELETED", logEntry.getStatusTo());
            assertEquals("Camera deleted", logEntry.getReason());
            assertEquals("CAM-001", logEntry.getCameraId());
            assertEquals("A", logEntry.getBuilding());
            assertNull(logEntry.getFpsAtTime());
            assertNotNull(logEntry.getCreatedAt());
        }
    }

    // ──────────────────────────────────────────────
    // Log failure handling tests
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("Log failure handling")
    class LogFailureHandling {

        @Test
        @DisplayName("log failure does not throw, main operation succeeds")
        void logFailureShouldNotThrowMainOperation() {
            when(cameraMapper.selectCount(null)).thenReturn(10L);
            when(cameraMapper.insert(any(DormCamera.class))).thenReturn(1);
            doThrow(new RuntimeException("DB error")).when(cameraLogMapper).insert(any(DormCameraLog.class));

            CameraCreateDTO dto = new CameraCreateDTO();
            dto.setCameraId("CAM-001");
            dto.setBuilding("A");
            dto.setName("Main Entrance");
            dto.setRtspUrl("rtsp://192.168.1.100:554/stream1");

            // Should NOT throw despite cameraLogMapper.insert failing
            DormCamera result = assertDoesNotThrow(() -> cameraService.registerCamera(dto));
            assertNotNull(result);
            assertEquals("CAM-001", result.getCameraId());

            verify(cameraMapper).insert(any(DormCamera.class));
            verify(cameraLogMapper).insert(any(DormCameraLog.class));
        }
    }
}
