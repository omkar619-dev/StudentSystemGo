// Package mailer abstracts email sending behind an interface so we can
// swap mock for real SMTP later without touching consumer code.
//
// Implementations:
//   - MockMailer: logs to stdout. For dev + early prod testing.
//   - (future) SMTPMailer / SESMailer / SendGridMailer.
package mailer

import (
	"context"
	"fmt"
	"os"
)

// Mailer is the contract every email backend must satisfy.
type Mailer interface {
	SendPasswordReset(ctx context.Context, msg PasswordResetMessage) error
}

// PasswordResetMessage is the payload published to the queue and consumed
// by the worker. JSON-serialised over the wire.
//
// Keep this stable — changing fields requires a queue drain or migration.
type PasswordResetMessage struct {
	To         string `json:"to"`           // recipient email
	Username   string `json:"username"`     // for personalisation in the email body
	ResetToken string `json:"reset_token"`  // the URL-safe token
	ResetURL   string `json:"reset_url"`    // full link to click — built by the producer
	ExpiresAt  string `json:"expires_at"`   // RFC3339 — when the token expires
}

// New builds a Mailer based on MAILER_TYPE env var.
//
//	mock  → MockMailer (default)
//	smtp  → not implemented yet
//	ses   → not implemented yet
func New() (Mailer, error) {
	switch os.Getenv("MAILER_TYPE") {
	case "", "mock":
		return &MockMailer{}, nil
	default:
		return nil, fmt.Errorf("mailer: unknown MAILER_TYPE=%s", os.Getenv("MAILER_TYPE"))
	}
}
