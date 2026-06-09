package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/enums"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
	redisclient "github.com/sims/campusvision/dormitory-service-go/internal/redis"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// EventConsumer consumes t_dorm_event messages from Kafka and processes them
// into the dormitory-service database with Redis-based deduplication.
type EventConsumer struct {
	logger *zap.Logger
	rdb    *redisclient.Client
	reader *kafka.Reader

	buildingRepo *repository.BuildingRepository
	eventLogRepo *repository.EventLogRepository
	studentRepo  *repository.StudentRepository
	alertRepo    *repository.AlertRepository
	strangerRepo *repository.StrangerRecordRepository
	cameraRepo   *repository.CameraRepository

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
			// Do not commit offset — message will be re-delivered for retry.
			// Deserialisation and validation failures return nil from processMessage
			// (poison committed), so only DB failures reach this path.
			continue
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
		c.logger.Error("Failed to deserialize event message, committing offset to skip poison pill",
			zap.Error(err),
			zap.Int64("offset", msg.Offset),
		)
		return nil
	}

	c.logger.Debug("Received event",
		zap.String("camera_id", event.CameraID),
		zap.String("event_type", event.EventType),
		zap.String("student_id", event.StudentID),
		zap.Bool("is_stranger", event.IsStranger),
		zap.String("source", event.Source),
	)

	// 2. Validate
	if event.CameraID == "" {
		c.logger.Error("Event missing camera_id, committing offset to skip poison pill",
			zap.Int64("offset", msg.Offset),
		)
		return nil
	}
	if event.EventType == "" {
		c.logger.Error("Event missing event_type, committing offset to skip poison pill",
			zap.Int64("offset", msg.Offset),
		)
		return nil
	}

	// 3. Source-based routing — non-face events skip all face-specific processing
	// (no eventlog, no face match, no stranger logic, no camera status update, no dedup)
	if event.Source == "behavior" || event.Source == "alert" {
		c.logger.Debug("Skipping face-specific processing for non-face event",
			zap.String("source", event.Source),
			zap.String("camera_id", event.CameraID),
			zap.String("event_type", event.EventType),
		)
		return nil
	}

	// 4. Pre-DB dedup check (read-only — don't set yet)
	isExisting, err := c.rdb.ExistsDedup(ctx, event.CameraID, event.FrameSequence)
	if err != nil {
		c.logger.Warn("Dedup pre-check failed, processing anyway",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
			zap.Int("frame_sequence", event.FrameSequence),
		)
		// Proceed on Redis error (defensive)
	} else if isExisting {
		c.logger.Debug("Skipping duplicate event",
			zap.String("camera_id", event.CameraID),
			zap.Int("frame_sequence", event.FrameSequence),
		)
		return nil
	}

	// 5. Persist event log
	eventLog := c.buildEventLog(event, event.Building)
	if _, err := c.eventLogRepo.Create(ctx, eventLog); err != nil {
		return fmt.Errorf("persist event log: %w", err)
	}

	// 6. Stranger detection
	if event.IsStranger {
		if err := c.handleStrangerEvent(ctx, event); err != nil {
			return fmt.Errorf("stranger event processing: %w", err)
		}
	}

	// 7. Update camera last_event_time
	if err := c.cameraRepo.UpdateLastEventTime(ctx, event.CameraID); err != nil {
		return fmt.Errorf("update camera last_event_time: %w", err)
	}

	// 8. Set dedup AFTER DB persistence succeeds
	if _, err := c.rdb.CheckAndSetDedupEventID(ctx, eventLog.EventID); err != nil {
		c.logger.Warn("Failed to set dedup mark after DB persistence",
			zap.Error(err),
			zap.String("event_id", eventLog.EventID),
		)
		// Non-fatal — DB already has the event
	}

	return nil
}

// buildEventLog creates a DormEventLog entity from the incoming event message.
func (c *EventConsumer) buildEventLog(event dto.FaceEventMessage, buildingCode string) *entity.DormEventLog {
	eventLog := &entity.DormEventLog{
		EventID:     fmt.Sprintf("evt-%s-%d", event.CameraID, event.FrameSequence),
		EventType:   event.EventType,
		IsStranger:  event.IsStranger,
		IsProcessed: true,
		Building:    buildingCode,
		CreatedAt:   time.Now(),
	}

	if event.Timestamp > 0 {
		eventLog.Timestamp = time.UnixMilli(event.Timestamp)
	} else {
		eventLog.Timestamp = time.Now()
	}
	if event.CameraID != "" {
		eventLog.CameraID = jsontype.NewNullString(event.CameraID)
	}
	if event.StudentID != "" {
		eventLog.StudentID = jsontype.NewNullString(event.StudentID)
	}
	if event.Name != "" {
		eventLog.StudentName = jsontype.NewNullString(event.Name)
	}
	if event.Confidence > 0 {
		eventLog.Confidence = jsontype.NewNullFloat64(event.Confidence)
	}
	if event.SnapshotPath != "" {
		eventLog.FaceSnapshotURL = jsontype.NewNullString(event.SnapshotPath)
<<<<<<< HEAD
	}
	if event.Source != "" {
		eventLog.Source = event.Source
=======
>>>>>>> 2fee2d0 (fix(model): 自定义JSON Null序列化类型(sql.Null* -> jsontype.Null*))
	}

	return eventLog
}

// handleStrangerEvent creates an alert and a stranger record when an unknown person is detected.
// Returns an error if the alert creation fails (fatal). Stranger record creation failure is
// non-fatal: it is logged but does not return an error.
func (c *EventConsumer) handleStrangerEvent(ctx context.Context, event dto.FaceEventMessage) error {
	now := time.Now()

	// Create alert record
	alert := &entity.DormAlert{
		AlertType: string(enums.AlertTypeStrangerEntry),
		Building:  jsontype.NewNullString(event.Building),
		Severity:  string(enums.SeverityMedium),
		Description: jsontype.NewNullString(
			fmt.Sprintf("Stranger detected at camera %s (building %s)", event.CameraID, event.Building),
		),
		IsRead:     false,
		IsResolved: false,
		OccurredAt: now,
		CreatedAt:  now,
	}
	if event.StudentID != "" {
		alert.StudentID = jsontype.NewNullString(event.StudentID)
	}
	if event.SnapshotPath != "" {
		alert.FaceSnapshotURL = jsontype.NewNullString(event.SnapshotPath)
	}

	if _, err := c.alertRepo.Create(ctx, alert); err != nil {
		c.logger.Error("Failed to create stranger alert",
			zap.Error(err),
			zap.String("camera_id", event.CameraID),
		)
		return fmt.Errorf("create stranger alert: %w", err)
	}

	// Create stranger record (non-fatal)
	strangerRecord := &entity.DormStrangerRecord{
		Building:     event.Building,
		EventType:    event.EventType,
		DetectedTime: now,
		Status:       string(enums.StrangerStatusUnconfirmed),
		CreatedAt:    now,
	}
	if event.SnapshotPath != "" {
		strangerRecord.FaceSnapshotURL = jsontype.NewNullString(event.SnapshotPath)
	}
	if event.Confidence > 0 {
		strangerRecord.Confidence = jsontype.NewNullFloat64(event.Confidence)
	}
	if event.Name != "" {
		strangerRecord.Remark = jsontype.NewNullString(
			fmt.Sprintf("Detected at %s: name=%s, camera=%s", now.Format(time.RFC3339), event.Name, event.CameraID),
		)
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
	return nil
}
