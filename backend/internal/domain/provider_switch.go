package domain

// ProviderSwitchMaxBootstrapChars is the hard upper bound for the text sent to
// a target provider during a handoff.
const ProviderSwitchMaxBootstrapChars = 32_000

// ProviderSwitchRecentExchangeLimit is the number of complete user/final
// assistant exchanges preferred in a handoff bootstrap.
const ProviderSwitchRecentExchangeLimit = 5

// ProviderSwitchRefusalKind is a stable reason a provider switch was not
// admitted. The HTTP layer can map these values to retryable UI actions without
// parsing provider prose.
type ProviderSwitchRefusalKind string

const (
	ProviderSwitchRefusalBusy               ProviderSwitchRefusalKind = "busy"
	ProviderSwitchRefusalDraft              ProviderSwitchRefusalKind = "draft"
	ProviderSwitchRefusalPendingInteraction ProviderSwitchRefusalKind = "pending_interaction"
	ProviderSwitchRefusalUnsupported        ProviderSwitchRefusalKind = "unsupported"
	ProviderSwitchRefusalUnavailable        ProviderSwitchRefusalKind = "unavailable"
	ProviderSwitchRefusalBootstrapTooLarge  ProviderSwitchRefusalKind = "bootstrap_too_large"
	ProviderSwitchRefusalRecoveryRequired   ProviderSwitchRefusalKind = "recovery_required"
)

// Valid reports whether the refusal kind is one AO understands.
func (k ProviderSwitchRefusalKind) Valid() bool {
	switch k {
	case ProviderSwitchRefusalBusy,
		ProviderSwitchRefusalDraft,
		ProviderSwitchRefusalPendingInteraction,
		ProviderSwitchRefusalUnsupported,
		ProviderSwitchRefusalUnavailable,
		ProviderSwitchRefusalBootstrapTooLarge,
		ProviderSwitchRefusalRecoveryRequired:
		return true
	default:
		return false
	}
}

// Retryable reports whether waiting or retrying can reasonably change the
// result. Unsupported and over-budget payloads require a different choice.
func (k ProviderSwitchRefusalKind) Retryable() bool {
	switch k {
	case ProviderSwitchRefusalBusy,
		ProviderSwitchRefusalPendingInteraction,
		ProviderSwitchRefusalUnavailable,
		ProviderSwitchRefusalRecoveryRequired:
		return true
	default:
		return false
	}
}

// ProviderSwitchRequest is the provider-neutral input to a future daemon
// switch operation. The idempotency key is caller-owned and is never sent to a
// provider as conversation content.
type ProviderSwitchRequest struct {
	TargetHarness  AgentHarness
	IdempotencyKey string
	// Model and Effort are what the target starts on. Empty keeps its default.
	Model  string
	Effort string
	// FastMode starts the target on its faster tier, where its model has one.
	FastMode bool
	// Ultracode and AutoCompactWindow are Claude Code launch choices for a
	// terminal target. See AgentLaunchOptions.
	Ultracode         bool
	AutoCompactWindow string
	// HandoffNote is the note the target provider starts with. The switch
	// front door builds it; it is never accepted from a client.
	HandoffNote string
}

// ChatProviderSwitch is an internal launch reservation. The source controller
// is fenced before the target starts; Start uses this proof to reuse the same
// AO conversation while creating a fresh provider branch.
type ChatProviderSwitch struct {
	RequestID        string
	SourceSessionID  SessionID
	SourceHarness    AgentHarness
	TargetHarness    AgentHarness
	ConversationID   string
	SourceBranchID   string
	SourceSequence   int64
	SourceGeneration string
	Bootstrap        string
}

// ProviderSwitchCapability describes whether one target can be offered for the
// current session. Wire details stay behind ports.ChatDriver.
type ProviderSwitchCapability struct {
	Harness   AgentHarness
	Supported bool
	Reason    ProviderSwitchRefusalKind
	Retryable bool
}

// ProviderSwitchError crosses the Chat/TUI dispatch boundary without coupling
// the session manager to either transport controller.
type ProviderSwitchError struct {
	Kind   ProviderSwitchRefusalKind
	Detail string
	Cause  error
}

func (e *ProviderSwitchError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return string(e.Kind)
}
func (e *ProviderSwitchError) Unwrap() error { return e.Cause }

type ProviderSwitchResult struct {
	SessionID            SessionID
	ConversationID       string
	BranchID             string
	SourceHarness        AgentHarness
	TargetHarness        AgentHarness
	ProviderConversation string
}
