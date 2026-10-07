package chat

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func TestBuildProviderHandoffContextUsesLatestCompactionAndFiveCompleteExchanges(t *testing.T) {
	now := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	record := domain.SessionRecord{
		ID:        "session-1",
		ProjectID: "project-1",
		Harness:   domain.HarnessClaudeCode,
		Metadata: domain.SessionMetadata{
			WorkspacePath: "/tmp/worktree",
			Branch:        "feature/demo",
		},
	}
	rows := ConversationRows{
		Conversation: domain.ConversationRecord{
			ID:            "conversation-1",
			ProviderTitle: "Source thread",
		},
	}
	for index := 1; index <= 7; index++ {
		turnID := "turn-" + string(rune('0'+index))
		rows.Turns = append(rows.Turns, domain.ConversationTurn{
			ID: turnID, State: domain.TurnStateCompleted, RequestedAt: now.Add(time.Duration(index) * time.Minute),
		})
		rows.Messages = append(rows.Messages,
			domain.ConversationMessage{
				ID: "user-" + turnID, TurnID: turnID, Sequence: int64(index * 2),
				Role: domain.MessageRoleUser, Origin: domain.MessageOriginHuman, Text: "user " + turnID,
			},
			domain.ConversationMessage{
				ID: "assistant-" + turnID, TurnID: turnID, Sequence: int64(index*2 + 1),
				Role: domain.MessageRoleAssistant, Origin: domain.MessageOriginProvider, Text: "assistant " + turnID,
			},
		)
	}
	rows.Activities = []domain.ConversationActivity{
		{Kind: domain.ActivityKindSystem, Status: domain.ActivityStatusCompleted,
			Summary: "old summary", Detail: []byte("{\"event\":\"compaction\"}"), Sequence: 30},
		{Kind: domain.ActivityKindSystem, Status: domain.ActivityStatusCompleted,
			Summary: "latest summary", Detail: []byte("{\"event\":\"compaction\"}"), Sequence: 31},
	}

	got := BuildProviderHandoffContext(record, rows)
	if got.CompactionSummary != "latest summary" {
		t.Fatalf("summary = %q, want latest summary", got.CompactionSummary)
	}
	if len(got.Exchanges) != domain.ProviderSwitchRecentExchangeLimit {
		t.Fatalf("exchanges = %d, want %d", len(got.Exchanges), domain.ProviderSwitchRecentExchangeLimit)
	}
	if got.Exchanges[0].User != "user turn-3" || got.Exchanges[4].Assistant != "assistant turn-7" {
		t.Fatalf("exchanges = %+v, want turns 3 through 7", got.Exchanges)
	}
	for _, want := range []string{"Source provider: claude-code", "AO session: session-1",
		"Workspace: /tmp/worktree", "Latest compaction summary:\nlatest summary",
		"assistant turn-7"} {
		if !strings.Contains(got.Bootstrap, want) {
			t.Errorf("bootstrap missing %q:\n%s", want, got.Bootstrap)
		}
	}
}

