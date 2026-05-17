package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.common.exception.BusinessException;
import com.sims.dormitory.common.response.ErrorCode;
import com.sims.dormitory.model.dto.AlertDTO;
import com.sims.dormitory.model.entity.DormAlert;
import com.sims.dormitory.repository.DormAlertMapper;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDateTime;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("DormitoryAlertServiceImpl Unit Tests")
class DormitoryAlertServiceImplTest {

    @Mock
    private DormAlertMapper alertMapper;

    @InjectMocks
    private DormitoryAlertServiceImpl alertService;

    // ──────────────────────────────────────────────
    // createAlert
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("createAlert()")
    class CreateAlert {

        @Test
        @DisplayName("creates alert with default values")
        void shouldCreateAlert() {
            // Arrange
            DormAlert input = new DormAlert();
            input.setAlertType("STRANGER");
            input.setBuildingId(1L);
            input.setMessage("Unknown person detected");
            input.setDetails("details about the event");

            when(alertMapper.insert(any(DormAlert.class))).thenReturn(1);

            // Act
            DormAlert result = alertService.createAlert(input);

            // Assert
            assertNotNull(result);
            assertEquals("STRANGER", result.getAlertType());
            assertEquals(1L, result.getBuildingId());
            assertEquals("Unknown person detected", result.getMessage());
            assertFalse(result.getAcknowledged());
            assertNotNull(result.getCreatedAt());

            verify(alertMapper).insert(isA(DormAlert.class));
        }

        @Test
        @DisplayName("creates alert with different alert types")
        void shouldCreateAlertWithDifferentTypes() {
            // Arrange
            DormAlert input = new DormAlert();
            input.setAlertType("LATE_RETURN");
            input.setBuildingId(2L);
            input.setMessage("Student returned after curfew");

            // Act
            alertService.createAlert(input);

            // Assert
            verify(alertMapper).insert(isA(DormAlert.class));
        }
    }

    // ──────────────────────────────────────────────
    // acknowledgeAlert
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("acknowledgeAlert()")
    class AcknowledgeAlert {

        @Test
        @DisplayName("acknowledges alert when found")
        void shouldAcknowledgeAlert() {
            // Arrange
            DormAlert alert = new DormAlert();
            alert.setId(1L);
            alert.setAcknowledged(false);
            alert.setAlertType("STRANGER");
            when(alertMapper.selectById(1L)).thenReturn(alert);

            // Act
            alertService.acknowledgeAlert(1L, "admin");

            // Assert
            verify(alertMapper).updateById(isA(DormAlert.class));
        }

        @Test
        @DisplayName("throws NOT_FOUND when alert does not exist")
        void shouldThrowWhenNotFound() {
            // Arrange
            when(alertMapper.selectById(999L)).thenReturn(null);

            // Act & Assert
            BusinessException ex = assertThrows(BusinessException.class,
                    () -> alertService.acknowledgeAlert(999L, "admin"));
            assertEquals(ErrorCode.NOT_FOUND.getCode(), ex.getCode());
            verify(alertMapper, never()).updateById(isA(DormAlert.class));
        }

        @Test
        @DisplayName("throws NOT_FOUND with null id")
        void shouldThrowWhenIdIsNull() {
            // Arrange
            when(alertMapper.selectById(null)).thenReturn(null);

            // Act & Assert
            assertThrows(BusinessException.class,
                    () -> alertService.acknowledgeAlert(null, "admin"));
        }
    }

