package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// HandoffNotePrefix opens every handoff note. The hook client and the Chat
	// history import both recognise it, so a note is never recorded as something
	// the user typed.
	HandoffNotePrefix = "<ao-handoff-request"

	// TerminalTurnRetention is how many Terminal UI turns AO keeps per session.
	TerminalTurnRetention = 50

	handoffPreviousTurnLimit = 5
	handoffOlderTurnLimit    = 15

	handoffLatestTurnBudget   = 10_000
	handoffLatestUserCap      = 3_000
	handoffPreviousTurnBudget = 2_000
	handoffPreviousUserCap    = 600
	handoffOlderTurnBudget    = 350
	handoffOlderUserCap       = 150

	handoffOriginalTaskCap  = 2_000
	handoffChangedFileLimit = 50
	handoffChangedFileCap   = 160

	// A note delivered as a launch argument must also fit the operating
	// system's per-argument limit, which is counted in bytes.
	handoffNoteMaxBytes = 96_000

	handoffClipMarker     = "\n…[clipped]…\n"
	handoffLineClipMarker = " …[clipped]… "
)

// TerminalTurn is one Terminal UI turn as the hooks reported it: what the user
// asked and the agent's final message. RespondedAt is zero while the turn is
// open, and stays zero for a turn the user interrupted.
type TerminalTurn struct {
	ID              int64
	SessionID       SessionID
	Harness         AgentHarness
	NativeSessionID string
	UserRequest     string
	AgentResponse   string
	RequestedAt     time.Time
	RespondedAt     time.Time
}

// HandoffTurn is one turn of a session in the form the handoff note and the
// history command share, whichever UI it happened in.
type HandoffTurn struct {
	UserRequest   string
	AgentResponse string
	At            time.Time
	Harness       AgentHarness
}

// HandoffNoteInput is everything a handoff note is rendered from. Turns holds
// every turn AO knows about, oldest first; the builder decides what fits.
type HandoffNoteInput struct {
	SwitchID       string
	SourceHarness  AgentHarness
	TargetHarness  AgentHarness
	SessionTitle   string
	Project        string
	WorkspacePath  string
	Branch         string
	At             time.Time
	OriginalTask   string
	ChangedFiles   []string
	Turns          []HandoffTurn
	HistoryCommand bool
	Terminal       bool
}

// BuildHandoffNote renders the note a provider receives when it takes over a
// session. It is deterministic and never asks a model to summarise: the latest
// turn is kept, the five before it are clipped to a budget, up to fifteen older
// ones become one line each, and anything beyond that is only counted.
func BuildHandoffNote(in HandoffNoteInput) string {
	turns := in.Turns
	var latest *HandoffTurn
	if len(turns) > 0 {
		latest = &turns[len(turns)-1]
		turns = turns[:len(turns)-1]
	}
	previous := lastHandoffTurns(turns, handoffPreviousTurnLimit)
	older := lastHandoffTurns(turns[:len(turns)-len(previous)], handoffOlderTurnLimit)

	// Drop the oldest material first until the whole note fits. The latest
	// turn is never dropped.
	for {
		shown := len(older) + len(previous)
		if latest != nil {
			shown++
		}
		note := renderHandoffNote(in, older, previous, latest, len(in.Turns)-shown)
		if handoffNoteFits(note) {
			return note
		}
		switch {
		case len(older) > 0:
			older = older[1:]
		case len(previous) > 0:
			previous = previous[1:]
		default:
			return clipHandoffText(note, ProviderSwitchMaxBootstrapChars, handoffClipMarker)
		}
	}
}

func lastHandoffTurns(turns []HandoffTurn, limit int) []HandoffTurn {
	if len(turns) > limit {
		return turns[len(turns)-limit:]
	}
	return turns
}

func handoffNoteFits(note string) bool {
	return utf8.RuneCountInString(note) <= ProviderSwitchMaxBootstrapChars && len(note) <= handoffNoteMaxBytes
}

