package services

import (
	"context"
	"errors"
	"testing"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

func TestCoordinatorConflict(t *testing.T) {
	c := NewCoordinator()
	ctx := context.Background()
	_, release, err := c.Begin(ctx, "activate", false)
	if err != nil {
		t.Fatalf("first begin: %v", err)
	}
	if c.Current() != "activate" {
		t.Fatalf("current = %q", c.Current())
	}
	if _, _, err := c.Begin(ctx, "refresh", false); !errors.Is(err, domain.ErrOperationConflict) {
		t.Fatalf("second begin err = %v, want conflict", err)
	}
	release()
	if c.Current() != "" {
		t.Fatalf("current after release = %q", c.Current())
	}
	// After release, a new operation can begin.
	if _, r2, err := c.Begin(ctx, "refresh", false); err != nil {
		t.Fatalf("begin after release: %v", err)
	} else {
		r2()
	}
}

func TestCoordinatorReentrant(t *testing.T) {
	c := NewCoordinator()
	nctx, release, err := c.Begin(context.Background(), "maintain", false)
	if err != nil {
		t.Fatalf("outer begin: %v", err)
	}
	defer release()
	// A nested call carrying the token must not conflict.
	_, innerRelease, err := c.Begin(nctx, "probe", false)
	if err != nil {
		t.Fatalf("nested begin should be reentrant, got %v", err)
	}
	innerRelease() // no-op; must not free the outer lock
	if c.Current() != "maintain" {
		t.Fatalf("nested release freed outer lock; current = %q", c.Current())
	}
}
