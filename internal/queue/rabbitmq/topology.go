package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Queue names. Centralised here so producers and consumers can't typo
// each other into separate queues that nobody connects.
const (
	QueuePasswordReset       = "email.password_reset"
	QueuePasswordResetFailed = "email.password_reset.failed"
)

// DeclareTopology idempotently creates queues + DLX wiring.
// Safe to call repeatedly — RabbitMQ "declare" verifies existing
// definitions match what we requested. Mismatch = error.
//
// Topology:
//
//	[publisher] ──→ default exchange ──→ email.password_reset
//	                                          │
//	                                          │ on NACK(requeue=false)
//	                                          │ or message TTL expiry
//	                                          ▼
//	                                    (default exchange,
//	                                     routing key = "email.password_reset.failed")
//	                                          │
//	                                          ▼
//	                                  email.password_reset.failed
//
// Failed-queue messages sit there until ops manually inspects/replays.
// Production: add monitoring (alert when failed queue depth > 0).
func DeclareTopology(c *Client) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer ch.Close()

	// Failed queue — declare FIRST because main queue's args reference it.
	// No special args. Messages land here, sit, wait for human.
	if _, err := ch.QueueDeclare(
		QueuePasswordResetFailed,
		true,  // durable: queue survives broker restart
		false, // autoDelete: do NOT delete when last consumer disconnects
		false, // exclusive: shared (not tied to one connection)
		false, // noWait: wait for broker confirmation
		nil,   // args: none
	); err != nil {
		return fmt.Errorf("rabbitmq: declare failed queue: %w", err)
	}

	// Main queue with DLX wiring.
	// x-dead-letter-exchange = ""  → use default exchange
	// x-dead-letter-routing-key = name of failed queue → routes there
	if _, err := ch.QueueDeclare(
		QueuePasswordReset,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": QueuePasswordResetFailed,
		},
	); err != nil {
		return fmt.Errorf("rabbitmq: declare main queue: %w", err)
	}

	return nil
}
