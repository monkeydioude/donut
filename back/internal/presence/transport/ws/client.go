package wstransport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"

	"donut/internal/presence/domain"
)

const (
	// queueSize is how many events may wait for a player; past that, the player reads too slowly. Walking into a
	// crowd brings a burst of joined messages, and a normal player mustn't be dropped for it (ADR 0003).
	queueSize    = 256
	writeTimeout = 5 * time.Second
)

// client is a player's websocket as a room sees it: events go through a queue, written by one goroutine.
type client struct {
	conn   *websocket.Conn
	events chan domain.Event
	done   chan struct{}
	once   sync.Once
	log    *slog.Logger
}

func newClient(conn *websocket.Conn, log *slog.Logger) *client {
	c := &client{conn: conn, events: make(chan domain.Event, queueSize), done: make(chan struct{}), log: log}
	go c.writeLoop()
	return c
}

// Send queues an event, false when the queue is full. A closing client takes everything and sends nothing.
func (c *client) Send(event domain.Event) bool {
	select {
	case <-c.done:
		return true
	default:
	}
	select {
	case c.events <- event:
		return true
	default:
		return false
	}
}

// Close ends the connection without waiting for the closing handshake.
func (c *client) Close() {
	c.once.Do(func() {
		c.log.Debug("ws closing", "queued_events_dropped", len(c.events))
		close(c.done)
		go c.conn.Close(websocket.StatusNormalClosure, "")
	})
}

func (c *client) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case event := <-c.events:
			if err := c.write(event); err != nil {
				c.log.Debug("ws write failed", "event", fmt.Sprintf("%T", event), "error", err)
				c.Close()
				return
			}
		}
	}
}

func (c *client) write(event domain.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()
	message := encodeEvent(event)
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	c.log.Debug("ws send", "message", logged(event, message), "bytes", len(data), "queued", len(c.events))
	return c.conn.Write(ctx, websocket.MessageText, data)
}
