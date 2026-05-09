package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DefaultPublisher is the package-level publisher used by handlers.
// Set by InitPublisher at app startup. Nil until then.
var DefaultPublisher *Publisher

// InitPublisher creates DefaultPublisher from the given client.
// Call once at startup, after Connect. Returns the publisher so the
// caller can also `defer pub.Close()`.
func InitPublisher(c *Client) (*Publisher, error) {
	pub, err := NewPublisher(c)
	if err != nil {
		return nil, err
	}
	DefaultPublisher = pub
	return pub, nil
}

// Publisher wraps a long-lived channel for publishing.
//
// Why a dedicated channel: publish operations on the same channel are
// strictly ordered. Sharing a channel between publishers means one slow
// publish blocks the others. So: each goroutine that needs to publish
// gets its own Publisher (one channel each).
//
// In our app there's just ONE publisher (the API server uses it from
// the forgot-password handler), so a singleton is fine.
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher opens a channel for publishing. Channel is closed when
// pub.Close() is called. Caller typically defers Close at startup.
func NewPublisher(c *Client) (*Publisher, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: open publish channel: %w", err)
	}

	// Enable publisher confirms — broker sends an ACK back when message
	// is durably stored. Without this, Publish returns immediately and
	// network errors mid-flight are silently lost.
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		return nil, fmt.Errorf("rabbitmq: enable confirms: %w", err)
	}

	return &Publisher{ch: ch}, nil
}

func (p *Publisher) Close() error {
	if p.ch == nil {
		return nil
	}
	return p.ch.Close()
}

// PublishJSON marshals body to JSON and publishes to the named queue
// using the default exchange (which routes by queue name).
//
// Persistent delivery + publisher confirm: message survives broker
// restart, and we wait for the broker to confirm storage before returning.
func (p *Publisher) PublishJSON(ctx context.Context, queue string, body any) error {
	bytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("rabbitmq: marshal body: %w", err)
	}

	// PublishWithDeferredConfirmWithContext returns a confirmation handle
	// that we wait on below. Modern equivalent of the old confirm-channel
	// dance.
	conf, err := p.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		"",    // exchange = default (routes by queue name)
		queue, // routing key = queue name
		true,  // mandatory: error if no queue matches the routing key
		false, // immediate: deprecated, must be false
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // 2 = persistent, write to disk
			Timestamp:    time.Now(),
			Body:         bytes,
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq: publish to %s: %w", queue, err)
	}

	// Wait for broker ACK. If it NACKs, the message wasn't durably stored.
	// 5 sec timeout — if broker is wedged, fail fast rather than hang.
	ackCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ack, err := conf.WaitContext(ackCtx)
	if err != nil {
		return fmt.Errorf("rabbitmq: wait for confirm on %s: %w", queue, err)
	}
	if !ack {
		return fmt.Errorf("rabbitmq: broker NACKed message to %s", queue)
	}
	return nil
}
