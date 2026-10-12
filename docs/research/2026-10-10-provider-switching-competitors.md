# Provider switching: competitor hunt and what AO should build

Research dates: 2026-10-10 to 2026-10-11. Branch: `feature/provider-switching`.

"Provider switching" here means: the user continues the same task on a different coding agent (for example Claude Code to Codex), and the new agent receives a handoff of context from the old one.

---

## 1. Summary

1. **The space is crowded and young.** We screened more than 150 products and tools. About 30 apps and about 45 small utilities do real cross-agent handoff. Most appeared in 2026 and most have few users.
2. **The closest competitors agree with the design we chose.** T3 Code, Synara, Happier, Jean, Kanna and Orbital all build the handoff from the app's own stored conversation, with no model-written summary. T3 Code's docs say it outright: a handoff is "a budgeted selection, not an agent-written summary".
3. **Nobody we verified depends on a compaction summary.** Dropping it from AO's note is consistent with the field.
4. **Most in-place switchers deliver the note with the user's next message.** AO delivers it immediately as the launch prompt. That is the largest design difference and the main open decision (section 7.2).
5. **Terminal is where everyone is weak.** Orca and Superset start a new session for the other agent. Of the in-place switchers, only Happier works on an agent running as a terminal program, for Claude only and not yet released. Orca has a cluster of open issues where the pasted prompt never reaches the new agent's terminal.
6. **Usage limits are the main reason users switch.** They want to see each agent's remaining quota before switching, and to be nudged before the limit, not after.
7. **Some products refuse to switch on purpose.** Amp locks the model per thread, Nimbalyst blocks agent changes after a session starts, and CodePilot locks the runtime after the first run. Their reasons are worth reading before we ship (section 6.4).

Recommendation in one line: keep the messages-only design, adopt the framing and selection rules the best competitors converged on, fix delivery confirmation and failure recovery, and treat terminal delivery as the part that needs the most care.

---

## 2. How this was done

| Step | What ran |
|---|---|
| Discovery | Six research agents in parallel: the five named competitors, desktop apps, terminal tools, IDEs and single-agent harnesses, GitHub session-transfer tools, community and launch channels |
| Deep dives | Three agents reading source code: Orbital and Happier; eight smaller apps; vendor importers and VS Code |
| Fact-check | Two agents independently re-checking 37 key claims against source: 27 confirmed, 10 partly right, 0 wrong. Corrections are applied below |
| Synthesis | Main session, plus an earlier direct read of Synara's handoff code |

All agents ran on Sonnet 5.5. Nothing was cloned and run; source was read through the GitHub API and raw files.

**Evidence labels used below**

- **Code**: seen in the product's source code.
- **Docs**: stated in official docs, README or changelog.
- **Claimed**: marketing page, launch post or third-party write-up only.

**Limits**

