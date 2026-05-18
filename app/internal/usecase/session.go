package usecase

import (
	"errors"
	"strings"
)

var ErrInvalidEmail = errors.New("email required")

type SessionUseCase struct{}

func NewSessionUseCase() *SessionUseCase {
	return &SessionUseCase{}
}

func (uc *SessionUseCase) CreateSession(email string) (string, error) {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" || !strings.Contains(trimmed, "@") {
		return "", ErrInvalidEmail
	}
	return SenderFromEmail(trimmed), nil
}

// SenderFromEmail derives transport sender from email (local part).
func SenderFromEmail(email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "@") {
		return strings.Split(trimmed, "@")[0]
	}
	return trimmed
}
