# Dormitory Service — Go 后端架构设计

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

### 1.1 Go Module 结构

```go
// go.mod 关键依赖
module github.com/sims/campusvision/dormitory-service-go

go 1.26

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/redis/go-redis/v9 v9.7.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/spf13/viper v1.19.0
	go.uber.org/zap v1.27.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/robfig/cron/v3 v3.0.1
)
```

### 1.2 目录骨架

```
dormitory-service-go/
├── go.mod
├── go.sum
├── config.yaml                          # 本地开发配置
├── config.docker.yaml                   # Docker Compose 覆盖
├── Dockerfile
│
├── cmd/
│   └── dormitory-service/
│       └── main.go                      # 入口：依赖注入 + 路由注册 + Kafka 启动
│
├── internal/
│   ├── client/                          # PushClient → stream-gateway 通知
│   │   └── push_client.go
│   │
│   ├── config/                          # Viper 配置加载 (YAML + 环境变量)
│   │   └── config.go
│   │
│   ├── consumer/                        # Kafka 消费者
│   │   ├── event_consumer.go            # t_dorm_event 消费
│   │   ├── alert_consumer.go            # t_dorm_alert 消费 (骨架)
│   │   └── manager.go                   # 消费者管理器
│   │
│   ├── handler/                         # Gin HTTP 处理器
│   │   ├── camera_handler.go            # 摄像头 CRUD
│   │   ├── record_handler.go            # 考勤/事件查询
│   │   ├── alert_handler.go             # 告警管理
│   │   ├── config_handler.go            # 系统配置
│   │   ├── face.go                      # 人脸匹配/特征提取
│   │   ├── auth.go                      # 登录认证
│   │   └── response.go                  # 统一响应格式
│   │
│   ├── middleware/                       # 中间件
│   │   ├── auth.go                      # JWT 认证
│   │   └── cors.go                      # CORS 跨域
│   │
│   ├── model/                           # 数据模型
│   │   ├── dto/                         # 请求/响应类型
│   │   │   ├── camera_dto.go
│   │   │   ├── record_dto.go
│   │   │   ├── alert_dto.go
│   │   │   └── config_dto.go
│   │   │
│   │   ├── entity/                      # 数据库实体 (db tag 映射)
│   │   │   ├── student_assignment.go
│   │   │   ├── student_status.go
│   │   │   ├── entry_exit_event.go
│   │   │   ├── nightly_report.go
│   │   │   ├── nightly_detail.go
│   │   │   ├── stranger_record.go
│   │   │   ├── alert_record.go
│   │   │   ├── config.go
│   │   │   ├── sync_log.go
│   │   │   ├── camera.go
│   │   │   └── camera_log.go
│   │   │
│   │   └── enums/                       # 枚举
│   │       ├── event_type.go            # ENTRY / EXIT
│   │       ├── alert_type.go            # STRANGER_ENTRY / ...
│   │       ├── camera_status.go         # ONLINE / OFFLINE / IDLE / UNKNOWN
│   │       └── student_status.go        # IN / OUT / UNKNOWN
│   │
│   ├── redis/                           # go-redis 封装 + 去重
│   │   └── client.go
│   │
│   ├── repository/                      # DAO 层 (sqlx)
│   │   ├── base.go                      # 泛型 BaseRepository[T]
│   │   ├── camera_repository.go
│   │   ├── student_repository.go
│   │   ├── event_log_repository.go
│   │   ├── alert_repository.go
│   │   ├── config_repository.go
│   │   └── ...
│   │
│   ├── service/                         # 业务逻辑层
│   │   ├── camera_service.go
│   │   ├── record_service.go
│   │   ├── alert_service.go
│   │   ├── config_service.go
│   │   └── report_service.go
│   │
│   ├── scheduler/                       # 定时任务 (robfig/cron)
│   │   ├── manager.go                   # 任务管理器
│   │   ├── nightly_report_job.go        # 每晚查宿
│   │   └── health_check_job.go          # 摄像头健康检查
│   │
│   └── util/                            # 工具
│       └── crypto.go                    # AES-256-GCM 加密
│
├── test/
│   └── repository/
│       └── base_test.go                # BaseRepository 泛型 CRUD 测试
```

## 2. 分层架构

### 2.1 层间调用链

