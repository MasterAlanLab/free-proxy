package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
	"github.com/masteralanlab/free-proxy/internal/store"
)

// JobFunc is the body of an async job; its returned map becomes the job result.
type JobFunc func(ctx context.Context) (map[string]any, error)

// JobService runs long operations in the background and persists their status.
type JobService struct {
	repo *store.JobRepository
	root context.Context
}

// NewJobService creates a JobService. root bounds job goroutines to process life.
func NewJobService(repo *store.JobRepository, root context.Context) *JobService {
	return &JobService{repo: repo, root: root}
}

// Initialize cancels jobs left unfinished by a prior run.
func (s *JobService) Initialize(ctx context.Context) error {
	return s.repo.CancelUnfinished(ctx, time.Now().UTC())
}

// Submit records a pending job and runs fn in the background, returning the job.
func (s *JobService) Submit(ctx context.Context, name string, fn JobFunc) (domain.JobRead, error) {
	id := newHexID()
	now := time.Now().UTC()
	if err := s.repo.Create(ctx, id, name, now); err != nil {
		return domain.JobRead{}, err
	}
	go func() {
		jctx := context.WithoutCancel(s.root)
		_ = s.repo.MarkRunning(jctx, id, time.Now().UTC())
		res, err := fn(jctx)
		if err != nil {
			slog.Warn("job failed", "module", "jobs", "job", name, "err", err)
			_ = s.repo.MarkFailed(jctx, id, time.Now().UTC(), err.Error())
			return
		}
		_ = s.repo.MarkSucceeded(jctx, id, time.Now().UTC(), res)
	}()
	return s.repo.Get(ctx, id)
}

// Get returns a job by id.
func (s *JobService) Get(ctx context.Context, id string) (domain.JobRead, error) {
	return s.repo.Get(ctx, id)
}

func newHexID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
