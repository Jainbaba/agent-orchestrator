package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// ProviderSwitchError is a stable, user-facing refusal from the provider
// switch admission gate. The wrapped cause is retained for logs and tests.
type ProviderSwitchError = domain.ProviderSwitchError

// ProviderSwitchResult is the durable identity returned after target
// publication. The bootstrap itself is deliberately never returned over HTTP.
type ProviderSwitchResult = domain.ProviderSwitchResult

// TerminalProviderSwitcher is late-bound during daemon construction.
type TerminalProviderSwitcher interface {
	SwitchTUIProvider(context.Context, domain.SessionID, domain.ProviderSwitchRequest) (domain.ProviderSwitchResult, error)
	TUIProviderSwitchCapabilities(context.Context, domain.SessionID, domain.AgentHarness) (domain.ProviderSwitchCapability, error)
}

func (s *Service) SetTerminalProviderSwitcher(switcher TerminalProviderSwitcher) {
	s.terminalSwitcher = switcher
}

type providerSwitchSessionWriter interface {
	UpdateSession(context.Context, domain.SessionRecord) error
}

type providerSwitchCommitter interface {
	CommitProviderSwitch(context.Context, domain.SessionRecord, domain.ConversationBranch, string, domain.ConversationActivity) error
}

type providerSwitchActivityDetail struct {
	Event         string              `json:"event"`
	SourceHarness domain.AgentHarness `json:"sourceHarness"`
	TargetHarness domain.AgentHarness `json:"targetHarness"`
	// The models and efforts on each side, where AO knows them. A side that
	// ran on its agent's default has none recorded.
	SourceModel  string `json:"sourceModel,omitempty"`
	SourceEffort string `json:"sourceEffort,omitempty"`
	TargetModel  string `json:"targetModel,omitempty"`
	TargetEffort string `json:"targetEffort,omitempty"`
	BranchID     string `json:"branchId"`
}

