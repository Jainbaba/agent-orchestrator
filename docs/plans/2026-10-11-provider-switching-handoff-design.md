# Provider switching: handoff design

Status: design agreed on 2026-10-11. The v0 backend is implemented on this branch, uncommitted; see section 13.
Branch: `feature/provider-switching`.

This document records what was decided for the handoff: what AO stores, what the handoff note contains, and how a switch works in the Chat UI and the Terminal UI. It replaces the "latest compaction summary plus five exchanges" contents described in [2026-10-07-provider-switching.md](2026-10-07-provider-switching.md). The switch lifecycle in that plan (idle-only, stop the source, start the target, roll back on failure) still stands.

Background research: [competitor hunt](../research/2026-10-10-provider-switching-competitors.md).

---

## 1. What a switch is

The user continues the same AO session on a different coding agent. The old agent is stopped, a new one starts on a fresh thread of its own, and it receives a written handoff note built from AO's own record of the conversation.

- Same AO session, same workspace, same history on screen.
- The new agent always starts a fresh native thread in v0.
- A switch is allowed only when the session is idle: no running turn, no pending question or approval, no unsent draft.

---

## 2. Decisions

| Decision | Detail |
|---|---|
| Messages only | The note is built from user requests and agent responses. No model writes a summary |
| No compaction summary | Dropped completely, not kept as an option. Chat's stored "summary" is a status line, Codex's is encrypted, and Claude's can exceed the whole note |
| One turn = request plus final response | Per turn, AO keeps the user's request and the agent's final message. Tool calls and reasoning are not carried |
| AO's own record is the source | Nothing is parsed from the agents' session files and nothing is read from the terminal screen |
| Two stores, combined when the note is built | Chat keeps its conversation tables. Terminal gets a new turn table fed by hooks |
| One note format for both UIs | Section 5 |
| Earlier history is available on request | Through an `ao` command, not a file |
| The note is stored for every switch | Terminal in the switch record, chat on the branch row. Section 4 |

---

## 3. Storage and schema

Everything is in AO's SQLite database (`ao.db` in the AO data directory). Nothing is kept in separate files.

Compared with `main`, this feature adds two tables, adds one column, removes six old tables, and changes nothing else.

### 3.1 Added: `tui_provider_switches`

Already on this branch (migration `0230`), unchanged by this design. One row per terminal switch. It lets AO recover if something fails mid-switch.

```sql
CREATE TABLE tui_provider_switches (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    request_id  TEXT NOT NULL,
    phase       TEXT NOT NULL CHECK (phase IN ('prepared','source_stopping',
                'target_starting','completed','failed','recovery_required')),
    record_json TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    UNIQUE (session_id, request_id)
);

-- only one unfinished switch per session
CREATE UNIQUE INDEX tui_provider_switch_active
    ON tui_provider_switches (session_id)
    WHERE phase NOT IN ('completed','failed');
```

Two triggers, on insert and on update, write a `session_updated` event to the existing `change_log` table so the UI sees the switch phase change.

Fields inside `record_json`:

| Field | Holds |
|---|---|
| `SourceHarness`, `TargetHarness` | Provider switched from and to |
| `SourceNativeID`, `TargetNativeID` | Each provider's thread ID |
| `SourceLaunchID`, `SourceRuntimeID`, `TargetLaunchID` | Process identifiers, for recovery |
| `SourceModel`, `SourceEffort` | The old provider's settings, for rollback |
| `Bootstrap` | The handoff note |
| `Error` | Failure detail, if any |
| `Material` | Summary and exchanges read from the agent. Unused under this design; to be removed |

Rows are kept. There is no retention rule for now.

### 3.2 Added: `session_terminal_turns`

New. One row per terminal turn, fed by hooks.

```sql
CREATE TABLE session_terminal_turns (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id        TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    harness           TEXT NOT NULL,
    native_session_id TEXT NOT NULL DEFAULT '',
    user_request      TEXT NOT NULL,
    agent_response    TEXT NOT NULL DEFAULT '',
    requested_at      TIMESTAMP NOT NULL,
    responded_at      TIMESTAMP
);

CREATE INDEX idx_session_terminal_turns_session
    ON session_terminal_turns (session_id, id);
```