- Reddit and X could not be fetched, so user-demand evidence comes mostly from Hacker News and dev.to.
- Closed-source products (Conductor, Agentastic, Cursor, Clinch's site claims) are known only from docs and changelogs.
- No competitor was installed or exercised. "Code" means the code says so, not that we watched it work.
- Product counts are approximate. The six discovery lists overlap and were merged by hand.

**Related file.** Another session wrote [provider-switching-competitor-analysis.md](provider-switching-competitor-analysis.md) with four evidence notebooks during the same window. It covers some products this hunt did not, and reaches one different conclusion. Section 9 compares the two.

---

## 3. The landscape

Four different things get advertised as "switching":

| Class | Meaning | Roughly how many found |
|---|---|---|
| **A. Cross-agent handoff** | The same task continues on a different agent with context carried over | About 30 apps, about 45 small tools |
| **B. Model picker** | One agent or harness keeps control and changes the model underneath | About 35 |
| **C. Side by side** | Several agents run in parallel; no conversation moves between them | About 35 |
| **F. Failover** | Automatic or semi-automatic switch on a usage limit or error | About 12 |

Only class A is what AO is building. Class B is useful for engineering patterns. Class F is where user demand points.

---

## 4. Products with real cross-agent switching

### 4.1 In place: same thread or session keeps going

These are the direct comparisons for AO.

| Product | Stars | What is handed over | Size limit | When delivered | Evidence |
|---|---|---|---|---|---|
| **T3 Code** | 27k | App's own history: messages, command results, file-change names, plans | 16,000 by default, counted as 1 token per byte; ceiling 64,000 | With the next message | Code, fact-checked |
| **Synara** | n/a | App's own history: last 6 messages (2,400 chars each), older ones clipped to 320 chars | 32,000 chars | With the next message | Code (read directly) |
| **Happier** | 1.9k | App's own transcript: user and assistant text, the old agent's plan marked as past | 120,000 chars | With the next message | Code, fact-checked; not in a release yet |
| **Jean** | 1.3k | App's own run log: last 40 runs as User/Assistant lines | 60,000 chars | Hidden block on the next message | Code, fact-checked |
| **Kanna** | 0.7k | App's own transcript including tool calls; large tool output older than the most recent 100,000 chars is elided | 100,000 tokens, counted as 400,000 chars | Prepended to the first prompt on the new agent | Code, fact-checked |
| **Atlas** | 9.6k | Previous conversation attached as a "past session" | Not established | With the next message | Code, fact-checked |
| **Agent of Empires** | 3.3k | Recap from its event log: 20 turns, tool arguments stripped | 24,000 chars | Prefilled in the composer, never auto-sent | Code, fact-checked |
| **Helix** | 0.8k | Prior transcript repopulated into a fresh thread | 400,000 bytes, oldest dropped first | At switch | Code, fact-checked |
| **Kandev** | 0.9k | "Continuation package": task, user messages, summaries, repo status, failure reason | 4,000 chars per field | At automatic failover | Code, fact-checked; switched off by default |
| **Orbital** | 0.3k | Shared chat transcript, only the part the target has not seen | 60,000 chars | Prefix on the next message | Code |

Notes:

- **T3 Code is the richest reference.** It always keeps the latest user message, the latest answer and the *first* user message, then fills newest to oldest. It never truncates the user's new message. It sends history to Codex as real history items and to other agents as a text preamble. When the user returns to an agent used earlier, it resumes that agent's own thread and sends only what it missed.
- **T3 Code does not require idle.** A message sent mid-turn is queued and the handoff is built when that queued run starts.
- **Happier is the closest to AO's rules:** strictly idle, old agent confirmed stopped, then cutover. It has five typed outcomes for a switch and an idempotent retry, but no rollback to the old agent.
- **Orbital is not a handoff.** It is a "Talking to" dropdown that routes messages inside one shared chat. The old agent is not stopped and there is no idle check. It was first reported as the closest competitor; the code says otherwise.
- **Kanna cleans up on switch:** it clears the old agent's session token and stops the warm Codex process so a later switch back cannot silently resume a stale thread.
- **Kandev's automatic failover is built but not live.** The code is wired in behind a feature flag that is off in every environment, and its spec is still marked draft.
- **Atlas avoids switching a busy chat in place**, with two exceptions: a chat whose first message has not been accepted yet, and a turn it stops itself after an optional "remember" step.
- **Only Happier switches a terminal agent in place.** Jean, Kanna, Kandev and Helix do it for structured chat sessions only. Agent of Empires can swap the tool in a terminal session, but carries no context when it does. Happier injects its note into the first terminal prompt, and only for Claude.

### 4.2 New session or tab: the old one stays as it was

| Product | Stars | What is handed over | Size limit | Delivery | Evidence |
|---|---|---|---|---|---|
| **Orca** | 89k | A pointer to the old agent's own transcript file plus the last prompt and answer; or "read the whole file" | 36,000 chars for the fallback terminal capture | Prefilled draft for Claude; pasted when ready for others | Code, fact-checked |
| **Superset** | 15k | Old agent's transcript file (Claude and Codex only), else recorded terminal output, else the visible screen | 36,000 chars | Inlined in the new agent's launch prompt | Code, fact-checked |
| **Conductor** | closed | A summary of the conversation, very likely model-written | Not published | New chat tab opens at once; summary arrives after | Changelog |
| **Maestro** | 3.4k | Context "groomed" by a model pass; option to send verbatim | Token estimate shown | New tab | Code |
| **VS Code + Copilot** | n/a | Transcript attachment: user and assistant prose only | Last 20,000 chars | New session | Code, docs |
| **Agentrium** | 43 | Editable brief: task, latest context, changed files from git | n/a | Staged in the prompt editor, never typed into the terminal | Code |
| **Kangentic** | 126 | Pointer only: the path of the old agent's session file | n/a | New session prompt | Code |
| **Clinch** | 6 | Pointer: session id, transcript path, working directory | n/a | New tab | Code |
| **Paseo** | 20k | A briefing the outgoing agent writes | n/a | New agent | Code |
| **claw-orchestrator** | 0.6k | Its own recorded transcript, opening request pinned | 240,000 chars | First message to a headless engine | Code |

Notes:

- **Orca and Superset are terminal-first, and both chose a new session over an in-place swap.** Superset's chat pane only forks the same agent.
- **Superset shows the estimated size before the user confirms**, and disables the button if it could not read the transcript.
- **Conductor is the main product using a summary instead of messages.** Its changelog once said the tab waited for the summary to generate; a later release opens the tab immediately.

### 4.3 The vendors' own importers

| Vendor | What exists | Direction | Notes |
|---|---|---|---|
| **OpenAI Codex** | Built-in importer, reachable from the terminal (`/import`) and its app-server | Claude Code and Cursor into Codex | Tool calls flattened to tagged text (2,000 and 4,000 char caps), thinking dropped, duplicate imports blocked by a content-hash ledger |
| **OpenAI** | `/codex:transfer` plugin for Claude Code | Claude Code into Codex | Calls the same Codex importer |
| **xAI Grok** | `grok import` | Claude Code into Grok; its picker also lists Codex and Cursor sessions | Conversion code is not public |
| **Anthropic** | Nothing | n/a | No import of another agent's session. Resuming a hand-written session file works but relies on an internal, unstable format |

Codex also has an app-server call (`thread/inject_items`) that adds real history items to an existing thread with no model call. T3 Code uses it. AO's Codex chat driver talks to the same app-server, so this is available to us (section 8, "Later").

### 4.4 Small transfer tools

About 45 utilities move a session between agents. They fall into five approaches:

| Approach | Examples | Trade-off |
|---|---|---|
| **Transplant**: write the conversation into the target's own session format | casr, transession, harness-convert, Skillsync/txcript, Continuo | Target's resume shows full history. Depends on private formats that change between versions; several have open bugs where the converted session will not resume |
| **Target's own import** | Codex importer, omnisession | Survives format changes. Exists for few targets |
| **Generated brief as a prompt** | cli-continues (1.5k stars), Hermes, passoff | Works for any target. Limited by command-line size |
| **Agent-written handoff file** | many `/handoff` skills, Paseo, baton | Captures intent and dead ends. Needs a live, responsive outgoing agent |
| **Pull on demand** | session-hub, Codeg, AMLink | Small prompt. The new agent has to be told to pull |

None of these tools scrapes the terminal to get the conversation. The ones that support terminal agents read the agent's own session files.

### 4.5 Automatic switching

| Product | Trigger | What moves |
|---|---|---|
| **SpecWeave 3** | 90% of the usage limit, read from hooks and status lines | Git state and a one-page note, not the conversation |
| **Sno Station** | 2% quota remaining, and only if the other vendor has headroom | A checksummed brief; the receiver must acknowledge before it may edit |
| **Kandev** | Provider errors and usage limits, down an ordered list of agents | A continuation package. Built, but switched off by default |
| **Hydra** | Rate-limit text in CLI output; user confirms with a keypress | Copied to the clipboard for the user to paste |
| **ccswarm** | Rate limit | Nothing: it drops the conversation and re-prompts fresh |

The two that trigger fully on their own (SpecWeave, Sno Station) hand over a thin brief, never the transcript.

---

## 5. Products without cross-agent switching

Listed so the "do they switch?" question has an answer for each.

**The five named competitors**

| Product | Verdict |
|---|---|
| Orca | Yes, new session (4.2) |
| T3 Code | Yes, in place (4.1) |
| Conductor | Yes, new tab with a summary (4.2) |
| Superset | Yes, terminal only, new session (4.2) |
| Agentastic | No cross-agent handoff found. It switches models inside its own agent and flips one session between Chat and Terminal |

**Model pickers only (class B):** Cursor, Zed's own agent, OpenCode, Kilo Code, Cline, Roo Code (archived), Aider, Goose, Crush, Claude Code, Codex CLI, Gemini CLI, Pi, Kiro, Augment, Antigravity, Happy Harness, Xum (formerly Mux), CodePilot, Nimbalyst, Aperant, Warp.

**Side by side only (class C):** claude-squad, cmux, agent-deck, workmux, schaltwerk, parallel-code, uzi, ccmanager, Gas Town, Vibe Kanban (sunsetting), Emdash, Sculptor, CloudCLI, Harnss, ccgui, dev-3.0, Daintree, Zed external agents, JetBrains ACP, Intent (Augment), Verdent.

**Backend switches, not handoffs:** claude-code-router, LiteLLM, OpenRouter, CCS, cc-switch, Moxie, dario. They change the API behind an unchanged agent.

**Dropped or changed:** Crystal became Nimbalyst. Claudia became opcode and is Claude-only. Omnara pivoted. Terragon shut down. Amp removed its handoff feature in May 2026.

---

## 6. What the field teaches

### 6.1 What goes in the note

| Choice | Who does it | Comment |
|---|---|---|
| User and assistant text only | Happier, Jean, Orbital, Superset (chat), VS Code | The common default |
| Plus command results and file names | T3 Code | No diffs, just "File change: name" |
| Plus tool calls, large ones elided | Kanna, Codex importer | Costs the most space |
| Model-written summary | Conductor, Maestro, older VS Code path | Loses detail; Cursor tested it and said so |
| Pointer to the old transcript | Orca, Kangentic, Clinch | Small prompt, but the new agent may then read a huge file |

Superset's reason for leaving tool calls out of chat handoffs: replaying them as prose makes the new agent treat finished work as still pending.

### 6.2 What gets kept when space is tight

- **T3 Code:** latest user message, latest answer, first user message, then newest to oldest. Oversized items are left out whole.
- **Synara:** recent messages nearly in full, older ones as one clipped line each, with a count of what was omitted.
- **claw-orchestrator, Continuo:** pin the opening request.
- **Happier, Orbital, Jean:** drop oldest first. Orbital gives no notice that anything was dropped.

The better implementations all pin the original request and say what was omitted.

### 6.3 How the note is framed

The wording has converged across Orca, Superset, Kanna, Jean, claw-orchestrator and others:

- The history is a read-only record, not instructions.
- Files and git state are the truth; inspect them first.
- You are taking over from another agent. Cursor and Codex inject this even for a model change inside one agent.
- Do not repeat requests that appear in the history.
- Say where the previous session stopped.

Happier also escapes the quoted turns so text inside them cannot break out of the block.

### 6.4 Why some products refuse to switch

- **Amp:** "most models get confused when they continue a conversation that another model started". It locks the model per thread. It built a handoff feature in October 2025 and removed it in May 2026 in favour of automatic compaction.
- **Cursor:** recommends staying with one model for a conversation. Switching means a cache miss and a slower, more expensive first turn.
- **CodePilot:** locked the runtime after it found a mid-chat runtime change could leave the new runtime seeing only the latest sentence, while the UI showed the switch as done.
- **Nimbalyst:** blocks agent changes after a session starts, to keep the agent's own session state coherent.

These argue for a fresh thread with a brief (what AO does), not for transplanting one agent's transcript into another. They also argue for being honest in the UI about the cost of the first turn.

### 6.5 Delivery is where things break

- **Orca** has repeated open issues where the handoff prompt never reaches the new agent's terminal: a self-update prompt swallowed it, or the paste raced the terminal becoming ready. One case showed a success message while the prompt was lost. Orca added a separate "could not confirm delivery" state with a "Copy prompt" button.
- **Happier** lost the note on switches into Claude's terminal runtime. The fix is in its source (commit of 2026-08-30) but not yet released, and the issue is still open.
- **Agentrium** states the rule plainly: never inject a brief into an unknown terminal input state, because a new CLI may be showing a trust or login prompt.
- **cli-continues** fails on its largest preset because the prompt exceeds the operating system's command-line size limit.
- **Superset** caps at 36,000 chars for the same reason.
- **T3 Code and Kandev** track delivery as its own state (pending, then delivered) and treat an unclear result as "start a fresh thread, do not resend".

### 6.6 What users say

From Hacker News, dev.to and launch threads:

1. **Usage limits are the main reason to switch.** Second is wanting a review from a different model.
2. **By hand, they ask for a HANDOFF.md or paste a summary.** What goes wrong: the new agent starts blank, redoes finished work, and loses the approaches already rejected.
3. **Summaries drop detail.** Users prefer a goal-directed handoff to a compacted one.
4. **Agent session formats are undocumented and break between versions.**
5. **They want quota visible per agent before choosing**, and a handoff before the limit hits.
6. **They dislike the new agent charging ahead on its own.** A cli-continues issue asks for it to stop and ask first.

One sceptical point worth keeping: the same context can reproduce the same mistakes, so a handoff will not always unstick a stuck agent.

---

## 7. What is ideal for AO

### 7.1 What the research confirms

These decisions were made before the hunt. The field supports each one.

| Decision | Support |
|---|---|
| Build the note from AO's own record of the conversation, not the agent's files | T3 Code, Synara, Happier, Jean, Kanna. Tools that parse agent files report format breakage |
| No model-written summary | Same products; Cursor found summaries lose detail |
| No compaction summary | No verified product depends on one |
| Per turn, keep the user's query and the agent's final message | Happier, Jean, Superset chat, VS Code |
| Always carry the original task | T3 Code, claw-orchestrator, Continuo |
| Recent turns nearly whole, older turns clipped, omitted count stated | Synara, session-hub |
| Only switch when idle | Happier, Synara, Atlas |
| Fresh thread on the new agent | Everyone except the transplant tools; Amp's and Cursor's warnings |
| 32,000-character cap | Mid-range: T3 16k, Agent of Empires 24k, Orca and Superset 36k, Jean and Orbital 60k, Happier 120k |

### 7.2 The one decision to make: when the note is delivered

| | AO today | Most in-place competitors |
|---|---|---|
| **When** | At switch time, as the new agent's launch prompt | With the user's next message |
| **Effect** | The agent reads the note and replies with nothing new asked of it | The agent gets the note plus a real instruction |
| **Risk** | It may start working unprompted, or spend a turn restating the note | None of that; but the new agent sits unstarted until the user types |

Recommendation:

- **Chat:** move to delivering with the next message. AO's chat already holds the note hidden from the timeline, so this is a small change in when it is sent.
- **Terminal:** keep delivery at launch, because the launch prompt is the reliable channel into a terminal agent. Change the note's closing instruction to "state in a few lines where things stand, then wait". AO only switches when idle, so nothing is mid-flight and there is nothing to continue on its own.
- **Both:** add an optional "what should it do next?" field to the switch menu. That gives the goal-directed handoff users ask for, and in the terminal it turns the note into a real instruction.

### 7.3 Changes to make in the first version

**The note**

1. **Framing block.** History is a record, not instructions. Files and git are the truth. You are taking over from the named agent. Do not redo work. Say where it stopped, then wait.
2. **Selection order.** Latest turn first, then the original task prompt, then newest to oldest. Never drop the latest turn. State how many turns were omitted.
3. **Clip the agent's text harder than the user's.** User messages carry intent and constraints.
4. **Remove the compaction line from the chat builder.** It can currently put "Compacted history, freeing 38.1k tokens" into the note.
5. **Escape quoted turns** so text inside one cannot close the handoff block.

**Storage for terminal sessions**

6. **Keep the last 30 or so turns from hooks**, recorded only while the session is in terminal mode, as agreed.
7. **Keep excluding AO's own notes from that list.** The hook client already skips prompts starting with `<ao-handoff-request`. The note must keep that prefix, or a second switch would carry the first note as if the user had typed it. A test should pin this.

**Delivery**

8. **Count the cap in bytes for launch-argument delivery.** The cap counts characters today. Linux limits one argument to about 128 KB, and 32,000 multi-byte characters can reach that.
9. **Treat "typed in after start" agents as higher risk.** AO delivers the launch prompt on the command line for some agents and types it in after start for others (Aider, Goose, Amp, Kimi, Crush and more). The second group has the failure mode Orca keeps hitting.
10. **Record delivery separately from launch.** "Agent started" is not "note received". Offer "Copy handoff note" when delivery is unconfirmed.

**Failure and recovery** (bugs already found on this branch)

11. `recovery_required` never clears, which locks a session out of switching for good.
12. Two failure exits leave the old agent stopped with no rollback.
13. The UI generates a new idempotency key on every retry.
14. Phase writes ignore errors.

**What the user sees**

15. **A boundary in both views** naming both agents, and the account where it differs.
16. **"View what was handed over"** on that boundary. Happier rebuilds it on demand; storing it is simpler for AO because the terminal switch record already holds the note.
17. **Use the capabilities endpoint** so the menu only offers agents that can accept a switch.

**Test**

18. **Recall test.** After a switch, the new agent must quote one specific earlier user turn and one specific earlier answer. Also run A to B to A and check nothing is duplicated.

### 7.4 Where AO can stand out

- **In-place switching in the terminal, for every agent.** Orca and Superset start a new session. Of six in-place switchers checked, five work on structured chat only. Happier is the exception, for Claude only and unreleased. Tandem CLI is reported by the other analysis to do it by mirroring session files (section 9). A reliable in-place terminal swap across agents would put AO ahead.
- **One switch for chat and terminal.** Most competitors support one or the other.
- **Rollback to the old agent on failure.** Happier and Orbital do not roll back.

---

## 8. Ideas kept from the hunt

**Next, after the first version works**

| Idea | From | Why |
|---|---|---|
| Show each agent's remaining quota in the switch menu | Orbital | The top user request; AO has usage reporting on other branches |
| Suggest a switch near the limit, user confirms | SpecWeave, Sno Station, Hydra | Hand off before the limit, while the agent can still respond |
| Return to an earlier agent: resume its own thread, send only what it missed | T3 Code, Happier, Orbital | Cheaper and keeps that agent's own memory. Needs a per-agent marker of what it has seen |
| Size preview before confirming | Superset, Maestro | Cheap trust-builder |
| A command the new agent can run to read older turns | T3 Code, Kanna, Happy Harness | Keeps the note small without losing access. AO already ships a CLI inside sessions |
| Carry a prompt the old agent never processed | Agent of Empires | Covers the "hit the limit mid-request" case |
| Stop warm background processes for the old agent on switch | Kanna | Prevents a stale resume on switch-back |

**Later, needs measuring first**

| Idea | From | Caution |
|---|---|---|
| Send real history items to Codex through its app-server | T3 Code | Item format must match the installed Codex version |
| Use Codex's own importer for Claude-to-Codex | OpenAI plugin | One direction only, text-only result |
| Switch policy as a setting: reset, new tab, or handoff | Atlas | More surface to test |
| Optional briefing written by the outgoing agent | Paseo, Amp, Kandev | Costs a turn; impossible once the agent has hit its limit. Never block the switch on it |
| Automatic failover down an ordered list | Kandev | Ownership and cost questions; start with a suggestion instead |
| Redact secrets from the note | sctxx, omnisession | A switch sends one vendor's conversation to another |
| Receiver acknowledges the note before it may edit | Sno Station | Closes a race they hit in practice |

**Avoid**

| Idea | Why |
|---|---|
| Parsing each agent's session files | Private formats that change between versions; Codex stores compaction encrypted |
| Writing sessions in another agent's format | Same fragility, plus open "will not resume" bugs in every tool that tries |
| A model-written summary as the default | Loses detail, costs a turn, unavailable when the agent is rate-limited |
| Reading the terminal screen for context | Nobody does it for handoff; lossy |
| Letting the new agent continue on its own by default | Users object; AO only switches when idle anyway |
| Pasting into a terminal that may not be ready | Orca's and Happier's open bugs |

---

## 9. How this compares with the other analysis in this folder

[provider-switching-competitor-analysis.md](provider-switching-competitor-analysis.md) was written by another session at about the same time.

**It found products this hunt did not verify:** agentctl, Tandem CLI, Voyager/sessionFlow, Vibeyard, 1Code (archived), showagent, Omnigent, agentproto, Turntrail, Agent Switcher, context-bridge, OpenHands.

- **Tandem CLI matters for section 7.4.** That file describes it as rotating real terminal agents inside one paired session by continuously mirroring history into each agent's own storage. If so, it is an in-place terminal switcher, built on the transplant approach this report advises against.
- **agentctl** is described there as keeping delivery receipts so a note is never replayed blindly, which supports item 10 above.

**This hunt found products that file does not cover:** Happier, Jean, Kanna, Atlas, Agent of Empires, Kandev, Helix, Maestro, Agentrium, Paseo, Kangentic, Orbital, Clinch, the vendor importers, and the automatic switchers.

**One real disagreement.** That file recommends keeping "the latest readable semantic checkpoint" (a compaction summary) in the note when one can be captured. This report follows the later decision to drop the compaction summary completely. The evidence here supports dropping it: no in-place competitor uses one, Codex's is encrypted, and Claude's can exceed the whole note budget.

Both agree on the rest: one AO session, fresh thread on the target, AO's own history as the source, idle-only, the original task as an anchor, history treated as data, and separate tracking of launch and delivery.

---

## 10. Open items

| Item | Status |
|---|---|
| When to deliver the note (7.2) | Needs a decision |
| Whether the first version includes the "what next?" field | Needs a decision |
| Tandem CLI, agentctl and the others from the second analysis | Not verified by this hunt |
| Conductor's summary mechanism, size and failure handling | Not public |
| Whether Kandev's failover will be turned on | Built, but behind a flag that is off everywhere |
| Reddit and X demand evidence | Could not be fetched |
| T3 Code issue 15997 (failed handoff leaves both providers stuck) | A single unconfirmed user report; not relied on here |

---

## 11. Sources

Detailed evidence, with file paths and links for every claim, is in the notebooks in [2026-10-10-provider-switching-evidence/](2026-10-10-provider-switching-evidence/):

| Notebook | Covers |
|---|---|
| `discovery-1-named.md` | Orca, T3 Code, Conductor, Superset, Agentastic, Happier, Paseo, Emdash, Nimbalyst, Vibe Kanban |
| `discovery-2-desktop-apps.md` | 42 desktop and web apps |
| `discovery-3-terminal.md` | 57 terminal and CLI tools |
| `discovery-4-ide-harness.md` | 28 IDEs and harnesses, 17 engineering patterns |
| `discovery-5-transfer-tools.md` | 32 session-transfer tools and their mechanisms |
| `discovery-6-community.md` | About 56 products from launch channels; user demand and pain |
| `deep-1-orbital-happier.md` | Source-level detail on Orbital and Happier |
| `deep-2-smaller-apps.md` | Kangentic, Codeg, Clinch, Continuo, Skillsync, SpecWeave, Sno Station, claw-orchestrator |
| `deep-3-first-party.md` | Codex importer, `/codex:transfer`, Grok import, Anthropic, VS Code, Cursor |
| `verify-1-key-claims.md` | Fact-check of 22 claims about T3 Code, Orca, Superset, Conductor |
| `verify-2-second-batch.md` | Fact-check of 15 claims about Jean, Kanna, Atlas, Agent of Empires, Kandev, Helix, Happy Harness, Happier; terminal in-place switching per product |

Main repositories read: [T3 Code](https://github.com/pingdotgg/t3code), [Synara](https://github.com/Emanuele-web04/synara), [Orca](https://github.com/stablyai/orca), [Superset](https://github.com/superset-sh/superset), [Happier](https://github.com/happier-dev/happier), [Jean](https://github.com/coollabsio/jean), [Kanna](https://github.com/jakemor/kanna), [Atlas](https://github.com/pacifio/atlas), [Agent of Empires](https://github.com/agent-of-empires/agent-of-empires), [Kandev](https://github.com/kdlbs/kandev), [Helix](https://github.com/helixml/helix), [Orbital](https://github.com/zqiren/Orbital), [Maestro](https://github.com/RunMaestro/Maestro), [OpenAI Codex](https://github.com/openai/codex), [cli-continues](https://github.com/yigitkonur/cli-continues). Conductor: [changelog](https://www.conductor.build/changelog).
