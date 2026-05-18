package com.sims.dormitory.client;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Component;
import org.springframework.http.client.SimpleClientHttpRequestFactory;
import org.springframework.web.client.RestTemplate;

@Component
public class CameraPushClient {

    private static final Logger log = LoggerFactory.getLogger(CameraPushClient.class);

    private final RestTemplate restTemplate;
    private final String managementBaseUrl;
    private final String managementKey;

    public CameraPushClient(
            @Value("${camera.management.base-url}") String managementBaseUrl,
            @Value("${camera.management.key:}") String managementKey) {
        var factory = new SimpleClientHttpRequestFactory();
        factory.setConnectTimeout(2 * 1000);
        factory.setReadTimeout(5 * 1000);
        this.restTemplate = new RestTemplate(factory);
        this.managementBaseUrl = managementBaseUrl;
        this.managementKey = managementKey;
    }

    // For testing with custom RestTemplate
    CameraPushClient(RestTemplate restTemplate, String managementBaseUrl, String managementKey) {
        this.restTemplate = restTemplate;
        this.managementBaseUrl = managementBaseUrl;
        this.managementKey = managementKey;
    }

    private HttpHeaders buildHeaders() {
        HttpHeaders headers = new HttpHeaders();
        headers.set("Content-Type", "application/json");
        if (managementKey != null && !managementKey.isEmpty()) {
            headers.set("X-Management-Key", managementKey);
        }
        return headers;
    }

    public void notifyRegister(Object cameraConfig) {
        try {
            HttpEntity<Object> entity = new HttpEntity<>(cameraConfig, buildHeaders());
            ResponseEntity<String> response = restTemplate.exchange(
                managementBaseUrl + "/cameras",
                HttpMethod.POST,
                entity,
                String.class
            );
            log.info("Push register success: camera={}, status={}", cameraConfig, response.getStatusCode());
        } catch (Exception e) {
            log.warn("Push register failed for camera {}: {} — DB poll will pick up changes",
                cameraConfig, e.getMessage());
        }
    }

    public void notifyUpdate(String cameraId, Object cameraConfig) {
        try {
            HttpEntity<Object> entity = new HttpEntity<>(cameraConfig, buildHeaders());
            ResponseEntity<String> response = restTemplate.exchange(
                managementBaseUrl + "/cameras",
                HttpMethod.POST,
                entity,
                String.class
            );
            log.info("Push update success: cameraId={}, status={}", cameraId, response.getStatusCode());
        } catch (Exception e) {
            log.warn("Push update failed for cameraId {}: {} — DB poll will pick up changes",
                cameraId, e.getMessage());
        }
    }

    public void notifyDelete(String cameraId) {
        try {
            HttpEntity<?> entity = new HttpEntity<>(buildHeaders());
            ResponseEntity<String> response = restTemplate.exchange(
                managementBaseUrl + "/cameras/" + cameraId,
                HttpMethod.DELETE,
                entity,
                String.class
            );
            log.info("Push delete success: cameraId={}, status={}", cameraId, response.getStatusCode());
        } catch (Exception e) {
            log.warn("Push delete failed for cameraId {}: {} — DB poll will pick up changes",
                cameraId, e.getMessage());
        }
    }
}
