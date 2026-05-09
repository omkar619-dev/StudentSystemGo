package rabbitmq

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// HandlerFunc processes one message. Return nil for success → ACK.
// Return error → NACK with requeue=false → message dead-letters to
// the queue's configured DLX (our `email.password_reset.failed`).
type HandlerFunc func(ctx context.Context, body []byte) error

// Consume blocks forever, dispatching incoming messages to handler.
// Designed for the worker process — call once at startup, never returns.
//
// Concurrency: prefetch=1 means broker hands ONE message at a time per
// channel. Worker can't get overwhelmed. If you want parallelism, run
// multiple worker containers (horizontal) rather than threads-per-worker.
func Consume(ctx context.Context, c *Client, queue string, handler HandlerFunc) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open consume channel: %w", err)
	}
	defer ch.Close()

	// QoS / prefetch. Without this, broker would push ALL queued messages
	// to one consumer, then the others sit idle.
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("rabbitmq: set qos: %w", err)
	}

	deliveries, err := ch.Consume(
		queue,
		"",    // consumer tag: empty = broker generates one
		false, // autoAck: NO — we ACK explicitly after handler succeeds
		false, // exclusive: queue can have multiple consumers
		false, // noLocal: not used by RabbitMQ
		false, // noWait: wait for broker to confirm Consume started
		nil,   // args: none
	)
	if err != nil {
		return fmt.Errorf("rabbitmq: consume %s: %w", queue, err)
	}

	log.Printf("[rabbitmq] consuming %s (prefetch=1)", queue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[rabbitmq] context cancelled, stopping consumer")
			return nil

		case d, ok := <-deliveries:
			if !ok {
				// Channel closed (broker disconnected, channel error, etc.).
				// Bubble up so caller can decide to reconnect or crash.
				return fmt.Errorf("rabbitmq: deliveries channel closed")
			}
			handleOne(ctx, d, handler)
		}
	}
}

// handleOne processes a single delivery and ACKs/NACKs accordingly.
//
// Recovers from handler panics so one bad message can't crash the worker.
// Panicked messages get NACK'd (no requeue) → DLX → failed queue.
func handleOne(ctx context.Context, d amqp.Delivery, handler HandlerFunc) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[rabbitmq] handler PANIC on msg=%s: %v", d.MessageId, r)
			// requeue=false → goes to DLX → dead letters to failed queue
			_ = d.Nack(false, false)
		}
	}()

	if err := handler(ctx, d.Body); err != nil {
		log.Printf("[rabbitmq] handler error on msg=%s: %v — NACKing", d.MessageId, err)
		_ = d.Nack(false, false) // dead-letter
		return
	}

	if err := d.Ack(false); err != nil {
		log.Printf("[rabbitmq] ACK failed on msg=%s: %v", d.MessageId, err)
	}
}
