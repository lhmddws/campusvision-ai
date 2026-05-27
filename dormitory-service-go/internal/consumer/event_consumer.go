package consumer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/enums"
	redisclient "github.com/sims/campusvision/dormitory-service-go/internal/redis"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// buildingCacheTTL is how long a resolved building code→ID mapping stays in Redis.
const buildingCacheTTL = 1 * time.Hour

// buildingCacheKeyPrefix is the Redis key prefix for building code→ID lookups.
const buildingCacheKeyPrefix = "dorm:building:code:"

// EventConsumer consumes t_dorm_event messages from Kafka and processes them
// into the dormitory-service database with Redis-based deduplication.
type EventConsumer struct {
	logger       *zap.Logger
	rdb          *redisclient.Client
	reader       *kafka.Reader

	buildingRepo  *repository.BuildingRepository
	eventLogRepo  *repository.EventLogRepository
	studentRepo   *repository.StudentRepository
	alertRepo     *repository.AlertRepository
	strangerRepo  *repository.StrangerRecordRepository
	cameraRepo    *repository.CameraRepository

	maxPollRecords int
	cancel         context.CancelFunc
}

// NewEventConsumer creates a new EventConsumer.
func NewEventConsumer(
	logger *zap.Logger,
	rdb *redisclient.Client,
	brokers []string,
	topic string,
	groupID string,
	maxPollRecords int,
	buildingRepo *repository.BuildingRepository,
	eventLogRepo *repository.EventLogRepository,
	studentRepo *repository.StudentRepository,
	alertRepo *repository.AlertRepository,
	strangerRepo *repository.StrangerRecordRepository,
	cameraRepo *repository.CameraRepository,
) *EventConsumer {
	if maxPollRecords <= 0 {
		maxPollRecords = 500
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,       // 1 byte
		MaxBytes:    5 << 20, // 5MB (matches t_dorm_frame max message size)
		MaxWait:     1 * time.Second,
	})

	return &EventConsumer{
		logger:         logger,
		rdb:            rdb,
		reader:         reader,
		buildingRepo:   buildingRepo,
		eventLogRepo:   eventLogRepo,
		studentRepo:    studentRepo,
		alertRepo:      alertRepo,
		strangerRepo:   strangerRepo,
		cameraRepo:     cameraRepo,
		maxPollRecords: maxPollRecords,
	}
}

// Start launches the consumer loop in a background goroutine.
func (c *EventConsumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.consumeLoop(ctx)
	c.logger.Info("Event consumer started",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group_id", c.reader.Config().GroupID),
	)
}

// Stop gracefully shuts down the consumer.
func (c *EventConsumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	return c.reader.Close()
}

// consumeLoop reads messages from Kafka in a loop until context is cancelled.
func (c *EventConsumer) consumeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Event consumer loop exiting due to context cancellation")
			return
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.logger.Error("Failed to read Kafka message", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := c.processMessage(ctx, msg); err != nil {
			c.logger.Error("Failed to process event message",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
				zap.Int("partition", msg.Partition),
			)
			// Commit anyway to avoid re-processing poison messages
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Warn("Failed to commit Kafka message offset",
				zap.Error(err),
				zap.Int64("offset", msg.Offset),
			)
		}
	}
}

// processMessage handles a single Kafka event message through the full pipeline.
func (c *EventConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	// 1. Deserialize
	var event dto.FaceEventMessage
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("deserialize event: %w", err)
	}

	c.logger.Debug("Received event",
		zap.String("camera_id", event.CameraID),
		zap.String("event_type", event.EventType),
		zap.String("student_id", event.StudentID),
		zap.Bool("is_stranger", event.IsStranger),
	)

	// 2. Validate
	if event.CameraID == "" {
		return fmt.Errorf("event missing camera_id")
	}
	if event.EventType == "" {
		return fmt.Errorf("event missing event_type")
	}

	// 3. Redis dedup
	isNew, err := c.rdb.CheckAndSetDedup(ctx, event.CameraID, event.FrameSequence)
	if err != nil {
		c.logger.Warn("Dedup check failed, processing anyway",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
			zap.Int("frame_sequence", event.FrameSequence),
		)
		// Proceed on Redis error (defensive)
	} else if !isNew {
		c.logger.Debug("Skipping duplicate event",
			zap.String("camera_id", event.CameraID),
			zap.Int("frame_sequence", event.FrameSequence),
		)
		return nil
	}

	// 5. Persist event log
	eventLog := c.buildEventLog(event, event.Building)
	if _, err := c.eventLogRepo.Create(ctx, eventLog); err != nil {
		c.logger.Error("Failed to persist event log",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
		)
	}

	// 6. Stranger detection
	if event.IsStranger {
		c.handleStrangerEvent(ctx, event)
	}

	// 7. Update camera last_event_time
	if err := c.cameraRepo.UpdateLastEventTime(ctx, event.CameraID); err != nil {
		c.logger.Warn("Failed to update camera last_event_time",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
		)
	}

	return nil
}