| Column | Holds | Why |
|---|---|---|
| `id` | Auto-incrementing number | Gives a stable order within a session |
| `session_id` | The AO session | Rows are deleted with the session |
| `harness` | Which provider handled the turn | The note can say who answered |
| `native_session_id` | That provider's thread ID | Decides which rows an import replaces (section 8) and supports "continue existing" (section 9.2) |
| `user_request` | The user's request text | Queued requests are appended to the same row, separated by a blank line |
| `agent_response` | The agent's final message | Empty until the agent stops, or if the turn was interrupted |
| `requested_at` | When the first request arrived | Used to merge with chat turns by time |
| `responded_at` | When the agent stopped | Empty means the turn is still open |

`harness` has no fixed list of allowed values, so adding a provider needs no migration for this table. The table has no triggers.

**How rows are written**

The source is the two hook events AO already receives: the prompt-submitted event and the stop event.

1. **Request hook arrives.**
   - If the session's newest row is still open and on the same thread, append the text to it.
   - Otherwise insert a new row.
2. **Stop hook arrives.** Fill `agent_response` and `responded_at` on the newest open row. If none is open, ignore the event.
3. **After an insert.** Delete everything beyond the newest 50 rows for that session.

A hook event is recorded only if all of these hold:

- The session is in terminal mode. The same hook-fed fields also update in chat mode, so without this check chat turns would land in both stores.
- It is not one of AO's own messages. Handoff notes and coordination messages are skipped. The hook client already recognises the `<ao-handoff-request` prefix; the note must keep that prefix.
- It comes from the session's current launch. AO already rejects hook events from a previous launch, and this table uses the same check.

**Which agents feed this table**

The table only holds what an agent's hooks report, and that differs by agent:

| Agent | User's request | Agent's final message | Basis |
|---|---|---|---|
| Claude Code, Continue | Yes | Yes | Tested |
| Codex | Yes | Yes | Tested at the hook client |
| OpenCode, Kilocode, Mimo Code | Yes | Yes; AO's plugin sends the turn's last assistant text on stop | Tested with stubbed events |
| Pi, Oh My Pi, Prime Agent | Yes | Yes if the agent's end-of-run event lists its messages | Request tested; response tested with a stubbed event only |
| Gemini, Devin, Muse, Droid, Kimchi, Qwen, Cursor, Kimi, Copilot, Autohand | Yes if the hook payload has `prompt` | Yes if it has `last_assistant_message` or `prompt_response` | Field names unverified; a real turn will show |
| Auggie | No; it installs no prompt hook | Same condition as the row above | Unverified |
| Antigravity, Vibe, OpenCode 2 | No | No | Their hooks use other event names, or carry no text |
| DeepSeek Harness | No | No | It has no hooks in the terminal |

The hook client reads turn text from every agent outside the replay-checkpoint group (Claude Code, Codex, Continue), under several field names, and sends it in two fields that feed this table only: `turnUserRequest` and `finalAssistantMessage`. Neither is ever used as a replay checkpoint, which is why text can be taken from agents whose hooks do not guarantee main-turn boundaries.

An agent with no rows hands over the session facts, the original task and the changed files, but no turns. A turn with a request and no recorded response shows "(no final response recorded)" in the note.

**Signals from a previous agent**

After a switch the old agent's hook config stays in the workspace. Every hook signal now names the agent that sent it, and the daemon drops a signal whose agent is not the one the session runs.

**How rows are read**

- **Building the note:** the newest rows for the session, merged with chat turns by time (section 5).
- **`ao session history`:** the same rows, ten per page, newest first (section 6).

### 3.3 Added column: `conversation_branches.handoff_note`

```sql
ALTER TABLE conversation_branches
    ADD COLUMN handoff_note TEXT NOT NULL DEFAULT '';
```

