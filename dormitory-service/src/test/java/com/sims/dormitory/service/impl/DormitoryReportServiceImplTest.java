package com.sims.dormitory.service.impl;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.sims.dormitory.model.dto.NightlyReportDTO;
import com.sims.dormitory.model.entity.DormNightlyReport;
import com.sims.dormitory.repository.DormNightlyReportMapper;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.LocalDate;
import java.time.LocalDateTime;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("DormitoryReportServiceImpl Unit Tests")
class DormitoryReportServiceImplTest {

    @Mock
    private DormNightlyReportMapper nightlyReportMapper;

    @InjectMocks
    private DormitoryReportServiceImpl reportService;

    @Captor
    private ArgumentCaptor<DormNightlyReport> reportCaptor;

    @Captor
    private ArgumentCaptor<LambdaQueryWrapper<DormNightlyReport>> wrapperCaptor;

    // ──────────────────────────────────────────────
    // generateNightlyReport
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("generateNightlyReport()")
    class GenerateNightlyReport {

        @Test
        @DisplayName("generates report with zero counts (stub implementation)")
        void shouldGenerateReport() {
            // Arrange
            when(nightlyReportMapper.insert(any(DormNightlyReport.class))).thenReturn(1);

            // Act
            DormNightlyReport result = reportService.generateNightlyReport(1L,
                    LocalDate.of(2026, 5, 17));

            // Assert
            assertNotNull(result);
            assertEquals(1L, result.getBuildingId());
            assertEquals(LocalDate.of(2026, 5, 17), result.getReportDate());
            assertEquals(0, result.getTotalStudents());
            assertEquals(0, result.getPresentCount());
            assertEquals(0, result.getAbsentCount());
            assertEquals(0, result.getLateCount());
            assertEquals(0, result.getStrangerCount());
            assertNotNull(result.getGeneratedAt());

            verify(nightlyReportMapper).insert(reportCaptor.capture());
            DormNightlyReport captured = reportCaptor.getValue();
            assertEquals(1L, captured.getBuildingId());
            assertEquals(0, captured.getTotalStudents());
        }

        @Test
        @DisplayName("generates report for different building and date")
        void shouldGenerateReportForDifferentBuilding() {
            // Arrange
            when(nightlyReportMapper.insert(any(DormNightlyReport.class))).thenReturn(1);

            // Act
            DormNightlyReport result = reportService.generateNightlyReport(2L,
                    LocalDate.of(2026, 5, 16));

            // Assert
            assertEquals(2L, result.getBuildingId());
            assertEquals(LocalDate.of(2026, 5, 16), result.getReportDate());
        }
    }

    // ──────────────────────────────────────────────
    // getNightlyReport
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getNightlyReport()")
    class GetNightlyReport {

        @Test
        @DisplayName("returns report DTO when found")
        void shouldReturnReport() {
            // Arrange
            LocalDate reportDate = LocalDate.of(2026, 5, 17);
            DormNightlyReport report = buildReport(1L, 1L, reportDate, 100, 95, 3, 1, 1);
            when(nightlyReportMapper.findByBuildingAndDate(1L, reportDate)).thenReturn(report);

            // Act
            NightlyReportDTO result = reportService.getNightlyReport(1L, reportDate);

            // Assert
            assertNotNull(result);
            assertEquals(1L, result.getId());
            assertEquals(1L, result.getBuildingId());
            assertEquals(reportDate, result.getReportDate());
            assertEquals(100, result.getTotalStudents());
            assertEquals(95, result.getPresentCount());
            assertEquals(3, result.getAbsentCount());
            assertEquals(1, result.getLateCount());
            assertEquals(1, result.getStrangerCount());
            assertNotNull(result.getGeneratedAt());
        }

        @Test
        @DisplayName("returns null when report not found")
        void shouldReturnNullWhenNotFound() {
            // Arrange
            when(nightlyReportMapper.findByBuildingAndDate(1L, LocalDate.of(2026, 5, 17)))
                    .thenReturn(null);

            // Act
            NightlyReportDTO result = reportService.getNightlyReport(1L,
                    LocalDate.of(2026, 5, 17));

            // Assert
            assertNull(result);
        }

        @Test
        @DisplayName("returns null for non-existent building-date combo")
        void shouldReturnNullForNonexistent() {
            // Arrange
            when(nightlyReportMapper.findByBuildingAndDate(999L, LocalDate.of(2026, 1, 1)))
                    .thenReturn(null);

            // Act
            NightlyReportDTO result = reportService.getNightlyReport(999L,
                    LocalDate.of(2026, 1, 1));

            // Assert
            assertNull(result);
        }
    }

