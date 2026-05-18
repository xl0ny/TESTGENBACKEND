package usecase

import "testing"

func TestSenderFromEmail(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"user@mail.ru", "user"},
		{"  ivan@test.com  ", "ivan"},
		{"plainname", "plainname"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := SenderFromEmail(tt.in); got != tt.want {
			t.Errorf("SenderFromEmail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSessionUseCase_CreateSession(t *testing.T) {
	uc := NewSessionUseCase()
	sender, err := uc.CreateSession("demo@example.com")
	if err != nil || sender != "demo" {
		t.Fatalf("CreateSession: sender=%q err=%v", sender, err)
	}
	if _, err := uc.CreateSession("bad"); err != ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}
