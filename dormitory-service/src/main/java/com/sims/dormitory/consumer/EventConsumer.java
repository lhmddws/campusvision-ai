package com.sims.dormitory.consumer;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sims.dormitory.common.constant.Constants;
import com.sims.dormitory.common.constant.RedisKeys;
import com.sims.dormitory.model.dto.FaceEventMessage;
import com.sims.dormitory.model.entity.DormAlert;
import com.sims.dormitory.model.entity.DormBuilding;
import com.sims.dormitory.model.entity.DormEventLog;
import com.sims.dormitory.model.entity.DormStrangerRecord;
import com.sims.dormitory.model.entity.DormStudent;
import com.sims.dormitory.model.enums.AlertType;
import com.sims.dormitory.model.enums.StudentStatus;
import com.sims.dormitory.repository.DormAlertMapper;
import com.sims.dormitory.repository.DormBuildingMapper;
import com.sims.dormitory.repository.DormEventLogMapper;
import com.sims.dormitory.repository.DormStrangerRecordMapper;
import com.sims.dormitory.repository.DormStudentMapper;
import com.sims.dormitory.service.CameraService;
import org.apache.kafka.clients.consumer.ConsumerRecord;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.stereotype.Component;

import java.time.LocalDateTime;
import java.util.List;
import java.time.ZoneOffset;
import java.util.concurrent.TimeUnit;

@Component
public class EventConsumer {

    private static final Logger log = LoggerFactory.getLogger(EventConsumer.class);

    private final ObjectMapper objectMapper;
    private final StringRedisTemplate stringRedisTemplate;
    private final DormStudentMapper studentMapper;
    private final DormEventLogMapper eventLogMapper;
    private final DormAlertMapper alertMapper;
    private final DormStrangerRecordMapper strangerRecordMapper;
    private final DormBuildingMapper buildingMapper;
    private final CameraService cameraService;

    public EventConsumer(ObjectMapper objectMapper,
                         StringRedisTemplate stringRedisTemplate,
                         DormStudentMapper studentMapper,
                         DormEventLogMapper eventLogMapper,
                         DormAlertMapper alertMapper,
                         DormStrangerRecordMapper strangerRecordMapper,
                         DormBuildingMapper buildingMapper,
                         CameraService cameraService) {
        this.objectMapper = objectMapper;
        this.stringRedisTemplate = stringRedisTemplate;
        this.studentMapper = studentMapper;
        this.eventLogMapper = eventLogMapper;
        this.alertMapper = alertMapper;
        this.strangerRecordMapper = strangerRecordMapper;
        this.buildingMapper = buildingMapper;
        this.cameraService = cameraService;
    }

    @KafkaListener(
            topics = Constants.TOPIC_EVENT,
            groupId = "dormitory-service-group",
            containerFactory = "kafkaListenerContainerFactory"
    )
    public void consumeEvents(List<ConsumerRecord<String, String>> records, Acknowledgment ack) {
        for (ConsumerRecord<String, String> record : records) {
            try {
                processEvent(record.value());
            } catch (Exception e) {
                log.error("Failed to process dorm event: {}", record.value(), e);
            }
        }
        ack.acknowledge();
    }

    private void processEvent(String json) {
        // 1. Deserialize raw JSON from Python face-recognition service
        FaceEventMessage msg;
        try {
            msg = objectMapper.readValue(json, FaceEventMessage.class);
        } catch (Exception e) {
            log.warn("Failed to deserialize event message: {}", json, e);
            return;
        }
        if (msg.getCameraId() == null || msg.getEventType() == null) {
            log.warn("Invalid event message (missing cameraId or eventType): {}", json);
            return;
        }

        // 2. Redis dedup check (TTL-based idempotency guard)
        String dedupKey = RedisKeys.eventProcessed(msg.getCameraId() + ":" + msg.getFrameSequence());
        Boolean firstSeen = stringRedisTemplate.opsForValue()
                .setIfAbsent(dedupKey, "1", Constants.DEDUP_TTL_SECONDS, TimeUnit.SECONDS);
        if (Boolean.FALSE.equals(firstSeen)) {
            log.debug("Duplicate event skipped: cameraId={}, frameSeq={}",
                    msg.getCameraId(), msg.getFrameSequence());
            return;
        }

        // 3. Map building code (A/B/C/D) → building ID
        Long buildingId = resolveBuildingId(msg.getBuilding());
        if (buildingId == null) {
            log.warn("Unknown building code '{}', event dropped: {}", msg.getBuilding(), json);
            return;
        }

        LocalDateTime eventTime = msg.getTimestamp() != null
                ? LocalDateTime.ofEpochSecond(msg.getTimestamp() / 1000, 0, ZoneOffset.ofHours(8))
                : LocalDateTime.now();

        // 4. Update DormStudent IN/OUT status when a known student is matched
        if (msg.getStudentId() != null && !msg.getStudentId().isEmpty()) {
            updateStudentStatus(msg.getStudentId(), msg.getEventType(), eventTime);
        }

        // 5. Persist event log
        DormEventLog eventLog = new DormEventLog();
        eventLog.setCameraId(msg.getCameraId());
        eventLog.setBuildingId(buildingId);
        eventLog.setEventType(msg.getEventType());
        eventLog.setStudentId(msg.getStudentId());
        eventLog.setIsStranger(Boolean.TRUE.equals(msg.getIsStranger()));
        eventLog.setConfidence(msg.getConfidence());
        eventLog.setSnapshotPath(msg.getSnapshotPath() != null ? msg.getSnapshotPath() : "");
        eventLog.setTimestamp(eventTime);
        eventLog.setCreatedAt(LocalDateTime.now());
        eventLogMapper.insert(eventLog);

        // 6. Stranger detection → alert + stranger record
        if (Boolean.TRUE.equals(msg.getIsStranger())) {
            createStrangerAlert(msg, buildingId, eventTime);
        }

        // 7. Update camera last-event timestamp
        try {
            cameraService.updateLastEventTime(msg.getCameraId(), msg.getTimestamp());
        } catch (Exception e) {
            log.warn("Failed to update camera last event time: cameraId={}", msg.getCameraId(), e);
        }
    }