```
HTTP Request
    │
    ▼
┌─────────────────────────────────────┐
│  Handler 层 (Gin)                    │
│  func(c *gin.Context)               │
│  • 参数绑定与校验 (c.ShouldBindJSON)   │
│  • 调用 Service                      │
│  • 返回统一响应 (handler.OK)          │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│  Service 层                          │
│  struct { ... }                     │
│  • 业务逻辑编排                       │
│  • 调用 Repository / Redis           │
│  • 返回 error 或业务结果              │
└────────────────┬────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────┐
│  Repository 层                       │
│  BaseRepository[T] + sqlx           │
│  • 数据库 CRUD (泛型)                 │
│  • 复杂查询使用手写 SQL                │
└────────────────┬────────────────────┘
                 │
                 ▼
               MariaDB
```

### 2.2 Kafka 消费链路

```
Kafka t_dorm_event
    │
    ▼
┌───────────────────────────┐
│  EventConsumer            │
│  consumer.EventConsumer   │
│                           │
│  ① 解析 JSON 消息          │
│  ② 幂等校验 (Redis dedup) │
│  ③ 调用 Repository        │
│  ④ 手动 Commit            │
└──────────┬────────────────┘
           │
           ▼
┌───────────────────────────┐
│  Repository 层 (直调)      │
│  • 更新学生状态            │
│  • 写入事件记录            │
│  • 陌生人 → 创建告警       │
└───────────────────────────┘
```

> **注意**: EventConsumer 直接调用 Repository，绕过 Service 层。这是当前实现的设计。

### 2.3 定时任务链路

```
Cron Scheduler (robfig/cron)
    │
    ▼
┌───────────────────────────┐
│  NightlyReportJob         │
│  cron: "0 0 23 * * *"     │
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

### 3.1 config.yaml

```yaml
server:
  port: 8083

database:
  dsn: "root:root_dev@tcp(localhost:3306)/dormitory?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"
  driver: mysql
  max_open_conns: 25
  max_idle_conns: 10

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0
  password: ""

kafka:
  brokers: ["localhost:9092"]
  event_topic: "t_dorm_event"
  alert_topic: "t_dorm_alert"
  group_id: "dormitory-service-group"
  max_poll_record: 500

jwt:
  secret: "${JWT_SECRET:your-256-bit-secret}"
  expiration_hours: 24

auth:
  admin_username: "admin"
  admin_password: "${ADMIN_PASSWORD:admin123}"

face:
  match_threshold: 0.6
  match_key: "${FACE_MATCH_KEY:}"

camera:
  max_cameras: 50

log:
  level: "info"
```

### 3.2 配置加载 (Viper)

```go
// internal/config/config.go
package config

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Face     FaceConfig     `mapstructure:"face"`
	Camera   CameraConfig   `mapstructure:"camera"`
	Log      LogConfig      `mapstructure:"log"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	// 兼容主流框架风格的环境变量映射 (SPRING_DATASOURCE_URL 等)
	v.BindEnv("database.dsn", "SPRING_DATASOURCE_URL")
	v.BindEnv("kafka.brokers", "KAFKA_BOOTSTRAP_SERVERS")
	// ...
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
```

### 3.3 Kafka 消费者配置

```go
// internal/consumer/event_consumer.go
type EventConsumer struct {
	logger       *zap.Logger
	redis        *redis.Client
	brokers      []string
	topic        string
	groupID      string
	maxPollRec   int
	// repositories...
}

func NewEventConsumer(
	logger *zap.Logger,
	redis *redis.Client,
	brokers []string,
	topic string,
	groupID string,
	maxPollRec int,
	// repos...
) *EventConsumer {
	return &EventConsumer{
		logger:     logger,
		redis:      redis,
		brokers:    brokers,
		topic:      topic,
		groupID:    groupID,
		maxPollRec: maxPollRec,
	}
}

func (c *EventConsumer) Start(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		Topic:    c.topic,
		GroupID:  c.groupID,
		MinBytes: 1024,
		MaxBytes: 10e6,
	})
	go c.consumeLoop(ctx, reader)
}
```

### 3.4 Redis 客户端封装