Holds the note for a chat switch. A chat switch already creates one branch row for the new provider's thread, in the same transaction that commits the switch.

### 3.4 Removed

Migration `0179`, already committed on this branch, drops the schema of the earlier switching feature:

- `agent_switches`
- `agent_native_sessions`
- `agent_switch_failure_policy`
- `agent_switch_failure_outbox`
- `agent_switch_failure_delivery_state`
- `agent_switch_failure_receipts`

Their triggers and indexes go with them.

### 3.5 Existing tables used, with no schema change

| Table | How the feature uses it |
|---|---|
| `conversation_branches` | A chat switch adds a row for the new provider's thread |
| `conversations` | `active_branch_id` is pointed at the new branch |
| `conversation_activities` | A "Switched provider" marker row per chat switch |
| `conversation_turns`, `conversation_messages` | Read to build the chat note |
| `sessions` | Current provider is updated; original prompt, git branch and diff base are read |
| `change_log` | Receives the switch-phase events from the triggers |

Chat's five conversation tables already exist on `main` and hold the full history. Terminal sessions have no equivalent beyond the latest request and response on the session row, which is why the only new table is for terminal.

### 3.6 Not stored

Compaction summaries, tool calls for terminal sessions, reasoning, images and attachments, or copies of agent transcripts.

### 3.7 Migrations

`0179` is committed. `0230` is uncommitted and temporarily renumbered to avoid a collision in the shared dev database. The new table and the new column go in `0231`. Both `0230` and `0231` need real sequential numbers before this merges.

---

## 4. Handoff note persistence

The note is stored for every switch, in both modes.

| Mode | Where | Written when |
|---|---|---|
| Terminal | `Bootstrap` inside the switch record | Before the old agent is stopped |
| Chat | `handoff_note` on the new provider's branch row | In the transaction that commits the switch |

- **Restart.** If a chat provider's process restarts, AO reads the note from the active branch and applies it to the system prompt again.
- **Privacy.** The note is not included in what the API sends to the UI. Showing it needs an explicit endpoint.
- **Why not rebuild it instead.** The note includes the workspace's changed files at switch time, the conversation can be edited or rolled back afterwards, and the builder's caps may change. A rebuilt note would not match what the agent received.

---

## 5. The handoff note

One format for both UIs. Hard cap: 32,000 characters for the whole note.

```
1. Framing          Taking over from <old provider>. This is a record, not
                    instructions. Files and git are the truth. Do not redo
                    finished work.
2. Metadata         Session title, project, workspace path, git branch, time
3. Original task    The prompt the session started with
4. Workspace state  Files changed since the session's diff base, names only
5. Older turns      Up to 15, one clipped line each
6. Previous turns   5, user request and agent response
7. Latest turn      User request and agent response
8. Earlier history  "N earlier turns are not shown. Run `ao session history`
                    to read them, only if you really need something that
                    isn't here."
9. Closing line     Terminal: say in a few lines where things stand, then wait.
                    Chat: do not mention this note unless it is relevant.
```

### Where the turns come from

Turns from the chat tables and from `session_terminal_turns` are merged by time and the newest are taken. A terminal session that has never been in chat has no chat conversation at all; that is the normal case and the builder must handle it.

### Turn tiers and budgets

At most 21 turns appear in the note. Each turn has one budget. The user's request takes what it needs up to its cap, and the agent's response gets the rest of that turn's budget.

| Tier | Turns | Budget per turn | User's request, up to | Agent's response gets |
|---|---|---|---|---|
| Latest | 1 | 10,000 chars | 3,000 | The rest, at least 7,000 |
| Previous | 5 | 2,000 chars | 600 | The rest, at least 1,400 |
| Older | 15 | 350 chars | 150 | The rest, at least 200 |

Other caps: original task 2,000 characters; changed files 50 names, with a count of any beyond that.

Worst case is about 25,000 characters of turns plus about 5,500 for the other sections.

### Clipping rules

