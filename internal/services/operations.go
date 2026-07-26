// Package services holds the use-case orchestration layer, wiring repositories,
// the proxy gateway, tunnel manager, and network helpers into product behavior.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

type opTokenKey struct{}

// Coordinator serializes mutually-exclusive network operations while allowing
// trusted nested calls (via a context token) to re-enter without deadlocking.
type Coordinator struct {
	sem chan struct{}

	mu        sync.Mutex
	current   string
	startedAt time.Time
	waiting   int
}

// NewCoordinator creates a Coordinator.
func NewCoordinator() *Coordinator { return &Coordinator{sem: make(chan struct{}, 1)} }

// Current returns the name of the running operation, or "".
func (c *Coordinator) Current() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Snapshot reports the coordinator state for diagnostics.
func (c *Coordinator) Snapshot() (operation string, waiting int, startedAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current, c.waiting, c.startedAt
}

// Begin acquires the operation lock. If ctx already carries a token (a trusted
// nested call), it returns immediately. When wait is false and the lock is held,
// it returns domain.ErrOperationConflict.
func (c *Coordinator) Begin(ctx context.Context, op string, wait bool) (context.Context, func(), error) {
	if ctx.Value(opTokenKey{}) != nil {
		return ctx, func() {}, nil
	}
	if wait {
		c.mu.Lock()
		c.waiting++
		c.mu.Unlock()
		c.sem <- struct{}{}
		c.mu.Lock()
		c.waiting--
		c.mu.Unlock()
	} else {
		select {
		case c.sem <- struct{}{}:
		default:
			return ctx, func() {}, fmt.Errorf("%w: %s", domain.ErrOperationConflict, c.Current())
		}
	}
	c.mu.Lock()
	c.current = op
	c.startedAt = time.Now()
	c.mu.Unlock()

	var once sync.Once
	release := func() {
		once.Do(func() {
			c.mu.Lock()
			c.current = ""
			c.mu.Unlock()
			<-c.sem
		})
	}
	return context.WithValue(ctx, opTokenKey{}, op), release, nil
}

// Run executes fn while holding the operation lock.
func (c *Coordinator) Run(ctx context.Context, op string, wait bool, fn func(context.Context) error) error {
	nctx, release, err := c.Begin(ctx, op, wait)
	if err != nil {
		return err
	}
	defer release()
	return fn(nctx)
}
