package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.model.dto.CameraCreateDTO;
import com.sims.dormitory.model.dto.CameraUpdateDTO;
import com.sims.dormitory.model.entity.DormCamera;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.repository.DormCameraMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
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

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("CameraServiceImpl Unit Tests")
class CameraServiceImplTest {

    @Mock
    private DormCameraMapper cameraMapper;

    @Mock
    private DormEventLogMapper eventLogMapper;

    @InjectMocks
    private CameraServiceImpl cameraService;

    // ──────────────────────────────────────────────
    // registerCamera
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("registerCamera()")
    class RegisterCamera {

        @Test
        @DisplayName("registers camera successfully when limit not exceeded")
        void shouldRegisterWhenUnderLimit() {
            when(cameraMapper.selectCount(null)).thenReturn(10L);
            when(cameraMapper.insert(any(DormCamera.class))).thenReturn(1);

            CameraCreateDTO dto = new CameraCreateDTO();
            dto.setCameraId("CAM-001");
            dto.setBuildingId(1L);
            dto.setName("Main Entrance");
            dto.setRtspUrl("rtsp://192.168.1.100:554/stream1");

            DormCamera result = cameraService.registerCamera(dto);

            assertNotNull(result);
            assertEquals("CAM-001", result.getCameraId());
            assertEquals(1L, result.getBuildingId());
            assertEquals("Main Entrance", result.getName());
            assertEquals("rtsp://192.168.1.100:554/stream1", result.getRtspUrl());
            assertEquals("unknown", result.getStatus());
            assertTrue(result.getEnabled());
            assertNotNull(result.getCreatedAt());
            assertNotNull(result.getUpdatedAt());

            verify(cameraMapper).insert(isA(DormCamera.class));
        }

        @Test
        @DisplayName("throws CAMERA_LIMIT_EXCEEDED when count >= 50")
        void shouldThrowWhenLimitExceeded() {
            when(cameraMapper.selectCount(null)).thenReturn(50L);

            CameraCreateDTO dto = new CameraCreateDTO();
            dto.setCameraId("CAM-002");
            dto.setBuildingId(2L);
            dto.setName("Overflow Cam");
            dto.setRtspUrl("rtsp://192.168.1.101:554/stream1");

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> cameraService.registerCamera(dto));
            assertEquals(ErrorCode.CAMERA_LIMIT_EXCEEDED.getCode(), ex.getCode());
            verify(cameraMapper, never()).insert(isA(DormCamera.class));
        }