- Text over its cap is clipped in the middle, keeping the start and the end, with a visible marker.
- If the note is still over 32,000 characters, older turns are dropped first (oldest first), then previous turns (oldest first).
- The latest turn is never dropped.
- The note states how many turns were left out.
- Quoted turns are escaped so text inside one cannot close the handoff block.

### Left out of the note

Tool calls and their output, reasoning, images and attachments, approval and permission state, earlier handoff notes, and AO coordination messages.

---

## 6. Reading earlier history

A new command lets the agent read turns that are not in the note.

- **Command:** `ao session history`, defaulting to the session the agent is running in.
- **Output:** full, unclipped turns, newest first, ten per call, with an option to page further back.
- **Access:** an agent can read only its own session's history.
- **Coverage:** in chat, the whole conversation. In terminal, the stored terminal turns (up to 50) plus any chat turns.
- **Storage:** none of its own. It reads the conversation tables and `session_terminal_turns`.

If the command cannot be reached from a given provider's agent, that provider's note drops the line and says the older turns are not available.

---

## 7. How a provider switch works

### 7.1 In chat

1. The user picks a provider from the session menu. The menu entry is enabled only when the session is idle.
2. The daemon builds the note from the chat tables: completed turns that were not rolled back, human prompts only, the final agent message of each turn.
3. It stops the old provider and starts the new one on a new branch of the same conversation. The note is added to the new provider's system prompt.
4. The new branch (with the note), the session's new owner and a "Switched provider" marker are saved in one transaction. If the new provider fails to start, the old one is restored.
5. The user sees the same chat with a divider. The new agent speaks when the user sends the next message.

### 7.2 In terminal

1. Same menu, same idle rules.
2. The daemon builds the note from `session_terminal_turns`, merged with chat turns if the session has any.
3. It saves the switch record with the note, drains terminal input, stops the old agent and confirms it has stopped.
4. It starts the new agent in the same terminal with the note as its launch prompt. If that fails, it relaunches the old agent on its old thread.
5. The new agent replies once with where things stand, then waits. The user sees the old agent's screen replaced by the new one.

A terminal agent acts on its launch prompt immediately, which is why the terminal closing line tells it to report and wait.

---

## 8. Switching between Terminal and Chat

This is the existing interface switch: same provider, same provider thread, different UI. This design only adds what happens to the terminal turn rows.

```
TERMINAL                   CHAT                         TERMINAL
turns -> terminal rows     turns -> chat tables         turns -> terminal rows
                    |                            |
        import provider's history        nothing is copied;
        into the chat tables, then       the chat tables just
        delete the terminal rows         stop growing
        for that thread
```

- **Terminal to Chat.** The chat side resumes the same provider thread and imports its history into the chat tables. When the import succeeds, the matching terminal rows are deleted.
- **Chat to Terminal.** Nothing is copied. The chat tables keep what they have and new turns go to terminal rows.
- **Back to Chat again.** The import runs again and the matching terminal rows are deleted again. The import skips anything it has already brought in, by matching each item's provider ID.

### Which rows an import deletes

Only the rows whose `native_session_id` matches the imported thread. Clearing every row for the session would be wrong after a provider switch:

```
Terminal on Claude  ->  switch to Codex (terminal)  ->  Terminal to Chat on Codex
```

The import brings in only Codex's thread. The Claude rows were never imported, so they must stay or the next note loses them.

### How duplicates are prevented

1. **A turn is written to one store only.** Terminal rows are recorded only in terminal mode; chat turns go only to the chat tables.
2. **After an import, the terminal copy is deleted**, so a turn never sits in both places.
3. **If there is no import** (the provider cannot replay its history), the terminal rows stay and the chat tables never received those turns. Still one copy.
4. **The handoff note is never stored as a turn.** The hook client and the chat controller both recognise the `<ao-handoff-request` prefix.

### Edge cases

- **A late hook at the boundary.** A stop event from the old mode can arrive just after the UI flips. The launch check in section 3.2 rejects it.
- **A row with no thread ID.** If the hook had not reported the thread ID yet, the row will not match on import. For those rows, match on provider and time instead.