```go
// internal/redis/client.go
package redis

import (
	"context"
	"fmt"
	redigo "github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redigo.Client
}

func NewClient(host string, port int, db int, password string) *Client {
	rdb := redigo.NewClient(&redigo.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		DB:       db,
		Password: password,
	})
	return &Client{rdb: rdb}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// CheckAndSetDedup 幂等去重: key=dedup:{camera_id}:{frame_sequence}, TTL=3600s
func (c *Client) CheckAndSetDedup(ctx context.Context, cameraID, frameSeq string) (bool, error) {
	key := fmt.Sprintf("dedup:%s:%s", cameraID, frameSeq)
	set, err := c.rdb.SetNX(ctx, key, "1", 3600*time.Second).Result()
	return set, err
}
```

### 3.5 sqlx 数据库连接

```go
// cmd/dormitory-service/main.go
import (
	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, _ := config.Load(cfgPath)

	db, err := sqlx.Connect(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.Database.MaxOpenConn)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConn)
	db.SetConnMaxLifetime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
}
```

---

## 4. Kafka 消费设计

### 4.1 消费者实现

```go
// internal/consumer/event_consumer.go
package consumer

type EventConsumer struct {
	logger  *zap.Logger
	redis   *redis.Client
	brokers []string
	topic   string
	groupID string
	// repositories...
}

// 消息体结构:
// {
//   "event_id": "evt_xxx",
//   "camera_id": "cam-a",
//   "building": "A",
//   "student_id": "S2024001",
//   "student_name": "张三",
//   "event_type": "entry",
//   "confidence": 0.95,
//   "face_snapshot": "/9j/4AAQ...",
//   "timestamp_unix_ms": 1747305000000,
//   "is_stranger": false,
//   "extra": { "class": "计算机2101班", "dorm_room": "A-301" }
// }

func (c *EventConsumer) Start(ctx context.Context) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.brokers,
		Topic:    c.topic,
		GroupID:  c.groupID,
		MinBytes: 1024,
		MaxBytes: 10e6,
	})
	go c.consumeLoop(ctx, reader)
}

func (c *EventConsumer) consumeLoop(ctx context.Context, reader *kafka.Reader) {
	defer reader.Close()
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			c.logger.Error("fetch message failed", zap.Error(err))
			continue
		}
		c.processMessage(ctx, msg, reader)
	}
}

func (c *EventConsumer) processMessage(ctx context.Context, msg kafka.Message, reader *kafka.Reader) {
	var event DormEventMessage
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.logger.Error("parse message failed", zap.Error(err), zap.String("value", string(msg.Value)))
		reader.CommitMessages(ctx, msg)
		return
	}

	// 幂等去重
	dedupKey := fmt.Sprintf("dedup:%s:%s", event.CameraID, event.EventID)
	isSet, _ := c.redis.CheckAndSetDedup(ctx, event.CameraID, event.EventID)
	if !isSet {
		c.logger.Debug("duplicate event, skip", zap.String("event_id", event.EventID))
		reader.CommitMessages(ctx, msg)
		return
	}

	// 处理事件
	if err := c.handleEvent(ctx, event); err != nil {
		c.logger.Error("process event failed", zap.Error(err), zap.String("event_id", event.EventID))
	}

	reader.CommitMessages(ctx, msg)
}
```

### 4.2 事件处理

```go
// internal/consumer/event_consumer.go
func (c *EventConsumer) handleEvent(ctx context.Context, msg DormEventMessage) error {
	// Step 1: 判断是否为本楼住宿学生
	student, err := c.studentRepo.FindByStudentID(ctx, msg.StudentID)
	if err != nil {
		// 陌生人处理
		return c.handleStranger(ctx, msg)
	}

	// Step 2: 更新在校状态
	if msg.EventType == "entry" {
		c.studentRepo.UpdateStatus(ctx, msg.StudentID, map[string]interface{}{
			"is_in_dorm":      true,
			"last_entry_time": msg.Timestamp,
			"today_status":    "in",
		})
	} else {
		c.studentRepo.UpdateStatus(ctx, msg.StudentID, map[string]interface{}{
			"is_in_dorm":   false,
			"last_exit_time": msg.Timestamp,
		})
	}

	// Step 3: 写入事件记录
	event := model.EntryExitEvent{
		EventID:       msg.EventID,
		CameraID:      msg.CameraID,
		Building:      msg.Building,
		StudentID:     msg.StudentID,
		StudentName:   msg.StudentName,
		EventType:     msg.EventType,
		Confidence:    msg.Confidence,
		IsStranger:    false,
		Timestamp:     msg.Timestamp,
	}
	return c.eventLogRepo.Create(ctx, &event)
}

func (c *EventConsumer) handleStranger(ctx context.Context, msg DormEventMessage) error {
	// 记录陌生人事件
	c.logger.Warn("stranger detected",
		zap.String("building", msg.Building),
		zap.Float64("confidence", msg.Confidence),
	)
	// 创建陌生人告警
	alert := model.AlertRecord{
		AlertID:     fmt.Sprintf("alert_%s_%d", msg.Building, time.Now().UnixMilli()),
		AlertType:   "STRANGER_ENTRY",
		Building:    msg.Building,
		Severity:    "high",
		Description: fmt.Sprintf("陌生人进入 %s 栋", msg.Building),
		OccurredAt:  msg.Timestamp,
	}
	return c.alertRepo.Create(ctx, &alert)
}
```

