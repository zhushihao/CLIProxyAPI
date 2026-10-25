package auth

import (
	"context"
	"fmt"
	"testing"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

// TestWRRWeight73Repro mirrors production: two workbuddy creds, same priority
// tier (-1), weights 7:3, queried through the session-affinity wrapper's
// fallback path (highestPriorityAuths -> WeightedRoundRobinSelector.Pick).
func TestWRRWeight73Repro(t *testing.T) {
	mk := func(id string, w string) *Auth {
		return &Auth{
			ID:         id,
			Provider:   "workbuddy",
			Attributes: map[string]string{"priority": "-1", "weight": w},
		}
	}
	shawn := mk("workbuddy-442e36a5", "7")
	anyu := mk("workbuddy-6f862498", "3")

	wrr := &WeightedRoundRobinSelector{}
	ctx := context.Background()

	// Plain WRR path (what the affinity wrapper calls on miss):
	// highestPriorityAuths(available) -> wrr.Pick
	tier := highestPriorityAuths([]*Auth{shawn, anyu})
	t.Logf("highest tier size = %d", len(tier))

	counts := map[string]int{}
	for i := 0; i < 12; i++ {
		auth, err := wrr.Pick(ctx, "mixed", "deepseek-v4.1-flash", cliproxyexecutor.Options{}, tier)
		if err != nil {
			t.Fatalf("pick %d: %v", i, err)
		}
		counts[auth.ID]++
	}
	t.Logf("plain WRR counts: shawn=%d anyu=%d", counts[shawn.ID], counts[anyu.ID])

	// Through the affinity wrapper, LCP-less requests (no session id metadata):
	aff := NewSessionAffinitySelectorWithConfig(SessionAffinityConfig{Fallback: wrr, TTL: 3600e9})
	counts2 := map[string]int{}
	for i := 0; i < 12; i++ {
		auth, err := aff.Pick(ctx, "mixed", "deepseek-v4.1-flash", cliproxyexecutor.Options{}, []*Auth{shawn, anyu})
		if err != nil {
			t.Fatalf("affinity pick %d: %v", i, err)
		}
		counts2[auth.ID]++
	}
	t.Logf("affinity counts: shawn=%d anyu=%d", counts2[shawn.ID], counts2[anyu.ID])

	fmt.Println("done")
}
