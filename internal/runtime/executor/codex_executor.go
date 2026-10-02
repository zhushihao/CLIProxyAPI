package executor

import (
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

// CodexExecutor is a stateless executor for Codex (OpenAI Responses API entrypoint).
// If api_key is unavailable on auth, it falls back to legacy via ClientAdapter.
type CodexExecutor struct {
	cfg *config.Config
}

func NewCodexExecutor(cfg *config.Config) *CodexExecutor { return &CodexExecutor{cfg: cfg} }

func (e *CodexExecutor) Identifier() string { return "codex" }

func (e *CodexExecutor) modelLevelCooling() bool {
	return e != nil && e.cfg != nil && e.cfg.Codex.ModelLevelCooling
}

func (e *CodexExecutor) overloadRetryCount() int {
	if e == nil || e.cfg == nil {
		return 2
	}
	return e.cfg.Codex.OverloadRetryCount()
}

func (e *CodexExecutor) overloadRetryDelay() time.Duration {
	if e == nil || e.cfg == nil {
		return time.Second
	}
	return e.cfg.Codex.OverloadRetryDelayDuration()
}

// SupportsApplyPatch reports the actual executor contract, independent of its provider name.
func (e *CodexExecutor) SupportsApplyPatch() bool { return e != nil }
