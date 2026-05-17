package com.sims.dormitory.model.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import lombok.ToString;

/**
 * Kafka event message published by the Face Recognition (Python) service.
 * <p>
 * Received from {@code t_dorm_event} topic, produced by
 * {@code face-recognition/app/main.py}.
 */
@Getter
@Setter
@ToString
@NoArgsConstructor
@AllArgsConstructor
public class FaceEventMessage {

    @JsonProperty("camera_id")
    private String cameraId;

    @JsonProperty("building")
    private String building;

    @JsonProperty("event_type")
    private String eventType;

    @JsonProperty("student_id")
    private String studentId;

    @JsonProperty("name")
    private String name;

    @JsonProperty("confidence")
    private Double confidence;

    @JsonProperty("timestamp")
    private Long timestamp;

    @JsonProperty("frame_sequence")
    private Integer frameSequence;

    @JsonProperty("is_stranger")
    private Boolean isStranger;

    @JsonProperty("snapshot_path")
    private String snapshotPath;

    @JsonProperty("direction_method")
    private String directionMethod;
}