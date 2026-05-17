package com.sims.dormitory.common.response;

import lombok.Getter;

@Getter
public enum ErrorCode {

    SUCCESS(0, "success"),
    BAD_REQUEST(400, "bad request"),
    NOT_FOUND(404, "not found"),
    INTERNAL_ERROR(500, "internal server error"),
    CAMERA_OFFLINE(1001, "camera is offline"),
    STUDENT_NOT_FOUND(1002, "student not found"),
    MATCH_FAILED(1003, "face match failed"),
    BUILDING_INVALID(1004, "invalid building"),
    REPORT_ALREADY_EXISTS(1005, "report already exists for this date"),
    CAMERA_LIMIT_EXCEEDED(1006, "camera limit exceeded"),
    INVALID_PARAMETER(1007, "invalid parameter");

    private final int code;
    private final String message;

    ErrorCode(int code, String message) {
        this.code = code;
        this.message = message;
    }
}
