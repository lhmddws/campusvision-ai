# 摄像头功能实现 — 技术设计

> **文档归属**: 摄像头功能实现 → 技术设计  
> **对应 PRD**: PRD-005 (摄像头功能实现)  
> **版本**: v1.0 · **更新**: 2026-05-15  

---

## 目录

1. [功能概述](#1-功能概述)
2. [架构设计](#2-架构设计)
3. [健康检查设计](#3-健康检查设计)
4. [离线检测与告警](#4-离线检测与告警)
5. [Service 实现](#5-service-实现)
6. [Controller 实现](#6-controller-实现)
7. [数据模型](#7-数据模型)
8. [与感知层的接口约定](#8-与感知层的接口约定)

---

## 1. 功能概述

摄像头模块负责在主服务中管理 4 个宿舍入口摄像头的设备信息、运行状态和配置。

**永远记住**: 真正的拉流/解码/抽帧是感知层（Stream Gateway）做的。本模块只管注册了哪些摄像头、它们现在是否在线、RTSP 地址是什么。

---

## 2. 架构设计

### 2.1 模块位置

```
┌──────────────────────────────────────┐
│  Dormitory Service (SpringBoot)       │
│                                       │
│  ┌──────────────────────────────┐    │
│  │  CameraController            │    │
│  │  /api/dormitory/cameras/*    │    │
│  └──────────┬───────────────────┘    │
│             │                          │
│  ┌──────────▼───────────────────┐    │
│  │  CameraService               │    │
│  │  • CRUD                      │    │
│  │  • 健康检查                    │    │
│  │  • 离线告警                    │    │
│  └──────────┬───────────────────┘    │
│             │                          │
│  ┌──────────▼───────────────────┐    │
│  │  CameraMapper (MyBatis-Plus)  │    │
│  │  CameraLogMapper             │    │
│  └──────────┬───────────────────┘    │
│             │                          │
│  ┌──────────▼───────────────────┐    │
│  │  PostgreSQL: dorm_camera      │    │
│  │  PostgreSQL: dorm_camera_log  │    │
│  └──────────────────────────────┘    │
│                                       │
│  ┌──────────────────────────────┐    │
│  │  CameraHealthCheckTask       │    │
│  │  (@Scheduled, 每30s)         │─────► Stream Gateway health API
│  └──────────────────────────────┘    │
└──────────────────────────────────────┘
```

### 2.2 依赖关系

| 依赖 | 方向 | 说明 |
|------|------|------|
| Stream Gateway health API | 本模块 → 感知层 | 定时拉取摄像头状态 |
| `t_dorm_event` (Kafka) | 本模块消费 | 通过事件时间判断摄像头 idle |
| `dorm_student_assignment` | 本模块查询 | 从 building 关联楼栋名 |
| `dorm_entry_exit_event` | 本模块查询 | 按 camera_id 查抓拍历史 |

---

## 3. 健康检查设计

### 3.1 健康检查流程

```
CameraHealthCheckTask (每 30s)
    │
    ▼
┌────────────────────────────────────┐
│  HTTP GET → Stream Gateway health  │
│  http://stream-gateway:8080/health │
└──────────┬─────────────────────────┘
           │
     ┌─────▼─────┐
     │ 请求成功?   │
     └─────┬─────┘
           │
    ┌──────▼──────┐         ┌──────────────┐
    │ 成功         │         │ 失败          │
    │             │         │              │
    │ 更新摄像头    │         │ failCount++  │
    │ 状态为在线    │         │              │
    │ fps_current  │         │ failCount    │
    │ = 返回值     │         │ ≥ 3?         │──是──► 触发离线告警
    │ last_heart   │         │              │
    │ beat = now   │         │ 更新状态为     │
    │ failCount=0  │         │ offline      │
    └──────┬───────┘         └──────────────┘
           │
           ▼
    ┌──────────────┐
    │ 若状态从 offline  │
    │ 变为 online:    │──► 记录恢复日志
    │ 记录上线日志     │
    └──────────────┘
```

### 3.2 Health Check Task 实现

```java
@Component
@Slf4j
public class CameraHealthCheckTask {

    @Autowired
    private CameraService cameraService;

    private static final int FAIL_THRESHOLD = 3;
    private final Map<String, Integer> failCountMap = new ConcurrentHashMap<>();

    /**
     * 每 30 秒执行一次健康检查
     */
    @Scheduled(fixedRateString = "${camera.health-check.interval-ms:30000}")
    public void checkAllCameras() {
        List<Camera> cameras = cameraService.listEnabledCameras();
        for (Camera camera : cameras) {
            checkSingleCamera(camera);
        }
    }

    private void checkSingleCamera(Camera camera) {
        try {
            // 调用 Stream Gateway health API
            CameraHealthResponse health = cameraService.fetchHealth(camera.getCameraId());

            String previousStatus = camera.getStatus();
            camera.setStatus("online");
            camera.setFpsCurrent(health.getFps());
            camera.setLastHeartbeat(LocalDateTime.now());
            cameraService.updateById(camera);

            failCountMap.put(camera.getCameraId(), 0);

            // 状态从 offline → online, 记录恢复
            if ("offline".equals(previousStatus)) {
                cameraService.logStatusChange(camera, previousStatus, "online", "健康检查恢复");
                log.info("摄像头恢复在线: {}", camera.getCameraId());
            }

        } catch (Exception e) {
            // 健康检查失败
            int failCount = failCountMap.getOrDefault(camera.getCameraId(), 0) + 1;
            failCountMap.put(camera.getCameraId(), failCount);

            if (failCount >= FAIL_THRESHOLD) {
                if (!"offline".equals(camera.getStatus())) {
                    camera.setStatus("offline");
                    cameraService.updateById(camera);
                    cameraService.logStatusChange(camera, camera.getStatus(), "offline",
                            "连续" + failCount + "次健康检查失败");
                    // 触发告警
                    cameraService.triggerOfflineAlert(camera);
                    log.warn("摄像头离线: {}, failCount={}", camera.getCameraId(), failCount);
                }
            } else {
                log.debug("摄像头健康检查失败({}/{}): {}",
                        failCount, FAIL_THRESHOLD, camera.getCameraId());
            }
        }
    }
}
```

### 3.3 Idle 检测消费延迟

在 `DormEventConsumer` 中，每收到一个事件就更新对应摄像头的 `last_event_time`：

```java
// 在 EventService.processEvent 结尾
cameraService.updateLastEventTime(msg.getCameraId(), msg.getTimestamp());
```

然后在定时任务中检查:

```java
// 在 CameraHealthCheckTask 中
private void checkIdleCameras() {
    List<Camera> cameras = cameraService.listOnlineCameras();
    for (Camera camera : cameras) {
        if (camera.getLastEventTime() == null) continue;
        long idleMinutes = ChronoUnit.MINUTES.between(
                camera.getLastEventTime(), LocalDateTime.now());
        if (idleMinutes > IDLE_THRESHOLD_MIN && !"idle".equals(camera.getStatus())) {
            camera.setStatus("idle");
            cameraService.updateById(camera);
            log.warn("摄像头无事件: {}, 已闲置 {} 分钟", camera.getCameraId(), idleMinutes);
        }
    }
}
```

---

## 4. 离线检测与告警

### 4.1 告警触发条件

| 场景 | 检测方式 | 延迟 |
|------|---------|------|
| 摄像头硬件断连 | Stream Gateway health API 连续 3 次失败 | ~90 秒 (30s×3) |
| 摄像头卡顿/fps=0 | health API 返回 fps=0 | ~30 秒 |
| 摄像头画面静止无事件 | Kafka 消费无事件超过 5 分钟 | ~5 分钟 |
| Stream Gateway 进程挂 | health API HTTP 连接拒绝 | ~90 秒 |

### 4.2 告警联动

```java
public void triggerOfflineAlert(Camera camera) {
    // 推送到 t_dorm_alert
    AlertMessage alert = AlertMessage.builder()
            .alertType("SYSTEM")
            .building(camera.getBuilding())
            .severity("critical")
            .description(String.format(
                    "%s 摄像头离线 (连续健康检查失败)", camera.getName()))
            .timestamp(LocalDateTime.now())
            .build();

    kafkaTemplate.send("t_dorm_alert", JSON.toJSONString(alert));

    // 同时写入 dorm_alert_record
    AlertRecord record = new AlertRecord();
    record.setAlertId(UUID.randomUUID().toString());
    record.setAlertType("SYSTEM");
    record.setBuilding(camera.getBuilding());
    record.setSeverity("critical");
    record.setDescription(alert.getDescription());
    alertRecordMapper.insert(record);
}
```

---

## 5. Service 实现

### 5.1 CameraService

```java
@Service
@Slf4j
public class CameraService {

    @Autowired
    private CameraMapper cameraMapper;
    @Autowired
    private CameraLogMapper cameraLogMapper;
    @Autowired
    private AlertRecordMapper alertRecordMapper;
    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;
    @Autowired
    private RestTemplate restTemplate;

    // ===== CRUD =====

    public List<Camera> listEnabledCameras() {
        return cameraMapper.selectList(
                Wrappers.<Camera>lambdaQuery().eq(Camera::getEnabled, true));
    }

    public Camera getByCameraId(String cameraId) {
        Camera camera = cameraMapper.selectOne(
                Wrappers.<Camera>lambdaQuery().eq(Camera::getCameraId, cameraId));
        if (camera == null) {
            throw new BusinessException(ErrorCode.NOT_FOUND, Map.of("cameraId", cameraId));
        }
        return camera;
    }

    public Camera create(Camera camera) {
        // 检查数量限制
        long count = cameraMapper.selectCount(Wrappers.emptyWrapper());
        if (count >= MAX_CAMERA_COUNT) {
            throw new BusinessException(ErrorCode.CAMERA_LIMIT_EXCEEDED);
        }
        camera.setStatus("unknown");
        cameraMapper.insert(camera);
        log.info("新增摄像头: cameraId={}, building={}", camera.getCameraId(), camera.getBuilding());
        return camera;
    }

    public void update(String cameraId, Camera update) {
        Camera existing = getByCameraId(cameraId);
        // 复制可更新字段
        if (update.getRtspUrl() != null) existing.setRtspUrl(update.getRtspUrl());
        if (update.getName() != null) existing.setName(update.getName());
        if (update.getResolution() != null) existing.setResolution(update.getResolution());
        if (update.getRemark() != null) existing.setRemark(update.getRemark());
        cameraMapper.updateById(existing);
        log.info("更新摄像头: cameraId={}", cameraId);
    }

    public void delete(String cameraId) {
        cameraMapper.delete(
                Wrappers.<Camera>lambdaQuery().eq(Camera::getCameraId, cameraId));
        log.info("删除摄像头: cameraId={}", cameraId);
    }

    // ===== 健康检查 =====

    public CameraHealthResponse fetchHealth(String cameraId) {
        // 从 Stream Gateway health API 获取该摄像头状态
        String url = String.format("http://stream-gateway:8080/health", cameraId);
        ResponseEntity<GatewayHealthResponse> response = restTemplate.getForEntity(
                url, GatewayHealthResponse.class, 5000);

        // 从返回的 cameras 列表中找对应的
        return response.getBody().getCameras().stream()
                .filter(c -> c.getCameraId().equals(cameraId))
                .findFirst()
                .orElseThrow(() -> new RuntimeException("摄像头不存在于 Gateway: " + cameraId));
    }

    public void updateLastEventTime(String cameraId, long timestampMs) {
        Camera camera = cameraMapper.selectOne(
                Wrappers.<Camera>lambdaQuery().eq(Camera::getCameraId, cameraId));
        if (camera != null) {
            camera.setLastEventTime(
                    LocalDateTime.ofInstant(
                            Instant.ofEpochMilli(timestampMs),
                            ZoneId.of("Asia/Shanghai")));
            cameraMapper.updateById(camera);
        }
    }

    // ===== 状态日志 =====

    public void logStatusChange(Camera camera, String from, String to, String reason) {
        CameraLog logEntry = new CameraLog();
        logEntry.setCameraId(camera.getCameraId());
        logEntry.setBuilding(camera.getBuilding());
        logEntry.setStatusFrom(from);
        logEntry.setStatusTo(to);
        logEntry.setReason(reason);
        logEntry.setFpsAtTime(camera.getFpsCurrent());
        cameraLogMapper.insert(logEntry);
    }

    // ===== 告警 =====

    public void triggerOfflineAlert(Camera camera) {
        String alertId = "alert_offline_" + camera.getCameraId() + "_"
                + LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyyMMddHHmmss"));

        AlertRecord record = new AlertRecord();
        record.setAlertId(alertId);
        record.setAlertType("SYSTEM");
        record.setBuilding(camera.getBuilding());
        record.setSeverity("critical");
        record.setDescription(camera.getName() + " 离线: 连续健康检查失败");
        record.setOccurredAt(LocalDateTime.now());
        alertRecordMapper.insert(record);

        // Kafka 推送
        Map<String, Object> msg = new HashMap<>();
        msg.put("alertId", alertId);
        msg.put("type", "SYSTEM");
        msg.put("building", camera.getBuilding());
        msg.put("severity", "critical");
        msg.put("description", record.getDescription());
        msg.put("timestamp", LocalDateTime.now().toString());
        kafkaTemplate.send("t_dorm_alert", JSON.toJSONString(msg));

        log.warn("触发摄像头离线告警: alertId={}, cameraId={}",
                alertId, camera.getCameraId());
    }
}
```

---

## 6. Controller 实现

### 6.1 CameraController

```java
@RestController
@RequestMapping("/api/dormitory/cameras")
@Validated
@Slf4j
public class CameraController {

    @Autowired
    private CameraService cameraService;

    /**
     * 获取所有摄像头列表
     */
    @GetMapping
    public ApiResponse<Map<String, Object>> listCameras() {
        List<Camera> cameras = cameraService.listEnabledCameras();
        Map<String, Object> result = new HashMap<>();
        result.put("total", cameras.size());
        result.put("cameras", cameras);
        return ApiResponse.success(result);
    }

    /**
     * 获取摄像头详情
     */
    @GetMapping("/{cameraId}")
    public ApiResponse<Camera> getCamera(@PathVariable String cameraId) {
        return ApiResponse.success(cameraService.getByCameraId(cameraId));
    }

    /**
     * 新增摄像头
     */
    @PostMapping
    public ApiResponse<Camera> createCamera(@RequestBody @Valid Camera camera) {
        Camera created = cameraService.create(camera);
        return ApiResponse.success(created);
    }

    /**
     * 更新摄像头
     */
    @PutMapping("/{cameraId}")
    public ApiResponse<Map<String, Object>> updateCamera(
            @PathVariable String cameraId,
            @RequestBody Camera update) {
        cameraService.update(cameraId, update);
        return ApiResponse.success(Map.of("cameraId", cameraId, "updated", true));
    }

    /**
     * 删除摄像头
     */
    @DeleteMapping("/{cameraId}")
    public ApiResponse<Void> deleteCamera(@PathVariable String cameraId) {
        cameraService.delete(cameraId);
        return ApiResponse.success(null);
    }

    /**
     * 摄像头实时状态看板
     */
    @GetMapping("/status")
    public ApiResponse<Map<String, Object>> getCameraStatus(
            @RequestParam(required = false) String building) {
        List<Camera> cameras;
        if (building != null) {
            cameras = cameraService.listByBuilding(building);
        } else {
            cameras = cameraService.listEnabledCameras();
        }

        List<Map<String, Object>> buildings = cameras.stream().map(c -> {
            Map<String, Object> item = new HashMap<>();
            item.put("building", c.getBuilding());
            item.put("cameraId", c.getCameraId());
            item.put("status", c.getStatus());
            item.put("fps", c.getFpsCurrent());
            item.put("lastEventTime", c.getLastEventTime());
            return item;
        }).collect(Collectors.toList());

        long online = cameras.stream().filter(c -> "online".equals(c.getStatus())).count();
        long offline = cameras.stream().filter(c -> "offline".equals(c.getStatus())).count();
        long idle = cameras.stream().filter(c -> "idle".equals(c.getStatus())).count();

        Map<String, Object> result = new HashMap<>();
        result.put("buildings", buildings);
        result.put("summary", Map.of(
                "total", cameras.size(),
                "online", online,
                "offline", offline,
                "idle", idle
        ));
        return ApiResponse.success(result);
    }

    /**
     * 按摄像头查询抓拍历史
     */
    @GetMapping("/{cameraId}/snapshots")
    public ApiResponse<PageResponse<SnapshotVO>> getSnapshots(
            @PathVariable String cameraId,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE_TIME)
                    LocalDateTime startTime,
            @RequestParam(required = false) @DateTimeFormat(iso = DateTimeFormat.ISO.DATE_TIME)
                    LocalDateTime endTime,
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "20") int size) {

        PageResponse<SnapshotVO> snapshots = cameraService.querySnapshots(
                cameraId, startTime, endTime, page, size);
        return ApiResponse.success(snapshots);
    }
}
```

---

## 7. 数据模型

### 7.1 Camera 实体

```java
@Data
@EqualsAndHashCode(callSuper = false)
@TableName("dorm_camera")
public class Camera implements Serializable {

    @TableId(type = IdType.AUTO)
    private Long id;

    /** 摄像头唯一 ID */
    private String cameraId;

    /** 显示名称 */
    private String name;

    /** 所在楼栋 A/B/C/D */
    private String building;

    /** RTSP 拉流地址 */
    private String rtspUrl;

    /** 监控方向 */
    private String direction;

    /** 分辨率 */
    private String resolution;

    /** online / offline / idle / unknown */
    private String status;

    /** 当前帧率 */
    private BigDecimal fpsCurrent;

    /** 累计帧数 */
    private Long totalFrames;

    /** 最近心跳时间 */
    private LocalDateTime lastHeartbeat;

    /** 最近事件时间 */
    private LocalDateTime lastEventTime;

    /** 是否启用 */
    private Boolean enabled;

    /** 摄像头级配置(JSON) */
    private String configJson;

    /** 备注 */
    private String remark;

    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
```

### 7.2 CameraLog 实体

```java
@Data
@EqualsAndHashCode(callSuper = false)
@TableName("dorm_camera_log")
public class CameraLog implements Serializable {

    @TableId(type = IdType.AUTO)
    private Long id;

    private String cameraId;
    private String building;
    private String statusFrom;
    private String statusTo;
    private String reason;
    private BigDecimal fpsAtTime;
    private LocalDateTime createdAt;
}
```

### 7.3 Stream Gateway Health Response DTO

```java
@Data
public class GatewayHealthResponse {
    private String status;
    private List<CameraHealthItem> cameras;

    @Data
    public static class CameraHealthItem {
        private String cameraId;
        private String building;
        private boolean connected;
        private double fps;
        private String lastFrameTime;
        private long framesSent;
        private long uptimeSeconds;
    }
}
```

---

## 8. 与感知层的接口约定

### 8.1 感知层 Stream Gateway 必须提供的

```
GET http://stream-gateway:8080/health

Response:
{
  "status": "UP",
  "cameras": [
    {
      "camera_id": "cam-a",
      "building": "A",
      "connected": true,
      "fps": 4.8,
      "last_frame_time": "2026-05-15T14:30:00+08:00",
      "frames_sent": 12345,
      "uptime_seconds": 86400
    }
  ]
}
```

### 8.2 字段映射

| Stream Gateway 返回字段 | Camera 实体字段 | 说明 |
|------------------------|----------------|------|
| `camera_id` | `cameraId` | 匹配标识 |
| `connected` | `status` | true→online, false→offline |
| `fps` | `fpsCurrent` | 实时帧率 |
| `last_frame_time` | `lastHeartbeat` | 最后帧时间 |
| `frames_sent` | `totalFrames` | 累积帧数 |

### 8.3 部署注意事项

| 注意点 | 说明 |
|--------|------|
| 网络可达 | 本模块必须能 HTTP 访问 `http://stream-gateway:8080` |
| 超时设置 | HTTP 超时 5 秒（不可达时快速失败） |
| camera_id 对齐 | 两个服务使用相同的 camera_id 体系（cam-a/b/c/d） |
| 启动顺序 | Stream Gateway 先启动，再启动本模块 |

---

> **本文件属于**: `doc/design/camera/01-camera-feature.md`  
> **面向读者**: 摄像头功能开发者（搭档）  
> **依赖感知层**: Stream Gateway health API  
> **对应 PRD**: PRD-005