---

## 9. After v0

### 9.1 Optional "what should we do next?" note

A text field in the switch menu.

- **Chat:** it becomes the user's first message to the new provider, visible in the timeline.
- **Terminal:** it is appended to the launch prompt as the user's instruction, and the closing line changes from "wait" to "do this".
- It is recorded as a normal user turn, so a later switch carries it.

### 9.2 New agent session, or continue the existing one

When switching to a provider this AO session has used before, the user can choose between a fresh thread and that provider's existing thread. Only the latest thread per provider is kept as "existing".

| | New session | Continue existing |
|---|---|---|
| Thread | Fresh | That provider's latest thread in this AO session |
| Note | Full | Only the turns since that provider was last active |
| Offered when | Always | The provider was used before and its thread can still be resumed |

- If the resume fails, fall back to a new session with the full note.
- The shorter note adds one framing line: files have changed since you last worked here, so check them again.
- No extra table is needed. For terminal, the newest row per provider in `session_terminal_turns` gives the thread ID and last-active time. For chat, `conversation_branches` has them.

### 9.3 What v0 must do so these are not blocked

- Keep the outgoing provider's thread ID when switching. Chat keeps it on the branch; terminal keeps it on the switch record and on the turn rows.
- Timestamp every stored turn and record which provider answered it.
- Build the note from a list of turns passed in, so the same builder produces both the full note and the shorter one.

---

## 10. Still to verify

| Item | Why it matters |
|---|---|
| `ao session history` is reachable from chat-mode agents as well as terminal ones | Section 6 depends on it |
| A Terminal to Chat to Terminal to Chat round trip produces no duplicate chat messages | Section 8 relies on the import's stated contract, which has not been tested here |
| Re-applying the stored note on a chat restart does not give the provider the note twice | Section 4 |
| The 32,000-character cap is safe for launch-argument delivery | The cap counts characters; Linux limits one argument to about 128 KB, and multi-byte text can approach that |
| Agents whose launch prompt is typed in after start receive the note reliably | Competitors report lost prompts on this path |
| The changed-files list is cheap enough on large worktrees | It runs at switch time |

---

## 11. Size budget

**Hard cap: about 3,000 production lines for the whole branch.** This replaces the 5,000-line soft target in the 7 October plan. Production lines exclude tests and generated code (query bindings, API schema files).

The branch was at about 1,900 when the cap was set, which leaves about 1,100 of net growth.

Proposed fit, not yet confirmed. The figures are estimates from the sizes of comparable code on the branch.

| Work | Net lines | Running total |
|---|---|---|
| On the branch today | | 1,900 |
| This design, with a slimmer `ao session history` | +570 | 2,470 |
| Cleanup of duplicate and unused code already on the branch | -100 | 2,370 |
| The six bugs in section 12 | +150 | 2,520 |
| Post-v0: "what should we do next?" note | +100 | 2,620 |
| Post-v0: new session or continue existing | +325 | 2,945 |

Deferred to stay under the cap: delivery tracking, "Copy handoff note", and the "view what was handed over" UI (about 250 lines). The note is already stored, so these can be added later without schema work.

If the count runs over while building, cut in this order:

1. Slim `ao session history` further, or defer it. Without it the note says older turns are not available.
2. Ship "continue existing" for chat first and terminal later.

The running production-line count is reported at each step of the build.

---

## 12. Recommended alongside this design, not yet confirmed

These came out of the code review and the competitor research. They were not part of the handoff discussion and need a decision before they are scheduled.

**Bugs found on this branch**