// SwitchProvider performs an idle-only Chat-to-Chat handoff. It retains the
// canonical AO conversation and publishes a new provider branch only after the
// target has started successfully.
func (s *Service) SwitchProvider(
	ctx context.Context,
	id domain.SessionID,
	request domain.ProviderSwitchRequest,
) (ProviderSwitchResult, error) {
	if request.TargetHarness == "" {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: "target provider is required"}
	}
	if record, found, err := s.sessions.GetSession(ctx, id); err != nil {
		return ProviderSwitchResult{}, err
	} else if found && domain.NormalizeSessionMode(record.Mode) == domain.SessionModeTUI && s.terminalSwitcher != nil {
		// One builder serves both UIs. A note that cannot be built must not
		// block the switch: the terminal switcher falls back to its own.
		note, noteErr := s.HandoffNote(ctx, record, request.TargetHarness, strings.TrimSpace(request.IdempotencyKey))
		if noteErr != nil {
			s.log.Warn("build terminal handoff note", "session", id, "error", noteErr)
		}
		request.HandoffNote = note
		return s.terminalSwitcher.SwitchTUIProvider(ctx, id, request)
	}
	record, err := s.requireChatSession(ctx, id)
	if err != nil {
		return ProviderSwitchResult{}, err
	}
	if key := strings.TrimSpace(request.IdempotencyKey); key != "" {
		if committed, found, lookupErr := s.committedProviderSwitch(ctx, id, key, request.TargetHarness); lookupErr != nil {
			return ProviderSwitchResult{}, lookupErr
		} else if found {
			return committed, nil
		}
		s.switchMu.Lock()
		previous, found := s.switchResults[string(id)+"\x00"+key]
		s.switchMu.Unlock()
		if found {
			if previous.TargetHarness != request.TargetHarness {
				return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "idempotency key belongs to a different provider switch"}
			}
			return previous, nil
		}
	}
	if record.IsTerminated {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "terminated sessions must be restored before switching"}
	}
	if !s.SupportsChat(request.TargetHarness) {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: fmt.Sprintf("%s does not support Chat", request.TargetHarness)}
	}
	if err := s.PreflightChat(ctx, request.TargetHarness, ports.PermissionModeAuto); err != nil {
		kind := domain.ProviderSwitchRefusalUnavailable
		if errors.Is(err, ports.ErrChatUnsupported) {
			kind = domain.ProviderSwitchRefusalUnsupported
		}
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: kind, Detail: err.Error(), Cause: err}
	}

	owner := domain.SessionConversationOwner(id)
	gate := s.controllerGate(owner)
	if err := gate.lock(ctx); err != nil {
		return ProviderSwitchResult{}, err
	}
	defer gate.unlock()
	if key := strings.TrimSpace(request.IdempotencyKey); key != "" {
		if committed, found, lookupErr := s.committedProviderSwitch(ctx, id, key, request.TargetHarness); lookupErr != nil {
			return ProviderSwitchResult{}, lookupErr
		} else if found {
			return committed, nil
		}
		s.switchMu.Lock()
		previous, found := s.switchResults[string(id)+"\x00"+key]
		s.switchMu.Unlock()
		if found {
			if previous.TargetHarness != request.TargetHarness {
				return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "idempotency key belongs to a different provider switch"}
			}
			return previous, nil
		}
	}

	source, err := s.Controller(id)
	if err != nil {
		if errors.Is(err, ErrNoController) {
			return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "the source Chat controller is not ready", Cause: err}
		}
		return ProviderSwitchResult{}, err
	}
	if source.harness == request.TargetHarness {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalUnsupported, Detail: "target provider is already active"}
	}
	if source.busy() {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: "the provider is still working"}
	}
	conversation, err := s.store.ConversationForSession(ctx, id)
	if err != nil {
		return ProviderSwitchResult{}, err
	}
	if pending, pendingErr := s.store.HasPendingConversationInteractions(ctx, conversation.ID); pendingErr != nil {
		return ProviderSwitchResult{}, pendingErr
	} else if pending {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalPendingInteraction, Detail: "answer the pending provider interaction before switching"}
	}
	handoffNote, err := s.HandoffNote(ctx, record, request.TargetHarness, strings.TrimSpace(request.IdempotencyKey))
	if err != nil {
		return ProviderSwitchResult{}, err
	}
	if handoffNote == "" || len([]rune(handoffNote)) > domain.ProviderSwitchMaxBootstrapChars {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalBootstrapTooLarge, Detail: "handoff context exceeds the provider budget"}
	}
	if err := source.BeginIdleBranchHandoff(ctx); err != nil {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalBusy, Detail: err.Error(), Cause: err}
	}
	abortSource := true
	defer func() {
		if abortSource {
			source.AbortHandoff()
		}
	}()

	s.mu.RLock()
	startCfg, found := s.startConfigs[source.owner()]
	s.mu.RUnlock()
	if !found {
		return ProviderSwitchResult{}, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "source launch configuration is unavailable"}
	}
	// The saved settings can name the session's owner from before an earlier
	// switch. Starting the target, or restoring the source, checks that owner
	// against the session, so take it from the session as it is now.
	startCfg.ExpectedControllerOwner = record.ControllerOwner()
	activeBranch, err := s.store.ConversationBranch(ctx, conversation.ID, conversation.ActiveBranchID)
	if err != nil {
		return ProviderSwitchResult{}, err
	}
	// What the outgoing provider was set to, for the handoff marker. Read before
	// it is stopped; a provider left on its default has neither.
	sourceSettings := source.Settings()
	sourceModel, sourceEffort := sourceSettings.Model, sourceSettings.ReasoningEffort
	if sourceModel == "" {
		sourceModel, sourceEffort = record.Metadata.Model, record.Metadata.Effort
	}
	sourceDriver, err := s.drivers.Driver(source.harness)
	if err != nil {
		return ProviderSwitchResult{}, err
	}
	operationCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), nativeEditHandoffLimit)
	defer cancel()
	if err := source.closeForBranchHandoff(operationCtx); err != nil {
		return ProviderSwitchResult{}, fmt.Errorf("stop source provider: %w", err)
	}

	targetCfg := cloneStartConfig(startCfg)
	targetCfg.Harness = request.TargetHarness
	targetCfg.ProviderConversationID = ""
	// The source may have been rejoined after a daemon restart. The target is a
	// new provider with no host to rejoin, so it must be allowed to start one.
	targetCfg.ReconnectOnly = false
	targetCfg.ProviderScopeID = ""
	targetCfg.ProviderHandoff = nil
	targetCfg.ProviderSwitch = &domain.ChatProviderSwitch{
		RequestID:        request.IdempotencyKey,
		SourceSessionID:  id,
		SourceHarness:    source.harness,
		TargetHarness:    request.TargetHarness,
		ConversationID:   conversation.ID,
		SourceBranchID:   activeBranch.ID,
		SourceSequence:   conversation.LatestSequence,
		SourceGeneration: record.Metadata.ControllerGeneration,
		Bootstrap:        handoffNote,
	}
	targetCfg.Model = request.Model
	targetCfg.Effort = request.Effort
	targetCfg.Permissions = ports.PermissionModeAuto
	targetCfg.ControllerReady = func(started StartResult) (ControllerCommit, error) {
		if started.ProviderBoundary == nil {
			return ControllerCommit{}, errors.New("provider switch target did not reserve a branch")
		}
		targetRecord := record
		targetRecord.Harness = request.TargetHarness
		targetRecord.Mode = domain.SessionModeChat
		targetRecord.IsTerminated = false
		targetRecord.Activity = domain.Activity{State: domain.ActivityIdle, LastActivityAt: s.now()}
		targetRecord.Metadata.ProviderConversationID = started.ProviderConversationID
		targetRecord.Metadata.ControllerGeneration = started.ControllerGeneration
		targetRecord.Metadata.Model = request.Model
		targetRecord.Metadata.Effort = request.Effort
		targetRecord.UpdatedAt = s.now()
		detail, _ := json.Marshal(providerSwitchActivityDetail{
			Event: "provider_switch", SourceHarness: source.harness,
			TargetHarness: request.TargetHarness, BranchID: started.ProviderBoundary.ID,
			SourceModel: sourceModel, SourceEffort: sourceEffort,
			TargetModel: request.Model, TargetEffort: request.Effort,
		})
		switchActivity := domain.ConversationActivity{
			ID: started.ProviderBoundary.ID + ":provider-switch", Kind: domain.ActivityKindSystem,
			Status:  domain.ActivityStatusCompleted,
			Summary: "Switched provider from " + string(source.harness) + " to " + string(request.TargetHarness),
			Detail:  detail, RequestID: strings.TrimSpace(request.IdempotencyKey),
			ProviderItemID: started.ProviderBoundary.ID + ":provider-switch",
		}
		if committer, ok := s.store.(providerSwitchCommitter); ok {
			// The note is stored with the branch it started, in the same
			// transaction that publishes the switch.
			boundary := *started.ProviderBoundary
			boundary.HandoffNote = handoffNote
			if err := committer.CommitProviderSwitch(ctx, targetRecord, boundary, started.ControllerGeneration, switchActivity); err != nil {
				return ControllerCommit{}, err
			}
		} else {
			if err := s.store.CreateAndActivateConversationBranch(ctx, id, *started.ProviderBoundary, started.ControllerGeneration, s.now()); err != nil {
				return ControllerCommit{}, err
			}
			writer, ok := s.sessions.(providerSwitchSessionWriter)
			if !ok {
				return ControllerCommit{}, errors.New("provider switch session writer is unavailable")
			}
			if err := writer.UpdateSession(ctx, targetRecord); err != nil {
				return ControllerCommit{}, err
			}
		}
		committed := started.Conversation
		committed.ActiveBranchID = started.ProviderBoundary.ID
		committed.SessionID = id
		return ControllerCommit{Conversation: committed, ControllerOwner: domain.SessionControllerOwner{
			Harness: request.TargetHarness, Mode: domain.SessionModeChat,
			ProviderConversationID: started.ProviderConversationID,
			ControllerGeneration:   started.ControllerGeneration,
		}}, nil
	}

	target, err := s.startLocked(operationCtx, targetCfg)
	if err != nil {
		if restoreErr := s.restoreClosedSourceController(operationCtx, id, source, activeBranch, startCfg, sourceDriver); restoreErr != nil {
			err = errors.Join(err, restoreErr)
		} else {
			abortSource = false
		}
		return ProviderSwitchResult{}, fmt.Errorf("start target provider: %w", err)
	}
	source.completeBranchHandoff()
	abortSource = false
	if request.FastMode {
		// Fast mode is a setting of the running controller, so it is applied to
		// the target once it exists rather than passed through its launch.
		settings := target.Settings()
		settings.FastMode = true
		if err := target.SetSettings(ctx, settings); err != nil {
			s.log.Warn("apply fast mode after provider switch", "session", id, "error", err)
		}
	}
	result := ProviderSwitchResult{
		SessionID: id, ConversationID: conversation.ID, BranchID: target.conversation.ActiveBranchID,
		SourceHarness: source.harness, TargetHarness: request.TargetHarness,
		ProviderConversation: target.ProviderConversationID(),
	}
	if key := strings.TrimSpace(request.IdempotencyKey); key != "" {
		s.switchMu.Lock()
		if len(s.switchResults) >= 1024 {
			// Request keys are a retry aid, not a second durable store. Bound the
			// process-local cache; the source branch remains the recovery authority.
			s.switchResults = make(map[string]ProviderSwitchResult)
		}
		s.switchResults[string(id)+"\x00"+key] = result
		s.switchMu.Unlock()
	}
	// The canonical AO timeline records the ownership change after the target is
	// durable. The bootstrap remains hidden; this marker is the user-visible fact
	// that explains why the provider label changed without adding a fake message.
	detail, _ := json.Marshal(providerSwitchActivityDetail{
		Event: "provider_switch", SourceHarness: result.SourceHarness,
		TargetHarness: result.TargetHarness, BranchID: result.BranchID,
		SourceModel: sourceModel, SourceEffort: sourceEffort,
		TargetModel: request.Model, TargetEffort: request.Effort,
	})
	if err := s.store.UpsertActivity(ctx, conversation.ID, "ao-provider-switch-"+result.BranchID, domain.ConversationActivity{
		ID: result.BranchID + ":provider-switch", Kind: domain.ActivityKindSystem, Status: domain.ActivityStatusCompleted,
		Summary: "Switched provider from " + string(result.SourceHarness) + " to " + string(result.TargetHarness), Detail: detail,
		RequestID: strings.TrimSpace(request.IdempotencyKey), ProviderItemID: result.BranchID + ":provider-switch",
	}, s.now()); err != nil {
		s.log.Warn("record provider switch activity", "session", id, "error", err)
	}
	return result, nil
}

