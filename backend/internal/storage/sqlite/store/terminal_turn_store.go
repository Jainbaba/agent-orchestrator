package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/storage/sqlite/gen"
)

// RecordTerminalTurnRequest stores what the user asked a Terminal UI agent. A
// request that arrives while the session's newest turn is still open on the
// same thread was queued behind it, so it joins that turn instead of starting
// another.
func (s *Store) RecordTerminalTurnRequest(ctx context.Context, turn domain.TerminalTurn) error {
	text := strings.TrimSpace(turn.UserRequest)
	if text == "" {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.inTx(ctx, "record terminal turn request", func(q *gen.Queries) error {
		latest, err := q.LatestTerminalTurn(ctx, string(turn.SessionID))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("read latest terminal turn: %w", err)
		}
		sameThread := latest.Harness == string(turn.Harness) &&
			(latest.NativeSessionID == turn.NativeSessionID || latest.NativeSessionID == "" || turn.NativeSessionID == "")
		if err == nil && !latest.RespondedAt.Valid && sameThread {
			return q.AppendTerminalTurnRequest(ctx, gen.AppendTerminalTurnRequestParams{UserRequest: text, ID: latest.ID})
		}
		if err := q.InsertTerminalTurn(ctx, gen.InsertTerminalTurnParams{
			SessionID: string(turn.SessionID), Harness: string(turn.Harness),
			NativeSessionID: turn.NativeSessionID, UserRequest: text, RequestedAt: turn.RequestedAt,
		}); err != nil {
			return fmt.Errorf("insert terminal turn: %w", err)
		}
		return q.TrimTerminalTurns(ctx, gen.TrimTerminalTurnsParams{
			SessionID: string(turn.SessionID), Keep: domain.TerminalTurnRetention,
		})
	})
}

// RecordTerminalTurnResponse closes the session's newest turn with the agent's
// final message. A stop with no open turn has nothing to answer and is ignored.
func (s *Store) RecordTerminalTurnResponse(ctx context.Context, id domain.SessionID, response string, at time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.inTx(ctx, "record terminal turn response", func(q *gen.Queries) error {
		latest, err := q.LatestTerminalTurn(ctx, string(id))
		if errors.Is(err, sql.ErrNoRows) || (err == nil && latest.RespondedAt.Valid) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read latest terminal turn: %w", err)
		}
		return q.CloseTerminalTurn(ctx, gen.CloseTerminalTurnParams{
			AgentResponse: strings.TrimSpace(response),
			RespondedAt:   sql.NullTime{Time: at, Valid: true},
			ID:            latest.ID,
		})
	})
}

// ListTerminalTurns returns up to limit of a session's stored terminal turns,
// oldest first.
func (s *Store) ListTerminalTurns(ctx context.Context, id domain.SessionID, limit int) ([]domain.TerminalTurn, error) {
	rows, err := s.qr.ListTerminalTurns(ctx, gen.ListTerminalTurnsParams{SessionID: string(id), MaxRows: int64(limit)})
	if err != nil {
		return nil, fmt.Errorf("list terminal turns: %w", err)
	}
	turns := make([]domain.TerminalTurn, len(rows))
	for index, row := range rows {
		turns[len(rows)-1-index] = domain.TerminalTurn{
			ID: row.ID, SessionID: domain.SessionID(row.SessionID), Harness: domain.AgentHarness(row.Harness),
			NativeSessionID: row.NativeSessionID, UserRequest: row.UserRequest, AgentResponse: row.AgentResponse,
			RequestedAt: row.RequestedAt, RespondedAt: row.RespondedAt.Time,
		}
	}
	return turns, nil
}

// DeleteTerminalTurnsForThread removes the turns of one provider thread after
// its history was imported into the conversation tables, so a turn never sits
// in both stores. Rows recorded before the hook reported a thread id match on
// the provider alone. Other providers' turns were not imported and stay.
func (s *Store) DeleteTerminalTurnsForThread(ctx context.Context, id domain.SessionID, harness domain.AgentHarness, nativeSessionID string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.qw.DeleteTerminalTurnsForThread(ctx, gen.DeleteTerminalTurnsForThreadParams{
		SessionID: string(id), Harness: string(harness), NativeSessionID: nativeSessionID,
	}); err != nil {
		return fmt.Errorf("delete imported terminal turns: %w", err)
	}
	return nil
}
