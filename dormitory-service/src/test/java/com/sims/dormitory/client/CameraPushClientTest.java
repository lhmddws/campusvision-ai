package com.sims.dormitory.client;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpMethod;
import org.springframework.http.MediaType;
import org.springframework.test.web.client.MockRestServiceServer;
import org.springframework.web.client.RestTemplate;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;
import static org.springframework.test.web.client.match.MockRestRequestMatchers.*;
import static org.springframework.test.web.client.response.MockRestResponseCreators.*;

@ExtendWith(MockitoExtension.class)
@DisplayName("CameraPushClient Tests")
class CameraPushClientTest {

    private RestTemplate restTemplate;
    private MockRestServiceServer mockServer;
    private CameraPushClient pushClient;

    @BeforeEach
    void setUp() {
        restTemplate = new RestTemplate();
        mockServer = MockRestServiceServer.bindTo(restTemplate).build();
        pushClient = new CameraPushClient(restTemplate, "http://127.0.0.1:8081", "test-key");
    }

    @Test
    @DisplayName("notifyRegister sends POST with correct headers and body")
    void notifyRegisterSendsPostRequest() {
        mockServer.expect(requestTo("http://127.0.0.1:8081/cameras"))
            .andExpect(method(HttpMethod.POST))
            .andExpect(header("X-Management-Key", "test-key"))
            .andExpect(header("Content-Type", "application/json"))
            .andRespond(withSuccess("{\"status\":\"added\"}", MediaType.APPLICATION_JSON));

        Map<String, String> config = Map.of("id", "cam-a", "building", "A");
        assertDoesNotThrow(() -> pushClient.notifyRegister(config));
        mockServer.verify();
    }

    @Test
    @DisplayName("notifyDelete sends DELETE with correct cameraId")
    void notifyDeleteSendsDeleteRequest() {
        mockServer.expect(requestTo("http://127.0.0.1:8081/cameras/cam-a"))
            .andExpect(method(HttpMethod.DELETE))
            .andExpect(header("X-Management-Key", "test-key"))
            .andRespond(withSuccess());

        assertDoesNotThrow(() -> pushClient.notifyDelete("cam-a"));
        mockServer.verify();
    }

    @Test
    @DisplayName("push failure does NOT throw exception")
    void pushFailureDoesNotThrow() {
        mockServer.expect(requestTo("http://127.0.0.1:8081/cameras"))
            .andExpect(method(HttpMethod.POST))
            .andRespond(withServerError());

        Map<String, String> config = Map.of("id", "cam-a", "building", "A");
        assertDoesNotThrow(() -> pushClient.notifyRegister(config));
        mockServer.verify();
    }

    @Test
    @DisplayName("notifyUpdate sends POST with correct config")
    void notifyUpdateSendsPostRequest() {
        mockServer.expect(requestTo("http://127.0.0.1:8081/cameras"))
            .andExpect(method(HttpMethod.POST))
            .andExpect(header("X-Management-Key", "test-key"))
            .andExpect(header("Content-Type", "application/json"))
            .andRespond(withSuccess());

        Map<String, String> config = Map.of("id", "cam-a", "building", "A");
        assertDoesNotThrow(() -> pushClient.notifyUpdate("cam-a", config));
        mockServer.verify();
    }

    @Test
    @DisplayName("works without management key")
    void worksWithoutKey() {
        RestTemplate rt = new RestTemplate();
        MockRestServiceServer mss = MockRestServiceServer.bindTo(rt).build();
        CameraPushClient client = new CameraPushClient(rt, "http://127.0.0.1:8081", "");

        mss.expect(requestTo("http://127.0.0.1:8081/cameras/cam-a"))
            .andExpect(method(HttpMethod.DELETE))
            .andExpect(headerDoesNotExist("X-Management-Key"))
            .andRespond(withSuccess());

        assertDoesNotThrow(() -> client.notifyDelete("cam-a"));
        mss.verify();
    }
}
