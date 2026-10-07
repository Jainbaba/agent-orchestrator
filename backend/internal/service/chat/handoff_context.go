package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

const (
	handoffTruncationMarker = "\n…[truncated]…\n"
	handoffSummaryEvent     = "compaction"
)

// HandoffExchange is one complete user/final-assistant pair copied into a
// target provider's bootstrap. Tool activities and streamed deltas deliberately
// do not become transcript turns.
type HandoffExchange struct {
	User      string
	Assistant string
}

// ProviderHandoffContext is the deterministic handoff material prepared before
// the source controller is stopped. Bootstrap is the only field sent to a
// target provider; the structured fields make the result inspectable and easy
// to test without parsing the rendered prompt.
type ProviderHandoffContext struct {
	SourceHarness     domain.AgentHarness
	SessionID         domain.SessionID
	ConversationID    string
	ProjectID         domain.ProjectID
	WorkspacePath     string
	Branch            string
	SourceTitle       string
	CompactionSummary string
	Exchanges         []HandoffExchange
	Bootstrap         string
	Truncated         bool
}

// BuildProviderHandoffContext reads only the supplied durable AO snapshot. It
// never asks a provider to summarize, and it does not create or mutate storage.
// Callers should build this before source-controller shutdown.
func BuildProviderHandoffContext(
	record domain.SessionRecord,
	rows ConversationRows,
) ProviderHandoffContext {
	summary := latestCompactionSummary(rows.Activities)
	exchanges := latestCompleteExchanges(rows.Turns, rows.Messages)
	if len(exchanges) > domain.ProviderSwitchRecentExchangeLimit {
		exchanges = exchanges[len(exchanges)-domain.ProviderSwitchRecentExchangeLimit:]
	}

	context := ProviderHandoffContext{
		SourceHarness:     record.Harness,
		SessionID:         record.ID,
		ConversationID:    rows.Conversation.ID,
		ProjectID:         record.ProjectID,
		WorkspacePath:     record.Metadata.WorkspacePath,
		Branch:            record.Metadata.Branch,
		SourceTitle:       rows.Conversation.ProviderTitle,
		CompactionSummary: summary,
		Exchanges:         exchanges,
	}
	context.Bootstrap, context.Truncated = renderHandoffBootstrap(context)
	return context
}

type handoffTurnPair struct {
	user         domain.ConversationMessage
	assistant    domain.ConversationMessage
	hasUser      bool
	hasAssistant bool
}