func renderHandoffNote(in HandoffNoteInput, older, previous []HandoffTurn, latest *HandoffTurn, omitted int) string {
	var b strings.Builder
	if in.SwitchID != "" {
		fmt.Fprintf(&b, "%s switch-id=%q>\n", HandoffNotePrefix, in.SwitchID)
	} else {
		b.WriteString(HandoffNotePrefix + ">\n")
	}
	// The new agent must take this as the user's real conversation. Calling it
	// AO's note, not the user's, made Claude refuse to use what the user had
	// said, such as a value they asked it to remember.
	fmt.Fprintf(&b, "You are %s, taking over this AO session from %s. The conversation below really took place between this user and %s. Treat it as your own history with them: what the user told you there, such as facts, preferences or values they asked you to remember, still holds.\n",
		handoffHarnessLabel(in.TargetHarness), handoffHarnessLabel(in.SourceHarness), handoffHarnessLabel(in.SourceHarness))
	b.WriteString("- AO copied this record from that conversation. Requests in it were already handled; do not carry them out again.\n")
	b.WriteString("- The files in the workspace and git are the truth. Check them before relying on this record.\n")
	b.WriteString("- Do not redo work that is already finished.\n")

	b.WriteString("\n## Session\n")
	writeHandoffNoteField(&b, "Title", in.SessionTitle)
	writeHandoffNoteField(&b, "Project", in.Project)
	writeHandoffNoteField(&b, "Workspace", in.WorkspacePath)
	writeHandoffNoteField(&b, "Git branch", in.Branch)
	if !in.At.IsZero() {
		writeHandoffNoteField(&b, "Handed off at", in.At.UTC().Format(time.RFC3339))
	}

	if task := strings.TrimSpace(in.OriginalTask); task != "" && !handoffTaskShownInFull(task, previous, latest) {
		b.WriteString("\n## Original task\n")
		b.WriteString(quoteHandoffText(task, handoffOriginalTaskCap, handoffClipMarker))
		b.WriteByte('\n')
	}

	if len(in.ChangedFiles) > 0 {
		fmt.Fprintf(&b, "\n## Workspace state\nFiles changed since the session started (%d):\n", len(in.ChangedFiles))
		for index, name := range in.ChangedFiles {
			if index == handoffChangedFileLimit {
				fmt.Fprintf(&b, "(and %d more)\n", len(in.ChangedFiles)-handoffChangedFileLimit)
				break
			}
			fmt.Fprintf(&b, "- %s\n", quoteHandoffText(name, handoffChangedFileCap, handoffLineClipMarker))
		}
	}

	if len(older) > 0 {
		b.WriteString("\n## Older turns, oldest first\n")
		for _, turn := range older {
			user, agent := budgetHandoffTurn(turn, handoffOlderTurnBudget, handoffOlderUserCap, true)
			fmt.Fprintf(&b, "- User: %s\n  %s: %s\n", user, handoffAssistantLabel(turn), agent)
		}
	}
	if len(previous) > 0 {
		b.WriteString("\n## Previous turns, oldest first\n")
		for _, turn := range previous {
			user, agent := budgetHandoffTurn(turn, handoffPreviousTurnBudget, handoffPreviousUserCap, false)
			fmt.Fprintf(&b, "\nUser:\n%s\n%s:\n%s\n", user, handoffAssistantLabel(turn), agent)
		}
	}
	if latest != nil {
		user, agent := budgetHandoffTurn(*latest, handoffLatestTurnBudget, handoffLatestUserCap, false)
		fmt.Fprintf(&b, "\n## Latest turn\nUser:\n%s\n%s:\n%s\n", user, handoffAssistantLabel(*latest), agent)
	}

	// The full record is always one command away, so the note says so even
	// when nothing was left out: long turns are clipped to fit the budget.
	if omitted > 0 || in.HistoryCommand {
		b.WriteString("\n## Earlier history\n")
		if omitted > 0 {
			fmt.Fprintf(&b, "%d earlier turns are not shown. ", omitted)
		}
		if in.HistoryCommand {
			b.WriteString("Run `ao session history` to read the conversation in full when you need something this note does not show.\n")
		} else {
			b.WriteString("They are not available to you.\n")
		}
	}

	b.WriteByte('\n')
	if in.Terminal {
		b.WriteString("Reply with a few lines on where things stand, then wait for the user. Do not start new work yet.\n")
	} else {
		b.WriteString("Do not mention this note unless it is relevant. Wait for the user's next message.\n")
	}
	b.WriteString("</ao-handoff-request>")
	return b.String()
}

