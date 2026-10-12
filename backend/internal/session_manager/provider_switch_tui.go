package sessionmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// tuiSwitchPreflightTimeout bounds the target checks a switch runs before it
// touches the running agent.
const tuiSwitchPreflightTimeout = 10 * time.Second

type tuiProviderSwitchStore interface {
	CreateTUIProviderSwitch(context.Context, domain.TUIProviderSwitch) (bool, error)
	AdvanceTUIProviderSwitch(context.Context, domain.TUIProviderSwitch, string) (bool, error)
	GetTUIProviderSwitch(context.Context, domain.SessionID, string) (domain.TUIProviderSwitch, bool, error)
	ListActiveTUIProviderSwitches(context.Context) ([]domain.TUIProviderSwitch, error)
	HasActiveTUIProviderSwitch(context.Context, domain.SessionID) (bool, error)
}

// TUIProviderSwitchError uses the provider-neutral refusal contract so the HTTP
// controller can classify Chat and Terminal failures identically.
type TUIProviderSwitchError = domain.ProviderSwitchError

// TUIProviderSwitchResult is the durable identity returned after a target TUI
// has started. The AO session remains the same; only its active harness and
// provider-native session identity change.
type TUIProviderSwitchResult = domain.ProviderSwitchResult

// SwitchTUIProvider replaces one terminal provider with another in the same AO
// session and terminal surface. The target always starts a fresh native thread
// seeded with bounded AO handoff text; provider-native histories cannot be
// merged across harnesses safely.
func (m *Manager) SwitchTUIProvider(
	ctx context.Context,
	id domain.SessionID,
	request domain.ProviderSwitchRequest,
) (domain.ProviderSwitchResult, error) {
	target := request.TargetHarness
	idempotencyKey := request.IdempotencyKey
	if strings.TrimSpace(string(target)) == "" {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: "target provider is required"}
	}
	if err := m.beginAgentOperation(ctx, id, agentOperationProviderSwitch); err != nil {
		if errors.Is(err, errAgentOperationInProgress) {
			return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "another session operation is in progress", Cause: err}
		}
		return TUIProviderSwitchResult{}, err
	}
	defer m.endAgentOperation(id, agentOperationProviderSwitch)

	rec, found, err := m.store.GetSession(ctx, id)
	if err != nil {
		return TUIProviderSwitchResult{}, err
	}
	if !found {
		return TUIProviderSwitchResult{}, ErrNotFound
	}
	if domain.NormalizeSessionMode(rec.Mode) != domain.SessionModeTUI {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: "provider switching for this session is handled by the Chat controller"}
	}
	if rec.IsTerminated {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "terminated sessions must be restored before switching"}
	}
	var durable tuiProviderSwitchStore
	if candidate, ok := m.store.(tuiProviderSwitchStore); ok {
		durable = candidate
	}
	if durable != nil {
		if err := m.settleRecoveredTUISwitches(ctx, durable, rec); err != nil {
			return TUIProviderSwitchResult{}, err
		}
		if key := strings.TrimSpace(request.IdempotencyKey); key != "" {
			if previous, found, readErr := durable.GetTUIProviderSwitch(ctx, id, key); readErr != nil {
				return TUIProviderSwitchResult{}, readErr
			} else if found {
				if previous.TargetHarness != target {
					return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "idempotency key belongs to a different provider switch"}
				}
				if previous.Phase == "completed" {
					return domain.ProviderSwitchResult{SessionID: id, SourceHarness: previous.SourceHarness, TargetHarness: previous.TargetHarness, ProviderConversation: previous.TargetNativeID}, nil
				}
				if !previous.Active() {
					return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnavailable, Detail: "this switch request already failed: " + previous.Error}
				}
				return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "a provider switch is already in progress"}
			}
		}
		if active, readErr := durable.HasActiveTUIProviderSwitch(ctx, id); readErr != nil {
			return TUIProviderSwitchResult{}, readErr
		} else if active {
			return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "a provider switch is already in progress"}
		}
	}
	if rec.Harness == target {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: "target provider is already active"}
	}
	active, err := m.hasActiveInterfaceTransition(ctx, id)
	if err != nil {
		return TUIProviderSwitchResult{}, err
	}
	if active {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "an interface switch is already in progress"}
	}
	targetAgent, ok := m.agents.Agent(target)
	if !ok || targetAgent == nil {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: fmt.Sprintf("%s is not available for Terminal UI", target)}
	}
	if rec.Activity.State == domain.ActivityActive {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "the provider is still working"}
	}
	if rec.Activity.State == domain.ActivityWaitingInput || rec.Activity.State == domain.ActivityBlocked {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalPendingInteraction, Detail: "answer the pending provider interaction before switching"}
	}

	switchID := strings.TrimSpace(idempotencyKey)
	if switchID == "" {
		switchID = m.newLaunchID()
	}
	durableRequestID := strings.TrimSpace(request.IdempotencyKey)
	if durableRequestID == "" {
		durableRequestID = switchID
	}
	// The switch front door builds the note from every turn AO holds for the
	// session. A caller that supplies none still gets one from the terminal
	// turns stored here.
	bootstrap := strings.TrimSpace(request.HandoffNote)
	if bootstrap == "" {
		bootstrap = m.terminalHandoffNote(ctx, rec, target, switchID)
	}
	if bootstrap == "" || len([]rune(bootstrap)) > domain.ProviderSwitchMaxBootstrapChars {
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBootstrapTooLarge, Detail: "handoff context exceeds the provider budget"}
	}
	// Everything the relaunch needs is read before the source is stopped, so a
	// failed lookup cannot leave the session without a provider.
	project, err := m.loadProject(ctx, rec.ProjectID)
	if err != nil {
		return TUIProviderSwitchResult{}, err
	}
	ws := workspaceInfo(rec)
	if err := preflightTUISwitchTarget(ctx, target, targetAgent); err != nil {
		return TUIProviderSwitchResult{}, err
	}

	var durableRecord domain.TUIProviderSwitch
	if durable != nil {
		durableRecord = domain.TUIProviderSwitch{ID: switchID, SessionID: id, RequestID: durableRequestID, SourceHarness: rec.Harness, TargetHarness: target, SourceNativeID: rec.Metadata.AgentSessionID, SourceLaunchID: rec.Metadata.RuntimeLaunchID, SourceRuntimeID: rec.Metadata.RuntimeHandleID, SourceModel: rec.Metadata.Model, SourceEffort: rec.Metadata.Effort, Phase: "prepared", Bootstrap: bootstrap, CreatedAt: m.clock()}
		if created, createErr := durable.CreateTUIProviderSwitch(ctx, durableRecord); createErr != nil {
			return TUIProviderSwitchResult{}, createErr
		} else if !created {
			return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "a provider switch is already in progress"}
		}
	}
	// advance moves the durable record to its next phase. The record is how a
	// switch interrupted by a crash is recognised, so a failed write is never
	// silent, and it outlives a cancelled request.
	advance := func(phase, cause string) error {
		if durable == nil {
			return nil
		}
		next := durableRecord
		next.Phase, next.Error = phase, cause
		moved, advanceErr := durable.AdvanceTUIProviderSwitch(context.WithoutCancel(ctx), next, durableRecord.Phase)
		if advanceErr == nil && !moved {
			advanceErr = fmt.Errorf("switch record is no longer in phase %q", durableRecord.Phase)
		}
		if advanceErr != nil {
			m.logger.Error("advance TUI provider switch", "sessionID", id, "phase", phase, "error", advanceErr)
			return advanceErr
		}
		durableRecord = next
		return nil
	}

	lastInputAt, releaseInput := m.beginTerminalInputDrain(rec)
	if releaseInput != nil {
		defer releaseInput()
	}
	if err := m.prepareSourceHandoff(ctx, rec, domain.SessionInterfaceTransitionDrain, lastInputAt); err != nil {
		_ = advance("failed", err.Error())
		return TUIProviderSwitchResult{}, tuiProviderSwitchRefusal(err)
	}
	// Stopping the source cannot be undone, so it does not happen unless the
	// record already says a stop is under way.
	if err := advance("source_stopping", ""); err != nil {
		_ = advance("failed", err.Error())
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnavailable, Detail: "could not record the provider switch: " + err.Error(), Cause: err}
	}
	if err := m.stopSourceControllerConclusive(rec); err != nil {
		_ = advance("recovery_required", err.Error())
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "AO could not confirm that the source provider stopped: " + err.Error(), Cause: err}
	}

	source := rec
	source.Metadata.RuntimeHandleID = ""
	source.Metadata.RuntimeLaunchID = ""
	source.Metadata.AgentSessionIDLaunchID = ""
	source.Activity = domain.Activity{State: domain.ActivityIdle, LastActivityAt: m.clock()}
	source.UpdatedAt = m.clock()

	// From here the source is stopped. Any failure brings it back on its old
	// native thread, so the user is never left with a blank terminal.
	// The target's launch choices replace the source's. They are saved before
	// the target starts, and put back if the source has to be restored.
	launchStore, _ := m.store.(launchOptionsStore)
	sourceLaunch := m.sessionLaunchOptions(ctx, id)
	restoreSource := func(cause error, what string) (TUIProviderSwitchResult, error) {
		if launchStore != nil {
			_ = launchStore.SetSessionLaunchOptions(context.WithoutCancel(ctx), id, sourceLaunch)
		}
		if restoreErr := m.restoreTUIProvider(ctx, source, project, ws); restoreErr != nil {
			joined := errors.Join(cause, restoreErr)
			_ = advance("recovery_required", joined.Error())
			return TUIProviderSwitchResult{}, &TUIProviderSwitchError{
				Kind:   domain.ProviderSwitchRefusalRecoveryRequired,
				Detail: fmt.Sprintf("%s and source restoration also failed: %v; %v", what, cause, restoreErr),
				Cause:  joined,
			}
		}
		_ = advance("failed", cause.Error())
		return TUIProviderSwitchResult{}, &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnavailable, Detail: what + ": " + cause.Error(), Cause: cause}
	}

	targetRecord := source
	targetRecord.Harness = target
	targetRecord.Metadata.AgentSessionID = ""
	targetRecord.Metadata.NativeTranscriptPath = ""
	targetRecord.Metadata.NativeIdentityObservedAt = time.Time{}
	targetRecord.Metadata.Model = request.Model
	targetRecord.Metadata.Effort = request.Effort
	targetRecord.Metadata.LatestUserPrompt = ""
	targetRecord.Metadata.LatestAssistantUpdate = ""
	targetRecord.Metadata.ConversationCheckpointState = domain.ConversationCheckpointEmpty
	targetRecord.Metadata.ConversationCheckpointGeneration = ""
	targetRecord.Metadata.ConversationCheckpointNativeID = ""
	targetRecord.Metadata.ConversationCheckpointTurnID = ""
	targetRecord.Metadata.ConversationCheckpointUnsettled = false
	targetRecord.UpdatedAt = m.clock()
	if err := m.store.UpdateSession(ctx, targetRecord); err != nil {
		return restoreSource(err, "could not publish target provider ownership")
	}
	if launchStore != nil {
		targetLaunch := domain.AgentLaunchOptions{
			FastMode: request.FastMode, Ultracode: request.Ultracode, AutoCompactWindow: request.AutoCompactWindow,
		}
		if err := launchStore.SetSessionLaunchOptions(ctx, id, targetLaunch); err != nil {
			return restoreSource(err, "could not save the target's launch options")
		}
	}
	_ = advance("target_starting", "")

	started, err := m.relaunchSessionWithPolicyAndGenerationPrompt(
		context.WithoutCancel(ctx), "switch TUI provider", targetRecord, project, ws, nil,
		true, false, "", domain.SessionInterfaceTransitionHistoryStrict, &bootstrap,
	)
	if err != nil {
		return restoreSource(err, "target provider failed to start")
	}
	if err := m.confirmTUISwitchTargetAlive(context.WithoutCancel(ctx), started.Session); err != nil {
		// The target's terminal outlives it to keep its last output. It has to
		// go before the source can have the session's terminal back.
		m.destroySpawnRuntimeAfterFailure(context.WithoutCancel(ctx), runtimeHandle(started.Session.Metadata))
		return restoreSource(err, "target provider did not stay running")
	}
	durableRecord.TargetNativeID = started.Session.Metadata.AgentSessionID
	_ = advance("completed", "")
	return domain.ProviderSwitchResult{
		SessionID: id, SourceHarness: rec.Harness, TargetHarness: target,
		ProviderConversation: started.Session.Metadata.AgentSessionID,
	}, nil
}