// committedProviderSwitch makes idempotency durable across daemon restarts.
// The SQLite implementation writes the marker in the same transaction as the
// branch/session publication; this read is only a recovery lookup, never the
// ownership authority.
func (s *Service) committedProviderSwitch(ctx context.Context, id domain.SessionID, key string, target domain.AgentHarness) (ProviderSwitchResult, bool, error) {
	if s.reader == nil || strings.TrimSpace(key) == "" {
		return ProviderSwitchResult{}, false, nil
	}
	conversation, err := s.store.ConversationForSession(ctx, id)
	if err != nil {
		return ProviderSwitchResult{}, false, err
	}
	rows, err := s.reader.LoadConversationSnapshot(ctx, conversation.ID)
	if err != nil {
		return ProviderSwitchResult{}, false, err
	}
	for _, activity := range rows.Activities {
		if activity.RequestID != key || activity.Status != domain.ActivityStatusCompleted {
			continue
		}
		var detail providerSwitchActivityDetail
		if json.Unmarshal(activity.Detail, &detail) != nil || detail.Event != "provider_switch" {
			continue
		}
		if detail.TargetHarness != target {
			return ProviderSwitchResult{}, false, &ProviderSwitchError{Kind: domain.ProviderSwitchRefusalRecoveryRequired, Detail: "idempotency key belongs to a different provider switch"}
		}
		branchID := detail.BranchID
		if branchID == "" {
			branchID = strings.TrimSuffix(activity.ProviderItemID, ":provider-switch")
		}
		branch, branchErr := s.store.ConversationBranch(ctx, conversation.ID, branchID)
		if branchErr != nil {
			return ProviderSwitchResult{}, false, branchErr
		}
		return ProviderSwitchResult{
			SessionID: id, ConversationID: conversation.ID, BranchID: branchID,
			SourceHarness: detail.SourceHarness, TargetHarness: detail.TargetHarness,
			ProviderConversation: branch.ProviderConversationID,
		}, true, nil
	}
	return ProviderSwitchResult{}, false, nil
}