| # | Bug | Status |
|---|---|---|
| 1 | A switch record in `recovery_required` never clears, which locks that session out of switching | Fixed: the record is closed once the session has a running provider again |
| 2 | Two failure exits leave the old agent stopped with no rollback | Fixed: the project is loaded before the stop, and a failed ownership write restores the old agent |
| 3 | The UI generates a new idempotency key on every retry | Left as is. A failed switch keeps its key's row, so reusing the key would be refused; a fresh key per click is what the backend needs today |
| 4 | Phase writes to the switch record ignore errors | Fixed: failures are logged, and the old agent is not stopped unless the "stopping" phase was recorded |
| 5 | The chat note can include a status line such as "Compacted history, freeing 38.1k tokens" | Fixed by removing the compaction summary |
| 6 | The switch menu uses a hardcoded provider list and ignores the capabilities endpoint | Partly fixed: the menu now reads agent readiness (hides uninstalled agents, greys out signed-out ones). It still does not call the per-session capabilities endpoint |
| 7 | Switching away from Codex was refused with "unsent text in the terminal composer" whenever Codex showed its "3 warnings · f2 to view" badge | Fixed: the composer ends at the first status row |
| 8 | Switching back to Claude Code killed it with "Session ID … is already in use" while the switch reported success | Fixed: a handoff launch gets a fresh native session id |
| 9 | A switch reported success when the new agent exited as soon as it started | Fixed: the new agent is watched for 1.5 seconds; if it exits, its terminal is removed, the old agent is resumed, and the error carries the agent's last line |
| 10 | GitHub Copilot's answers were never recorded: its stop hook names an event log instead of carrying the text, and that log is written in batches about 100 ms apart, after the hook has run | Fixed: the hook reads the log and waits for the line that records this stop |
| 11 | A stop event with no text left the turn open, so every later request was appended to the same row | Fixed: a main-agent stop always closes the turn |
| 12 | The chat note left out requests sent through AO (`ao send`, an orchestrator) and did not say which agent answered each turn | Fixed: those requests are included, as they already were in terminal mode, and each chat turn is credited from the switch markers |
| 13 | After a session switched from Claude Code to an agent the usage tracker does not cover, the tracker logged a constraint failure every three minutes | Fixed: a discovered Claude transcript is filed under Claude Code, not under the session's current agent |
| 14 | GitHub Copilot was reported as installed when only VS Code's launcher was present. Launching it stopped at "Install GitHub Copilot CLI? [y/N]" instead of starting | Fixed: the resolver passes over VS Code's launcher, on PATH and in its install folder, and keeps looking for the CLI itself |

**From the research**

- Record "note delivered" separately from "agent started", and offer "Copy handoff note" when delivery is unconfirmed.
- Show a boundary in both UIs naming both agents, with a way to view what was handed over. The note is now stored (section 4); the UI for it is not yet decided.
- A recall test: after a switch, the new agent must quote one earlier user turn and one earlier response. Also run A to B to A and check nothing is duplicated.
- A retention rule for switch records, which are kept indefinitely for now.
- Later candidates: remaining quota per provider in the switch menu, a suggestion to switch near a usage limit, the previous agent's plan in chat notes.

---

## 13. Implementation status

Built on this branch, uncommitted, as of 2026-10-11.

| Part of the design | State |
|---|---|
| `session_terminal_turns` table and `handoff_note` column (migration `0231`) | Done |
| Recording terminal turns from hooks, terminal mode only | Done |
| Shared note builder with the three turn tiers, budgets and clipping | Done |
| Chat switch: note built from both stores, stored on the branch, re-applied on restart | Done |
| Terminal switch: same note, stored in the switch record | Done |
| Deleting a thread's terminal turns after a Terminal to Chat import | Done |
| `ao session history` and its endpoint | Done |
| Bugs 1, 2, 4 and 5 from section 12 | Done |
| Target is checked for install and sign-in before the running agent is stopped | Done |
| Turn text forwarded from every agent whose hook carries it; plugin changes for OpenCode, Kilocode, Mimo Code, Pi, Oh My Pi, Prime Agent | Done |
| Hook signals tagged with their agent; signals from a previous agent dropped | Done |
| Switch menu: a "Switch agent" submenu listing 24 terminal agents or 13 chat agents by session mode, hiding uninstalled ones (bug 6) | Done. Typechecked, component tests pass, terminal menu used in the running app |
| Bugs 7 to 14 from section 12, found in live testing | Done |
| The two post-v0 features in section 9 | Not started |

