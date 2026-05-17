package com.sims.dormitory.service.impl;

import com.sims.dormitory.model.dto.AttendanceStatsDTO;
import com.sims.dormitory.model.dto.DailySummaryDTO;
import com.sims.dormitory.model.entity.DormAttendanceRecord;
import com.sims.dormitory.repository.DormAttendanceRecordMapper;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
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
@DisplayName("DormitoryRecordServiceImpl Unit Tests")
class DormitoryRecordServiceImplTest {

    @Mock
    private DormAttendanceRecordMapper attendanceRecordMapper;

    @InjectMocks
    private DormitoryRecordServiceImpl recordService;

    // ──────────────────────────────────────────────
    // handleAttendance
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("handleAttendance()")
    class HandleAttendance {

        @Test
        @DisplayName("inserts attendance record successfully")
        void shouldInsertRecord() {
            // Arrange
            DormAttendanceRecord record = new DormAttendanceRecord();
            record.setStudentId(1001L);
            record.setBuildingId(1L);
            record.setRoomId(101L);
            record.setDate(LocalDate.of(2026, 5, 17));
            record.setStatus("PRESENT");
            record.setCheckTime(LocalDateTime.now());
            record.setRemark("On time");

            // Act
            recordService.handleAttendance(record);

            // Assert
            verify(attendanceRecordMapper).insert(isA(DormAttendanceRecord.class));
        }

        @Test
        @DisplayName("inserts attendance record with ABSENT status")
        void shouldInsertAbsentRecord() {
            // Arrange
            DormAttendanceRecord record = new DormAttendanceRecord();
            record.setStudentId(1002L);
            record.setBuildingId(1L);
            record.setDate(LocalDate.of(2026, 5, 17));
            record.setStatus("ABSENT");

            // Act
            recordService.handleAttendance(record);

            // Assert
            verify(attendanceRecordMapper).insert(isA(DormAttendanceRecord.class));
        }

        @Test
        @DisplayName("inserts attendance record with LATE status")
        void shouldInsertLateRecord() {
            // Arrange
            DormAttendanceRecord record = new DormAttendanceRecord();
            record.setStudentId(1003L);
            record.setBuildingId(2L);
            record.setDate(LocalDate.of(2026, 5, 17));
            record.setStatus("LATE");
            record.setRemark("Returned 30 min late");

            // Act
            recordService.handleAttendance(record);

            // Assert
            verify(attendanceRecordMapper).insert(isA(DormAttendanceRecord.class));
        }

        @Test
        @DisplayName("handles record with minimal fields")
        void shouldHandleMinimalRecord() {
            // Arrange
            DormAttendanceRecord record = new DormAttendanceRecord();
            record.setStudentId(1004L);
            record.setStatus("PRESENT");

            // Act
            recordService.handleAttendance(record);

            // Assert
            verify(attendanceRecordMapper).insert(isA(DormAttendanceRecord.class));
        }
    }

    // ──────────────────────────────────────────────
    // getAttendanceStats
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getAttendanceStats()")
    class GetAttendanceStats {

        @Test
        @DisplayName("returns default stats (stub implementation)")
        void shouldReturnDefaultStats() {
            // Act
            AttendanceStatsDTO result = recordService.getAttendanceStats(1L,
                    LocalDate.of(2026, 5, 1),
                    LocalDate.of(2026, 5, 17));

            // Assert
            assertNotNull(result);
            assertEquals(0, result.getTotal());
            assertEquals(0, result.getPresent());
            assertEquals(0, result.getAbsent());
            assertEquals(0, result.getLate());
            assertEquals(0, result.getStranger());
            assertEquals(0.0, result.getRate(), 0.001);
            // Stub implementation always returns zeros
            verify(attendanceRecordMapper, never()).insert(isA(DormAttendanceRecord.class));
        }

        @Test
        @DisplayName("handles null buildingId")
        void shouldHandleNullBuilding() {
            // Act
            AttendanceStatsDTO result = recordService.getAttendanceStats(null,
                    LocalDate.of(2026, 5, 1), null);

            // Assert
            assertNotNull(result);
        }

        @Test
        @DisplayName("handles null date range")
        void shouldHandleNullDateRange() {
            // Act
            AttendanceStatsDTO result = recordService.getAttendanceStats(1L, null, null);

            // Assert
            assertNotNull(result);
        }
    }

    // ──────────────────────────────────────────────
    // getDailySummary
    // ──────────────────────────────────────────────

    @Nested
    @DisplayName("getDailySummary()")
    class GetDailySummary {

        @Test
        @DisplayName("returns empty list (stub implementation)")
        void shouldReturnEmptyList() {
            // Act
            List<DailySummaryDTO> result = recordService.getDailySummary(1L,
                    LocalDate.of(2026, 5, 1),
                    LocalDate.of(2026, 5, 17));

            // Assert
            assertNotNull(result);
            assertTrue(result.isEmpty());
        }

        @Test
        @DisplayName("handles null buildingId")
        void shouldHandleNullBuilding() {
            // Act
            List<DailySummaryDTO> result = recordService.getDailySummary(null,
                    LocalDate.of(2026, 5, 1),
                    LocalDate.of(2026, 5, 17));

            // Assert
            assertNotNull(result);
            assertTrue(result.isEmpty());
        }

        @Test
        @DisplayName("handles null date range")
        void shouldHandleNullDateRange() {
            // Act
            List<DailySummaryDTO> result = recordService.getDailySummary(1L, null, null);

            // Assert
            assertNotNull(result);
            assertTrue(result.isEmpty());
        }
    }
}
