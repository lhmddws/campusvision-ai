package com.sims.dormitory.common.exception;

import com.sims.dormitory.common.response.ErrorCode;
import lombok.Getter;

import java.util.HashMap;
import java.util.Map;

@Getter
public class BusinessException extends RuntimeException {

    private final int code;
    private final String errorMessage;
    private final Map<String, Object> details;

    public BusinessException(ErrorCode errorCode) {
        super(errorCode.getMessage());
        this.code = errorCode.getCode();
        this.errorMessage = errorCode.getMessage();
        this.details = new HashMap<>();
    }

    public BusinessException(ErrorCode errorCode, Map<String, Object> details) {
        super(errorCode.getMessage());
        this.code = errorCode.getCode();
        this.errorMessage = errorCode.getMessage();
        this.details = details;
    }

    public BusinessException(int code, String message) {
        super(message);
        this.code = code;
        this.errorMessage = message;
        this.details = new HashMap<>();
    }
}
