package com.sims.dormitory.common.interceptor;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sims.dormitory.common.response.ApiResponse;
import com.sims.dormitory.common.util.JwtUtils;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import org.springframework.util.StringUtils;
import org.springframework.web.servlet.HandlerInterceptor;

@Component
@Slf4j
public class DormitoryAuthInterceptor implements HandlerInterceptor {

    private static final ObjectMapper mapper = new ObjectMapper();

    @Override
    public boolean preHandle(HttpServletRequest request, HttpServletResponse response, Object handler) throws Exception {
        // Skip OPTIONS preflight
        if ("OPTIONS".equalsIgnoreCase(request.getMethod())) {
            return true;
        }

        String uri = request.getRequestURI();
        log.debug("Auth check: {}", uri);

        String authHeader = request.getHeader("Authorization");
        if (!StringUtils.hasLength(authHeader) || !authHeader.startsWith("Bearer ")) {
            log.warn("Missing or malformed Authorization header");
            response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
            response.setContentType("application/json;charset=UTF-8");
            ApiResponse<?> error = ApiResponse.error(401, "NOT_LOGIN");
            response.getWriter().write(mapper.writeValueAsString(error));
            return false;
        }

        String jwt = authHeader.substring(7);
        try {
            JwtUtils.parseJWT(jwt);
            log.debug("JWT valid: {}", uri);
            return true;
        } catch (Exception e) {
            log.warn("Invalid JWT: {}", e.getMessage());
            response.setStatus(HttpServletResponse.SC_UNAUTHORIZED);
            response.setContentType("application/json;charset=UTF-8");
            ApiResponse<?> error = ApiResponse.error(401, "NOT_LOGIN");
            response.getWriter().write(mapper.writeValueAsString(error));
            return false;
        }
    }
}