---

## 5. Redis 缓存设计

### 5.1 Key 命名规范

```go
// internal/redis/keys.go
package redis

import "fmt"

const (
	PrefixStudentStatus = "dorm:student:%s:status"
	PrefixBuildingStudents = "dorm:building:%s:students"
	PrefixEventProcessed = "dorm:event:processed:%s"
	PrefixBuildingStatus = "dorm:building:%s:status"
	PrefixTodayReport   = "dorm:report:today:%s"
	PrefixConfig        = "dorm:config"
)

func StudentStatusKey(studentID string) string {
	return fmt.Sprintf(PrefixStudentStatus, studentID)
}

func BuildingStudentsKey(building string) string {
	return fmt.Sprintf(PrefixBuildingStudents, building)
}

func EventProcessedKey(eventID string) string {
	return fmt.Sprintf(PrefixEventProcessed, eventID)
}

func BuildingStatusKey(building string) string {
	return fmt.Sprintf(PrefixBuildingStatus, building)
}

func TodayReportKey(building string) string {
	return fmt.Sprintf(PrefixTodayReport, building)
}
```

### 5.2 状态更新操作

```go
// 更新学生在校状态 (entry / exit)
func (c *Client) UpdateStudentStatus(ctx context.Context, studentID string, fields map[string]interface{}) error {
	key := StudentStatusKey(studentID)
	for field, value := range fields {
		if err := c.rdb.HSet(ctx, key, field, value).Err(); err != nil {
			return err
		}
	}
	// TTL 次日 06:00 过期
	tomorrow6am := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Add(6 * time.Hour)
	return c.rdb.ExpireAt(ctx, key, tomorrow6am).Err()
}

// 查询学生状态
func (c *Client) GetStudentStatus(ctx context.Context, studentID string) (map[string]string, error) {
	key := StudentStatusKey(studentID)
	return c.rdb.HGetAll(ctx, key).Result()
}

// 批量查询楼栋学生
func (c *Client) GetBuildingStudents(ctx context.Context, building string) ([]string, error) {
	key := BuildingStudentsKey(building)
	return c.rdb.SMembers(ctx, key).Result()
}

// 幂等检查
func (c *Client) CheckAndSetDedup(ctx context.Context, cameraID, frameSeq string) (bool, error) {
	key := fmt.Sprintf("dedup:%s:%s", cameraID, frameSeq)
	set, err := c.rdb.SetNX(ctx, key, "1", 3600*time.Second).Result()
	return set, err
}
```

---

## 6. 调度任务设计

### 6.1 每晚查宿统计

```go
// internal/scheduler/nightly_report_job.go
package scheduler

type NightlyReportJob struct {
	logger        *zap.Logger
	reportService *service.ReportService
}

func NewNightlyReportJob(logger *zap.Logger, reportService *service.ReportService) *NightlyReportJob {
	return &NightlyReportJob{logger: logger, reportService: reportService}
}

// Run 默认 23:00 执行
func (j *NightlyReportJob) Run() {
	j.logger.Info("=== 开始每晚查宿统计 ===")
	if err := j.reportService.GenerateForAllBuildings(time.Now()); err != nil {
		j.logger.Error("每晚查宿统计失败", zap.Error(err))
		return
	}
	j.logger.Info("=== 每晚查宿统计完成 ===")
}
```

