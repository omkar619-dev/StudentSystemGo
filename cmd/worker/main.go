// Worker process — consumes RabbitMQ queues and runs async work.
//
// Architecture:
//
//	HTTP server (cmd/api) ──publish──→ RabbitMQ ──consume──→ worker (this binary)
//
// One queue per task type. For now, only password-reset emails. Add
// more handlers as new async tasks come up (welcome emails, notifications,
// nightly reports, etc.).
//
// Graceful shutdown: on SIGTERM (Docker stop), stop accepting new
// messages, finish in-flight, exit. RabbitMQ redelivers anything that
// wasn't ACKed yet to the next available worker.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"restapi/internal/mailer"
	mq "restapi/internal/queue/rabbitmq"
)

func main() {
	log.Println("[worker] starting")

	// ── Connect to RabbitMQ ───────────────────────────────
	client, err := mq.Connect()
	if err != nil {
		log.Fatalf("[worker] connect rabbitmq: %v", err)
	}
	defer client.Close()

	// Declare topology (idempotent — same one publisher uses).
	if err := mq.DeclareTopology(client); err != nil {
		log.Fatalf("[worker] declare topology: %v", err)
	}

	// ── Build the mailer ──────────────────────────────────
	m, err := mailer.New()
	if err != nil {
		log.Fatalf("[worker] mailer init: %v", err)
	}

	// ── Set up graceful shutdown ──────────────────────────
	// ctx is cancelled when SIGTERM/SIGINT is received. The consumer
	// loop watches this context and exits cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Consume ───────────────────────────────────────────
	// One handler per queue. Block here forever (until ctx done).
	if err := mq.Consume(ctx, client, mq.QueuePasswordReset, makePasswordResetHandler(m)); err != nil {
		log.Fatalf("[worker] consume: %v", err)
	}

	log.Println("[worker] shutting down cleanly")
}

// makePasswordResetHandler returns a HandlerFunc bound to the given mailer.
// Closure pattern — handler signature has no mailer parameter, so we
// capture it via this factory.
func makePasswordResetHandler(m mailer.Mailer) mq.HandlerFunc {
	return func(ctx context.Context, body []byte) error {
		var msg mailer.PasswordResetMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			// Bad JSON = permanent failure. Returning error → NACK → DLX.
			// We never want to keep retrying a malformed message.
			return fmt.Errorf("decode message: %w", err)
		}

		// Send. Mock always succeeds; real SMTP can fail.
		if err := m.SendPasswordReset(ctx, msg); err != nil {
			return fmt.Errorf("send: %w", err)
		}

		log.Printf("[worker] sent password reset to %s", msg.To)
		return nil
	}
}
