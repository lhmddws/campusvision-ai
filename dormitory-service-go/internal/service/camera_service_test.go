package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
	"github.com/sims/campusvision/dormitory-service-go/internal/model/jsontype"
)

// MockCameraRepository is a mock implementation of CameraRepository interface
// for testing GetCamerasStatus method
//go:generate mockery --name CameraRepository --filename mock_camera_repository.go

type MockCameraRepository struct {
	mock.Mock
}

func (m *MockCameraRepository) FindByBuilding(ctx context.Context, building string) ([]entity.DormCamera, error) {
	args := m.Called(ctx, building)
	return args.Get(0).([]entity.DormCamera), args.Error(1)
}

func (m *MockCameraRepository) FindAll(ctx context.Context, orderBy ...string) ([]entity.DormCamera, error) {
	args := m.Called(ctx, orderBy)
	return args.Get(0).([]entity.DormCamera), args.Error(1)
}

func (m *MockCameraRepository) FindByCameraID(ctx context.Context, cameraID string) (*entity.DormCamera, error) {
	args := m.Called(ctx, cameraID)
	return args.Get(0).(*entity.DormCamera), args.Error(1)
}

func (m *MockCameraRepository) Update(ctx context.Context, camera *entity.DormCamera) error {
	args := m.Called(ctx, camera)
	return args.Error(0)
}

func (m *MockCameraRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCameraRepository) FindByStatus(ctx context.Context, status string) ([]entity.DormCamera, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]entity.DormCamera), args.Error(1)
}

func (m *MockCameraRepository) FindEnabled(ctx context.Context) ([]entity.DormCamera, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.DormCamera), args.Error(1)
}

func (m *MockCameraRepository) UpdateStatus(ctx context.Context, cameraID string, status string, fps float64, totalFrames int64) error {
	args := m.Called(ctx, cameraID, status, fps, totalFrames)
	return args.Error(0)
}

func (m *MockCameraRepository) UpdateLastEventTime(ctx context.Context, cameraID string) error {
	args := m.Called(ctx, cameraID)
	return args.Error(0)
}

func (m *MockCameraRepository) UpdateHealthCheck(ctx context.Context, cameraID string) error {
	args := m.Called(ctx, cameraID)
	return args.Error(0)
}

func (m *MockCameraRepository) FindWithPagination(ctx context.Context, building string, page, size int) ([]entity.DormCamera, int64, error) {
	args := m.Called(ctx, building, page, size)
	return args.Get(0).([]entity.DormCamera), args.Get(1).(int64), args.Error(2)
}

// MockBuildingRepository is a mock implementation of BuildingRepository interface
// for testing GetCamerasStatus method
//go:generate mockery --name BuildingRepository --filename mock_building_repository.go

type MockBuildingRepository struct {
	mock.Mock
}

func (m *MockBuildingRepository) FindAll(ctx context.Context, orderBy ...string) ([]entity.DormBuilding, error) {
	args := m.Called(ctx, orderBy)
	return args.Get(0).([]entity.DormBuilding), args.Error(1)
}

func (m *MockBuildingRepository) FindByID(ctx context.Context, id int64) (*entity.DormBuilding, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.DormBuilding), args.Error(1)
}

func (m *MockBuildingRepository) Create(ctx context.Context, building *entity.DormBuilding) (int64, error) {
	args := m.Called(ctx, building)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBuildingRepository) Update(ctx context.Context, building *entity.DormBuilding) error {
	args := m.Called(ctx, building)
	return args.Error(0)
}

func (m *MockBuildingRepository) Delete(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockBuildingRepository) FindWithPagination(ctx context.Context, where string, args []interface{}, orderBy string, page, size int) ([]entity.DormBuilding, int64, error) {
	queryArgs := []interface{}{ctx, where, args, orderBy, page, size}
	resultArgs := m.Called(queryArgs...)
	return resultArgs.Get(0).([]entity.DormBuilding), resultArgs.Get(1).(int64), resultArgs.Error(2)
}

type statusItem struct {
	Building        string      `json:"building"`
	CameraID        string      `json:"camera_id"`
	Status          string      `json:"status"`
	LastHealthCheck interface{} `json:"last_health_check"`
}

type TestCameraService struct {
	cameraRepo  *MockCameraRepository
	buildingRepo *MockBuildingRepository
}

func (s *TestCameraService) GetCamerasStatus(ctx context.Context, building string) (map[string]interface{}, error) {
	var cameras []entity.DormCamera
	var err error
	if building != "" {
		cameras, err = s.cameraRepo.FindByBuilding(ctx, building)
	} else {
		cameras, err = s.cameraRepo.FindAll(ctx)
	}
	if err != nil {
		return nil, err
	}

	items := make([]statusItem, 0, len(cameras))
	var total, online, offline, idle int
	for _, c := range cameras {
		total++
		switch c.Status {
		case "online":
			online++
		case "offline":
			offline++
		case "idle":
			idle++
		}
		var lastCheck interface{}
		if c.LastHealthCheck.Valid {
			lastCheck = c.LastHealthCheck.Time
		}
		items = append(items, statusItem{
			Building:        c.Building,
			CameraID:        c.CameraID,
			Status:          c.Status,
			LastHealthCheck: lastCheck,
		})
	}

	return map[string]interface{}{
		"total":   total,
		"online":  online,
		"offline": offline,
		"error":   idle,
		"cameras": items,
	}, nil
}

