package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/dto"
	redisclient "github.com/sims/campusvision/dormitory-service-go/internal/redis"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
)

// newTestEventConsumer creates an EventConsumer backed by sqlmock for DB
// and a fast-failing Redis client (no actual Redis needed). Tests call
// processMessage directly (bypassing the Kafka reader) and inspect the
// returned error to verify offset-commit decisions.
func newTestEventConsumer(t *testing.T) (sqlmock.Sqlmock, *EventConsumer) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = mockDB.Close() })

	sqlxDB := sqlx.NewDb(mockDB, "mysql")

	// Fast-failing Redis client — dial to port 1 fails immediately,
	// consumer logs a warning and continues (defensive behaviour).
	rdb := &redisclient.Client{
		Client: goredis.NewClient(&goredis.Options{
			Addr:        "127.0.0.1:1",
			DialTimeout: time.Millisecond,
		}),
	}
	t.Cleanup(func() { _ = rdb.Close() })

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"127.0.0.1:1"},
		Topic:   "test",
		GroupID: "test-group",
		MaxWait: 10 * time.Millisecond,
	})
	t.Cleanup(func() { _ = reader.Close() })

	consumer := &EventConsumer{
		logger:       zap.NewNop(),
		rdb:          rdb,
		reader:       reader,
		buildingRepo: repository.NewBuildingRepository(sqlxDB),
		eventLogRepo: repository.NewEventLogRepository(sqlxDB),
		studentRepo:  repository.NewStudentRepository(sqlxDB),
		alertRepo:    repository.NewAlertRepository(sqlxDB),
		strangerRepo: repository.NewStrangerRecordRepository(sqlxDB),
		cameraRepo:   repository.NewCameraRepository(sqlxDB),
	}

	return mock, consumer
}

// buildTestMessage creates a kafka.Message with the given event serialized as JSON.
func buildTestMessage(t *testing.T, event dto.FaceEventMessage) kafka.Message {
	t.Helper()
	data, err := json.Marshal(event)
	require.NoError(t, err)
	return kafka.Message{
		Value: data,
	}
}

// ---------- poison message (Unmarshal failure) ----------

func TestEventConsumer_processMessage_PoisonJSON(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// No DB expectations — poison messages should never reach a repository.
	msg := kafka.Message{
		Value: []byte(`{invalid json`),
		Topic: "t_dorm_event",
	}

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "poison messages must return nil so the offset is committed")
	assert.NoError(t, mock.ExpectationsWereMet(), "no DB operation should have been attempted")
}

// ---------- eventLogRepo.Create failure ----------

func TestEventConsumer_processMessage_EventLogCreateFails(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// Expect the INSERT and make it fail.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnError(fmt.Errorf("database connection lost"))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-1",
		Building:      "A",
		EventType:     "entry",
		FrameSequence: 1,
	})

	err := c.processMessage(context.Background(), msg)
	assert.Error(t, err, "processMessage must return error when event log creation fails")
	assert.Contains(t, err.Error(), "database connection lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- cameraRepo.UpdateLastEventTime failure ----------

func TestEventConsumer_processMessage_CameraUpdateFails(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// 1. Event log INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnResult(sqlmock.NewResult(100, 1))
	// 2. Camera UPDATE fails.
	mock.ExpectExec(`UPDATE dorm_camera SET last_event_time = NOW\(\)`).
		WillReturnError(fmt.Errorf("database timeout"))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-1",
		Building:      "A",
		EventType:     "entry",
		FrameSequence: 10,
	})

	err := c.processMessage(context.Background(), msg)
	assert.Error(t, err, "processMessage must return error when camera update fails")
	assert.Contains(t, err.Error(), "database timeout")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- alertRepo.Create failure (stranger event) ----------

func TestEventConsumer_processMessage_StrangerAlertFails(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// 1. Event log INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnResult(sqlmock.NewResult(101, 1))
	// 2. Alert INSERT fails.
	mock.ExpectExec(`INSERT INTO dorm_alert_record \(`).
		WillReturnError(fmt.Errorf("alert table full"))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-2",
		Building:      "B",
		EventType:     "entry",
		FrameSequence: 20,
		IsStranger:    true,
	})

	err := c.processMessage(context.Background(), msg)
	assert.Error(t, err, "processMessage must return error when stranger alert creation fails")
	assert.Contains(t, err.Error(), "alert table full")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- happy path ----------

func TestEventConsumer_processMessage_HappyPath(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// 1. Event log INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnResult(sqlmock.NewResult(102, 1))
	// 2. Camera UPDATE succeeds.
	mock.ExpectExec(`UPDATE dorm_camera SET last_event_time = NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-3",
		Building:      "C",
		EventType:     "exit",
		FrameSequence: 30,
	})

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "happy path must return nil")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- happy path with stranger (alert + stranger record) ----------

func TestEventConsumer_processMessage_HappyPathStranger(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// 1. Event log INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnResult(sqlmock.NewResult(103, 1))
	// 2. Alert INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_alert_record \(`).
		WillReturnResult(sqlmock.NewResult(10, 1))
	// 3. Stranger record INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_stranger_record \(`).
		WillReturnResult(sqlmock.NewResult(20, 1))
	// 4. Camera UPDATE succeeds.
	mock.ExpectExec(`UPDATE dorm_camera SET last_event_time = NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-4",
		Building:      "D",
		EventType:     "entry",
		FrameSequence: 40,
		IsStranger:    true,
	})

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "happy path with stranger must return nil")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- behavior source routing: no DB calls ----------

func TestEventConsumer_processMessage_BehaviorSourceSkipsProcessing(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// No DB expectations — behavior events should never reach a repository.
	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-b1",
		Building:      "A",
		EventType:     "entry",
		FrameSequence: 0,
		Source:        "behavior",
	})

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "behavior events must return nil (skip all processing)")
	assert.NoError(t, mock.ExpectationsWereMet(), "no DB operation should have been attempted")
}

// ---------- alert source routing: no DB calls ----------

func TestEventConsumer_processMessage_AlertSourceSkipsProcessing(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// No DB expectations — alert events should never reach a repository.
	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-a1",
		Building:      "B",
		EventType:     "alert",
		FrameSequence: 0,
		Source:        "alert",
	})

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "alert events must return nil (skip all processing)")
	assert.NoError(t, mock.ExpectationsWereMet(), "no DB operation should have been attempted")
}

// ---------- face source routing: normal processing (same as happy path) ----------

func TestEventConsumer_processMessage_FaceSourceFullProcessing(t *testing.T) {
	mock, c := newTestEventConsumer(t)

	// 1. Event log INSERT succeeds.
	mock.ExpectExec(`INSERT INTO dorm_entry_exit_event \(`).
		WillReturnResult(sqlmock.NewResult(104, 1))
	// 2. Camera UPDATE succeeds.
	mock.ExpectExec(`UPDATE dorm_camera SET last_event_time = NOW\(\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	msg := buildTestMessage(t, dto.FaceEventMessage{
		CameraID:      "cam-f1",
		Building:      "A",
		EventType:     "entry",
		FrameSequence: 50,
		Source:        "face",
	})

	err := c.processMessage(context.Background(), msg)
	assert.NoError(t, err, "face source events must process normally")
	assert.NoError(t, mock.ExpectationsWereMet())
}
