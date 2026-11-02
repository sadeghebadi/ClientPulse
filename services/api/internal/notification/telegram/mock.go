package telegram

import (
	"context"
	"sync"
)

// MockSender implements TelegramSender and TelegramBotAPI for unit and integration testing.
type MockSender struct {
	mu               sync.Mutex
	SentMessages     []SentTelegramMessage
	CustomError      error
	BotUser          TelegramUser
	WebhookURL       string
	PendingUpdates   []TelegramUpdate
	PollCallCount    int
	DeleteWebhookErr error
	GetUpdatesErr    error
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
	m.WebhookURL = ""
	m.PendingUpdates = nil
	m.PollCallCount = 0
	m.DeleteWebhookErr = nil
	m.GetUpdatesErr = nil
}

func (m *MockSender) GetWebhookInfo(ctx context.Context) (*WebhookInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CustomError != nil {
		return nil, m.CustomError
	}
	return &WebhookInfo{
		URL:                m.WebhookURL,
		PendingUpdateCount: len(m.PendingUpdates),
		AllowedUpdates:     []string{"message"},
	}, nil
}

func (m *MockSender) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteWebhookErr != nil {
		return m.DeleteWebhookErr
	}
	m.WebhookURL = ""
	if dropPendingUpdates {
		m.PendingUpdates = nil
	}
	return nil
}

func (m *MockSender) GetUpdates(ctx context.Context, offset int64, limit int, timeout int, allowedUpdates []string) ([]TelegramUpdate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PollCallCount++
	if m.GetUpdatesErr != nil {
		return nil, m.GetUpdatesErr
	}

	var eligible []TelegramUpdate
	for _, u := range m.PendingUpdates {
		if int64(u.UpdateID) >= offset {
			eligible = append(eligible, u)
			if limit > 0 && len(eligible) >= limit {
				break
			}
		}
	}
	return eligible, nil
}

