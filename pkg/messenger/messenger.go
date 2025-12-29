package messenger

import (
	"context"
	"fmt"
)

type Message struct {
	To   string
	Body string
}

type Provider interface {
	Send(ctx context.Context, msg Message) error
}

// 1. The struct must be exported (starts with capital M) or
// the constructor must return the interface.
type mockProvider struct{}

// NewMockProvider creates a new instance of the mock SMS provider
func NewMockProvider() Provider {
	return &mockProvider{}
}

func (m *mockProvider) Send(ctx context.Context, msg Message) error {
	fmt.Printf("\n--- [MOCK SMS] ---\nTo: %s\nBody: %s\n------------------\n\n", msg.To, msg.Body)
	return nil
}