// ProviderSwitchCapabilities evaluates one target without starting it.
func (s *Service) ProviderSwitchCapabilities(
	ctx context.Context,
	id domain.SessionID,
	target domain.AgentHarness,
) (domain.ProviderSwitchCapability, error) {
	if record, found, err := s.sessions.GetSession(ctx, id); err != nil {
		return domain.ProviderSwitchCapability{}, err
	} else if found && domain.NormalizeSessionMode(record.Mode) == domain.SessionModeTUI && s.terminalSwitcher != nil {
		return s.terminalSwitcher.TUIProviderSwitchCapabilities(ctx, id, target)
	}
	record, err := s.requireChatSession(ctx, id)
	if err != nil {
		return domain.ProviderSwitchCapability{}, err
	}
	capability := domain.ProviderSwitchCapability{Harness: target}
	if record.IsTerminated {
		capability.Reason = domain.ProviderSwitchRefusalRecoveryRequired
		capability.Retryable = true
		return capability, nil
	}
	if controller, controllerErr := s.Controller(id); controllerErr == nil && controller.busy() {
		capability.Reason = domain.ProviderSwitchRefusalBusy
		capability.Retryable = true
		return capability, nil
	}
	if !s.SupportsChat(target) {
		capability.Reason = domain.ProviderSwitchRefusalUnsupported
		return capability, nil
	}
	if err := s.PreflightChat(ctx, target, ports.PermissionModeAuto); err != nil {
		capability.Reason = domain.ProviderSwitchRefusalUnavailable
		capability.Retryable = true
		return capability, nil
	}
	capability.Supported = true
	return capability, nil
}
