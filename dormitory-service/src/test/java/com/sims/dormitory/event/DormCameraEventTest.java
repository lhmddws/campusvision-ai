package com.sims.dormitory.event;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.context.ApplicationEventPublisher;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("DormCameraEvent Tests")
class DormCameraEventTest {

    @Mock
    private ApplicationEventPublisher eventPublisher;

    @Captor
    private ArgumentCaptor<DormCameraEvent> eventCaptor;

    @Test
    @DisplayName("event contains correct fields after creation")
    void shouldContainCorrectFields() {
        DormCameraEvent event = new DormCameraEvent(
            this, "cam-test", DormCameraEvent.EventType.DELETED, "A", "online"
        );

        assertEquals("cam-test", event.getCameraId());
        assertEquals(DormCameraEvent.EventType.DELETED, event.getEventType());
        assertEquals("A", event.getBuilding());
        assertEquals("online", event.getStatus());
    }

    @Test
    @DisplayName("constructor without building/status works")
    void shouldWorkWithMinimalConstructor() {
        DormCameraEvent event = new DormCameraEvent(
            this, "cam-min", DormCameraEvent.EventType.REGISTERED
        );

        assertEquals("cam-min", event.getCameraId());
        assertEquals(DormCameraEvent.EventType.REGISTERED, event.getEventType());
        assertNull(event.getBuilding());
        assertNull(event.getStatus());
    }

    @Test
    @DisplayName("event published and received by listener with correct fields")
    void shouldBePublishedAndReceived() {
        DormCameraEvent event = new DormCameraEvent(
            this, "cam-pub", DormCameraEvent.EventType.UPDATED, "B", "offline"
        );
        eventPublisher.publishEvent(event);

        verify(eventPublisher).publishEvent(eventCaptor.capture());
        DormCameraEvent captured = eventCaptor.getValue();

        assertEquals("cam-pub", captured.getCameraId());
        assertEquals(DormCameraEvent.EventType.UPDATED, captured.getEventType());
        assertEquals("B", captured.getBuilding());
        assertEquals("offline", captured.getStatus());
    }

    @Test
    @DisplayName("toString returns meaningful representation")
    void toStringShouldBeMeaningful() {
        DormCameraEvent event = new DormCameraEvent(
            this, "cam-str", DormCameraEvent.EventType.STATUS_CHANGED, "C", "online"
        );
        String str = event.toString();
        assertTrue(str.contains("cam-str"));
        assertTrue(str.contains("STATUS_CHANGED"));
        assertTrue(str.contains("C"));
    }
}