// resolveBuildingID converts a building code (A/B/C/D) to a numeric building ID.
// Uses Redis cache → DB fallback.
func (c *EventConsumer) resolveBuildingID(ctx context.Context, buildingCode string) int64 {
	if buildingCode == "" {
		return 0
	}

	code := strings.ToUpper(strings.TrimSpace(buildingCode))
	cacheKey := buildingCacheKeyPrefix + code

	// Check Redis cache
	cachedID, err := c.rdb.Get(ctx, cacheKey)
	if err == nil && cachedID != "" {
		id, parseErr := strconv.ParseInt(cachedID, 10, 64)
		if parseErr == nil {
			return id
		}
	}

	// Cache miss — query dorm_building table via repository
	building, err := c.buildingRepo.FindByCode(ctx, code)
	if err != nil {
		c.logger.Warn("Building code not found in database",
			zap.String("code", code),
			zap.Error(err),
		)
		return 0
	}

	// Cache for 1 hour
	if cacheErr := c.rdb.Set(ctx, cacheKey, strconv.FormatInt(building.ID, 10), buildingCacheTTL); cacheErr != nil {
		c.logger.Warn("Failed to cache building ID",
			zap.Error(cacheErr),
			zap.String("code", code),
		)
	}

	return building.ID
}

// buildEventLog creates a DormEventLog entity from the incoming event message.
func (c *EventConsumer) buildEventLog(event dto.FaceEventMessage, buildingCode string) *entity.DormEventLog {
	eventLog := &entity.DormEventLog{
		EventID:    fmt.Sprintf("evt-%s-%d", event.CameraID, event.FrameSequence),
		EventType:  event.EventType,
		IsStranger: event.IsStranger,
		IsProcessed: true,
		Building:   buildingCode,
		CreatedAt:  time.Now(),
	}

	if event.Timestamp > 0 {
		eventLog.Timestamp = time.UnixMilli(event.Timestamp)
	} else {
		eventLog.Timestamp = time.Now()
	}
	if event.CameraID != "" {
		eventLog.CameraID = sql.NullString{String: event.CameraID, Valid: true}
	}
	if event.StudentID != "" {
		eventLog.StudentID = sql.NullString{String: event.StudentID, Valid: true}
	}
	if event.Name != "" {
		eventLog.StudentName = sql.NullString{String: event.Name, Valid: true}
	}
	if event.Confidence > 0 {
		eventLog.Confidence = sql.NullFloat64{Float64: event.Confidence, Valid: true}
	}
	if event.SnapshotPath != "" {
		eventLog.FaceSnapshotURL = sql.NullString{String: event.SnapshotPath, Valid: true}
	}

	return eventLog
}

// handleStrangerEvent creates an alert and a stranger record when an unknown person is detected.
func (c *EventConsumer) handleStrangerEvent(ctx context.Context, event dto.FaceEventMessage) {
	now := time.Now()

	// Create alert record
	alert := &entity.DormAlert{
		AlertType: string(enums.AlertTypeStrangerEntry),
		Building:  sql.NullString{String: event.Building, Valid: event.Building != ""},
		Severity:  string(enums.SeverityMedium),
		Description: sql.NullString{
			String: fmt.Sprintf("Stranger detected at camera %s (building %s)", event.CameraID, event.Building),
			Valid:  true,
		},
		IsRead:     false,
		IsResolved: false,
		OccurredAt: now,
		CreatedAt:  now,
	}
	if event.StudentID != "" {
		alert.StudentID = sql.NullString{String: event.StudentID, Valid: true}
	}
	if event.SnapshotPath != "" {
		alert.FaceSnapshotURL = sql.NullString{String: event.SnapshotPath, Valid: true}
	}

	if _, err := c.alertRepo.Create(ctx, alert); err != nil {
		c.logger.Error("Failed to create stranger alert",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
		)
	}

	// Create stranger record
	strangerRecord := &entity.DormStrangerRecord{
		Building:  event.Building,
		EventType: event.EventType,
		DetectedTime: now,
		Status:    string(enums.StrangerStatusUnconfirmed),
		CreatedAt: now,
	}
	if event.SnapshotPath != "" {
		strangerRecord.FaceSnapshotURL = sql.NullString{String: event.SnapshotPath, Valid: true}
	}
	if event.Confidence > 0 {
		strangerRecord.Confidence = sql.NullFloat64{Float64: event.Confidence, Valid: true}
	}
	if event.Name != "" {
		strangerRecord.Remark = sql.NullString{
			String: fmt.Sprintf("Detected at %s: name=%s, camera=%s", now.Format(time.RFC3339), event.Name, event.CameraID),
			Valid:  true,
		}
	}

	if _, err := c.strangerRepo.Create(ctx, strangerRecord); err != nil {
		c.logger.Error("Failed to create stranger record",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
		)
	}

	c.logger.Warn("Stranger detected",
		zap.String("camera_id", event.CameraID),
		zap.String("building", event.Building),
		zap.String("event_type", event.EventType),
	)
}
