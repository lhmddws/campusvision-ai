# Dormitory Service — Java 后端架构设计

> **文档归属**: 后端开发 → 架构设计  
> **对应 PRD**: PRD-004 (主进程对接)  
> **版本**: v1.0 · **更新**: 2026-05-15  

---

## 目录

1. [项目骨架](#1-项目骨架)
2. [分层架构](#2-分层架构)
3. [核心配置](#3-核心配置)
4. [Kafka 消费设计](#4-kafka-消费设计)
5. [Redis 缓存设计](#5-redis-缓存设计)
6. [调度任务设计](#6-调度任务设计)
7. [异常处理框架](#7-异常处理框架)
8. [日志规范](#8-日志规范)
9. [包结构](#9-包结构)

---

## 1. 项目骨架

### 1.1 Maven 项目结构

```xml
<!-- pom.xml 关键依赖 (Phase 1 独立部署阶段) -->
<parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.5</version>
</parent>

<groupId>com.sims.dormitory</groupId>
<artifactId>dormitory-service</artifactId>
<version>1.0.0-SNAPSHOT</version>
<packaging>jar</packaging>

<properties>
    <java.version>17</java.version>
    <mybatis-plus.version>3.5.7</mybatis-plus.version>
    <kafka-client.version>3.7.0</kafka-client.version>
</properties>

<dependencies>
    <!-- Web -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-web</artifactId>
    </dependency>

    <!-- Kafka -->
    <dependency>
        <groupId>org.springframework.kafka</groupId>
        <artifactId>spring-kafka</artifactId>
    </dependency>

    <!-- Redis -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-data-redis</artifactId>
    </dependency>

    <!-- Database -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-jdbc</artifactId>
    </dependency>
    <dependency>
        <groupId>com.mysql</groupId>
        <artifactId>mysql-connector-j</artifactId>
        <scope>runtime</scope>
    </dependency>
    <!-- MariaDB (MySQL 兼容) -->
    <dependency>
        <groupId>org.mariadb.jdbc</groupId>
        <artifactId>mariadb-java-client</artifactId>
        <scope>runtime</scope>
    </dependency>

    <!-- MyBatis-Plus -->
    <dependency>
        <groupId>com.baomidou</groupId>
        <artifactId>mybatis-plus-spring-boot3-starter</artifactId>
        <version>${mybatis-plus.version}</version>
    </dependency>

    <!-- Validation -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-validation</artifactId>
    </dependency>

    <!-- Lombok -->
    <dependency>
        <groupId>org.projectlombok</groupId>
        <artifactId>lombok</artifactId>
        <optional>true</optional>
    </dependency>

    <!-- OpenAPI (Swagger) -->
    <dependency>
        <groupId>org.springdoc</groupId>
        <artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>
        <version>2.5.0</version>
    </dependency>

    <!-- Jackson (JSON) -->
    <dependency>
        <groupId>com.fasterxml.jackson.core</groupId>
        <artifactId>jackson-databind</artifactId>
    </dependency>
    <dependency>
        <groupId>com.fasterxml.jackson.datatype</groupId>
        <artifactId>jackson-datatype-jsr310</artifactId>
    </dependency>

    <!-- Actuator -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-actuator</artifactId>
    </dependency>

    <!-- Test -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-test</artifactId>
        <scope>test</scope>
    </dependency>
    <dependency>
        <groupId>org.springframework.kafka</groupId>
        <artifactId>spring-kafka-test</artifactId>
        <scope>test</scope>
    </dependency>
</dependencies>
```

### 1.2 目录骨架

```
dormitory-service/
├── pom.xml
├── src/
│   ├── main/
│   │   ├── java/com/sims/dormitory/
│   │   │   ├── DormitoryApplication.java       # 启动类
│   │   │   │
│   │   │   ├── config/                          # 配置类
│   │   │   │   ├── KafkaConfig.java
│   │   │   │   ├── RedisConfig.java
│   │   │   │   ├── MyBatisPlusConfig.java
│   │   │   │   ├── JacksonConfig.java
│   │   │   │   ├── SchedulingConfig.java
│   │   │   │   └── WebMvcConfig.java
│   │   │   │
│   │   │   ├── controller/                      # REST 控制器
│   │   │   │   ├── NightlyReportController.java
│   │   │   │   ├── StudentStatusController.java
│   │   │   │   ├── EventController.java
│   │   │   │   ├── AlertController.java
│   │   │   │   ├── SyncController.java
│   │   │   │   ├── ConfigController.java
│   │   │   │   ├── StatsController.java
│   │   │   │   └── CameraController.java
│   │   │   │
│   │   │   ├── consumer/                        # Kafka 消费者
│   │   │   │   └── DormEventConsumer.java
│   │   │   │
│   │   │   ├── service/                         # 业务逻辑层
│   │   │   │   ├── EventService.java
│   │   │   │   ├── StudentStatusService.java
│   │   │   │   ├── NightlyReportService.java
│   │   │   │   ├── AlertService.java
│   │   │   │   ├── SyncService.java
│   │   │   │   ├── ConfigService.java
│   │   │   │   ├── StatsService.java
│   │   │   │   ├── CameraService.java
│   │   │   │   └── HealthCheckService.java
│   │   │   │
│   │   │   ├── repository/                      # DAO 层 (MyBatis-Plus Mapper)
│   │   │   │   ├── StudentAssignmentMapper.java
│   │   │   │   ├── StudentStatusMapper.java
│   │   │   │   ├── EntryExitEventMapper.java
│   │   │   │   ├── NightlyReportMapper.java
│   │   │   │   ├── NightlyDetailMapper.java
│   │   │   │   ├── StrangerRecordMapper.java
│   │   │   │   ├── AlertRecordMapper.java
│   │   │   │   ├── ConfigMapper.java
│   │   │   │   ├── SyncLogMapper.java
│   │   │   │   ├── CameraMapper.java
│   │   │   │   └── CameraLogMapper.java
│   │   │   │
│   │   │   ├── model/                           # 数据模型
│   │   │   │   ├── entity/                      # 数据库实体
│   │   │   │   │   ├── StudentAssignment.java
│   │   │   │   │   ├── StudentStatus.java
│   │   │   │   │   ├── EntryExitEvent.java
│   │   │   │   │   ├── NightlyReport.java
│   │   │   │   │   ├── NightlyDetail.java
│   │   │   │   │   ├── StrangerRecord.java
│   │   │   │   │   ├── AlertRecord.java
│   │   │   │   │   ├── Config.java
│   │   │   │   │   ├── SyncLog.java
│   │   │   │   │   ├── Camera.java
│   │   │   │   │   └── CameraLog.java
│   │   │   │   │
│   │   │   │   ├── dto/                         # 数据传输对象
│   │   │   │   │   ├── DormEventMessage.java     # Kafka 消息
│   │   │   │   │   ├── NightlyReportVO.java
│   │   │   │   │   ├── StudentStatusVO.java
│   │   │   │   │   ├── AlertMessage.java
│   │   │   │   │   └── CameraStatusVO.java
│   │   │   │   │
│   │   │   │   ├── query/                       # 查询参数封装
│   │   │   │   │   ├── EventQuery.java
│   │   │   │   │   ├── StudentQuery.java
│   │   │   │   │   └── ReportQuery.java
│   │   │   │   │
│   │   │   │   └── enums/                       # 枚举
│   │   │   │       ├── EventType.java          # ENTRY / EXIT
│   │   │   │       ├── TodayStatus.java        # IN / OUT / UNKNOWN
│   │   │   │       ├── AlertType.java          # STRANGER_ENTRY / ...
│   │   │   │       ├── Severity.java           # LOW / MEDIUM / HIGH / CRITICAL
│   │   │   │       ├── CameraStatus.java       # ONLINE / OFFLINE / IDLE / UNKNOWN
│   │   │   │       └── SyncStatus.java         # SUCCESS / FAILED / IN_PROGRESS
│   │   │   │
│   │   │   ├── common/                          # 公共
│   │   │   │   ├── response/                    # 统一响应
│   │   │   │   │   ├── ApiResponse.java
│   │   │   │   │   ├── PageResponse.java
│   │   │   │   │   └── ErrorCode.java
│   │   │   │   ├── exception/                   # 异常定义
│   │   │   │   │   ├── BusinessException.java
│   │   │   │   │   └── GlobalExceptionHandler.java
│   │   │   │   └── constant/                    # 常量
│   │   │   │       └── RedisKeys.java
│   │   │   │
│   │   │   └── scheduler/                       # 定时任务
│   │   │       ├── NightlyReportTask.java
│   │   │       ├── SyncStudentTask.java
│   │   │       └── CameraHealthCheckTask.java
│   │   │
│   │   └── resources/
│   │       ├── application.yml                  # 主配置
│   │       ├── application-dev.yml              # 开发环境
│   │       ├── application-prod.yml             # 生产环境
│   │       ├── mapper/                          # MyBatis XML
│   │       │   ├── StudentAssignmentMapper.xml
│   │       │   ├── EntryExitEventMapper.xml
│   │       │   └── ...
│   │       ├── db/migration/                    # Flyway 迁移脚本
│   │       │   ├── V1__init_schema.sql
│   │       │   └── V2__seed_config.sql
│   │       └── logback-spring.xml               # 日志配置
│   │
│   └── test/java/com/sims/dormitory/
│       ├── service/
│       │   ├── EventServiceTest.java
│       │   ├── NightlyReportServiceTest.java
│       │   └── ...
│       └── consumer/
│           └── DormEventConsumerTest.java
```

---

## 2. 分层架构

### 2.1 层间调用链

```
HTTP Request
    │
    ▼
┌─────────────────────────────────────┐
│  Controller 层                       │
│  @RestController                     │
│  • 参数校验 (@Valid)                  │
│  • 调用 Service                      │
│  • 返回统一 ApiResponse              │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│  Service 层                          │
│  @Service                           │
│  • 业务逻辑编排                       │
│  • 事务管理 (@Transactional)          │
│  • 调用 Mapper / RedisTemplate       │
│  • 抛出 BusinessException            │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│  Repository 层                       │
│  @Mapper (MyBatis-Plus)             │
│  • 数据库 CRUD                       │
│  • 复杂查询使用 XML                   │
└────────────────┬────────────────────┘
                 │
                 ▼
              PostgreSQL
```

### 2.2 Kafka 消费链路

```
Kafka t_dorm_event
    │
    ▼
┌───────────────────────────┐
│  DormEventConsumer        │
│  @KafkaListener           │
│                           │
│  ① 反序列化 JSON          │
│  ② 幂等校验 (eventId)    │
│  ③ 调用 EventService     │
│  ④ 手动 commit           │
└──────────┬────────────────┘
           │
           ▼
┌───────────────────────────┐
│  EventService             │
│  @Service                 │
│  @Transactional           │
│                           │
│  ① 判断是否为陌生人        │
│  ② 更新 Redis 状态        │
│  ③ 写入 PostgreSQL       │
│  ④ 检查告警规则            │
└───────────────────────────┘
```

### 2.3 定时任务链路

```
Spring Scheduler (默认 23:00)
    │
    ▼
┌───────────────────────────┐
│  NightlyReportTask        │
│  @Scheduled(cron=...)     │
│                           │
│  ① 获取所有 active 学生   │
│  ② 查询今日 entry 事件   │
│  ③ 逐人判定状态            │
│  ④ 按楼栋/房间聚合         │
│  ⑤ 写入报表表 + 明细表    │
│  ⑥ 触发告警检查            │
└───────────────────────────┘
```

---

## 3. 核心配置

### 3.1 application.yml

```yaml
server:
  port: 8080
  servlet:
    context-path: /

spring:
  application:
    name: dormitory-service

  # 数据源 (Phase 1 独立数据库)
  datasource:
    url: jdbc:mariadb://localhost:3306/dormitory
    username: dormitory
    password: ${DB_PASSWORD}
    driver-class-name: org.mariadb.jdbc.Driver
    hikari:
      maximum-pool-size: 10
      minimum-idle: 2
      idle-timeout: 300000
      max-lifetime: 600000

  # Redis
  data:
    redis:
      host: ${REDIS_HOST:localhost}
      port: 6379
      password: ${REDIS_PASSWORD:}
      timeout: 3000ms
      lettuce:
        pool:
          max-active: 8
          max-idle: 4
          min-idle: 1

  # Kafka Consumer
  kafka:
    bootstrap-servers: ${KAFKA_BOOTSTRAP:localhost:9092}
    consumer:
      group-id: dormitory-service
      auto-offset-reset: latest
      enable-auto-commit: false  # 手动提交
      key-deserializer: org.apache.kafka.common.serialization.StringDeserializer
      value-deserializer: org.apache.kafka.common.serialization.StringDeserializer
      properties:
        max.poll.records: 500
        fetch.min.bytes: 1024
        fetch.max.wait.ms: 500
    listener:
      ack-mode: manual  # 手动 ACK
      concurrency: 3    # 3 个并发消费者（对应 4 个分区）

  # Jackson
  jackson:
    date-format: yyyy-MM-dd'T'HH:mm:ssXXX
    time-zone: Asia/Shanghai
    property-naming-strategy: LOWER_CAMEL_CASE
    serialization:
      write-dates-as-timestamps: false

  # Flyway
  flyway:
    enabled: true
    locations: classpath:db/migration
    baseline-on-migrate: true

# MyBatis-Plus
mybatis-plus:
  mapper-locations: classpath:mapper/*.xml
  type-aliases-package: com.sims.dormitory.model.entity
  configuration:
    map-underscore-to-camel-case: true
    log-impl: org.apache.ibatis.logging.stdout.StdOutImpl  # 开发环境
  global-config:
    db-config:
      id-type: AUTO
      logic-delete-field: deleted
      logic-delete-value: 1
      logic-not-delete-value: 0

# SpringDoc (Swagger)
springdoc:
  api-docs:
    path: /api-docs
  swagger-ui:
    path: /swagger-ui.html

# 日志
logging:
  level:
    com.sims.dormitory: DEBUG
    org.springframework.kafka: INFO
    com.baomidou.mybatisplus: INFO
```

### 3.2 Kafka 配置类

```java
@Configuration
@EnableKafka
public class KafkaConfig {

    /**
     * KafkaListenerContainerFactory 手动确认模式
     */
    @Bean
    public ConcurrentKafkaListenerContainerFactory<String, String>
            kafkaListenerContainerFactory(ConsumerFactory<String, String> factory) {
        ConcurrentKafkaListenerContainerFactory<String, String> containerFactory =
                new ConcurrentKafkaListenerContainerFactory<>();
        containerFactory.setConsumerFactory(factory);
        // 手动 Ack
        containerFactory.getContainerProperties().setAckMode(ContainerProperties.AckMode.MANUAL);
        // 并发消费者数
        containerFactory.setConcurrency(3);
        // 批量消费
        containerFactory.setBatchListener(true);
        return containerFactory;
    }

    /**
     * KafkaTemplate (用于生产告警消息到 t_dorm_alert)
     */
    @Bean
    public KafkaTemplate<String, String> kafkaTemplate(ProducerFactory<String, String> factory) {
        return new KafkaTemplate<>(factory);
    }
}
```

### 3.3 Redis 配置类

```java
@Configuration
public class RedisConfig {

    @Bean
    public RedisTemplate<String, Object> redisTemplate(RedisConnectionFactory factory) {
        RedisTemplate<String, Object> template = new RedisTemplate<>();
        template.setConnectionFactory(factory);

        // 使用 Jackson2JsonRedisSerializer 序列化 value
        Jackson2JsonRedisSerializer<Object> serializer =
                new Jackson2JsonRedisSerializer<>(Object.class);
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());
        mapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        mapper.setVisibility(PropertyAccessor.ALL, JsonAutoDetect.Visibility.ANY);
        mapper.activateDefaultTyping(
                LazyObjectMapper.defaultBaseType(),
                ObjectMapper.DefaultTyping.NON_FINAL);
        serializer.setObjectMapper(mapper);

        template.setKeySerializer(new StringRedisSerializer());
        template.setValueSerializer(serializer);
        template.setHashKeySerializer(new StringRedisSerializer());
        template.setHashValueSerializer(serializer);
        template.afterPropertiesSet();
        return template;
    }

    /**
     * StringRedisTemplate 用于简单 KV
     */
    @Bean
    public StringRedisTemplate stringRedisTemplate(RedisConnectionFactory factory) {
        return new StringRedisTemplate(factory);
    }
}
```

### 3.4 MyBatis-Plus 配置类

```java
@Configuration
@MapperScan("com.sims.dormitory.repository")
public class MyBatisPlusConfig {

    /**
     * 分页插件
     */
    @Bean
    public MybatisPlusInterceptor mybatisPlusInterceptor() {
        MybatisPlusInterceptor interceptor = new MybatisPlusInterceptor();
        interceptor.addInnerInterceptor(new PaginationInnerInterceptor(DbType.POSTGRE_SQL));
        return interceptor;
    }
}
```

---

## 4. Kafka 消费设计

### 4.1 消费者实现

```java
@Component
@Slf4j
public class DormEventConsumer {

    @Autowired
    private EventService eventService;

    /**
     * 消费 t_dorm_event，手动确认模式
     *
     * 消息体:
     * {
     *   "event_id": "evt_xxx",
     *   "camera_id": "cam-a",
     *   "building": "A",
     *   "student_id": "S2024001",
     *   "student_name": "张三",
     *   "event_type": "entry",
     *   "confidence": 0.95,
     *   "face_snapshot": "/9j/4AAQ...",
     *   "timestamp_unix_ms": 1747305000000,
     *   "is_stranger": false,
     *   "extra": { "class": "计算机2101班", "dorm_room": "A-301" }
     * }
     */
    @KafkaListener(
        topics = "${kafka.consumer.topic:t_dorm_event}",
        groupId = "${kafka.consumer.group:dormitory-service}",
        containerFactory = "kafkaListenerContainerFactory"
    )
    public void consume(List<ConsumerRecord<String, String>> records,
                        Acknowledgment ack) {
        for (ConsumerRecord<String, String> record : records) {
            try {
                DormEventMessage message = JsonUtils.parse(record.value(),
                        DormEventMessage.class);
                eventService.processEvent(message);
            } catch (Exception e) {
                // 记录失败消息到死信队列（日志 + 后续补偿）
                log.error("Failed to process dorm event: {}", record.value(), e);
                // 不阻塞其它消息，继续消费
            }
        }
        // 批量确认
        ack.acknowledge();
    }
}
```

### 4.2 事件处理 Service

```java
@Service
@Slf4j
public class EventService {

    @Autowired
    private StudentStatusService statusService;
    @Autowired
    private EntryExitEventMapper eventMapper;
    @Autowired
    private AlertService alertService;
    @Autowired
    private StringRedisTemplate redisTemplate;

    private static final long DEDUP_TTL_SEC = 3600; // 幂等去重 TTL

    /**
     * 处理单条进出事件
     */
    @Transactional
    public void processEvent(DormEventMessage msg) {
        // Step 1: 幂等校验
        String dedupKey = RedisKeys.eventProcessed(msg.getEventId());
        if (Boolean.TRUE.equals(redisTemplate.hasKey(dedupKey))) {
            log.debug("Duplicate event, skip: {}", msg.getEventId());
            return;
        }

        // Step 2: 持久化事件记录
        EntryExitEvent event = convertToEntity(msg);
        eventMapper.insert(event);

        // Step 3: 判断是否为本楼住宿学生
        StudentAssignment student = statusService.findStudent(
                msg.getBuilding(), msg.getStudentId());

        if (student == null) {
            // 陌生人处理
            handleStranger(msg);
        } else {
            // 更新在校状态
            statusService.updateStatus(student, msg.getEventType(), msg.getTimestamp());
        }

        // Step 4: 标记已处理（防重复）
        redisTemplate.opsForValue().set(dedupKey, "1", Duration.ofSeconds(DEDUP_TTL_SEC));
    }

    private void handleStranger(DormEventMessage msg) {
        // 记录陌生人事件
        eventMapper.updateStrangerFlag(msg.getEventId(), true);
        // 触发陌生人告警
        alertService.createAlert(AlertType.STRANGER_ENTRY, msg.getBuilding(),
                null, "陌生人进入 " + msg.getBuilding() + " 栋", Severity.HIGH);
    }
}
```

---

## 5. Redis 缓存设计

### 5.1 Key 命名规范

```java
public class RedisKeys {

    /** 学生在校状态 Hash: dorm:student:{studentId}:status */
    public static String studentStatus(String studentId) {
        return String.format("dorm:student:%s:status", studentId);
    }

    /** 楼栋学生集合 Set: dorm:building:{building}:students */
    public static String buildingStudents(String building) {
        return String.format("dorm:building:%s:students", building);
    }

    /** 事件已处理标记: dorm:event:processed:{eventId} */
    public static String eventProcessed(String eventId) {
        return String.format("dorm:event:processed:%s", eventId);
    }

    /** 楼栋状态缓存 Hash: dorm:building:{building}:status */
    public static String buildingStatus(String building) {
        return String.format("dorm:building:%s:status", building);
    }

    /** 今日查宿缓存 Hash: dorm:report:today:{building} */
    public static String todayReport(String building) {
        return String.format("dorm:report:today:%s", building);
    }

    /** 配置缓存 Hash: dorm:config */
    public static final String CONFIG = "dorm:config";
}
```

### 5.2 状态更新操作

```java
/**
 * 更新学生在校状态 (entry / exit)
 */
public void updateStatus(StudentAssignment student, String eventType, long timestamp) {
    String key = RedisKeys.studentStatus(student.getStudentId());

    Map<String, String> fields = new HashMap<>();
    fields.put("building", student.getBuilding());
    fields.put("room", student.getRoom());
    fields.put("studentName", student.getStudentName());

    if ("entry".equals(eventType)) {
        fields.put("isInDorm", "true");
        fields.put("lastEntryTime", formatTimestamp(timestamp));
        fields.put("todayStatus", "in");
    } else {
        fields.put("isInDorm", "false");
        fields.put("lastExitTime", formatTimestamp(timestamp));
    }

    // 写入 Redis Hash，TTL 次日 06:00 过期
    redisTemplate.opsForHash().putAll(key, fields);
    redisTemplate.expireAt(key, nextDay6am());

    // 同时更新楼栋聚合缓存
    updateBuildingCache(student.getBuilding());
}
```

---

## 6. 调度任务设计

### 6.1 每晚查宿统计

```java
@Component
@Slf4j
public class NightlyReportTask {

    @Autowired
    private NightlyReportService reportService;

    /**
     * 默认 23:00 执行，从配置动态读取
     */
    @Scheduled(cron = "${nightly_report.trigger_time:0 0 23 * * ?}")
    public void generateNightlyReport() {
        log.info("=== 开始每晚查宿统计 ===");
        try {
            reportService.generateForAllBuildings(LocalDate.now());
            log.info("=== 每晚查宿统计完成 ===");
        } catch (Exception e) {
            log.error("每晚查宿统计失败", e);
        }
    }
}
```

### 6.2 学管数据同步

```java
@Component
@Slf4j
public class SyncStudentTask {

    @Autowired
    private SyncService syncService;

    /**
     * 默认每 60 分钟执行
     */
    @Scheduled(fixedDelayString = "${sync.student.interval_ms:3600000}")
    public void syncStudents() {
        if (!syncService.isSyncEnabled()) {
            return;
        }
        log.info("开始同步学管宿舍数据...");
        SyncResult result = syncService.syncFromSIMS();
        log.info("同步完成: {}", result);
    }
}
```

### 6.3 摄像头健康检查

```java
@Component
@Slf4j
public class CameraHealthCheckTask {

    @Autowired
    private CameraService cameraService;

    /**
     * 每 30 秒检查一次摄像头状态
     */
    @Scheduled(fixedRateString = "${camera.health_check.interval_ms:30000}")
    public void checkCameras() {
        cameraService.checkAllCameras();
    }
}
```

---

## 7. 异常处理框架

### 7.1 统一响应体

```java
@Data
@NoArgsConstructor
@AllArgsConstructor
public class ApiResponse<T> {
    private int code;
    private String text;
    private T data;
    private String timestamp;
    private String requestId;

    public static <T> ApiResponse<T> success(T data) {
        return new ApiResponse<>(200, "success", data,
                LocalDateTime.now().toString(), MDC.get("requestId"));
    }

    public static <T> ApiResponse<T> error(int code, String message) {
        return new ApiResponse<>(code, message, null,
                LocalDateTime.now().toString(), MDC.get("requestId"));
    }

    public static <T> ApiResponse<T> error(ErrorCode errorCode) {
        return new ApiResponse<>(errorCode.getCode(), errorCode.getMessage(),
                null, LocalDateTime.now().toString(), MDC.get("requestId"));
    }
}
```

### 7.2 业务异常类

```java
@Getter
public class BusinessException extends RuntimeException {
    private final int code;
    private final String message;
    private final Map<String, Object> details;

    public BusinessException(ErrorCode errorCode) {
        super(errorCode.getMessage());
        this.code = errorCode.getCode();
        this.message = errorCode.getMessage();
        this.details = new HashMap<>();
    }

    public BusinessException(ErrorCode errorCode, Map<String, Object> details) {
        super(errorCode.getMessage());
        this.code = errorCode.getCode();
        this.message = errorCode.getMessage();
        this.details = details;
    }

    public BusinessException(int code, String message) {
        super(message);
        this.code = code;
        this.message = message;
        this.details = new HashMap<>();
    }
}
```

### 7.3 全局异常处理器

```java
@RestControllerAdvice
@Slf4j
public class GlobalExceptionHandler {

    @ExceptionHandler(BusinessException.class)
    public ResponseEntity<ApiResponse<Void>> handleBusiness(BusinessException e) {
        log.warn("业务异常: code={}, message={}", e.getCode(), e.getMessage());
        return ResponseEntity.status(HttpStatus.BAD_REQUEST)
                .body(ApiResponse.error(e.getCode(), e.getMessage()));
    }

    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ApiResponse<Void>> handleValidation(
            MethodArgumentNotValidException e) {
        String msg = e.getBindingResult().getFieldErrors().stream()
                .map(f -> f.getField() + ": " + f.getDefaultMessage())
                .collect(Collectors.joining(", "));
        return ResponseEntity.status(HttpStatus.BAD_REQUEST)
                .body(ApiResponse.error(400, msg));
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ApiResponse<Void>> handleUnknown(Exception e) {
        log.error("未知异常", e);
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                .body(ApiResponse.error(500, "服务器内部错误"));
    }
}
```

### 7.4 错误码枚举

```java
@Getter
public enum ErrorCode {
    INVALID_PARAMETER(400, "请求参数不合法"),
    UNAUTHORIZED(401, "认证失败"),
    NOT_FOUND(404, "资源不存在"),
    CONFLICT(409, "数据冲突"),
    BUILDING_INVALID(400, "楼栋参数不合法，仅支持 A/B/C/D"),
    STUDENT_NOT_FOUND(404, "未找到该学生"),
    REPORT_ALREADY_EXISTS(409, "该日期已存在查宿统计"),
    SYNC_IN_PROGRESS(409, "同步任务执行中"),
    CAMERA_LIMIT_EXCEEDED(400, "摄像头数量已达上限"),
    INTERNAL_ERROR(500, "服务器内部错误"),
    SERVICE_UNAVAILABLE(503, "服务暂不可用");

    private final int code;
    private final String message;

    ErrorCode(int code, String message) {
        this.code = code;
        this.message = message;
    }
}
```

---

## 8. 日志规范

### 8.1 logback-spring.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
    <include resource="org/springframework/boot/logging/logback/defaults.xml"/>

    <!-- 控制台输出 -->
    <appender name="CONSOLE" class="ch.qos.logback.core.ConsoleAppender">
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>

    <!-- 文件输出 -->
    <appender name="FILE" class="ch.qos.logback.core.rolling.RollingFileAppender">
        <file>${LOG_PATH:-logs}/dormitory-service.log</file>
        <rollingPolicy class="ch.qos.logback.core.rolling.TimeBasedRollingPolicy">
            <fileNamePattern>${LOG_PATH:-logs}/dormitory-service.%d{yyyy-MM-dd}.log.gz</fileNamePattern>
            <maxHistory>30</maxHistory>
        </rollingPolicy>
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>

    <!-- Kafka 消费日志单独文件 -->
    <appender name="KAFKA_FILE" class="ch.qos.logback.core.rolling.RollingFileAppender">
        <file>${LOG_PATH:-logs}/kafka-consumer.log</file>
        <rollingPolicy class="ch.qos.logback.core.rolling.TimeBasedRollingPolicy">
            <fileNamePattern>${LOG_PATH:-logs}/kafka-consumer.%d{yyyy-MM-dd}.log.gz</fileNamePattern>
            <maxHistory>7</maxHistory>
        </rollingPolicy>
        <encoder>
            <pattern>%d{yyyy-MM-dd HH:mm:ss.SSS} [%thread] %-5level %logger{36} - %msg%n</pattern>
        </encoder>
    </appender>

    <logger name="com.sims.dormitory.consumer" level="INFO" additivity="false">
        <appender-ref ref="KAFKA_FILE"/>
        <appender-ref ref="CONSOLE"/>
    </logger>

    <root level="INFO">
        <appender-ref ref="CONSOLE"/>
        <appender-ref ref="FILE"/>
    </root>
</configuration>
```

### 8.2 关键日志埋点

| 位置 | 日志事件 | 级别 | 说明 |
|------|---------|------|------|
| 事件消费 | 收到/处理完成 | INFO | 包含 eventId、building、eventType |
| 事件消费 | 解析失败/异常 | ERROR | 包含原始消息体 |
| 状态更新 | 更新成功 | DEBUG | 包含 studentId + 新状态 |
| 查宿统计 | 开始/完成/失败 | INFO | 包含 date、各楼栋计数 |
| 学管同步 | 开始/成功/失败 | INFO | 包含 syncId、数据量 |
| 告警触发 | 告警创建 | WARN | 包含 alertType、building |
| 摄像头检查 | 状态变更 | INFO | 包含 cameraId、old/new status |
| 配置更新 | 更新成功 | INFO | 包含 key、old/new value |

---

## 9. 包结构总结

```
com.sims.dormitory
├── DormitoryApplication.java
├── config/          # Spring Boot 配置类
├── controller/      # REST 控制器
├── consumer/        # Kafka 消费者
├── service/         # 业务逻辑
├── repository/      # MyBatis-Plus Mapper
├── model/
│   ├── entity/      # 数据库实体
│   ├── dto/         # 数据传输对象
│   ├── query/       # 查询参数
│   └── enums/       # 枚举
├── common/
│   ├── response/    # 统一响应
│   ├── exception/   # 异常处理
│   └── constant/    # 常量
└── scheduler/       # 定时任务
```

---

> **本文件属于**: `doc/design/backend/01-architecture.md`  
> **面向读者**: Java 后端开发（搭档）  
> **参考**: PRD-004 主进程对接、PRD-003 Dormitory Service
