package services

import (
	"testing"
	"time"

	"github.com/masteralanlab/free-proxy/internal/domain"
)

func TestProbeSlicePrioritisesStaleAndCapsTheCycle(t *testing.T) {
	at := func(d time.Duration) *time.Time { v := time.Now().Add(d); return &v }
	nodes := []domain.ProxyNodeRead{
		{ID: "recent", LastProbedAt: at(-time.Minute)},
		{ID: "old", LastProbedAt: at(-48 * time.Hour)},
		{ID: "never"},
		{ID: "middle", LastProbedAt: at(-2 * time.Hour)},
	}
	got := probeSlice(nodes)
	want := []string{"never", "old", "middle", "recent"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v (never-probed first, then oldest)", got, want)
		}
	}
}

func TestProbeSliceCapsAtBudget(t *testing.T) {
	nodes := make([]domain.ProxyNodeRead, probeBudget+50)
	for i := range nodes {
		nodes[i] = domain.ProxyNodeRead{ID: string(rune('a' + i%26))}
	}
	// A cycle must stay bounded no matter how large the pool grows, because it
	// holds the operation lock for its whole duration.
	if got := probeSlice(nodes); len(got) != probeBudget {
		t.Fatalf("cycle probes %d nodes, want it capped at %d", len(got), probeBudget)
	}
}
