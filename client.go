package main

import (
	"context"
	"errors"
	"math"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	url string

	mu   sync.RWMutex
	conn *amqp.Connection
	ch   *amqp.Channel

	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error

	ready chan struct{} // closed when a channel is ready
	done  chan struct{} // close to stop reconnect loop
}

func NewClient(ctx context.Context, url string) (*Client, error) {
	if url == "" {
		url = os.Getenv("AMQP_URL")
	}
	if url == "" {
		return nil, errors.New("AMQP_URL is not set")
	}
	c := &Client{
		url:   url,
		ready: make(chan struct{}),
		done:  make(chan struct{}),
	}
	go c.reconnectLoop(ctx)
	// wait until the first successful connect
	select {
	case <-c.ready:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) reconnectLoop(ctx context.Context) {
	var attempt int
	for {
		conn, err := amqp.Dial(c.url)
		if err != nil {
			attempt++
			backoff := time.Duration(math.Min(float64(30*time.Second), math.Pow(2, float64(attempt))*500*time.Millisecond.Seconds())) * time.Second
			log.Warnf("[amqp] dial failed: %v (retry in %v)", err, backoff)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				close(c.done)
				return
			}
		}
		attempt = 0 // reset

		c.mu.Lock()
		c.conn = conn
		c.notifyConnClose = conn.NotifyClose(make(chan *amqp.Error, 1))
		c.mu.Unlock()

		if err := c.initChannel(); err != nil {
			_ = conn.Close()
			continue
		}

		// signal ready (only the first time this has an effect)
		select {
		case <-c.ready:
			// already closed
		default:
			close(c.ready)
		}

		// Block here until something closes, then loop to reconnect
		select {
		case err := <-c.notifyConnClose:
			if err != nil {
				log.Warnf("[amqp] connection closed: %v", err)
			} else {
				log.Info("[amqp] connection closed")
			}
		case <-c.notifyChanClose:
			// channel closed: we’ll try to recreate the channel on the same connection
			// but if connection is gone, initChannel will fail and we’ll reconnect.
			log.Info("[amqp] channel closed; attempting to re-init")
			if err := c.initChannel(); err == nil {
				// channel resurrected; keep waiting for next close
				continue
			}
		case <-ctx.Done():
			_ = c.Close()
			close(c.done)
			return
		}

		// ensure everything is closed before next attempt
		c.mu.Lock()
		if c.ch != nil {
			_ = c.ch.Close()
			c.ch = nil
		}
		if c.conn != nil {
			_ = c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
	}
}

func (c *Client) initChannel() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	// example: configure QoS, declare exchanges/queues/bindings as needed
	// _ = ch.Qos(10, 0, false)
	// _ = ch.ExchangeDeclare("events", "topic", true, false, false, false, nil)
	// _, _ = ch.QueueDeclare("events.q", true, false, false, false, nil)
	// _ = ch.QueueBind("events.q", "#", "events", false, nil)

	c.mu.Lock()
	if c.ch != nil {
		_ = c.ch.Close()
	}
	c.ch = ch
	c.notifyChanClose = ch.NotifyClose(make(chan *amqp.Error, 1))
	c.mu.Unlock()
	return nil
}

func (c *Client) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ch == nil {
		return nil, errors.New("channel not ready")
	}
	return c.ch, nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch != nil {
		_ = c.ch.Close()
		c.ch = nil
	}
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}
