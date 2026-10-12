package sessionmanager

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestTerminalHandoffNoteCarriesSessionFacts(t *testing.T) {
	m := New(Deps{Agents: singleAgent{agent: fakeAgent{}}})
	got := m.terminalHandoffNote(context.Background(), domain.SessionRecord{
		ID: "session-1", Harness: domain.HarnessClaudeCode, Mode: domain.SessionModeTUI,
		Metadata: domain.SessionMetadata{Prompt: "ship the importer", Branch: "feature/importer"},
	}, domain.HarnessCodex, "switch-1")
	if !strings.HasPrefix(got, domain.HandoffNotePrefix) {
		t.Fatalf("note does not open with the handoff prefix: %s", got)
	}
	for _, want := range []string{"ship the importer", "feature/importer", "claude-code", "codex", "switch-1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("note is missing %q: %s", want, got)
		}
	}
}

func TestTUIProviderSwitchActivePhases(t *testing.T) {
	for _, phase := range []string{"prepared", "source_stopping", "target_starting", "recovery_required"} {
		if !(domain.TUIProviderSwitch{Phase: phase}).Active() {
			t.Fatalf("phase %q was not active", phase)
		}
	}
	for _, phase := range []string{"completed", "failed"} {
		if (domain.TUIProviderSwitch{Phase: phase}).Active() {
			t.Fatalf("phase %q was active", phase)
		}
	}
}

// recoveryOnlySwitchStore holds switch records and reports the active ones.
type recoveryOnlySwitchStore struct {
	tuiProviderSwitchStore
	rows []domain.TUIProviderSwitch
}

func (s *recoveryOnlySwitchStore) ListActiveTUIProviderSwitches(context.Context) ([]domain.TUIProviderSwitch, error) {
	var active []domain.TUIProviderSwitch
	for _, row := range s.rows {
		if row.Active() {
			active = append(active, row)
		}
	}
	return active, nil
}

func (s *recoveryOnlySwitchStore) AdvanceTUIProviderSwitch(_ context.Context, record domain.TUIProviderSwitch, from string) (bool, error) {
	for index, row := range s.rows {
		if row.ID == record.ID && row.Phase == from {
			s.rows[index] = record
			return true, nil
		}
	}
	return false, nil
}

func TestRecoveryRequiredSwitchSettlesOnceTheSessionRunsAgain(t *testing.T) {
	m := New(Deps{Agents: singleAgent{agent: fakeAgent{}}})
	newStore := func() *recoveryOnlySwitchStore {
		return &recoveryOnlySwitchStore{rows: []domain.TUIProviderSwitch{
			{ID: "stuck", SessionID: "session-1", Phase: "recovery_required"},
			{ID: "other-session", SessionID: "session-2", Phase: "recovery_required"},
			{ID: "in-flight", SessionID: "session-1", Phase: "target_starting"},
		}}
	}

	// No running provider yet: the record keeps refusing new switches.
	down := newStore()
	if err := m.settleRecoveredTUISwitches(context.Background(), down, domain.SessionRecord{ID: "session-1"}); err != nil {
		t.Fatal(err)
	}
	if down.rows[0].Phase != "recovery_required" {
		t.Fatalf("a session with no running provider settled its recovery record: %+v", down.rows[0])
	}

	// Running again: only this session's recovery record is closed.
	up := newStore()
	running := domain.SessionRecord{ID: "session-1", Metadata: domain.SessionMetadata{RuntimeHandleID: "runtime-1"}}
	if err := m.settleRecoveredTUISwitches(context.Background(), up, running); err != nil {
		t.Fatal(err)
	}
	if up.rows[0].Phase != "failed" || up.rows[1].Phase != "recovery_required" || up.rows[2].Phase != "target_starting" {
		t.Fatalf("settled the wrong records: %+v", up.rows)
	}
}

// childProbeRuntime adds the child-process probe the production runtime has.
type childProbeRuntime struct {
	*fakeRuntime
	childAlive bool
}

func (r *childProbeRuntime) IsChildAlive(context.Context, ports.RuntimeHandle) (bool, error) {
	return r.childAlive, nil
}

func TestSwitchTargetThatExitsAtOnceIsReportedWithItsLastLine(t *testing.T) {
	settle, poll := tuiSwitchTargetSettle, tuiSwitchTargetSettlePoll
	tuiSwitchTargetSettle, tuiSwitchTargetSettlePoll = 20*time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { tuiSwitchTargetSettle, tuiSwitchTargetSettlePoll = settle, poll })

	rec := domain.SessionRecord{ID: "s-1", Metadata: domain.SessionMetadata{RuntimeHandleID: "handle-1"}}
	dead := &childProbeRuntime{fakeRuntime: &fakeRuntime{outputs: []string{"starting\nError: Session ID 123 is already in use.\n\n"}}}
	err := (&Manager{runtime: dead}).confirmTUISwitchTargetAlive(context.Background(), rec)
	if err == nil || !strings.Contains(err.Error(), "Error: Session ID 123 is already in use.") {
		t.Fatalf("error = %v, want the agent's last line", err)
	}

	alive := &childProbeRuntime{fakeRuntime: &fakeRuntime{}, childAlive: true}
	if err := (&Manager{runtime: alive}).confirmTUISwitchTargetAlive(context.Background(), rec); err != nil {
		t.Fatalf("a running target was reported as failed: %v", err)
	}
	// A runtime that cannot say whether the child is alive never fails a switch.
	if err := (&Manager{runtime: &fakeRuntime{}}).confirmTUISwitchTargetAlive(context.Background(), rec); err != nil {
		t.Fatalf("a runtime without the probe failed the switch: %v", err)
	}
}
