package mailer

import (
	"context"
	"log"
)

// MockMailer "sends" emails by logging them to stdout.
// Used in dev and during early prod testing before real SMTP is wired up.
type MockMailer struct{}

func (m *MockMailer) SendPasswordReset(ctx context.Context, msg PasswordResetMessage) error {
	log.Printf("[mailer-mock] === PASSWORD RESET EMAIL ===")
	log.Printf("[mailer-mock] To:        %s", msg.To)
	log.Printf("[mailer-mock] Username:  %s", msg.Username)
	log.Printf("[mailer-mock] Reset URL: %s", msg.ResetURL)
	log.Printf("[mailer-mock] Expires:   %s", msg.ExpiresAt)
	log.Printf("[mailer-mock] === END EMAIL ===")
	return nil
}