func TestBuildProviderHandoffContextExcludesIncompleteAndNonCompactionActivities(t *testing.T) {
	record := domain.SessionRecord{ID: "session-1", Harness: domain.HarnessCodex}
	rolledBackAt := time.Now()
	rows := ConversationRows{
		Conversation: domain.ConversationRecord{ID: "conversation-1"},
		Turns: []domain.ConversationTurn{
			{ID: "complete", State: domain.TurnStateCompleted},
			{ID: "running", State: domain.TurnStateRunning},
			{ID: "rolled-back", State: domain.TurnStateCompleted, RolledBackAt: &rolledBackAt},
		},
		Messages: []domain.ConversationMessage{
			{TurnID: "complete", Sequence: 1, Role: domain.MessageRoleUser, Origin: domain.MessageOriginHuman, Text: "keep"},
			{TurnID: "complete", Sequence: 2, Role: domain.MessageRoleAssistant, Origin: domain.MessageOriginProvider, Text: "final"},
			{TurnID: "complete", Sequence: 3, Role: domain.MessageRoleUser, Origin: domain.MessageOriginDaemon, Text: "internal bootstrap"},
			{TurnID: "running", Sequence: 3, Role: domain.MessageRoleUser, Origin: domain.MessageOriginHuman, Text: "drop"},
			{TurnID: "running", Sequence: 4, Role: domain.MessageRoleAssistant, Origin: domain.MessageOriginProvider, Text: "drop"},
			{TurnID: "rolled-back", Sequence: 5, Role: domain.MessageRoleUser, Origin: domain.MessageOriginHuman, Text: "drop"},
			{TurnID: "rolled-back", Sequence: 6, Role: domain.MessageRoleAssistant, Origin: domain.MessageOriginProvider, Text: "drop"},
		},
		Activities: []domain.ConversationActivity{
			{Kind: domain.ActivityKindSystem, Status: domain.ActivityStatusCompleted,
				Summary: "not compaction", Detail: []byte("{\"event\":\"steer\"}")},
			{Kind: domain.ActivityKindSystem, Status: domain.ActivityStatusRunning,
				Summary: "running summary", Detail: []byte("{\"event\":\"compaction\"}")},
		},
	}

	got := BuildProviderHandoffContext(record, rows)
	if got.CompactionSummary != "" {
		t.Fatalf("summary = %q, want empty", got.CompactionSummary)
	}
	if len(got.Exchanges) != 1 || got.Exchanges[0].User != "keep" || got.Exchanges[0].Assistant != "final" {
		t.Fatalf("exchanges = %+v, want only the completed pair", got.Exchanges)
	}
}

func TestBuildProviderHandoffContextDropsOldestBeforeTrimmingNewest(t *testing.T) {
	record := domain.SessionRecord{ID: "session-1", Harness: domain.HarnessCodex}
	rows := ConversationRows{Conversation: domain.ConversationRecord{ID: "conversation-1"}}
	for index := 0; index < 5; index++ {
		turnID := "turn-" + string(rune('0'+index))
		rows.Turns = append(rows.Turns, domain.ConversationTurn{ID: turnID, State: domain.TurnStateCompleted})
		rows.Messages = append(rows.Messages,
			domain.ConversationMessage{TurnID: turnID, Sequence: int64(index*2 + 1),
				Role: domain.MessageRoleUser, Origin: domain.MessageOriginHuman, Text: strings.Repeat(string(rune('a'+index)), 6000)},
			domain.ConversationMessage{TurnID: turnID, Sequence: int64(index*2 + 2),
				Role: domain.MessageRoleAssistant, Origin: domain.MessageOriginProvider, Text: strings.Repeat(string(rune('A'+index)), 6000)},
		)
	}

	got := BuildProviderHandoffContext(record, rows)
	if charCount(got.Bootstrap) > domain.ProviderSwitchMaxBootstrapChars {
		t.Fatalf("bootstrap chars = %d, want <= %d", charCount(got.Bootstrap), domain.ProviderSwitchMaxBootstrapChars)
	}
	if !got.Truncated {
		t.Fatal("large bootstrap was not marked truncated")
	}
	if !strings.Contains(got.Bootstrap, "Exchange 1") {
		t.Fatal("newest exchange was not retained")
	}
	if strings.Contains(got.Bootstrap, strings.Repeat("a", 100)) {
		t.Fatal("oldest exchange survived when the cap required dropping it")
	}
}

func TestTrimToCharsHandlesUnicodeAndSmallLimits(t *testing.T) {
	value := "पहले🙂और बाद में"
	for limit := 1; limit <= charCount(value); limit++ {
		got := trimToChars(value, limit)
		if charCount(got) > limit {
			t.Fatalf("limit %d produced %d characters", limit, charCount(got))
		}
		if !utf8.ValidString(got) {
			t.Fatalf("limit %d produced invalid UTF-8", limit)
		}
	}
}