    private Long resolveBuildingId(String buildingCode) {
        if (buildingCode == null || buildingCode.isEmpty()) {
            return null;
        }
        String cacheKey = "dorm:building:code:" + buildingCode;

        // Redis cache hit
        String cachedId = stringRedisTemplate.opsForValue().get(cacheKey);
        if (cachedId != null) {
            return Long.parseLong(cachedId);
        }

        // DB lookup
        DormBuilding building = buildingMapper.findByCode(buildingCode);
        if (building == null) {
            return null;
        }

        // Cache for 1 hour
        stringRedisTemplate.opsForValue().set(cacheKey, String.valueOf(building.getId()), 1, TimeUnit.HOURS);
        return building.getId();
    }

    private void updateStudentStatus(String studentId, String eventType, LocalDateTime eventTime) {
        try {
            DormStudent student = studentMapper.findByStudentId(studentId);
            if (student == null) {
                log.warn("Student not found in dorm_student table: studentId={}", studentId);
                return;
            }

            String newStatus;
            if ("ENTRY".equalsIgnoreCase(eventType)) {
                newStatus = StudentStatus.IN.name();
            } else if ("EXIT".equalsIgnoreCase(eventType)) {
                newStatus = StudentStatus.OUT.name();
            } else {
                log.warn("Unknown event type, skipping status update: {}", eventType);
                return;
            }

            student.setStatus(newStatus);
            student.setLastEventTime(eventTime);
            student.setUpdatedAt(LocalDateTime.now());
            studentMapper.updateById(student);
            log.debug("Student status updated: studentId={} -> {}", studentId, newStatus);
        } catch (Exception e) {
            log.error("Failed to update student status: studentId={}", studentId, e);
        }
    }

    private void createStrangerAlert(FaceEventMessage msg, Long buildingId, LocalDateTime eventTime) {
        try {
            // High-level alert for administrators
            DormAlert alert = new DormAlert();
            alert.setBuildingId(buildingId);
            alert.setAlertType(AlertType.STRANGER_ENTRY.name());
            alert.setMessage("\u964c\u751f\u4eba\u8fdb\u5165\u5bbf\u820d\u697c: " + msg.getBuilding());
            alert.setDetails(String.format(
                    "camera=%s, confidence=%.2f, time=%s",
                    msg.getCameraId(),
                    msg.getConfidence() != null ? msg.getConfidence() : 0.0,
                    eventTime));
            alert.setAcknowledged(false);
            alert.setCreatedAt(LocalDateTime.now());
            alertMapper.insert(alert);

            // Per-detection stranger record for traceability
            DormStrangerRecord record = new DormStrangerRecord();
            record.setBuildingId(buildingId);
            record.setCameraId(msg.getCameraId());
            record.setSnapshotPath(msg.getSnapshotPath() != null ? msg.getSnapshotPath() : "");
            record.setConfidence(msg.getConfidence());
            record.setDetectedAt(eventTime);
            record.setAcknowledged(false);
            record.setCreatedAt(LocalDateTime.now());
            strangerRecordMapper.insert(record);

            log.info("Stranger alert created: buildingId={}, cameraId={}", buildingId, msg.getCameraId());
        } catch (Exception e) {
            log.error("Failed to create stranger alert: buildingId={}", buildingId, e);
        }
    }

    @KafkaListener(
            topics = Constants.TOPIC_ALERT,
            groupId = "dormitory-service-group",
            containerFactory = "kafkaListenerContainerFactory"
    )
    public void consumeAlerts(List<ConsumerRecord<String, String>> records, Acknowledgment ack) {
        for (ConsumerRecord<String, String> record : records) {
            try {
                String value = record.value();
                log.info("Received alert command: {}", value);
                // TODO: implement alert command routing
                // Expected format example:
                //   {"action":"acknowledge","alertId":123}
                //   {"action":"notify","type":"STRANGER_ENTRY","buildingId":1}
                // Dispatch:
                //   acknowledge → dormitoryAlertService.acknowledgeAlert(id, "system")
                //   notify       → push notification / WebSocket event
            } catch (Exception e) {
                log.error("Failed to process alert command: {}", record.value(), e);
            }
        }
        ack.acknowledge();
    }
}