### 6.2 学管数据同步

```go
// internal/scheduler/sync_student_job.go
package scheduler

type SyncStudentJob struct {
	logger      *zap.Logger
	syncService *service.SyncService
}

func NewSyncStudentJob(logger *zap.Logger, syncService *service.SyncService) *SyncStudentJob {
	return &SyncStudentJob{logger: logger, syncService: syncService}
}

// Run 默认每 60 分钟执行
func (j *SyncStudentJob) Run() {
	if !j.syncService.IsEnabled() {
		return
	}
	j.logger.Info("开始同步学管宿舍数据...")
	result, err := j.syncService.SyncFromSIMS()
	if err != nil {
		j.logger.Error("同步失败", zap.Error(err))
		return
	}
	j.logger.Info("同步完成", zap.Any("result", result))
}
```

### 6.3 摄像头健康检查

```go
// internal/scheduler/health_check_job.go
package scheduler

type HealthCheckJob struct {
	logger        *zap.Logger
	cameraService *service.CameraService
}

func NewHealthCheckJob(logger *zap.Logger, cameraService *service.CameraService) *HealthCheckJob {
	return &HealthCheckJob{logger: logger, cameraService: cameraService}
}

// Run 每 30 秒检查一次摄像头状态
func (j *HealthCheckJob) Run() {
	j.cameraService.CheckAllCameras()
}
```

### 6.4 调度管理器

```go
// internal/scheduler/manager.go
package scheduler

import "github.com/robfig/cron/v3"

type Manager struct {
	logger *zap.Logger
	cron   *cron.Cron
}

func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger: logger,
		cron:   cron.New(cron.WithSeconds()),
	}
}

func (m *Manager) AddJob(spec string, job cron.Job) {
	m.cron.AddJob(spec, job)
	m.logger.Info("scheduled job", zap.String("spec", spec))
}

func (m *Manager) Start() {
	m.cron.Start()
}

func (m *Manager) Stop() {
	m.cron.Stop()
}
```

---

## 7. 异常处理框架

### 7.1 统一响应体

```go
// internal/handler/response.go
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"requestId,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      200,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: c.GetString("requestId"),
	})
}

func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: c.GetString("requestId"),
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 400, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 404, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 500, message)
}
```

### 7.2 业务错误定义

```go
// internal/handler/errors.go
package handler

import "errors"

var (
	ErrInvalidParameter  = errors.New("请求参数不合法")
	ErrUnauthorized      = errors.New("认证失败")
	ErrNotFound          = errors.New("资源不存在")
	ErrConflict          = errors.New("数据冲突")
	ErrBuildingInvalid   = errors.New("楼栋参数不合法，仅支持 A/B/C/D")
	ErrStudentNotFound   = errors.New("未找到该学生")
	ErrReportExists      = errors.New("该日期已存在查宿统计")
	ErrSyncInProgress    = errors.New("同步任务执行中")
	ErrCameraLimit       = errors.New("摄像头数量已达上限")
	ErrInternal          = errors.New("服务器内部错误")
	ErrServiceUnavailable = errors.New("服务暂不可用")
)

type BusinessError struct {
	HTTPStatus int
	Code       int
	Message    string
}

func (e *BusinessError) Error() string {
	return e.Message
}
```

### 7.3 全局错误处理中间件

```go
// internal/middleware/error_handler.go
package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"your.module/internal/handler"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				handler.InternalError(c, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// Handler 层统一错误处理模式
func handleServiceError(c *gin.Context, err error) {
	var bizErr *handler.BusinessError
	if errors.As(err, &bizErr) {
		handler.Error(c, bizErr.HTTPStatus, bizErr.Code, bizErr.Message)
		return
	}
	handler.InternalError(c, err.Error())
}
```

### 7.4 Gin 参数校验

```go
// internal/model/dto/camera_dto.go
package dto

type RegisterCameraRequest struct {
	CameraID  string `json:"cameraId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Building  string `json:"building" binding:"required,oneof=A B C D"`
	RTSPURL   string `json:"rtspUrl" binding:"required"`
	Direction string `json:"direction"`
	Resolution string `json:"resolution"`
	Remark    string `json:"remark"`
}

