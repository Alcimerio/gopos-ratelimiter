package storage

import (
	"context"
	"sync"
	"time"
)

type MockStorage struct {
	counters    map[string]int64
	blocked     map[string]time.Time
	mutex       sync.RWMutex
	currentTime time.Time
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		counters:    make(map[string]int64),
		blocked:     make(map[string]time.Time),
		currentTime: time.Now(),
	}
}

func (m *MockStorage) Increment(ctx context.Context, key string, expiration time.Duration) (int64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.counters[key]++
	return m.counters[key], nil
}

func (m *MockStorage) IsBlocked(ctx context.Context, key string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if blockTime, exists := m.blocked[key]; exists {
		return blockTime.After(m.currentTime), nil
	}
	return false, nil
}

func (m *MockStorage) Block(ctx context.Context, key string, duration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.blocked[key] = m.currentTime.Add(duration)
	return nil
}

func (m *MockStorage) Reset(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.counters, key)
	delete(m.blocked, key)
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

// Test helper methods
func (m *MockStorage) SetCurrentTime(t time.Time) {
	m.currentTime = t
}

func (m *MockStorage) AdvanceTime(d time.Duration) {
	m.currentTime = m.currentTime.Add(d)
}