        @Test
        @DisplayName("throws CAMERA_LIMIT_EXCEEDED when count exceeds 50")
        void shouldThrowWhenCountExceedsFifty() {
            when(cameraMapper.selectCount(null)).thenReturn(100L);

            CameraCreateDTO dto = new CameraCreateDTO();
            dto.setCameraId("CAM-003");
            dto.setBuildingId(3L);
            dto.setName("Overflow Cam 2");
            dto.setRtspUrl("rtsp://192.168.1.102:554/stream1");

            assertThrows(BusinessException.class, () -> cameraService.registerCamera(dto));
            verify(cameraMapper, never()).insert(isA(DormCamera.class));
        }
    }

    // ──────────────────────────────────────────────
    // updateCamera
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("updateCamera()")
    class UpdateCamera {

        @Test
        @DisplayName("updates all fields when provided")
        void shouldUpdateAllFields() {
            DormCamera existing = new DormCamera();
            existing.setId(1L);
            existing.setCameraId("CAM-001");
            existing.setName("Old Name");
            existing.setRtspUrl("old://url");
            existing.setEnabled(false);
            existing.setStatus("OFFLINE");

            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(existing);

            CameraUpdateDTO dto = new CameraUpdateDTO();
            dto.setName("New Name");
            dto.setRtspUrl("new://url");
            dto.setEnabled(true);
            dto.setStatus("ONLINE");

            cameraService.updateCamera("CAM-001", dto);

            verify(cameraMapper).updateById(isA(DormCamera.class));
        }

        @Test
        @DisplayName("partial update only changes non-null fields")
        void shouldPartialUpdate() {
            DormCamera existing = new DormCamera();
            existing.setId(2L);
            existing.setCameraId("CAM-002");
            existing.setName("Keep Name");
            existing.setRtspUrl("keep://url");
            existing.setEnabled(true);
            existing.setStatus("ONLINE");

            when(cameraMapper.findByCameraId("CAM-002")).thenReturn(existing);

            CameraUpdateDTO dto = new CameraUpdateDTO();
            dto.setName("Updated Name");

            cameraService.updateCamera("CAM-002", dto);

            verify(cameraMapper).updateById(isA(DormCamera.class));
        }

        @Test
        @DisplayName("throws NOT_FOUND when camera does not exist")
        void shouldThrowWhenNotFound() {
            when(cameraMapper.findByCameraId("NONEXISTENT")).thenReturn(null);
            CameraUpdateDTO dto = new CameraUpdateDTO();
            dto.setName("Ghost");

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> cameraService.updateCamera("NONEXISTENT", dto));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
            verify(cameraMapper, never()).updateById(isA(DormCamera.class));
        }
    }

    // ──────────────────────────────────────────────
    // getCameraStatus
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getCameraStatus()")
    class GetCameraStatus {

        @Test
        @DisplayName("returns status summary for specific building")
        void shouldReturnStatusForBuilding() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 1L, "ONLINE");
            DormCamera cam3 = buildCamera("CAM-C", 1L, "OFFLINE");
            DormCamera cam4 = buildCamera("CAM-D", 1L, "IDLE");
            when(cameraMapper.selectList(any(LambdaQueryWrapper.class)))
                    .thenReturn(List.of(cam1, cam2, cam3, cam4));

            Map<String, Object> result = cameraService.getCameraStatus("1");

            assertNotNull(result);
            assertTrue(result.containsKey("cameras"));
            assertTrue(result.containsKey("summary"));

            @SuppressWarnings("unchecked")
            Map<String, Object> summary = (Map<String, Object>) result.get("summary");
            assertEquals(4, summary.get("total"));
            assertEquals(2L, summary.get("online"));
            assertEquals(1L, summary.get("offline"));
            assertEquals(1L, summary.get("idle"));
        }

        @Test
        @DisplayName("returns empty summary when no cameras")
        void shouldReturnEmptyWhenNoCameras() {
            when(cameraMapper.selectList(any(LambdaQueryWrapper.class))).thenReturn(List.of());

            Map<String, Object> result = cameraService.getCameraStatus("1");

            assertNotNull(result);
            @SuppressWarnings("unchecked")
            Map<String, Object> summary = (Map<String, Object>) result.get("summary");
            assertEquals(0, summary.get("total"));
            assertEquals(0L, summary.get("online"));
            assertEquals(0L, summary.get("offline"));
            assertEquals(0L, summary.get("idle"));
        }

        @Test
        @DisplayName("returns all cameras when building is null")
        void shouldReturnAllWhenBuildingNull() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 2L, "OFFLINE");
            when(cameraMapper.selectList(null)).thenReturn(List.of(cam1, cam2));

            Map<String, Object> result = cameraService.getCameraStatus(null);

            assertNotNull(result);
            verify(cameraMapper).selectList(null);
            @SuppressWarnings("unchecked")
            Map<String, Object> summary = (Map<String, Object>) result.get("summary");
            assertEquals(2, summary.get("total"));
        }
    }

    // ──────────────────────────────────────────────
    // getByCameraId
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getByCameraId()")
    class GetByCameraId {

        @Test
        @DisplayName("returns camera when found")
        void shouldReturnCamera() {
            DormCamera expected = new DormCamera();
            expected.setId(1L);
            expected.setCameraId("CAM-001");
            expected.setName("Test Cam");
            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(expected);

            DormCamera result = cameraService.getByCameraId("CAM-001");

            assertNotNull(result);
            assertEquals(1L, result.getId());
            assertEquals("Test Cam", result.getName());
        }

        @Test
        @DisplayName("throws NOT_FOUND when camera does not exist")
        void shouldThrowWhenNotFound() {
            when(cameraMapper.findByCameraId("GHOST")).thenReturn(null);

            BusinessException ex = assertThrows(BusinessException.class,
                    () -> cameraService.getByCameraId("GHOST"));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
        }
    }

    // ──────────────────────────────────────────────
    // getCameras
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getCameras()")
    class GetCameras {

        @Test
        @DisplayName("returns cameras for given buildingId")
        void shouldReturnCamerasForBuilding() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 1L, "OFFLINE");
            when(cameraMapper.findByBuildingId(1L)).thenReturn(List.of(cam1, cam2));

            List<DormCamera> result = cameraService.getCameras(1L);

            assertEquals(2, result.size());
            verify(cameraMapper).findByBuildingId(1L);
        }

        @Test
        @DisplayName("returns all cameras when buildingId is null")
        void shouldReturnAllWhenBuildingNull() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 2L, "ONLINE");
            when(cameraMapper.selectList(null)).thenReturn(List.of(cam1, cam2));

            List<DormCamera> result = cameraService.getCameras(null);

            assertEquals(2, result.size());
            verify(cameraMapper).selectList(null);
        }

        @Test
        @DisplayName("returns empty list when no cameras")
        void shouldReturnEmptyWhenNoCameras() {
            when(cameraMapper.findByBuildingId(999L)).thenReturn(List.of());

            List<DormCamera> result = cameraService.getCameras(999L);

            assertTrue(result.isEmpty());
        }
    }

    // ──────────────────────────────────────────────
    // healthCheck
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("healthCheck()")
    class HealthCheck {

        @Test
        @DisplayName("logs health check without side effects (no-op currently)")
        void shouldLogHealthCheck() {
            assertDoesNotThrow(() -> cameraService.healthCheck("CAM-001"));
        }
    }

    // ──────────────────────────────────────────────
    // listEnabledCameras
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("listEnabledCameras()")
    class ListEnabledCameras {

        @Test
        @DisplayName("returns only enabled cameras")
        void shouldReturnEnabledCameras() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 2L, "OFFLINE");
            when(cameraMapper.findEnabledCameras()).thenReturn(List.of(cam1, cam2));

            List<DormCamera> result = cameraService.listEnabledCameras();

            assertEquals(2, result.size());
            verify(cameraMapper).findEnabledCameras();
        }

        @Test
        @DisplayName("returns empty list when no enabled cameras")
        void shouldReturnEmptyWhenNoneEnabled() {
            when(cameraMapper.findEnabledCameras()).thenReturn(List.of());

            List<DormCamera> result = cameraService.listEnabledCameras();

            assertTrue(result.isEmpty());
        }
    }

    // ──────────────────────────────────────────────
    // listOnlineCameras
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("listOnlineCameras()")
    class ListOnlineCameras {

        @Test
        @DisplayName("returns only online enabled cameras")
        void shouldReturnOnlineCameras() {
            DormCamera cam1 = buildCamera("CAM-A", 1L, "ONLINE");
            DormCamera cam2 = buildCamera("CAM-B", 2L, "ONLINE");
            when(cameraMapper.selectList(any(LambdaQueryWrapper.class)))
                    .thenReturn(List.of(cam1, cam2));

            List<DormCamera> result = cameraService.listOnlineCameras();

            assertEquals(2, result.size());
        }

        @Test
        @DisplayName("returns empty list when no online cameras")
        void shouldReturnEmptyWhenNoneOnline() {
            when(cameraMapper.selectList(any(LambdaQueryWrapper.class)))
                    .thenReturn(List.of());

            List<DormCamera> result = cameraService.listOnlineCameras();

            assertTrue(result.isEmpty());
        }
    }

    // ──────────────────────────────────────────────
    // updateLastEventTime
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("updateLastEventTime()")
    class UpdateLastEventTime {

        @Test
        @DisplayName("updates with provided timestamp")
        void shouldUpdateWithTimestamp() {
            DormCamera existing = new DormCamera();
            existing.setId(1L);
            existing.setCameraId("CAM-001");
            when(cameraMapper.findByCameraId("CAM-001")).thenReturn(existing);

            cameraService.updateLastEventTime("CAM-001", 1700000000000L);

            verify(cameraMapper).updateById(isA(DormCamera.class));
        }

        @Test
        @DisplayName("uses current time when timestamp is null")
        void shouldUseCurrentTimeWhenNull() {
            DormCamera existing = new DormCamera();
            existing.setId(2L);
            existing.setCameraId("CAM-002");
            when(cameraMapper.findByCameraId("CAM-002")).thenReturn(existing);

            cameraService.updateLastEventTime("CAM-002", null);

            verify(cameraMapper).updateById(isA(DormCamera.class));
        }

        @Test
        @DisplayName("throws NOT_FOUND when camera does not exist")
        void shouldThrowWhenNotFound() {
            when(cameraMapper.findByCameraId("GHOST")).thenReturn(null);

            assertThrows(BusinessException.class,
                    () -> cameraService.updateLastEventTime("GHOST", 1000L));
        }
    }

    // ──────────────────────────────────────────────
    // querySnapshots
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("querySnapshots()")
    class QuerySnapshots {

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns paginated event logs")
        void shouldReturnPaginatedEvents() {
            DormEventLog log1 = new DormEventLog();
            log1.setId(1L);
            log1.setCameraId("CAM-001");
            log1.setTimestamp(LocalDateTime.now());

            DormEventLog log2 = new DormEventLog();
            log2.setId(2L);
            log2.setCameraId("CAM-001");

            Page<DormEventLog> mockPage = new Page<>(1, 10);
            mockPage.setRecords(List.of(log1, log2));
            mockPage.setTotal(2);

            when(eventLogMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            LocalDateTime start = LocalDateTime.now().minusDays(1);
            LocalDateTime end = LocalDateTime.now();

            Page<DormEventLog> result = cameraService.querySnapshots("CAM-001", start, end, 1, 10);

            assertNotNull(result);
            assertEquals(2, result.getRecords().size());
            assertEquals(2, result.getTotal());
            verify(eventLogMapper).selectPage(any(Page.class), any(LambdaQueryWrapper.class));
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns empty page when no events match")
        void shouldReturnEmptyWhenNoEvents() {
            Page<DormEventLog> emptyPage = new Page<>(1, 10);
            emptyPage.setRecords(List.of());
            emptyPage.setTotal(0);

            when(eventLogMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(emptyPage);

            Page<DormEventLog> result = cameraService.querySnapshots("CAM-999", null, null, 1, 10);

            assertTrue(result.getRecords().isEmpty());
            assertEquals(0, result.getTotal());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("works with null time range")
        void shouldWorkWithNullTimeRange() {
            Page<DormEventLog> mockPage = new Page<>(1, 20);
            mockPage.setRecords(List.of(new DormEventLog()));
            mockPage.setTotal(1);

            when(eventLogMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            Page<DormEventLog> result = cameraService.querySnapshots("CAM-001", null, null, 1, 20);

            assertEquals(1, result.getRecords().size());
        }
    }

    // ──────────────────────────────────────────────
    // Helpers
    // ──────────────────────────────────────────────

    private static DormCamera buildCamera(String cameraId, Long buildingId, String status) {
        DormCamera cam = new DormCamera();
        cam.setCameraId(cameraId);
        cam.setBuildingId(buildingId);
        cam.setStatus(status);
        cam.setEnabled(true);
        return cam;
    }
}
