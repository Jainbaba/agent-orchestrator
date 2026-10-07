package domain

import "testing"

func TestProviderSwitchRefusalKinds(t *testing.T) {
	for _, kind := range []ProviderSwitchRefusalKind{
		ProviderSwitchRefusalBusy,
		ProviderSwitchRefusalDraft,
		ProviderSwitchRefusalPendingInteraction,
		ProviderSwitchRefusalUnsupported,
		ProviderSwitchRefusalUnavailable,
		ProviderSwitchRefusalBootstrapTooLarge,
		ProviderSwitchRefusalRecoveryRequired,
	} {
		if !kind.Valid() {
			t.Errorf("refusal %q is not valid", kind)
		}
	}
	if ProviderSwitchRefusalKind("future").Valid() {
		t.Fatal("unknown refusal kind was accepted")
	}
}

func TestProviderSwitchRefusalRetryability(t *testing.T) {
	for _, tc := range []struct {
		kind      ProviderSwitchRefusalKind
		retryable bool
	}{
		{ProviderSwitchRefusalBusy, true},
		{ProviderSwitchRefusalPendingInteraction, true},
		{ProviderSwitchRefusalUnavailable, true},
		{ProviderSwitchRefusalRecoveryRequired, true},
		{ProviderSwitchRefusalDraft, false},
		{ProviderSwitchRefusalUnsupported, false},
		{ProviderSwitchRefusalBootstrapTooLarge, false},
	} {
		if got := tc.kind.Retryable(); got != tc.retryable {
			t.Errorf("%q.Retryable() = %v, want %v", tc.kind, got, tc.retryable)
		}
	}
}

func TestProviderSwitchLimits(t *testing.T) {
	if ProviderSwitchMaxBootstrapChars != 32_000 {
		t.Fatalf("bootstrap limit = %d, want 32000", ProviderSwitchMaxBootstrapChars)
	}
	if ProviderSwitchRecentExchangeLimit != 5 {
		t.Fatalf("exchange limit = %d, want 5", ProviderSwitchRecentExchangeLimit)
	}
}
