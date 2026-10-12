package lifecycle

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

type terminalTurnRecorder interface {
	RecordTerminalTurnRequest(context.Context, domain.TerminalTurn) error
	RecordTerminalTurnResponse(context.Context, domain.SessionID, string, time.Time) error
}

// recordTerminalTurn keeps AO's own record of a Terminal UI turn: the request
// from the prompt hook and the agent's final message from the stop hook. Chat
// turns live in the conversation tables, so the caller records only in terminal
// mode. A failure here must never reject the activity signal itself.
func (m *Manager) recordTerminalTurn(ctx context.Context, rec domain.SessionRecord, s ports.ActivitySignal, now time.Time) {
	recorder, ok := m.store.(terminalTurnRecorder)
	if !ok {
		return
	}
	at := timeOr(s.Timestamp, now)
	var err error
	switch s.Event {
	case "user-prompt-submit":
		request := strings.TrimSpace(s.TurnUserRequest)
		if request == "" {
			request = s.LatestUserPrompt
		}
		if s.ConversationCheckpointOrigin == domain.ConversationCheckpointOriginCoordination || request == "" {
			// The user did not ask for this turn: AO's own message opened it,
			// or the hook carried no text. Close any turn still open so the
			// coming stop cannot answer an earlier request.
			err = recorder.RecordTerminalTurnResponse(ctx, rec.ID, "", at)
			break
		}
		nativeID := s.AgentSessionID
		if nativeID == "" {
			nativeID = rec.Metadata.AgentSessionID
		}
		err = recorder.RecordTerminalTurnRequest(ctx, domain.TerminalTurn{
			SessionID: rec.ID, Harness: rec.Harness, NativeSessionID: nativeID,
			UserRequest: request, RequestedAt: at,
		})
	case "stop":
		if s.SubagentID != "" {
			break
		}
		response := strings.TrimSpace(s.FinalAssistantMessage)
		if response == "" {
			response = s.LatestAssistantUpdate
		}
		// The turn is over even when the hook carries no text. Closing it
		// keeps the next request from piling onto this one.
		err = recorder.RecordTerminalTurnResponse(ctx, rec.ID, response, at)
	}
	if err != nil {
		slog.Default().Warn("lifecycle: record terminal turn", "session", rec.ID, "event", s.Event, "error", err)
	}
}
