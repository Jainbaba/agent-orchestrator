package chat_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	chatsvc "github.com/aoagents/agent-orchestrator/backend/internal/service/chat"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/store"
)

func TestProviderSwitchPublishesApproximateBranchAndIsIdempotent(t *testing.T) {
	h, _, driver := newEditHarness(t, false)
	ctx := context.Background()

	// Give the source a complete durable exchange so the target assertion proves
	// the bootstrap came from AO's canonical rows, not provider scrollback.
	completeTurn(t, h, "keep this context", "provider-turn-1")
	h.awaitSnapshot(t, func(snapshot store.ConversationSnapshot) bool {
		return len(snapshot.Messages) == 2
	})

	first, err := h.svc.SwitchProvider(ctx, testSession, domain.ProviderSwitchRequest{
		TargetHarness:  domain.HarnessClaudeCode,
		IdempotencyKey: "switch-once",
	})
	if err != nil {
		t.Fatalf("SwitchProvider: %v", err)
	}
	if first.SourceHarness != domain.HarnessCodex || first.TargetHarness != domain.HarnessClaudeCode {
		t.Fatalf("switch result = %+v, want codex -> claude-code", first)
	}
	if first.BranchID == "" || first.ProviderConversation == "" {
		t.Fatalf("switch result missing durable target identity: %+v", first)
	}

	driver.mu.Lock()
	if len(driver.startConfigs) < 2 {
		driver.mu.Unlock()
		t.Fatalf("target start was not observed: %d starts", len(driver.startConfigs))
	}
	targetConfig := driver.startConfigs[len(driver.startConfigs)-1]
	driver.mu.Unlock()
	if !strings.Contains(targetConfig.SystemPrompt, "keep this context") {
		t.Fatalf("target system prompt did not contain AO handoff context: %q", targetConfig.SystemPrompt)
	}

	retry, err := h.svc.SwitchProvider(ctx, testSession, domain.ProviderSwitchRequest{
		TargetHarness:  domain.HarnessClaudeCode,
		IdempotencyKey: "switch-once",
	})
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if retry != first {
		t.Fatalf("idempotent retry = %+v, want original %+v", retry, first)
	}

	record, found, err := h.st.GetSession(ctx, testSession)
	if err != nil || !found {
		t.Fatalf("read switched session: found=%v err=%v", found, err)
	}
	if record.Harness != domain.HarnessClaudeCode || record.Mode != domain.SessionModeChat {
		t.Fatalf("durable session owner = %+v, want Claude Chat", record)
	}
	snapshot, err := h.st.LoadConversationSnapshot(ctx, first.ConversationID)
	if err != nil {
		t.Fatalf("read switched conversation: %v", err)
	}
	if snapshot.ActiveBranch.ID != first.BranchID {
		t.Fatalf("active branch = %q, want %q", snapshot.ActiveBranch.ID, first.BranchID)
	}
	if snapshot.ActiveBranch.Strategy != domain.ConversationBranchStrategyApproximateContext {
		t.Fatalf("branch strategy = %q, want approximate_context", snapshot.ActiveBranch.Strategy)
	}
	foundMarker := false
	for _, activity := range snapshot.Activities {
		if activity.RequestID == "switch-once" && strings.Contains(activity.Summary, "Switched provider") {
			foundMarker = true
			break
		}
	}
	if !foundMarker {
		t.Fatalf("durable idempotency marker was not committed with the branch")
	}
}

func TestProviderSwitchFailureRestoresSourceController(t *testing.T) {
	h, _, driver := newEditHarness(t, false)
	ctx := context.Background()
	driver.startErr = errProviderSwitchTarget

	_, err := h.svc.SwitchProvider(ctx, testSession, domain.ProviderSwitchRequest{
		TargetHarness: domain.HarnessClaudeCode,
	})
	if err == nil || !strings.Contains(err.Error(), "start target provider") {
		t.Fatalf("SwitchProvider error = %v, want target-start failure", err)
	}
	if _, err := h.svc.Controller(testSession); err != nil {
		t.Fatalf("source controller was not restored: %v", err)
	}
}

func TestProviderSwitchPublicationFailureRestoresSourceController(t *testing.T) {
	h, _, _ := newEditHarnessWithStore(t, false, func(st *store.Store) chatsvc.Store {
		return failingProviderSwitchStore{Store: st, err: errors.New("commit unavailable")}
	})

	_, err := h.svc.SwitchProvider(context.Background(), testSession, domain.ProviderSwitchRequest{
		TargetHarness: domain.HarnessClaudeCode,
	})
	if err == nil || !strings.Contains(err.Error(), "start target provider") {
		t.Fatalf("SwitchProvider error = %v, want publication failure", err)
	}
	if _, err := h.svc.Controller(testSession); err != nil {
		t.Fatalf("source controller was not restored after publication failure: %v", err)
	}
}

type failingProviderSwitchStore struct {
	*store.Store
	err error
}

func (s failingProviderSwitchStore) CommitProviderSwitch(context.Context, domain.SessionRecord, domain.ConversationBranch, string, domain.ConversationActivity) error {
	return s.err
}

var errProviderSwitchTarget = ports.ErrChatDriverUnavailable
