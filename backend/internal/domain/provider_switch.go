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
}

// ProviderSwitchCapability describes whether one target can be offered for the
// current session. Wire details stay behind ports.ChatDriver.
type ProviderSwitchCapability struct {
	Harness   AgentHarness
	Supported bool
	Reason    ProviderSwitchRefusalKind
	Retryable bool
}
