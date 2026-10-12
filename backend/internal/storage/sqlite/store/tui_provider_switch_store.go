package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// CreateTUIProviderSwitch claims the durable per-session provider-switch slot.
func (s *Store) CreateTUIProviderSwitch(ctx context.Context, record domain.TUIProviderSwitch) (bool, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("encode TUI provider switch: %w", err)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	err = s.qw.InsertTUIProviderSwitch(ctx, gen.InsertTUIProviderSwitchParams{
		ID: record.ID, SessionID: string(record.SessionID), RequestID: record.RequestID,
		Phase: record.Phase, RecordJson: string(payload), CreatedAt: record.CreatedAt,
	})
	if err == nil {
		return true, nil
	}
	if _, found, lookupErr := s.GetTUIProviderSwitch(ctx, record.SessionID, record.RequestID); lookupErr == nil && found {
		return false, nil
	}
	if active, lookupErr := s.HasActiveTUIProviderSwitch(ctx, record.SessionID); lookupErr == nil && active {
		return false, nil
	}
	// SQLite reports the unique active/request claim as a constraint error. The
	// caller re-reads the row to make retries idempotent.
	return false, fmt.Errorf("create TUI provider switch: %w", err)
}

func (s *Store) AdvanceTUIProviderSwitch(ctx context.Context, record domain.TUIProviderSwitch, from string) (bool, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("encode TUI provider switch: %w", err)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.UpdateTUIProviderSwitch(ctx, gen.UpdateTUIProviderSwitchParams{Phase: record.Phase, RecordJson: string(payload), ID: record.ID, Phase_2: from})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (s *Store) GetTUIProviderSwitch(ctx context.Context, sessionID domain.SessionID, requestID string) (domain.TUIProviderSwitch, bool, error) {
	row, err := s.qr.GetTUIProviderSwitch(ctx, gen.GetTUIProviderSwitchParams{SessionID: string(sessionID), RequestID: requestID})
	if err == sql.ErrNoRows {
		return domain.TUIProviderSwitch{}, false, nil
	}
	if err != nil {
		return domain.TUIProviderSwitch{}, false, err
	}
	var record domain.TUIProviderSwitch
	if err := json.Unmarshal([]byte(row), &record); err != nil {
		return record, false, err
	}
	return record, true, nil
}

func (s *Store) ListActiveTUIProviderSwitches(ctx context.Context) ([]domain.TUIProviderSwitch, error) {
	rows, err := s.qr.ActiveTUIProviderSwitches(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]domain.TUIProviderSwitch, 0, len(rows))
	for _, row := range rows {
		var record domain.TUIProviderSwitch
		if err := json.Unmarshal([]byte(row), &record); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, nil
}

func (s *Store) HasActiveTUIProviderSwitch(ctx context.Context, sessionID domain.SessionID) (bool, error) {
	return s.qr.HasActiveTUIProviderSwitch(ctx, string(sessionID))
}

func (s *Store) MarkTUIProviderSwitchOwner(ctx context.Context, id domain.SessionID, sourceHarness domain.AgentHarness, sourceLaunch, targetHarness, targetLaunch, nativeID, model, effort string, now time.Time) (bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.qw.ChangeTUIProviderOwner(ctx, gen.ChangeTUIProviderOwnerParams{
		ID: id, SourceHarness: sourceHarness, SourceLaunchID: sourceLaunch,
		TargetHarness: domain.AgentHarness(targetHarness), TargetLaunchID: targetLaunch, TargetNativeID: nativeID,
		Model: model, Effort: effort, Now: now,
	})
	return rows > 0, err
}

// SessionLaunchOptions returns the launch choices saved for a terminal session.
func (s *Store) SessionLaunchOptions(ctx context.Context, id domain.SessionID) (domain.AgentLaunchOptions, error) {
	raw, err := s.qr.GetSessionLaunchOptions(ctx, id)
	if err != nil || raw == "" {
		return domain.AgentLaunchOptions{}, err
	}
	var options domain.AgentLaunchOptions
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return domain.AgentLaunchOptions{}, fmt.Errorf("decode launch options for %s: %w", id, err)
	}
	return options, nil
}

// SetSessionLaunchOptions saves the launch choices for a terminal session.
func (s *Store) SetSessionLaunchOptions(ctx context.Context, id domain.SessionID, options domain.AgentLaunchOptions) error {
	raw := ""
	if options != (domain.AgentLaunchOptions{}) {
		encoded, err := json.Marshal(options)
		if err != nil {
			return err
		}
		raw = string(encoded)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.qw.SetSessionLaunchOptions(ctx, gen.SetSessionLaunchOptionsParams{LaunchOptions: raw, ID: id})
}
