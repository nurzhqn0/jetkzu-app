package smtp

import (
	"context"
	"fmt"
	"net/smtp"

	"go.uber.org/zap"
)

type Sender interface {
	Send(ctx context.Context, to, subject, body string) (status string, err error)
}

type RealSender struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func New(host string, port int, user, password, from string) *RealSender {
	return &RealSender{host: host, port: port, user: user, password: password, from: from}
}

func (s *RealSender) Send(_ context.Context, to, subject, body string) (string, error) {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body))
	if err := smtp.SendMail(addr, auth, s.from, []string{to}, msg); err != nil {
		return "failed", err
	}
	return "sent", nil
}

type MockSender struct {
	log *zap.Logger
}

func NewMock(log *zap.Logger) *MockSender { return &MockSender{log: log} }

func (m *MockSender) Send(_ context.Context, to, subject, body string) (string, error) {
	m.log.Info("mock email", zap.String("to", to), zap.String("subject", subject), zap.String("body", body))
	return "mock_sent", nil
}
