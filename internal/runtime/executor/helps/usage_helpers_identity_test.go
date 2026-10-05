package helps

import (
	"context"
	"testing"
)

type identityTestExecutor struct{}

func (identityTestExecutor) Identifier() string { return "claude" }

func TestUsageIdentityOverrideRelabelsReporter(t *testing.T) {
	ctx := context.Background()
	plain := NewExecutorUsageReporter(ctx, identityTestExecutor{}, "m", nil)
	if plain.provider != "claude" || plain.executorType != "identityTestExecutor" {
		t.Fatalf("no-override identity = %q/%q, want claude/identityTestExecutor", plain.provider, plain.executorType)
	}
	overridden := NewExecutorUsageReporter(WithUsageIdentityOverride(ctx, "kimi", "KimiExecutor"), identityTestExecutor{}, "m", nil)
	if overridden.provider != "kimi" || overridden.executorType != "KimiExecutor" {
		t.Fatalf("override identity = %q/%q, want kimi/KimiExecutor", overridden.provider, overridden.executorType)
	}
	partial := NewExecutorUsageReporter(WithUsageIdentityOverride(ctx, "kimi", ""), identityTestExecutor{}, "m", nil)
	if partial.provider != "kimi" || partial.executorType != "identityTestExecutor" {
		t.Fatalf("partial override identity = %q/%q, want kimi/identityTestExecutor", partial.provider, partial.executorType)
	}
}