    // ──────────────────────────────────────────────
    // getAlerts
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getAlerts()")
    class GetAlerts {

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns paginated alerts with no filters")
        void shouldReturnPaginatedAlerts() {
            // Arrange
            DormAlert alert1 = buildAlert(1L, 1L, "STRANGER", false);
            DormAlert alert2 = buildAlert(2L, 1L, "LATE_RETURN", true);

            Page<DormAlert> mockPage = new Page<>(1, 20);
            mockPage.setRecords(List.of(alert1, alert2));
            mockPage.setTotal(2);

            when(alertMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<AlertDTO> result = alertService.getAlerts(null, null, null, 1, 20);

            // Assert
            assertNotNull(result);
            assertEquals(2, result.getRecords().size());
            assertEquals(2, result.getTotal());
            assertEquals(1, result.getCurrent());

            AlertDTO dto1 = result.getRecords().get(0);
            assertEquals(1L, dto1.getId());
            assertEquals("STRANGER", dto1.getAlertType());
            assertFalse(dto1.getAcknowledged());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("filters by buildingId, alertType, and acknowledged")
        void shouldApplyFilters() {
            // Arrange
            DormAlert alert = buildAlert(1L, 2L, "ABSENCE", false);
            Page<DormAlert> mockPage = new Page<>(1, 10);
            mockPage.setRecords(List.of(alert));
            mockPage.setTotal(1);

            when(alertMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<AlertDTO> result = alertService.getAlerts(2L, "ABSENCE", false, 1, 10);

            // Assert
            assertEquals(1, result.getRecords().size());
            assertEquals("ABSENCE", result.getRecords().get(0).getAlertType());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns empty page when no alerts match")
        void shouldReturnEmptyWhenNoAlerts() {
            // Arrange
            Page<DormAlert> emptyPage = new Page<>(1, 20);
            emptyPage.setRecords(List.of());
            emptyPage.setTotal(0);

            when(alertMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(emptyPage);

            // Act
            Page<AlertDTO> result = alertService.getAlerts(999L, null, null, 1, 20);

            // Assert
            assertTrue(result.getRecords().isEmpty());
            assertEquals(0, result.getTotal());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns empty page with custom page size")
        void shouldReturnWithCustomPageSize() {
            // Arrange
            Page<DormAlert> mockPage = new Page<>(1, 50);
            mockPage.setRecords(List.of(buildAlert(1L, 1L, "STRANGER", false)));
            mockPage.setTotal(1);

            when(alertMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<AlertDTO> result = alertService.getAlerts(null, null, null, 1, 50);

            // Assert
            assertEquals(1, result.getRecords().size());
            assertEquals(50, result.getSize());
        }
    }

    // ──────────────────────────────────────────────
    // getAlertCount
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getAlertCount()")
    class GetAlertCount {

        @Test
        @DisplayName("returns count with no filters")
        void shouldReturnTotalCount() {
            // Arrange
            when(alertMapper.selectCount(any(LambdaQueryWrapper.class))).thenReturn(10L);

            // Act
            long count = alertService.getAlertCount(null, null);

            // Assert
            assertEquals(10L, count);
        }

        @Test
        @DisplayName("returns filtered count by building and acknowledged status")
        void shouldReturnFilteredCount() {
            // Arrange
            when(alertMapper.selectCount(any(LambdaQueryWrapper.class))).thenReturn(3L);

            // Act
            long count = alertService.getAlertCount(1L, true);

            // Assert
            assertEquals(3L, count);
        }

        @Test
        @DisplayName("returns zero when no alerts match")
        void shouldReturnZeroWhenNoAlerts() {
            // Arrange
            when(alertMapper.selectCount(any(LambdaQueryWrapper.class))).thenReturn(0L);

            // Act
            long count = alertService.getAlertCount(999L, false);

            // Assert
            assertEquals(0L, count);
        }
    }

    // ──────────────────────────────────────────────
    // Helpers
    // ──────────────────────────────────────────────

    private static DormAlert buildAlert(Long id, Long buildingId, String alertType, Boolean acknowledged) {
        DormAlert alert = new DormAlert();
        alert.setId(id);
        alert.setBuildingId(buildingId);
        alert.setAlertType(alertType);
        alert.setMessage("Test alert: " + alertType);
        alert.setDetails("Details for " + alertType);
        alert.setAcknowledged(acknowledged);
        alert.setCreatedAt(LocalDateTime.now());
        return alert;
    }
}