    // ──────────────────────────────────────────────
    // getReportHistory
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getReportHistory()")
    class GetReportHistory {

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns paginated report history without filters")
        void shouldReturnPaginatedHistory() {
            // Arrange
            DormNightlyReport r1 = buildReport(1L, 1L, LocalDate.of(2026, 5, 17), 100, 95, 3, 1, 1);
            DormNightlyReport r2 = buildReport(2L, 1L, LocalDate.of(2026, 5, 16), 100, 90, 5, 3, 2);

            Page<DormNightlyReport> mockPage = new Page<>(1, 20);
            mockPage.setRecords(List.of(r1, r2));
            mockPage.setTotal(2);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(null, null, null, 1, 20);

            // Assert
            assertNotNull(result);
            assertEquals(2, result.getRecords().size());
            assertEquals(2, result.getTotal());
            assertEquals(1, result.getCurrent());

            NightlyReportDTO dto = result.getRecords().get(0);
            assertEquals(1L, dto.getId());
            assertEquals(1L, dto.getBuildingId());
            assertEquals(95, dto.getPresentCount());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("filters by buildingId and date range")
        void shouldApplyFilters() {
            // Arrange
            DormNightlyReport report = buildReport(1L, 2L, LocalDate.of(2026, 5, 17), 50, 48, 1, 1, 0);
            Page<DormNightlyReport> mockPage = new Page<>(1, 10);
            mockPage.setRecords(List.of(report));
            mockPage.setTotal(1);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            LocalDate start = LocalDate.of(2026, 5, 1);
            LocalDate end = LocalDate.of(2026, 5, 31);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(2L, start, end, 1, 10);

            // Assert
            assertEquals(1, result.getRecords().size());
            assertEquals(2L, result.getRecords().get(0).getBuildingId());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("returns empty page when no reports match")
        void shouldReturnEmptyWhenNoReports() {
            // Arrange
            Page<DormNightlyReport> emptyPage = new Page<>(1, 20);
            emptyPage.setRecords(List.of());
            emptyPage.setTotal(0);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(emptyPage);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(999L,
                    LocalDate.of(2020, 1, 1), LocalDate.of(2020, 12, 31), 1, 20);

            // Assert
            assertTrue(result.getRecords().isEmpty());
            assertEquals(0, result.getTotal());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("honors pagination parameters")
        void shouldHonorPagination() {
            // Arrange
            DormNightlyReport report = buildReport(1L, 1L, LocalDate.of(2026, 5, 17), 50, 48, 1, 1, 0);
            Page<DormNightlyReport> mockPage = new Page<>(2, 5);
            mockPage.setRecords(List.of(report));
            mockPage.setTotal(6);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(1L, null, null, 2, 5);

            // Assert
            assertEquals(2, result.getCurrent());
            assertEquals(5, result.getSize());
            assertEquals(1, result.getRecords().size());
            assertEquals(6, result.getTotal());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("works with only startDate filter")
        void shouldWorkWithStartDateOnly() {
            // Arrange
            Page<DormNightlyReport> mockPage = new Page<>(1, 20);
            mockPage.setRecords(List.of(buildReport(1L, 1L, LocalDate.of(2026, 5, 17), 50, 48, 1, 1, 0)));
            mockPage.setTotal(1);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(null,
                    LocalDate.of(2026, 5, 1), null, 1, 20);

            // Assert
            assertEquals(1, result.getRecords().size());
        }

        @SuppressWarnings("unchecked")
        @Test
        @DisplayName("works with only endDate filter")
        void shouldWorkWithEndDateOnly() {
            // Arrange
            Page<DormNightlyReport> mockPage = new Page<>(1, 20);
            mockPage.setRecords(List.of(buildReport(1L, 1L, LocalDate.of(2026, 5, 17), 50, 48, 1, 1, 0)));
            mockPage.setTotal(1);

            when(nightlyReportMapper.selectPage(any(Page.class), any(LambdaQueryWrapper.class)))
                    .thenReturn(mockPage);

            // Act
            Page<NightlyReportDTO> result = reportService.getReportHistory(null,
                    null, LocalDate.of(2026, 5, 31), 1, 20);

            // Assert
            assertEquals(1, result.getRecords().size());
        }
    }

    // ──────────────────────────────────────────────
    // Helpers
    // ──────────────────────────────────────────────

    private static DormNightlyReport buildReport(Long id, Long buildingId, LocalDate date,
                                                  Integer total, Integer present, Integer absent,
                                                  Integer late, Integer stranger) {
        DormNightlyReport report = new DormNightlyReport();
        report.setId(id);
        report.setBuildingId(buildingId);
        report.setReportDate(date);
        report.setTotalStudents(total);
        report.setPresentCount(present);
        report.setAbsentCount(absent);
        report.setLateCount(late);
        report.setStrangerCount(stranger);
        report.setGeneratedAt(LocalDateTime.now());
        return report;
    }
}
