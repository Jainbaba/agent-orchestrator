package domain

import "time"

// TUIProviderSwitch persists the recovery coordinates before any process stops.
// Bootstrap is the handoff note: private daemon data, never an API response.
type TUIProviderSwitch struct {
	ID              string
	SessionID       SessionID
	RequestID       string
	SourceHarness   AgentHarness
	TargetHarness   AgentHarness
	SourceNativeID  string
	SourceLaunchID  string
	SourceRuntimeID string
	SourceModel     string
	SourceEffort    string
	TargetNativeID  string
	TargetLaunchID  string
	Phase           string
	Bootstrap       string
	Error           string
	CreatedAt       time.Time
}

func (s TUIProviderSwitch) Active() bool {
	return s.Phase != "completed" && s.Phase != "failed"
}