// handler 中使用
func (h *CameraHandler) RegisterCamera(c *gin.Context) {
	var req dto.RegisterCameraRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.BadRequest(c, err.Error())
		return
	}
	// ...
}
```

### 7.5 错误码速查

| HTTP 状态码 | 业务错误码            | 说明                 |
| ----------- | --------------------- | -------------------- |
| 400         | INVALID_PARAMETER     | 请求参数校验失败     |
| 400         | BUILDING_INVALID      | 楼栋参数不合法       |
| 400         | CAMERA_LIMIT_EXCEEDED | 摄像头数量已达上限   |
| 401         | UNAUTHORIZED          | 认证失败             |
| 404         | NOT_FOUND             | 资源不存在           |
| 404         | STUDENT_NOT_FOUND     | 学生未找到           |
| 409         | CONFLICT              | 数据冲突             |
| 409         | REPORT_ALREADY_EXISTS | 该日期已存在查宿统计 |
| 409         | SYNC_IN_PROGRESS      | 同步任务执行中       |
| 422         | UNPROCESSABLE_ENTITY  | 业务规则校验失败     |
| 429         | TOO_MANY_REQUESTS     | 请求频率超限         |
| 500         | INTERNAL_ERROR        | 服务器内部错误       |
| 503         | SERVICE_UNAVAILABLE   | 服务暂不可用         |

---

## 8. 日志规范

### 8.1 zap 日志配置

```go
// cmd/dormitory-service/main.go
func initLogger(level string) (*zap.Logger, error) {
	var lvl zapcore.Level
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      lvl == zapcore.DebugLevel,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return cfg.Build()
}
```

### 8.2 关键日志埋点

| 位置       | 日志事件       | 级别  | 说明                                |
| ---------- | -------------- | ----- | ----------------------------------- |
| 事件消费   | 收到/处理完成  | INFO  | 包含 event_id、building、event_type |
| 事件消费   | 解析失败/异常  | ERROR | 包含原始消息体                      |
| 状态更新   | 更新成功       | DEBUG | 包含 student_id + 新状态            |
| 查宿统计   | 开始/完成/失败 | INFO  | 包含 date、各楼栋计数               |
| 学管同步   | 开始/成功/失败 | INFO  | 包含 sync_id、数据量                |
| 告警触发   | 告警创建       | WARN  | 包含 alert_type、building           |
| 摄像头检查 | 状态变更       | INFO  | 包含 camera_id、old/new status      |
| 配置更新   | 更新成功       | INFO  | 包含 key、old/new value             |

### 8.3 日志使用示例

```go
// Kafka 消费日志
c.logger.Info("event processed",
	zap.String("event_id", msg.EventID),
	zap.String("building", msg.Building),
	zap.String("event_type", msg.EventType),
)

// 错误日志
c.logger.Error("process event failed",
	zap.Error(err),
	zap.String("event_id", msg.EventID),
	zap.String("raw_message", string(msg.Value)),
)

// 调试日志
c.logger.Debug("duplicate event, skip",
	zap.String("event_id", msg.EventID),
)
```

---

## 9. 包结构总结

```
github.com/sims/campusvision/dormitory-service-go
├── cmd/dormitory-service/main.go    # 入口：DI + 路由 + 启动
├── internal/
│   ├── client/                      # PushClient → stream-gateway
│   ├── config/                      # Viper 配置加载
│   ├── consumer/                    # Kafka 消费者
│   ├── handler/                     # Gin HTTP 处理器
│   ├── middleware/                  # JWT + CORS 中间件
│   ├── model/
│   │   ├── dto/                     # 请求/响应类型
│   │   ├── entity/                  # 数据库实体 (db tag)
│   │   └── enums/                   # 领域枚举
│   ├── redis/                       # go-redis 封装 + 去重
│   ├── repository/                  # sqlx 数据访问 (泛型 BaseRepository)
│   ├── service/                     # 业务逻辑
│   ├── scheduler/                   # robfig/cron 定时任务
│   └── util/                        # 工具 (AES 加密)
```

---

> **本文件属于**: `doc/design/backend/01-architecture.md`  
> **面向读者**: Go 后端开发（搭档）  
> **参考**: PRD-004 主进程对接、PRD-003 Dormitory Service
