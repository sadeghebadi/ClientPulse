package telegram

import (
	"context"
	"sync"
)

// MockSender implements TelegramSender for unit and integration testing.
type MockSender struct {
	mu           sync.Mutex
	SentMessages []SentTelegramMessage
	CustomError  error
	BotUser      TelegramUser
}

type SentTelegramMessage struct {
	ChatID    int64
	Text      string
	ParseMode string
}

func NewMockSender() *MockSender {
	return &MockSender{
		BotUser: TelegramUser{
			ID:        12345678,
			IsBot:     true,
			FirstName: "ClientPulse Test Bot",
			Username:  "ClientPulseTestBot",
		},
	}
}

func (m *MockSender) SendMessage(ctx context.Context, chatID int64, text string, parseMode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.CustomError != nil {
		return m.CustomError
	}

	m.SentMessages = append(m.SentMessages, SentTelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	})
	return nil
}

func (m *MockSender) GetMe(ctx context.Context) (*TelegramUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CustomError != nil {
		return nil, m.CustomError
	}
	return &m.BotUser, nil
}

func (m *MockSender) LastMessage() *SentTelegramMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.SentMessages) == 0 {
		return nil
	}
	last := m.SentMessages[len(m.SentMessages)-1]
	return &last
}

func (m *MockSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages = nil
	m.CustomError = nil
}