// How long a freshly started target is watched before the switch is reported
// as done. Variables so tests do not have to wait.
var (
	tuiSwitchTargetSettle     = 1500 * time.Millisecond
	tuiSwitchTargetSettlePoll = 250 * time.Millisecond
)

const tuiSwitchTargetExitLines = 8

// confirmTUISwitchTargetAlive watches the target for a moment after it starts.
// An agent that exits straight away (a refused flag or session id, a launcher
// that gives up) would otherwise be reported as a successful switch over a dead
// terminal. The error carries the last line the agent printed, which is usually
// its reason. A runtime that cannot answer does not fail the switch.
func (m *Manager) confirmTUISwitchTargetAlive(ctx context.Context, rec domain.SessionRecord) error {
	inspector, ok := m.runtime.(ports.RuntimeChildInspector)
	handle := runtimeHandle(rec.Metadata)
	if !ok || handle.ID == "" {
		return nil
	}
	deadline := time.Now().Add(tuiSwitchTargetSettle)
	for {
		if alive, err := inspector.IsChildAlive(ctx, handle); err == nil && !alive {
			reason := "it exited as soon as it started"
			if output, outputErr := m.runtime.GetOutput(ctx, handle, tuiSwitchTargetExitLines); outputErr == nil {
				lines := strings.Split(strings.TrimSpace(domain.SanitizeControlChars(output)), "\n")
				if last := strings.TrimSpace(lines[len(lines)-1]); last != "" {
					reason += ": " + truncateForSwitchError(last)
				}
			}
			return errors.New(reason)
		}
		if !time.Now().Before(deadline) {
			return nil
		}
		time.Sleep(tuiSwitchTargetSettlePoll)
	}
}

