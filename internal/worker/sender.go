package worker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type Sender interface {
	Send(content string, to string) (string, error)
}

// Simulates sending messages
type MockSender struct {
	successRate float64
}

// Create a new mock sender with the given success rate
func NewMockSender(successRate float64) *MockSender {
	return &MockSender{
		successRate: successRate,
	}
}

// Simulates sending a message
// Returns a provider message ID on success, or an error on failure
func (s *MockSender) Send(content string, to string) (string, error) {
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	if rand.Float64() > s.successRate {
		return "", fmt.Errorf("mock provider error: failed to deliver message to %s", to)
	}
	return fmt.Sprintf("mock-msg-%s", uuid.New().String()), nil
}