func TestGetCamerasStatus_ReturnsFlatFormat(t *testing.T) {
	// Setup
	mockCameraRepo := new(MockCameraRepository)
	mockBuildingRepo := new(MockBuildingRepository)

	service := &TestCameraService{
		cameraRepo:  mockCameraRepo,
		buildingRepo: mockBuildingRepo,
	}

	// Test data
	cameras := []entity.DormCamera{
		{
			Building:        "Building A",
			CameraID:        "cam1",
			Status:          "online",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building A",
			CameraID:        "cam2",
			Status:          "offline",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building B",
			CameraID:        "cam3",
			Status:          "online",
			LastHealthCheck: jsontype.NullTime{},
		},
	}

	// Mock expectations - FindAll is called with empty orderBy slice
	mockCameraRepo.On("FindAll", mock.Anything, mock.Anything).Return(cameras, nil)

	// Execute
	result, err := service.GetCamerasStatus(context.Background(), "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify flat format (no summary key)
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "online")
	assert.Contains(t, result, "offline")
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "cameras")

	// Verify no summary key
	assert.NotContains(t, result, "summary")

	// Verify values
	assert.Equal(t, 3, result["total"])
	assert.Equal(t, 2, result["online"])
	assert.Equal(t, 1, result["offline"])
	assert.Equal(t, 0, result["error"])

	camerasList, ok := result["cameras"].([]statusItem)
	assert.True(t, ok)
	assert.Len(t, camerasList, 3)

	// Verify mock expectations
	mockCameraRepo.AssertExpectations(t)
}

func TestGetCamerasStatus_EmptyData(t *testing.T) {
	// Setup
	mockCameraRepo := new(MockCameraRepository)
	mockBuildingRepo := new(MockBuildingRepository)

	service := &TestCameraService{
		cameraRepo:  mockCameraRepo,
		buildingRepo: mockBuildingRepo,
	}

	// Mock expectations - empty cameras (variadic orderBy, match both args)
	mockCameraRepo.On("FindAll", mock.Anything, mock.Anything).Return([]entity.DormCamera{}, nil)

	// Execute
	result, err := service.GetCamerasStatus(context.Background(), "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify flat format (no summary key)
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "online")
	assert.Contains(t, result, "offline")
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "cameras")

	// Verify no summary key
	assert.NotContains(t, result, "summary")

	// Verify zero values
	assert.Equal(t, 0, result["total"])
	assert.Equal(t, 0, result["online"])
	assert.Equal(t, 0, result["offline"])
	assert.Equal(t, 0, result["error"])

	camerasList, ok := result["cameras"].([]statusItem)
	assert.True(t, ok)
	assert.Len(t, camerasList, 0)

	// Verify mock expectations
	mockCameraRepo.AssertExpectations(t)
}

func TestGetCamerasStatus_MixedStatus(t *testing.T) {
	// Setup
	mockCameraRepo := new(MockCameraRepository)
	mockBuildingRepo := new(MockBuildingRepository)

	service := &TestCameraService{
		cameraRepo:  mockCameraRepo,
		buildingRepo: mockBuildingRepo,
	}

	// Test data with mixed statuses
	cameras := []entity.DormCamera{
		{
			Building:        "Building A",
			CameraID:        "cam1",
			Status:          "online",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building A",
			CameraID:        "cam2",
			Status:          "offline",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building B",
			CameraID:        "cam3",
			Status:          "online",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building C",
			CameraID:        "cam4",
			Status:          "idle",
			LastHealthCheck: jsontype.NullTime{},
		},
	}

	// Mock expectations (variadic orderBy, match both args)
	mockCameraRepo.On("FindAll", mock.Anything, mock.Anything).Return(cameras, nil)

	// Execute
	result, err := service.GetCamerasStatus(context.Background(), "")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify flat format (no summary key)
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "online")
	assert.Contains(t, result, "offline")
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "cameras")

	// Verify no summary key
	assert.NotContains(t, result, "summary")

	// Verify values (idle counts as error)
	assert.Equal(t, 4, result["total"])
	assert.Equal(t, 2, result["online"])
	assert.Equal(t, 1, result["offline"])
	assert.Equal(t, 1, result["error"])

	camerasList, ok := result["cameras"].([]statusItem)
	assert.True(t, ok)
	assert.Len(t, camerasList, 4)

	// Verify mock expectations
	mockCameraRepo.AssertExpectations(t)
}

func TestGetCamerasStatus_WithBuildingFilter(t *testing.T) {
	// Setup
	mockCameraRepo := new(MockCameraRepository)
	mockBuildingRepo := new(MockBuildingRepository)

	service := &TestCameraService{
		cameraRepo:  mockCameraRepo,
		buildingRepo: mockBuildingRepo,
	}

	// Test data - only cameras from Building A
	cameras := []entity.DormCamera{
		{
			Building:        "Building A",
			CameraID:        "cam1",
			Status:          "online",
			LastHealthCheck: jsontype.NullTime{},
		},
		{
			Building:        "Building A",
			CameraID:        "cam2",
			Status:          "offline",
			LastHealthCheck: jsontype.NullTime{},
		},
	}

	// Mock expectations - FindByBuilding called with "Building A"
	mockCameraRepo.On("FindByBuilding", mock.Anything, "Building A").Return(cameras, nil)

	// Execute
	result, err := service.GetCamerasStatus(context.Background(), "Building A")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify flat format (no summary key)
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "online")
	assert.Contains(t, result, "offline")
	assert.Contains(t, result, "error")
	assert.Contains(t, result, "cameras")

	// Verify no summary key
	assert.NotContains(t, result, "summary")

	// Verify values
	assert.Equal(t, 2, result["total"])
	assert.Equal(t, 1, result["online"])
	assert.Equal(t, 1, result["offline"])
	assert.Equal(t, 0, result["error"])

	camerasList, ok := result["cameras"].([]statusItem)
	assert.True(t, ok)
	assert.Len(t, camerasList, 2)

	// Verify mock expectations
	mockCameraRepo.AssertExpectations(t)
}