func truncateForSwitchError(line string) string {
	const limit = 240
	if runes := []rune(line); len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return line
}

// preflightTUISwitchTarget refuses a switch whose target cannot launch while
// the current agent is still running. Without it the target's missing binary or
// sign-in is discovered only after the source was stopped, and the user's agent
// is restarted for nothing. A probe that cannot answer does not block: the
// launch itself still validates.
func preflightTUISwitchTarget(ctx context.Context, harness domain.AgentHarness, target ports.Agent) error {
	ctx, cancel := context.WithTimeout(ctx, tuiSwitchPreflightTimeout)
	defer cancel()
	if resolver, ok := target.(ports.AgentBinaryResolver); ok {
		if _, err := resolver.ResolveBinary(ctx); errors.Is(err, ports.ErrAgentBinaryNotFound) {
			return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnavailable, Detail: fmt.Sprintf("%s is not installed", harness), Cause: err}
		}
	}
	if checker, ok := target.(ports.AgentAuthChecker); ok {
		if status, err := checker.AuthStatus(ctx); err == nil && status == ports.AgentAuthStatusUnauthorized {
			return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnavailable, Detail: fmt.Sprintf("%s is not signed in", harness), Cause: ports.ErrAgentAuthRequired}
		}
	}
	return nil
}

// settleRecoveredTUISwitches closes switch records left in recovery_required
// once their session has a running provider again. That phase means AO could
// not prove what state a switch ended in; a session that is back up has
// answered the question, and leaving the record open would refuse every later
// switch for good.
func (m *Manager) settleRecoveredTUISwitches(ctx context.Context, durable tuiProviderSwitchStore, rec domain.SessionRecord) error {
	if rec.IsTerminated || rec.Metadata.RuntimeHandleID == "" {
		return nil
	}
	rows, err := durable.ListActiveTUIProviderSwitches(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.SessionID != rec.ID || row.Phase != "recovery_required" {
			continue
		}
		row.Phase = "failed"
		if _, err := durable.AdvanceTUIProviderSwitch(ctx, row, "recovery_required"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) restoreTUIProvider(ctx context.Context, source domain.SessionRecord, project domain.ProjectRecord, ws ports.WorkspaceInfo) error {
	if err := m.store.UpdateSession(ctx, source); err != nil {
		return err
	}
	_, err := m.relaunchSessionWithPolicy(
		context.WithoutCancel(ctx), "restore source provider", source, project, ws, nil,
		false, false, domain.SessionInterfaceTransitionHistoryStrict,
	)
	return err
}

func tuiProviderSwitchRefusal(err error) error {
	if errors.Is(err, errDrainDraftPresent) {
		return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalDraft, Detail: err.Error(), Cause: err}
	}
	if errors.Is(err, errDrainDecisionPending) {
		return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalPendingInteraction, Detail: err.Error(), Cause: err}
	}
	if errors.Is(err, errDrainQuiescenceUnverified) {
		return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: err.Error(), Cause: err}
	}
	return &TUIProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: err.Error(), Cause: err}
}

