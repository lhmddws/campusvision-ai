package consumer

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Consumer defines the interface for a Kafka message consumer.
type Consumer interface {
	Start(ctx context.Context)
	Stop() error
}

// Manager manages the lifecycle of all Kafka consumers.
type Manager struct {
	logger    *zap.Logger
	consumers []Consumer
}

// NewManager creates a new Manager.
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		logger:    logger,
		consumers: make([]Consumer, 0),
	}
}

// Register adds a consumer to the manager.
func (m *Manager) Register(consumer Consumer) {
	m.consumers = append(m.consumers, consumer)
}

// Start starts all registered consumers.
func (m *Manager) Start(ctx context.Context) {
	m.logger.Info("Starting all Kafka consumers", zap.Int("count", len(m.consumers)))

	var wg sync.WaitGroup
	for _, c := range m.consumers {
		wg.Add(1)
		consumer := c
		go func() {
			defer wg.Done()
			consumer.Start(ctx)
		}()
	}
	wg.Wait()
}

// Stop gracefully stops all registered consumers.
func (m *Manager) Stop() {
	m.logger.Info("Stopping all Kafka consumers", zap.Int("count", len(m.consumers)))
	for _, c := range m.consumers {
		if err := c.Stop(); err != nil {
			m.logger.Error("Failed to stop consumer", zap.Error(err))
		}
	}
	m.logger.Info("All Kafka consumers stopped")
}
