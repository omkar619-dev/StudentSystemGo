// Package rabbitmq wraps the AMQP 0.9.1 client for our app.
//
// Two roles share this package:
//   - PUBLISHER (the API server): builds connection at startup, publishes
//     messages on every relevant request.
//   - CONSUMER (the worker): builds connection at startup, registers handlers,
//     blocks forever consuming messages.
//
// Connection model: ONE TCP connection, MANY channels. Channels are cheap;
// connections are not. We keep one connection per process and let callers
// open channels as needed.
package rabbitmq

import (
	"fmt"
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client wraps an AMQP connection.
//
// One per process. Constructed at startup, closed at shutdown.
// Concurrent operations open separate channels off the connection.
type Client struct {
	conn *amqp.Connection
}

// Conn exposes the raw connection. Caller is responsible for opening
// channels via conn.Channel() when needed.
func (c *Client) Conn() *amqp.Connection { return c.conn }

// Connect opens an AMQP connection from RABBITMQ_URL env var.
// URL format: amqp://user:pass@host:port/vhost
//
// Fail-closed: if RabbitMQ is unreachable at startup, return error so
// the caller can crash. Without a working broker, async work doesn't work.
func Connect() (*Client, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return nil, fmt.Errorf("rabbitmq: RABBITMQ_URL not set")
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: dial: %w", err)
	}

	log.Printf("[rabbitmq] connected to %s", redactURL(url))
	return &Client{conn: conn}, nil
}

// Close gracefully closes the connection. All channels are auto-closed.
// Call via defer at startup so it runs on shutdown.
func (c *Client) Close() error {
	if c.conn == nil || c.conn.IsClosed() {
		return nil
	}
	return c.conn.Close()
}

// redactURL hides the password from logs.
//   amqp://rabbit:rabbitpass@rabbitmq:5672/  →  amqp://rabbit:***@rabbitmq:5672/
func redactURL(url string) string {
	parsed, err := amqp.ParseURI(url)
	if err != nil {
		return "<invalid>"
	}
	parsed.Password = "***"
	return parsed.String()
}