func (m *Manager) TUIProviderSwitchCapabilities(ctx context.Context, id domain.SessionID, target domain.AgentHarness) (domain.ProviderSwitchCapability, error) {
	capability := domain.ProviderSwitchCapability{Harness: target}
	rec, found, err := m.store.GetSession(ctx, id)
	if err != nil {
		return capability, err
	}
	if !found {
		return capability, ErrNotFound
	}
	if domain.NormalizeSessionMode(rec.Mode) != domain.SessionModeTUI {
		capability.Reason = domain.ProviderSwitchRefusalUnsupported
		return capability, nil
	}
	if rec.IsTerminated {
		capability.Reason = domain.ProviderSwitchRefusalRecoveryRequired
		capability.Retryable = true
		return capability, nil
	}
	if rec.Harness == target {
		capability.Reason = domain.ProviderSwitchRefusalUnsupported
		return capability, nil
	}
	if durable, ok := m.store.(tuiProviderSwitchStore); ok {
		if err := m.settleRecoveredTUISwitches(ctx, durable, rec); err != nil {
			return capability, err
		}
		active, err := durable.HasActiveTUIProviderSwitch(ctx, id)
		if err != nil {
			return capability, err
		}
		if active {
			capability.Reason = domain.ProviderSwitchRefusalRecoveryRequired
			capability.Retryable = true
			return capability, nil
		}
	}
	if _, ok := m.agents.Agent(target); !ok {
		capability.Reason = domain.ProviderSwitchRefusalUnsupported
		return capability, nil
	}
	if rec.Activity.State == domain.ActivityActive {
		capability.Reason = domain.ProviderSwitchRefusalBusy
		capability.Retryable = true
		return capability, nil
	}
	if rec.Activity.State == domain.ActivityWaitingInput || rec.Activity.State == domain.ActivityBlocked {
		capability.Reason = domain.ProviderSwitchRefusalPendingInteraction
		capability.Retryable = true
		return capability, nil
	}
	capability.Supported = true
	return capability, nil
}