**API changes**

- New endpoint: `GET /api/v1/sessions/{sessionId}/history`, with `offset` and `limit`.
- New optional fields on the activity signal: `finalAssistantMessage`, `turnUserRequest` and `harness`.
- `openapi.yaml` and `frontend/src/api/schema.ts` are regenerated.

**Things that differ from the design text**

- A terminal switch that does not come through the chat service's front door builds its note from the stored terminal turns alone, without chat turns or changed files. Every production switch goes through the front door.
- Temporary migration numbers `230` and `231` are listed in the migration ledger test so the suite passes. Those two lines change when the files are renumbered.

**Checked**

Targeted tests pass for the note builder, the terminal turn store, hook recording, the chat switch (merged turns, stored note, history), the recovery-lock fix, the CLI command, and the API spec parity. `go vet ./...` is clean.

**Terminal agents in the menu**

24, not 25: Unreal Agent is chat-only and cannot launch in a terminal. The list is the other 12 chat agents plus Continue, Gemini, Devin, Muse, Kilocode, Mimo Code, Prime Agent, Auggie, Copilot, Antigravity, Autohand and Vibe.

**Not checked**

- The chat-mode switch menu in the running app. The chat switch itself was run through the CLI; the terminal menu was used in the app.
- Whether DeepSeek Harness can be switched away from in a terminal. It reports no activity there, so AO may be unable to prove it is idle.
- The full backend test suite and the linter. Targeted tests and `go vet` on the changed packages pass; the full suites are left to CI.
- Section 10, except where section 14 says otherwise.

---

## 14. Live test results, 2026-10-11

Run against real agents, first on a private test daemon and then in the dev app built from this branch.

**Works**

| What | Result |
|---|---|
| Terminal switch through the app's menu: Codex to Claude Code to GitHub Copilot | Each agent received the note, summarised where things stood and waited. Copilot named which agent had noted each code word |
| Terminal switches through the CLI: Claude Code to Codex, Codex to Kimchi, Kimchi to Pi, Pi to Claude Code, Codex to Claude Code and back | Notes correct; stored turns stayed at three with no duplicates through a round trip |
| Chat switch: Codex to Claude Code to Codex, and Codex to Kimchi | New branch active, note stored on the branch, switch marker shown. Claude Code and Codex both recalled earlier turns and who answered them |
| Turn recording, request and response | Claude Code, Codex, Kimchi, GitHub Copilot |
| The note itself is never stored as a turn | Confirmed on every switch |
| Target check before the running agent is stopped | "cursor is not signed in", Codex untouched |
| New agent exits at once | Switch fails with the agent's own error line and the old agent is resumed on its thread |
| `ao session history` | Prints the merged turns |
| Menu contents | Current agent left out, uninstalled agents hidden, Cursor greyed out with "sign in required" |

**Does not work yet**

| Problem | Detail |
|---|---|
| Kimchi in chat never sees the note | It ignores the appended system prompt (the flag is not in `kimchi --help`). The note needs to travel with the first user message for agents like this |
| DeepSeek Harness in chat | Its chat driver ignores the system prompt in code, so the same applies |
| A switch still reports success when the new agent starts but cannot work | Seen with Kimi (not logged in) and Pi (no API key): the process stays up showing its own error. Only an immediate exit is caught. Needs the "note delivered" signal from section 12 |

**Not covered**

- Turn recording for Antigravity, Vibe, OpenCode 2 and Auggie, and Cursor's responses: none of these could be run here.
- Kimi, Cursor and Pi beyond the launch: not signed in on this machine.
- The agents not installed on this machine.

**Section 10 items settled**

- `ao session history` works for a terminal agent. Not tried from a chat agent.
- Re-applying the stored chat note on a restart is still untested live. By the code, the note is added to the system prompt once per start.
