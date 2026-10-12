package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	aoprocess "github.com/aoagents/agent-orchestrator/backend/internal/process"
)

const handoffChangedFilesTimeout = 5 * time.Second

type terminalTurnReader interface {
	ListTerminalTurns(ctx context.Context, id domain.SessionID, limit int) ([]domain.TerminalTurn, error)
}

// SessionTurns returns every turn AO holds for a session, oldest first: the
// completed Chat exchanges merged by time with the stored Terminal UI turns.
// A turn is written to one of the two stores only, so merging never repeats it.
// A session that has never been in Chat simply has no conversation.
func (s *Service) SessionTurns(ctx context.Context, id domain.SessionID) ([]domain.HandoffTurn, error) {
	var turns []domain.HandoffTurn
	conversation, err := s.store.ConversationForSession(ctx, id)
	switch {
	case errors.Is(err, domain.ErrNoConversation):
	case err != nil:
		return nil, err
	case s.reader != nil:
		rows, readErr := s.reader.LoadConversationSnapshot(ctx, conversation.ID)
		if readErr != nil {
			return nil, fmt.Errorf("load conversation %s turns: %w", conversation.ID, readErr)
		}
		turns = chatHandoffTurns(rows.Turns, rows.Messages, rows.Activities)
	}
	if reader, ok := s.store.(terminalTurnReader); ok {
		stored, listErr := reader.ListTerminalTurns(ctx, id, domain.TerminalTurnRetention)
		if listErr != nil {
			return nil, listErr
		}
		for _, turn := range stored {
			turns = append(turns, domain.HandoffTurn{
				UserRequest: turn.UserRequest, AgentResponse: turn.AgentResponse,
				At: turn.RequestedAt, Harness: turn.Harness,
			})
		}
	}
	sort.SliceStable(turns, func(i, j int) bool { return turns[i].At.Before(turns[j].At) })
	return turns, nil
}

// chatHandoffTurns reduces a conversation to one request and one final response
// per completed turn. Tool activity and streamed deltas are not turns. A request
// is what a person typed or what was sent to the session through AO, which is
// how an orchestrator and `ao send` talk to it and what the Terminal UI history
// records too. The daemon's own messages are not requests.
//
// Each turn is credited to the provider that answered it, read from the markers
// earlier provider switches left: a marker names the provider of the branch it
// opened, and the first marker's source answered everything before it. A
// conversation that never switched has no marker, and its turns stay uncredited.
func chatHandoffTurns(turns []domain.ConversationTurn, messages []domain.ConversationMessage, activities []domain.ConversationActivity) []domain.HandoffTurn {
	branchHarness := make(map[string]domain.AgentHarness)
	var firstSource domain.AgentHarness
	for _, activity := range activities {
		var marker providerSwitchActivityDetail
		if activity.Kind != domain.ActivityKindSystem || json.Unmarshal(activity.Detail, &marker) != nil || marker.Event != "provider_switch" {
			continue
		}
		if len(branchHarness) == 0 {
			firstSource = marker.SourceHarness
		}
		branchHarness[marker.BranchID] = marker.TargetHarness
	}
	eligible := make(map[string]bool, len(turns))
	turnHarness := make(map[string]domain.AgentHarness, len(turns))
	for _, turn := range turns {
		if turn.State == domain.TurnStateCompleted && turn.RolledBackAt == nil {
			eligible[turn.ID] = true
			harness, switched := branchHarness[turn.BranchID]
			if !switched {
				harness = firstSource
			}
			turnHarness[turn.ID] = harness
		}
	}
	type exchange struct {
		user, assistant       domain.ConversationMessage
		hasUser, hasAssistant bool
	}
	exchanges := make(map[string]*exchange)
	for _, message := range messages {
		if message.TurnID == "" || !eligible[message.TurnID] || message.Streaming {
			continue
		}
		pair := exchanges[message.TurnID]
		if pair == nil {
			pair = &exchange{}
			exchanges[message.TurnID] = pair
		}
		switch message.Role {
		case domain.MessageRoleUser:
			if message.Origin != domain.MessageOriginHuman && message.Origin != domain.MessageOriginAutomation {
				continue
			}
			if !pair.hasUser || message.Sequence < pair.user.Sequence {
				pair.user = message
			}
			pair.hasUser = true
		case domain.MessageRoleAssistant:
			if !pair.hasAssistant || message.Sequence > pair.assistant.Sequence {
				pair.assistant = message
			}
			pair.hasAssistant = true
		}
	}
	complete := make([]*exchange, 0, len(exchanges))
	for _, pair := range exchanges {
		if pair.hasUser && pair.hasAssistant && strings.TrimSpace(pair.user.Text) != "" {
			complete = append(complete, pair)
		}
	}
	sort.Slice(complete, func(i, j int) bool { return complete[i].user.Sequence < complete[j].user.Sequence })
	out := make([]domain.HandoffTurn, 0, len(complete))
	for _, pair := range complete {
		out = append(out, domain.HandoffTurn{
			UserRequest: pair.user.Text, AgentResponse: pair.assistant.Text, At: pair.user.CreatedAt,
			Harness: turnHarness[pair.user.TurnID],
		})
	}
	return out
}