// reconcileInterruptedTUIProviderSwitches fences rows whose in-memory saga was
// lost with the previous daemon. The source process is never guessed at boot:
// the durable session is left for an explicit restore, and the API reports a
// recovery-required refusal until the operator has inspected it.
func (m *Manager) reconcileInterruptedTUIProviderSwitches(ctx context.Context) error {
	durable, ok := m.store.(tuiProviderSwitchStore)
	if !ok {
		return nil
	}
	rows, err := durable.ListActiveTUIProviderSwitches(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		from := row.Phase
		row.Phase = "recovery_required"
		if row.Error == "" {
			row.Error = "AO restarted while switching providers; restore the session before retrying"
		}
		if _, err := durable.AdvanceTUIProviderSwitch(ctx, row, from); err != nil {
			return err
		}
	}
	return nil
}

type terminalTurnLister interface {
	ListTerminalTurns(context.Context, domain.SessionID, int) ([]domain.TerminalTurn, error)
}

// terminalHandoffNote builds a handoff note from this session's stored Terminal
// UI turns alone. It is the fallback for a switch that did not come through the
// front door, which also merges Chat history and the workspace's changed files.
func (m *Manager) terminalHandoffNote(ctx context.Context, rec domain.SessionRecord, target domain.AgentHarness, switchID string) string {
	var turns []domain.HandoffTurn
	if lister, ok := m.store.(terminalTurnLister); ok {
		stored, err := lister.ListTerminalTurns(ctx, rec.ID, domain.TerminalTurnRetention)
		if err != nil {
			m.logger.Warn("read terminal turns for handoff", "session", rec.ID, "error", err)
		}
		for _, turn := range stored {
			turns = append(turns, domain.HandoffTurn{
				UserRequest: turn.UserRequest, AgentResponse: turn.AgentResponse,
				At: turn.RequestedAt, Harness: turn.Harness,
			})
		}
	}
	return domain.BuildHandoffNote(domain.HandoffNoteInput{
		SwitchID:      switchID,
		SourceHarness: rec.Harness,
		TargetHarness: target,
		SessionTitle:  rec.DisplayName,
		Project:       string(rec.ProjectID),
		WorkspacePath: rec.Metadata.WorkspacePath,
		Branch:        rec.Metadata.Branch,
		At:            m.clock(),
		OriginalTask:  rec.Metadata.Prompt,
		Turns:         turns,
		Terminal:      true,
		// `ao session history` reads terminal turns too.
		HistoryCommand: true,
	})
}