func latestCompleteExchanges(
	turns []domain.ConversationTurn,
	messages []domain.ConversationMessage,
) []HandoffExchange {
	eligible := make(map[string]bool, len(turns))
	for _, turn := range turns {
		if turn.State == domain.TurnStateCompleted && turn.RolledBackAt == nil {
			eligible[turn.ID] = true
		}
	}

	pairs := make(map[string]*handoffTurnPair)
	for _, message := range messages {
		if message.TurnID == "" || !eligible[message.TurnID] || message.Streaming {
			continue
		}
		pair := pairs[message.TurnID]
		if pair == nil {
			pair = &handoffTurnPair{}
			pairs[message.TurnID] = pair
		}
		switch message.Role {
		case domain.MessageRoleUser:
			// Only human-authored prompts form the portable transcript. Automation,
			// daemon coordination, and provider observations may share a turn-shaped
			// row, but replaying them as user intent would corrupt the target session.
			if message.Origin != domain.MessageOriginHuman {
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

	complete := make([]handoffTurnPair, 0, len(pairs))
	for _, pair := range pairs {
		if pair.hasUser && pair.hasAssistant &&
			strings.TrimSpace(pair.user.Text) != "" {
			complete = append(complete, *pair)
		}
	}
	sort.SliceStable(complete, func(i, j int) bool {
		return complete[i].user.Sequence < complete[j].user.Sequence
	})

	out := make([]HandoffExchange, 0, len(complete))
	for _, pair := range complete {
		out = append(out, HandoffExchange{
			User:      pair.user.Text,
			Assistant: pair.assistant.Text,
		})
	}
	return out
}

func latestCompactionSummary(activities []domain.ConversationActivity) string {
	var latest domain.ConversationActivity
	found := false
	for _, activity := range activities {
		if activity.Kind != domain.ActivityKindSystem ||
			activity.Status != domain.ActivityStatusCompleted {
			continue
		}
		var detail struct {
			Event string
		}
		if len(activity.Detail) == 0 || json.Unmarshal(activity.Detail, &detail) != nil ||
			detail.Event != handoffSummaryEvent {
			continue
		}
		if !found || activity.Sequence >= latest.Sequence {
			latest = activity
			found = true
		}
	}
	if found {
		return strings.TrimSpace(latest.Summary)
	}
	return ""
}

func renderHandoffBootstrap(context ProviderHandoffContext) (string, bool) {
	// Drop the oldest exchange first. This keeps the newest complete exchange
	// intact whenever the cap permits it.
	exchanges := append([]HandoffExchange(nil), context.Exchanges...)
	summary := context.CompactionSummary
	truncated := false
	for len(exchanges) > 0 {
		candidate := renderHandoff(context, summary, exchanges)
		if charCount(candidate) <= domain.ProviderSwitchMaxBootstrapChars {
			return candidate, truncated
		}
		if len(exchanges) == 1 {
			break
		}
		exchanges = exchanges[1:]
		truncated = true
	}

	// If one exchange plus framing is too large, preserve its newest content and
	// spend the remaining budget on the compaction summary first.
	exchanges = keepNewestExchange(exchanges)
	base := renderHandoff(context, "", exchanges)
	remaining := domain.ProviderSwitchMaxBootstrapChars - charCount(base)
	if remaining >= 0 {
		summary = trimToChars(summary, remaining)
		candidate := renderHandoff(context, summary, exchanges)
		if charCount(candidate) <= domain.ProviderSwitchMaxBootstrapChars {
			return candidate, truncated || summary != context.CompactionSummary
		}
	}

	// A pathological single message can exceed the entire cap. Trim assistant
	// prose before user prose, keeping both role labels and an explicit marker.
	if len(exchanges) == 1 {
		exchange := exchanges[0]
		exchange.Assistant = trimToChars(exchange.Assistant, domain.ProviderSwitchMaxBootstrapChars/2)
		exchange.User = trimToChars(exchange.User, domain.ProviderSwitchMaxBootstrapChars/2)
		exchanges[0] = exchange
	}
	candidate := renderHandoff(context, trimToChars(summary, domain.ProviderSwitchMaxBootstrapChars/4), exchanges)
	if charCount(candidate) > domain.ProviderSwitchMaxBootstrapChars {
		candidate = trimToChars(candidate, domain.ProviderSwitchMaxBootstrapChars)
	}
	return candidate, true
}

func keepNewestExchange(exchanges []HandoffExchange) []HandoffExchange {
	if len(exchanges) == 0 {
		return nil
	}
	return exchanges[len(exchanges)-1:]
}

func renderHandoff(
	context ProviderHandoffContext,
	summary string,
	exchanges []HandoffExchange,
) string {
	var b strings.Builder
	b.WriteString("AO provider handoff context\n\n")
	writeHandoffField(&b, "Source provider", string(context.SourceHarness))
	writeHandoffField(&b, "AO session", string(context.SessionID))
	writeHandoffField(&b, "AO conversation", context.ConversationID)
	writeHandoffField(&b, "Project", string(context.ProjectID))
	writeHandoffField(&b, "Workspace", context.WorkspacePath)
	writeHandoffField(&b, "Branch", context.Branch)
	writeHandoffField(&b, "Source title", context.SourceTitle)

	b.WriteString("\nLatest compaction summary:\n")
	if strings.TrimSpace(summary) == "" {
		b.WriteString("(none recorded)\n")
	} else {
		b.WriteString(summary)
		b.WriteByte('\n')
	}

	b.WriteString("\nRecent complete exchanges:\n")
	if len(exchanges) == 0 {
		b.WriteString("(none recorded)\n")
	}
	for index, exchange := range exchanges {
		fmt.Fprintf(&b, "\nExchange %d — user:\n%s\n", index+1, exchange.User)
		b.WriteString("Assistant final:\n")
		b.WriteString(exchange.Assistant)
		b.WriteByte('\n')
	}
	return b.String()
}

func writeHandoffField(b *strings.Builder, label, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	fmt.Fprintf(b, "%s: %s\n", label, value)
}

func charCount(value string) int {
	return utf8.RuneCountInString(value)
}

func trimToChars(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= utf8.RuneCountInString(handoffTruncationMarker) {
		return string(runes[:limit])
	}
	remaining := limit - utf8.RuneCountInString(handoffTruncationMarker)
	head := remaining / 2
	tail := remaining - head
	return string(runes[:head]) + handoffTruncationMarker + string(runes[len(runes)-tail:])
}