// HandoffNote builds the note a provider receives when it takes over record's
// session. It reads AO's own stores and the workspace only: no provider is
// asked to summarise, and nothing is written.
func (s *Service) HandoffNote(
	ctx context.Context,
	record domain.SessionRecord,
	target domain.AgentHarness,
	switchID string,
) (string, error) {
	turns, err := s.SessionTurns(ctx, record.ID)
	if err != nil {
		return "", err
	}
	return domain.BuildHandoffNote(domain.HandoffNoteInput{
		SwitchID:       switchID,
		SourceHarness:  record.Harness,
		TargetHarness:  target,
		SessionTitle:   record.DisplayName,
		Project:        string(record.ProjectID),
		WorkspacePath:  record.Metadata.WorkspacePath,
		Branch:         record.Metadata.Branch,
		At:             s.now(),
		OriginalTask:   record.Metadata.Prompt,
		ChangedFiles:   handoffChangedFiles(ctx, record.Metadata.WorkspacePath, record.Metadata.DiffBaseSHA),
		Turns:          turns,
		HistoryCommand: true,
		Terminal:       domain.NormalizeSessionMode(record.Mode) == domain.SessionModeTUI,
	}), nil
}

// SessionHistory returns one page of a session's turns, newest first, with the
// total number AO holds. It backs `ao session history`, which lets the agent
// that took over read what the handoff note left out.
func (s *Service) SessionHistory(ctx context.Context, id domain.SessionID, offset, limit int) ([]domain.HandoffTurn, int, error) {
	if _, found, err := s.sessions.GetSession(ctx, id); err != nil {
		return nil, 0, err
	} else if !found {
		return nil, 0, ports.ErrSessionNotFound
	}
	turns, err := s.SessionTurns(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	total := len(turns)
	page := make([]domain.HandoffTurn, 0, limit)
	for index := total - 1 - offset; index >= 0 && len(page) < limit; index-- {
		page = append(page, turns[index])
	}
	return page, total, nil
}

// handoffChangedFiles lists what the session changed in its workspace: tracked
// files that differ from the diff base, then untracked ones. It is the note's
// view of what is on disk, since tool calls are not carried. A workspace git
// cannot read yields no list; that must never block a switch.
func handoffChangedFiles(ctx context.Context, workspace, base string) []string {
	if strings.TrimSpace(workspace) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, handoffChangedFilesTimeout)
	defer cancel()
	if strings.TrimSpace(base) == "" {
		base = "HEAD"
	}
	seen := make(map[string]bool)
	var files []string
	for _, args := range [][]string{
		{"diff", "--name-only", "-z", base},
		{"ls-files", "-z", "-o", "--exclude-standard"},
	} {
		globalArgs := []string{"--no-pager", "--no-optional-locks", "-c", "core.hooksPath=" + os.DevNull, "-c", "core.fsmonitor=false", "-C", workspace}
		cmd := aoprocess.CommandContext(ctx, "git", append(globalArgs, args...)...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		for _, name := range bytes.Split(out, []byte{0}) {
			if file := string(name); file != "" && !seen[file] {
				seen[file] = true
				files = append(files, file)
			}
		}
	}
	return files
}
