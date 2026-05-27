# 主进程对接设计

> **文档归属**: 系统集成 → 主进程对接  
> **对应 PRD**: PRD-004 (主进程对接)  
> **版本**: v1.0 · **更新**: 2026-05-15  

---

## 目录

1. [目标与原则](#1-目标与原则)
2. [当前架构 (Phase 1)](#2-当前架构-phase-1)
3. [目标架构 (Phase 3)](#3-目标架构-phase-3)
4. [Maven 模块提取](#4-maven-模块提取)
5. [Controller 迁移](#5-controller-迁移)
6. [认证鉴权合并](#6-认证鉴权合并)
7. [数据源合并](#7-数据源合并)
8. [配置合并](#8-配置合并)
9. [迁移清单](#9-迁移清单)
10. [回退方案](#10-回退方案)

---

## 1. 目标与原则

### 1.1 总体目标

将独立部署的 Dormitory Service JAR 无缝接入学管主 SpringBoot 进程，成为主进程的一个模块，**不改变**现有功能和行为。

### 1.2 核心原则

| 原则 | 说明 |
|------|------|
| **不改业务代码** | Service / Mapper / 实体类 0 改动 |
| **路径兼容** | API 路径从 `/api/dormitory/*` 变为 `/api/sims/dormitory/*`，旧路径保留过渡期内可用 |
| **可回退** | 独立 JAR 模式保留，主进程出问题时能切回独立部署 |
| **逐步迁移** | 不是大爆炸迁移，分 3 步走 |

---

## 2. 当前架构 (Phase 1)

```
┌─────────────────────────────────────────────┐
│  学管主进程 (SpringBoot)                     │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Student  │  │ Class    │  │ Grade    │  │
│  │ Module   │  │ Module   │  │ Module   │  │
│  └──────────┘  └──────────┘  └──────────┘  │
│                                              │
│  独立部署:                                    │
│  Auth: JWT Filter                            │
│  DB: 共用 MySQL 实例                          │
│  Redis: 共用实例                               │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│  Dormitory Service (独立 JAR :8080)          │
│                                              │
│  Auth: 简单 Token (独立)                      │
│  DB: dormitory 库 (独立)                      │
│  Redis: dormitory 命名空间 (共用实例)           │
│  API: /api/dormitory/*                       │
└─────────────────────────────────────────────┘
```

---

## 3. 目标架构 (Phase 3)

```
┌─────────────────────────────────────────────┐
│  学管主进程 (SpringBoot) ← 统一进程            │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐│
│  │ Student  │  │ Class    │  │ Dormitory  ││
│  │ Module   │  │ Module   │  │ Module     ││  ← dormitory-core
│  └──────────┘  └──────────┘  └────────────┘│
│                                              │
│  统一:                                        │
│  Auth: Spring Security + JWT                  │
│  DB: 主进程统一配置的数据源                      │
│  Redis: 主进程统一配置的 RedisTemplate          │
│  API: /api/sims/dormitory/*                   │
│  Config: 主进程 application.yml 统一管理        │
└─────────────────────────────────────────────┘
```

---

## 4. Maven 模块提取

### 4.1 多模块结构

```
sims-main-process/                           ← 学管主项目
├── pom.xml                                  ← parent pom
├── sims-common/                             ← 公共类 (统一响应、异常、工具)
│   └── src/main/java/com/sims/common/
│       ├── response/ApiResponse.java
│       ├── exception/BusinessException.java
│       └── ...
│
├── sims-student/                            ← 学生管理模块
├── sims-class/                              ← 班级管理模块
│
└── dormitory-core/                          ← 从 dormitory-service 提取 (NEW)
    └── src/main/java/com/sims/dormitory/
        ├── service/                         ← 0 改动
        ├── repository/                      ← 0 改动
        ├── model/entity/                    ← 0 改动
        ├── model/enums/                     ← 0 改动
        ├── consumer/                        ← 0 改动 (Kafka 消费者)
        └── scheduler/                       ← 0 改动 (定时任务)
```

### 4.2 dormitory-core 模块

```xml
<!-- dormitory-core/pom.xml -->
<project>
    <parent>
        <groupId>com.sims</groupId>
        <artifactId>sims-main-process</artifactId>
        <version>1.0.0</version>
    </parent>

    <artifactId>dormitory-core</artifactId>
    <packaging>jar</packaging>

    <dependencies>
        <!-- 依赖 sims-common 复用统一响应 -->
        <dependency>
            <groupId>com.sims</groupId>
            <artifactId>sims-common</artifactId>
        </dependency>

        <!-- 移除独立的 spring-boot-starter-web (由主进程提供) -->
        <!-- 保留 kafka/redis/mybatis/数据校验 -->
        <dependency>
            <groupId>org.springframework.kafka</groupId>
            <artifactId>spring-kafka</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-data-redis</artifactId>
        </dependency>
        <dependency>
            <groupId>com.baomidou</groupId>
            <artifactId>mybatis-plus-spring-boot3-starter</artifactId>
        </dependency>
    </dependencies>
</project>
```

### 4.3 提取的模块包含

| 包路径 | 是否改动 | 说明 |
|--------|---------|------|
| `service/` | ✅ 0 改动 | 直接复制 |
| `repository/` | ✅ 0 改动 | 直接复制 |
| `model/entity/` | ✅ 0 改动 | 直接复制 |
| `model/enums/` | ✅ 0 改动 | 直接复制 |
| `consumer/` | ✅ 0 改动 | Kafka 消费者 |
| `scheduler/` | ✅ 0 改动 | 定时任务 |
| `common/response/` | ⚠️ 移除 | 改用 sims-common 的 ApiResponse |
| `common/exception/` | ⚠️ 移除 | 改用 sims-common 的 BusinessException |
| `config/` | ⚠️ 移除 | 配置移至主进程 |
| `controller/` | ⚠️ 重建 | 在主进程中新建，路径改为 `/api/sims/dormitory/*` |

---

## 5. Controller 迁移

### 5.1 新旧路径映射

| 旧路径 (独立 JAR) | 新路径 (主进程) | 说明 |
|-------------------|----------------|------|
| `/api/dormitory/nightly-report/today` | `/api/sims/dormitory/nightly-report/today` | 查宿概览 |
| `/api/dormitory/students/status` | `/api/sims/dormitory/students/status` | 人员状态 |
| `/api/dormitory/events` | `/api/sims/dormitory/events` | 事件查询 |
| `/api/dormitory/cameras` | `/api/sims/dormitory/cameras` | 摄像头管理 |
| `/api/dormitory/config` | `/api/sims/dormitory/config` | 配置管理 |
| `/api/dormitory/health` | `/api/sims/dormitory/health` | 健康检查 |

### 5.2 Controller 示例 (主进程版)

```java
@RestController
@RequestMapping("/api/sims/dormitory/nightly-report")
@Slf4j
public class NightlyReportController {

    @Autowired
    private NightlyReportService reportService;

    @GetMapping("/today")
    public ApiResponse<NightlyReportVO> getTodayReport() {
        return ApiResponse.success(reportService.getTodayReport());
    }

    @GetMapping("/{date}")
    public ApiResponse<NightlyReportVO> getReportByDate(@PathVariable LocalDate date) {
        return ApiResponse.success(reportService.getReportByDate(date));
    }

    @GetMapping("/{date}/building/{building}")
    public ApiResponse<BuildingReportVO> getBuildingReport(
            @PathVariable LocalDate date,
            @PathVariable @Pattern(regexp = "[A-D]") String building) {
        return ApiResponse.success(reportService.getBuildingReport(date, building));
    }

    @PostMapping("/trigger")
    public ApiResponse<TriggerResult> triggerReport(
            @RequestBody @Valid TriggerRequest request) {
        return ApiResponse.success(reportService.triggerReport(request));
    }
}
```

### 5.3 统一响应适配

独立阶段使用 `com.sims.dormitory.common.response.ApiResponse`  
主进程阶段使用 `com.sims.common.response.ApiResponse`

两个类签名完全一致（code/text/data/timestamp/requestId），主进程的 `GlobalExceptionHandler` 统一接管。

---

## 6. 认证鉴权合并

### 6.1 独立阶段: TokenFilter

```java
// 独立部署: 简单 Token 校验
@Component
public class DormitoryTokenFilter extends OncePerRequestFilter {
    @Override
    protected void doFilterInternal(HttpServletRequest request,
            HttpServletResponse response, FilterChain chain) {
        String token = request.getHeader("Authorization");
        if (token == null || !token.equals(apiToken)) {
            throw new BusinessException(ErrorCode.UNAUTHORIZED);
        }
        chain.doFilter(request, response);
    }
}
```

### 6.2 主进程阶段: 复用 Spring Security

```java
// 主进程中: 无需独立 filter，由主进程 SecurityConfig 统一管理
@Configuration
@EnableWebSecurity
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .authorizeHttpRequests(auth -> auth
                // 宿舍模块路径统一受保护
                .requestMatchers("/api/sims/dormitory/**").authenticated()
                // 健康检查开放
                .requestMatchers("/api/sims/dormitory/health").permitAll()
            )
            .oauth2ResourceServer(oauth2 -> oauth2
                .jwt(Customizer.withDefaults())
            );
        return http.build();
    }
}
```

**迁移要点**:

| 要点 | 操作 |
|------|------|
| 独立 JWT 校验 | 删除 `DormitoryTokenFilter` |
| 主进程 SecurityConfig | 添加 `/api/sims/dormitory/**` 到受保护路径 |
| 角色权限 | 宿舍数据可设 `ROLE_DORM_MANAGER` 角色 |
| 用户上下文 | 通过 `SecurityContextHolder` 获取当前用户 |

---

## 7. 数据源合并

### 7.1 独立阶段

```yaml
# dormitory-service application.yml
spring:
  datasource:
    url: jdbc:mariadb://localhost:3306/dormitory
    username: dormitory
    password: ${DB_PASSWORD}
```

### 7.2 主进程阶段

```yaml
# 主进程 application.yml
spring:
  datasource:
    # 共用学管主数据源
    url: jdbc:mariadb://localhost:3306/sims
    username: sims
    password: ${DB_PASSWORD}

mybatis-plus:
  # dormitory-core 的 Mapper 扫描路径
  mapper-locations:
    - classpath:mapper/sims/**/*.xml      # 学管原 mapper
    - classpath:mapper/dormitory/**/*.xml  # 宿舍模块 mapper

  # 实体扫描
  type-aliases-package: com.sims.common.entity, com.sims.dormitory.model.entity
```

### 7.3 表名冲突检查

| dormitory 表 | 学管已有表 | 冲突 |
|-------------|----------|------|
| `dorm_student_assignment` | ❌ 无 | ✅ 无冲突 |
| `dorm_student_status` | ❌ 无 | ✅ 无冲突 |
| `dorm_entry_exit_event` | ❌ 无 | ✅ 无冲突 |
| 全部 `dorm_*` 前缀 | 各异 | ✅ 无冲突 |

> 全部 dormitory 表使用 `dorm_` 前缀，与学管系统表名无冲突。

### 7.4 Redis 命名空间隔离

```yaml
spring:
  data:
    redis:
      # 共用 Redis 实例
      host: ${REDIS_HOST:localhost}
      port: 6379
```

独立部署时使用 `dorm:` 前缀  
主进程中仍保持 `dorm:` 前缀   
→ 与学管系统的 `sims:` 前缀天然隔离。

---

## 8. 配置合并

### 8.1 配置迁移清单

```yaml
# 主进程 application.yml — 宿舍模块配置段
---
# 宿舍模块配置
dormitory:
  nightly-report:
    trigger-time: "23:00"
    timezone: "Asia/Shanghai"
  late-return:
    threshold: "22:00"
  alert:
    stranger:
      enabled: true
    cooldown-seconds: 300
    max-per-minute: 100
  sync:
    student:
      enabled: true
      interval-min: 60
      api-url: "http://sims-backend:8080/sims/students/dormitory"
      timeout-sec: 30
      retry-max: 3
  camera:
    health-check:
      interval-sec: 30
    offline:
      alert-threshold: 3
    idle:
      threshold-min: 5

# Kafka 配置（与学管共用或独立）
spring:
  kafka:
    bootstrap-servers: ${KAFKA_BOOTSTRAP:localhost:9092}
    consumer:
      group-id: dormitory-service
      auto-offset-reset: latest
      enable-auto-commit: false
```

### 8.2 配置读取适配

独立部署阶段: 从 `application.yml` 直接读取  
主进程阶段: 从主进程 `application.yml` 的 `dormitory.*` 前缀读取

```java
// 兼容两种部署模式
@Component
@ConfigurationProperties(prefix = "dormitory")
public class DormitoryProperties {
    private NightlyReport nightlyReport = new NightlyReport();
    private LateReturn lateReturn = new LateReturn();
    private Alert alert = new Alert();
    private Sync sync = new Sync();
    private Camera camera = new Camera();

    @Data
    public static class NightlyReport {
        private String triggerTime = "23:00";
        private String timezone = "Asia/Shanghai";
    }
    // ...
}
```

---

## 9. 迁移清单

### 9.1 Phase 2 → Phase 3 迁移步骤

| 步骤 | 操作 | 风险 | 回退 |
|------|------|------|------|
| 1 | 创建 `dormitory-core` Maven 模块，复制 service/repository/model/consumer/scheduler | 低 | 删除模块 |
| 2 | 独立 JAR 中排除 controller/config，验证 dormitory-core 编译通过 | 低 | 回退代码 |
| 3 | 主进程新建 `DormitoryController`，路径 `/api/sims/dormitory/*`，调用 dormitory-core 的 Service | 中 | 保留旧 controller |
| 4 | 主进程添加宿舍模块配置段，Kafka/Redis/DB 复用主进程配置 | 中 | 切回独立配置 |
| 5 | 独立 JAR 去掉 authentication filter，注册主进程 SecurityConfig | 低 | 恢复 filter |
| 6 | 测试环境全面回归 | 高 | 修复问题 |
| 7 | 灰度上线，观察 3 天 | 高 | 切回独立 JAR |
| 8 | 下线独立 JAR 的 8080 端口 | 低 | 重启独立 JAR |

### 9.2 需主进程配合的准备工作

| 事项 | 谁做 | 说明 |
|------|------|------|
| `sims-common` 模块提供 `ApiResponse` + `BusinessException` | 主进程开发 | 确保与 dormitory 版本的签名一致 |
| 主进程 `application.yml` 添加 `dormitory.*` 配置段 | 主进程开发 | 独立部署时不需要 |
| 主进程 `SecurityConfig` 添加 `/api/sims/dormitory/**` 路由 | 主进程开发 | 统一认证 |
| 数据库表 `dorm_*` 确认无冲突 | 双方确认 | `dorm_` 前缀已保证无冲突 |
| Kafka Topic `t_dorm_event` 生产/消费确认 | 感知层/本模块确认 | 格式不变 |

---

## 10. 回退方案

### 10.1 什么情况下回退

| 问题 | 严重程度 | 操作 |
|------|---------|------|
| 接入后查宿 API 不可用 | critical | 停主进程宿舍模块，启独立 JAR |
| 接入后主进程启动失败 | critical | 回退 dormitory-core 代码版本 |
| 接入后性能下降 > 20% | high | 排查瓶颈，必要时独立 JAR |
| 接入后认证异常 | medium | 临时允许 `/api/sims/dormitory/**` 公开访问 |

### 10.2 回退操作

```bash
# 1. 停主进程宿舍模块（不改代码，改 nginx 路由即可）
# nginx: 将 /api/sims/dormitory/ 由指向主进程改为指向独立 JAR
location /api/sims/dormitory/ {
    proxy_pass http://dormitory-standalone:8080/api/dormitory/;
}

# 2. 启动独立 JAR
java -jar dormitory-service-standalone.jar --spring.profiles.active=prod

# 3. 验证
curl http://dormitory-standalone:8080/api/dormitory/health
```

> 回退全程无需修改代码，仅需 nginx 配置变更 + 启动备用 JAR。

---

> **本文件属于**: `doc/design/integration/01-main-process.md`  
> **面向读者**: 系统集成工程师（搭档）  
> **依赖**: PRD-004 主进程对接、sims-common 模块