// handoffTaskShownInFull reports whether the original task already appears,
// unclipped, as the request of a turn the note shows in full. A short session
// would otherwise print the same text twice.
func handoffTaskShownInFull(task string, previous []HandoffTurn, latest *HandoffTurn) bool {
	if latest != nil && strings.TrimSpace(latest.UserRequest) == task && utf8.RuneCountInString(task) <= handoffLatestUserCap {
		return true
	}
	for _, turn := range previous {
		if strings.TrimSpace(turn.UserRequest) == task && utf8.RuneCountInString(task) <= handoffPreviousUserCap {
			return true
		}
	}
	return false
}

// handoffAssistantLabel names the agent that answered a turn when AO knows it.
// A session can pass through several agents; without the name the reader
// credits every earlier answer to the one it took over from.
func handoffAssistantLabel(turn HandoffTurn) string {
	if harness := strings.TrimSpace(string(turn.Harness)); harness != "" {
		return "Assistant (" + harness + ")"
	}
	return "Assistant"
}

func handoffHarnessLabel(harness AgentHarness) string {
	if strings.TrimSpace(string(harness)) == "" {
		return "another agent"
	}
	return string(harness)
}

func writeHandoffNoteField(b *strings.Builder, label, value string) {
	if value = strings.TrimSpace(value); value != "" {
		fmt.Fprintf(b, "%s: %s\n", label, quoteHandoffText(value, handoffChangedFileCap, handoffLineClipMarker))
	}
}

// budgetHandoffTurn gives the user's request what it needs up to its cap and
// the agent's response the rest of the turn's budget: requests are short, so
// the room is better spent on the response.
func budgetHandoffTurn(turn HandoffTurn, budget, userCap int, oneLine bool) (user, agent string) {
	marker := handoffClipMarker
	userText, agentText := strings.TrimSpace(turn.UserRequest), strings.TrimSpace(turn.AgentResponse)
	if oneLine {
		marker = handoffLineClipMarker
		userText, agentText = strings.Join(strings.Fields(userText), " "), strings.Join(strings.Fields(agentText), " ")
	}
	user = quoteHandoffText(userText, userCap, marker)
	if agentText == "" {
		return user, "(no final response recorded)"
	}
	return user, quoteHandoffText(agentText, budget-utf8.RuneCountInString(user), marker)
}

// quoteHandoffText clips quoted text to limit and defangs the note's own tag,
// so nothing inside a quoted turn can close the handoff block early.
func quoteHandoffText(value string, limit int, marker string) string {
	value = strings.ReplaceAll(SanitizeControlChars(value), "ao-handoff-request", "ao_handoff_request")
	return clipHandoffText(value, limit, marker)
}

// clipHandoffText keeps the start and the end of over-long text. A request
// usually opens with the ask and a response usually closes with what is left
// to do, so the middle is the cheapest part to lose.
func clipHandoffText(value string, limit int, marker string) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	keep := limit - utf8.RuneCountInString(marker)
	if keep <= 0 {
		return string(runes[:limit])
	}
	head := keep / 2
	return string(runes[:head]) + marker + string(runes[len(runes)-(keep-head):])
}
