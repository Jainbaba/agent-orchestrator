# Discovery 3: terminal-first / CLI agent managers and cross-agent handoff

Researched 2026-10-10/11 with `gh api` (repo metadata, raw source, READMEs) plus WebSearch/WebFetch for blogs and docs. No repo was cloned or executed. Stars are GitHub stars at fetch time. "Last push" is the repo `pushed_at` date.

Class key: A = cross-agent handoff (same task or conversation continues on a different agent, context carried). B = model/provider/account switch inside one agent or harness. C = parallel or side-by-side agents, no conversation handoff. D = no switching. U = unclear.
Verified key: V = seen in source or official docs (URL or path given). I = inferred from a description or secondary source.

## Headline findings

1. Real cross-agent conversation handoff for terminal agents exists, but almost entirely as small single-purpose converters (stars: 0 to 1.5k), not inside the big multiplexers (claude-squad, agent-deck, cmux, workmux, emdash, etc.). None of the 16 multiplexers/worktree managers I checked moves a conversation between different agent CLIs.
2. Context never comes out of the TUI by scraping the terminal. Every A-class tool reads the outgoing agent's native session files on disk (`~/.claude/projects/**.jsonl`, `~/.codex/sessions/**/rollout-*.jsonl`, `~/.gemini/tmp/*/chats`, opencode SQLite, etc.), or uses hooks, or its own recorded transcript, or an agent-written handoff document. The only tool that scrapes a TUI (coder/agentapi, now archived) uses it for message parsing, not handoff.
3. Two injection styles: (a) write a converted session into the target's native store and use its resume command (`codex resume <id>`, `claude --resume <id>`), or (b) start the target fresh with a handoff text (launch prompt argument, `<conversation_history>` first message, or a file it is told to read).
4. OpenAI's own Codex CLI now ships first-party import of Claude Code (and Cursor) sessions (source verified). This is the strongest signal that native session import is becoming a vendor feature.
5. The only orchestrator I found that does provider switching on failure (ccswarm, rate-limit fallback) explicitly throws the conversation away and re-prompts fresh with a notice. Vibe Kanban docs explicitly say a new session with a different agent gets no conversation context.

## Summary table

| Name | Repo | Stars | Agents supported | Class | Context source | Verified? | One-line evidence |
|---|---|---|---|---|---|---|---|
| continues (cli-continues) | yigitkonur/cli-continues | 1564 | 16: Claude, Codex, Copilot, Gemini, Cursor, Amp, Cline, Roo, Kilo, Kiro, Crush, OpenCode, Droid, Antigravity, Kimi, Qwen | A | Native session files parsed, structured handoff markdown, launched as prompt arg | V | `src/utils/resume.ts` writes `.continues-handoff.md` and spawns target with prompt; README "240 handoff paths" |
| session-porter | liwala/session-porter | 46 | Claude, Codex | A | Native JSONL converted, written into target store + Codex sqlite | V | README `transfer --from claude --to codex --mode compact`; `src/handoff.ts` |
| session-bridge | FacuVCanale/session-bridge | 0 | Claude, Codex | A | Native JSONL to IR to target JSONL | V | README "convert", `claude_to_codex.py` |
| session-sync | npow/session-sync | 1 | Claude to Codex, Amp, Gemini | A | Claude hooks (PostToolUse, Stop) mirror into target session dirs live | V | README "How it works" |
| agent-session-bridge | bohdanpodvirnyi/agent-session-bridge | 105 | Pi, Claude, Codex | A | Hook-driven two-way mirror of native session stores + registry | V | README "How It Works" |
| authsec-bridge | authsec-ai/authsec-bridge | 9 | Claude, Codex, Gemini | A | Native session files to NeutralSession, native writer, HANDOFF.md fallback | V (blog; repo README empty) | authsec.ai blog 2026-05-07 |
| session-handoff (yuzushi) | yuzushi-dev/session-handoff | 67 | Claude Code, Codex | A | Agent-written handoff record via MCP + native session conversion engine | V | `server/migration_engine.py`, README "migrate codex" |
| session-handoff (fork) | beverlynngroup/session-handoff | 0 | Claude Code, Codex | A | Same as above (fork) | V | README identical, fork of yuzushi-dev |
| dotbabel /handoff | kaiohenricunha/dotbabel | 0 | Claude, Copilot CLI, Codex (+Gemini in toolkit) | A | Raw session JSONL, scrubbed digest `<handoff>`, optional private git repo transport | V | `docs/handoff-guide.md` |
| session-handover | hahahahahahahahah6/session-handover | 0 | Claude, Codex | A | Reads both transcripts, templated markdown (no LLM), PreCompact hook | V (blog) / I (injection) | dev.to post; injection not described |
| claw-orchestrator | Enderfga/claw-orchestrator | 588 | Claude, Codex, Antigravity, Grok, OpenCode, custom (headless) | A | Tool's own recorded transcript replayed as text in first message | V | `skills/references/sessions.md` "session_handoff" |
| Codex CLI native import | openai/codex | 128531 | imports Claude Code, Cursor sessions into Codex | A | Claude JSONL parsed (user/assistant text, tool calls as tagged notes) written as Codex rollout | V | `codex-rs/external-agent-migration/src/sessions/*` |
| Sno Station | sno-ai/sno-station | 500 | Claude Code, Codex, Cursor, Hermes, OpenClaw | A | Shared encrypted memory + mailbox; `sno reach call` | V (README) | README "How to use" step 1 |
| baton | blader/baton | 60 | Claude Code, Codex (skill) | A | Agent-written `.baton/*.md` verified against git | V | README |
| project-continuity | seriousz158/project-continuity | 99 | Codex, Claude, Kimi, OpenCode, ZCode | A | Shared `.relay/CURRENT.md` (64 KiB cap, JSON+MD) with writer lease | V | README |
| claude-codex-switch | gitgoready/claude-codex-switch | 0 | Claude, Codex | A | Session converter skill + Python CLI, registers in `~/.codex/state_5.sqlite` | V (README) | README features |
| ccswarm | nwiizo/ccswarm | 153 | Claude, Codex (Copilot unsupported) | B | None: fallback re-prompts fresh with a notice | V | `crates/ccswarm/src/session/bridge.rs` `apply_rate_limit_fallback` |
| agent-deck | asheshgoplani/agent-deck | 1054 | Claude, Gemini, OpenCode, Codex, Pi, Oh My Pi, Muse, more | B | Account switch moves Claude conversation file between config dirs, restarts with `--resume` (same agent only) | V | README "Switch a session's account on the fly" |
| CCS | kaitranntt/ccs | 2884 | Claude Code, Codex, Droid runtimes; many provider profiles | B | Env/proxy profile switch for the same harness; resume-lane warnings | V (README) / I (mid-chat) | README "Why CCS"; `src/auth/resume-lane-warning.ts` |
| Pi (pi-ai) | earendil-works/pi | 114077 | 20+ LLM providers | B | In-memory message context replayed to a new provider; thinking blocks to `<thinking>` text | V | `packages/ai/README.md` "Cross-Provider Handoffs" |
| OpenCode | anomalyco/opencode | 212523 | Many providers | B | Session history in its own store, `/models` in TUI | V (command) / I (history kept) | `packages/web/src/content/docs/tui.mdx` |
| Crush | charmbracelet/crush | 28591 | Many providers | B | Own session store | V | README "switch LLMs mid-session while preserving context" |
| Goose | aaif-goose/goose | 55132 | Many providers | B | Own session | V | docs "Switching Models Mid-Session ... /model" |
| Aider | Aider-AI/aider | 49456 | Many providers | B | Own chat history | V | docs `usage/commands.md` `/model` |
| Gemini CLI | google-gemini/gemini-cli | 107275 | Gemini models | B | Own session | V | `docs/cli/model.md` `/model` |
| Kimi Code CLI | MoonshotAI/kimi-code | 7823 | Kimi models + others | B | Own session; also `/import-from-cc-codex` (config only) | V | `docs/en/reference/slash-commands.md` |
| Qwen Code | QwenLM/qwen-code | 28404 | OpenAI/Anthropic/Gemini/Qwen APIs | B | Own session | V (README "Switch at runtime") | README |
| Happy | slopus/happy | 24084 | Claude Code, Codex, Grok | B | Mobile/desktop wrapper | V (README) | README "Switch models in the middle of a task" |
| CLIProxyAPI | router-for-me/CLIProxyAPI | 54736 | Wraps Codex/Claude/Grok/etc. as compatible APIs | B | Proxy re-targets a harness at another model | I (description) | repo description |
| claude-squad | smtg-ai/claude-squad | 8588 | Claude, Codex, OpenCode, Amp, any program | C | n/a (profiles pick the program at session creation) | V | README profiles |
| agent-of-empires | agent-of-empires/agent-of-empires | 3329 | Claude, OpenCode, more | C | n/a | I (README no mention) | README |
| uzi | devflowinc/uzi | 582 | claude, codex, any | C | n/a (stale: last push 2025-06-04) | V | README `uzi prompt --agents claude:3,codex:2` |
| workmux | raine/workmux | 2809 | Claude, Codex, Gemini, Copilot, more | C | `--fork` copies conversation within same agent | V | README `--fork` "Supports Claude Code and Codex" |
| schaltwerk | 2mawi2/schaltwerk | 287 | Copilot, Claude, OpenCode, Codex, Pi, Gemini, Qwen, Droid, Amp | C | n/a | V (README) | README |
| parallel-code | johannesjo/parallel-code | 1042 | Claude, Codex, Gemini, Copilot | C | n/a | V (README) | README |
| emdash | generalaction/emdash | 5951 | Claude, Codex, Cursor, OpenCode, Amp, Devin, Qwen, Droid, Copilot | C | n/a (hooks for status and resume) | V (README) | README "Agents" |
| jean | coollabsio/jean | 1321 | Claude, Codex, Cursor, OpenCode, Pi, Grok, Kimi, Antigravity | C | n/a | I | README |
| cmux | manaflow-ai/cmux | 28082 | Claude, Codex, Amp, others | C | Hooks save native session id for same-agent resume | V | README "Supported agent sessions can resume" |
| ccmanager | kbwo/ccmanager | 1261 | Claude, Gemini, Codex, Cursor, Copilot, more | C | Copies `~/.claude/projects/<src>` to `<target>` for new worktree (Claude only) | V | README "Session Data Copying" |
| Gas Town | gastownhall/gastown | 18320 | Claude (default), Codex, Copilot, Gemini, Cursor, Auggie | C | `gt handoff`: same-role fresh session fed by mail + hooked beads + `gt prime` | V | `internal/cmd/handoff.go` |
| Vibe Kanban | BloopAI/vibe-kanban | 28317 | 10+ agents | C | None: "New sessions don't inherit conversation history" | V | `docs/workspaces/sessions.mdx`; repo says sunsetting |
| CLI Agent Orchestrator | awslabs/cli-agent-orchestrator | 1406 | Kiro, Claude Code, Codex, Antigravity | C | Supervisor `handoff`/`assign` MCP tools send a task message to another agent's tmux terminal | V (docs/api.md) | `docs/api.md` |
| claude_codex_bridge (CCB) | SeemSeam/claude_codex_bridge | 3560 | Codex, Claude, Gemini, Kimi, Qwen, Cursor, Copilot, Pi, OMP | C | `/ask` delegation + shared `.ccb/ccb_memory.md` | V (README) | README |
| claude-codex-handoff (OpenMOSS) | OpenMOSS/claude-codex-handoff | 42 | Claude, Codex | C | Append-only message files in `.handoff/`, timer-polled | V | README |
| Ruflo (claude-flow) | ruvnet/ruflo | 74264 | Claude Code, Codex | C | Swarm orchestration, no session transfer found | I | README |
| codex-plugin-cc | openai/codex-plugin-cc | 34058 | Codex from inside Claude Code | C | Delegate task to Codex, results returned | I (description) | repo description |
| agentapi | coder/agentapi | 1498 | Claude, Goose, Aider, Gemini, Amp, Codex | D | In-memory terminal emulator + screen diff to build messages; ARCHIVED | V | README "How it works" |
| ACP | agentclientprotocol/agent-client-protocol | 4405 | protocol | D | `session/load` / `session/resume` are per-agent | V | agentclientprotocol.com RFD session-resume-replay |
| opensessions | Ataraxy-Labs/opensessions | 1234 | Amp, Claude, Codex, OpenCode | D | Read-only watchers of native session files for sidebar status | V | README |
| tmuxai | alvinunreal/tmuxai | 1955 | LLM terminal assistant (not an agent manager) | D | n/a | I | description |
| SuperClaude | SuperClaude-Org/SuperClaude_Framework | 23907 | Claude Code config framework | D | n/a | I | description |
| BMAD-METHOD | bmad-code-org/BMAD-METHOD | 54010 | Method/personas for many IDE agents | D | n/a | I | description |
| git-worktree-runner | coderabbitai/git-worktree-runner | 1791 | Worktree + AI tool launcher | D | n/a | I | README |
| oh-my-pi | can1357/oh-my-pi | 34915 | Own harness, multi-model roles | U | Model roles (advisor, etc.) not verified as mid-session switch | I | README |
| Amp CLI | closed source (sourcegraph) | n/a | Amp | U | Same-agent `handoff` (new thread with summary) per third-party write-ups; official page not fetched | I | hamel.dev notes, mitsuhiko/agent-commands |
| Droid CLI | Factory-AI (closed) | n/a | Droid | U | Docs fetched did not state mid-session model switch | U | docs.factory.com choosing-your-model |
| Omnara | omnara-ai/omnara | 2904 | now a managed agent platform | U | not investigated | U | repo description |

Counts: A = 16 rows (15 distinct tools plus 1 fork), B = 13, C = 17, D = 7, U = 4. Total 57 rows.

## Details (A and B tools)

### A-class

#### 1. continues / cli-continues (yigitkonur/cli-continues), A, V
- MIT, TypeScript, npm `continues` (alias `cont`). 1564 stars. Created before 2026-05, last push 2026-05-07 (5 months stale as of 2026-10-10). 16 agents, "240 handoff paths".
- Context source: reads the outgoing agent's native session files, read-only. Paths from README: `~/.claude/projects/`, `~/.codex/sessions/`, `~/.copilot/session-state/`, `~/.gemini/tmp/*/chats/`, opencode SQLite `~/.local/share/opencode/storage/`, `~/.factory/sessions/`, `~/.cursor/projects/*/agent-transcripts/`, `~/.local/share/amp/threads/`, `~/.crush/crush.db`, `~/.kimi/sessions/`, `~/.qwen/projects/*/chats/`, Kiro and Cline/Roo/Kilo VS Code globalStorage. One parser per tool in `src/parsers/*.ts`; adapter registry in `src/parsers/registry.ts`. Env overrides honoured (`CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `GEMINI_CLI_HOME`). Index cache `~/.continues/sessions.jsonl` (5-min TTL).
- Included: verbosity presets (README table): minimal 3 messages / 0 tool samples; standard (default) 10 messages / 5 tool samples / 500-char subagent detail; verbose 20 / 10 / 2000; full 50 / all. Plus a "Tool Activity" section (Bash/Edit/Grep counts and samples), session notes (model, tokens, thinking), and the original session file path so the target can trace back. YAML config `.continues.yml`.
- Injection: `src/utils/resume.ts` always writes `.continues-handoff.md` into the project dir, then spawns the target CLI in a PTY-inherited child process with the handoff as the first prompt argument. `crossToolArgs` per agent: `claude <prompt>`, `codex <prompt>`, `copilot -i <prompt>`, `opencode run <prompt>`, `droid exec <prompt>`, `cursor-agent <prompt>`, `amp <prompt>`, `--prompt` for Kimi. `--reference` mode sends a short prompt pointing at the file ("Read `.continues-handoff.md` first"). On Windows it always uses the short file-pointer prompt because of cmd.exe argument limits. Flags (model, sandbox, auto-approve, extra dirs) are mapped to the target's equivalents (`src/utils/forward-flags.ts`). `--debug-prompt` prints the exact prompt without launching.
- Notes for AO: this is the closest match to what AO wants (start target fresh, injected context), and it validates "launch prompt argument + file fallback" as the common pattern. It does not touch the target's native session store, so the target sees a brand new session.

#### 2. session-porter (liwala/session-porter), A, V
- MIT, TypeScript, npm `@liwala/session-porter`. 46 stars. Created 2026-05-18, last push 2026-05-20.
- Source: native JSONL for Claude and Codex (`src/claude.ts`, `src/codex.ts`), normalised to a `PortableSession`; "Universal Tool Taxonomy" maps Read/Write/Edit/Glob/Grep/Bash to Codex equivalents (`cat`, `rg`, `apply_patch`).
- Modes (README): `raw` exact chat (default), `full` chat + handoff, `compact` handoff + user/assistant only, `resume-only` context brief only. `fork --last-n 3` or `--from-turn 5`.
- Handoff content (`src/handoff.ts`): original task (first user message, 1000 chars), up to 5 "key decisions" (first assistant messages 20 to 500 chars), files modified, last user message (500 chars), last assistant message (500 chars), turn counts. Compact is "about 10% of original size". `generateResume()` text begins "You are continuing a coding session that was started in <agent>".
- Injection: writes the converted session directly into the target's session directory and registers it in the Codex SQLite DB so `codex resume <id>` lists it. `continue --in codex` transfers then spawns the target.
- Bonus: `COMPETITORS.md` (2026-05-09) is a useful landscape doc: sheal, Entire.io (`entire/checkpoints/v1` git branch storing `full.jsonl`), claude-conversation-extractor, simonw/claude-code-transcripts, Codex_Relay, cctrace. None transfer sessions.

#### 3. session-bridge (FacuVCanale/session-bridge), A, V
- Python, 0 stars, created and last pushed 2026-05-27. Claude to Codex and back.
- Parse to IR (`parsers/`, `models.py`), map tools (`tool_map.py`: `shell_command` to `Bash`, `apply_patch` to `Edit`), write target JSONL, place it where the target looks (`placement.py`). Resume with `codex resume <id>` or `claude --resume <id>`.
- Documented lossiness (README Limitations): encrypted Codex reasoning dropped, token counts zeroed, Edit vs apply_patch lossy, not a replay.

#### 4. session-sync (npow/session-sync), A, V
- Apache-2.0, TypeScript, 1 star, created 2026-03-03, last push 2026-03-09.
- Mechanism is unusual: installs Claude Code hooks, PostToolUse and Stop, in `.claude/settings.json` (`session-sync init`). Each tool call is translated (via `@npow/interchange-core`) and appended live to each detected agent's native session file (Codex rollout JSONL, Amp, Gemini). Stop hook sweeps text-only turns. When you want to switch: `session-sync status` prints `codex --continue <id>`. Hooks always exit 0 and print nothing.
- Pro: the target session is always up to date, so switching is instant and survives a crash or rate limit. Con: only Claude as source, a hook on every tool call.

#### 5. agent-session-bridge (bohdanpodvirnyi/agent-session-bridge), A, V (README)
- MIT, TypeScript monorepo (`packages/core,cli,pi,claude-code,codex,daemon`), npm `agent-session-bridge`. 105 stars. Created 2026-04-05, last push 2026-04-08.
- Pi, Claude Code, Codex. Mirrors resumable conversations continuously: detect source session, normalise to a shared message model, write mirror entries into the other tools' native stores, track offsets and identities in `~/.agent-session-bridge/registry.json`, prevent replay loops, `repair` malformed imports. Integration via per-tool hooks (`docs/HOOKS.md`), optional daemon for backfill. `import --tool codex --all` backfills.

#### 6. authsec-bridge (authsec-ai/authsec-bridge), A, V (blog; repo has no README text)
- Python (`session_bridge.cli`), 9 stars, last push 2026-05-08. Source: authsec.ai blog dated 2026-05-07.
- Reads `~/.claude/projects/*.jsonl`, `~/.codex/sessions` rollout JSONL, `~/.gemini/tmp/session-*.json` into a `NeutralSession` (ordered user/assistant turns, one-line prose summaries of tool calls, a banner telling the model the chat came from another CLI). Per-CLI writer creates a new native session and registers it with that CLI's resume mechanism (`codex resume <id>`, `gemini --resume <id>`, `/resume` in Claude). Optionally writes `HANDOFF.md` for any CLI. Tool calls are not replayed; long sessions bridged in full (a `--summarize` flag was planned); tested against specific CLI versions.

#### 7. session-handoff (yuzushi-dev/session-handoff; fork beverlynngroup/session-handoff), A, V
- Python plugin + MCP server + npm shim, 67 stars, last push 2026-10-07, active (versions 0.7.x, tested with Claude Code 2.1.263 and Codex 0.153.4 per README). Telemetry opt-in only.
- Two modes. (1) Handoff: the agent writes a handoff via `handoff_create`; stored as an immutable record outside the workspace at `handoff://<project-uuid>/<handoff-uuid>`; secrets redacted by `server/redaction.py`; the next session starts clean with the handoff. (2) `migrate codex` / `migrate claude`: `server/migration_engine.py` converts the native session between Claude JSONL (`~/.claude/projects/<slug>/<id>.jsonl`) and Codex rollout (`sessions/YYYY/...`), limits `MAX_RECORDS = 100_000`, `MAX_LINE_BYTES = 16 MiB`, validates tool-call relations, and drops what is not portable (with counts reported).
- Injection: a "managed launcher" (installed by setup, wraps `claude`/`codex` in the shell) opens the fresh or migrated session automatically after the handoff ("Automatic session switching", `server/session_switch.py`, 63 KB). Supervision covers CLI processes, not desktop apps.
- Also: `PreCompact` hook writes a small redacted checkpoint (git state + lifecycle event log) and reinjects only a pointer after compaction. The project publishes a compatibility matrix and treats missing live receipts as `not-run`, never passed.
- Notes for AO: the most mature design for supervising the switch itself (launcher takes over the terminal). Closest analogue to what AO's PTY owner could do natively.

#### 8. dotbabel `/handoff` (kaiohenricunha/dotbabel), A, V
- Node, created 2026-04-14, last push 2026-10-05, 0 stars. Part of a broader skills and governance toolkit.
- `docs/handoff-guide.md`: "moves live session context from one agentic CLI to another: Claude Code, GitHub Copilot CLI, OpenAI Codex CLI". Producer is a binary (`dotbabel handoff`), not the model: it builds a `<handoff>` digest from the raw session JSONL files. Digest caps: user prompts capped at 50 (prompt 1 pinned plus last 49), assistant turn sampling (first turn plus last 3). Secret scrubbing with eight patterns. Optional `--state-file` prepends an agent-written state block.
- Transport: local paste-ready `<handoff>` block, or a user-owned private git repo (`handoff/<project>/<cli>/<YYYY-MM>/<short-uuid>` branches) for cross-machine. `pull latest` renders it into the other CLI; query accepts Claude customTitle or Codex thread_name.

#### 9. session-handover (hahahahahahahahah6/session-handover), A (partial), V (blog) / I (injection)
- 0 stars, last push 2026-10-02. dev.to post (Sep 30, edited Oct 6). Reads `~/.claude/projects` and `~/.codex/sessions`, no LLM, produces a structured markdown (goal, what happened, what broke, what is open, what the next agent should do). v0.3 PreCompact hook writes the handover to `~/.cache/session-handover/precompact/` before the auto-summary replaces context; `audit-compact` reports what a compaction dropped. Post does not say how the document reaches the second agent.

#### 10. claw-orchestrator (Enderfga/claw-orchestrator), A, V
- TypeScript, 588 stars, created 2026-01-30, last push 2026-10-09. Engines: Claude Code, Codex (exec and app-server), Antigravity, Grok Build, Cursor (legacy), OpenCode, custom. All headless (stream-json, `codex exec --json`, `opencode run --format json`), not TUIs.
- `session_handoff` (`manager.handoffSession(name, {engine:'codex'})`) from `skills/references/sessions.md`: "No engine can resume another's session, and each keeps its history in its own undocumented on-disk format. So the conversation is replayed as text: a `<conversation_history>` block in front of the new session's first message". Source is the orchestrator's own per-session record of every message sent through `sendMessage` and every reply (not the session history buffer, which is capped by event count). Each turn is fenced so a reply containing the block's tags cannot close it early. Nothing is written into either engine's session store.
- Inherits cwd and engine-neutral settings (permission and sandbox mode, effort, spend cap, system prompts, extra dirs); does not inherit model, tool allowlists, resume ids, profiles. Hidden reasoning and tool-call detail do not carry; the framing tells the new agent to check the workspace before trusting the history. Old session keeps running.
- Budget: `maxChars` default 240,000 chars (about 60k tokens). Over that: keep the opening request, fill the remainder with the newest turns, and insert one line stating how many middle turns were omitted.
- Notes for AO: the cleanest write-up of "own transcript, replay as text, fenced, with a head-and-tail budget". AO already owns the Chat UI transcript, so this approach needs no parsing of foreign file formats.

#### 11. OpenAI Codex CLI native external-agent import (openai/codex), A, V
- 128,531 stars. `codex-rs/external-agent-migration/` and `codex-rs/app-server/src/external_agent_migration/session_importer.rs`. The `cla.rs` detector was last touched 2026-07-27 ("Parse Claude and Cursor session records separately (#35623)"). The Codex docs page lists "Import from another agent" in its navigation.
- Source: Claude Code `~/.claude/projects/*/*.jsonl` (`detect/sessions/cla.rs`) and Cursor (`cur.rs`, plus connector attribution). Import limits in `model.rs`: sessions modified within 30 days, max 50 sessions. Requires the session cwd to still exist. Custom titles and first user message become the title.
- Included: user and assistant messages only as `ConversationMessage`s; tool calls are serialised as text notes tagged `external_agent_tool_call` / `external_agent_tool_result` with tool results truncated to 4,000 chars (`records_common.rs`). A synthetic `<EXTERNAL SESSION IMPORTED>` marker message is appended; a ledger (`sessions/ledger.rs`) records imported sessions so they can be appended to or de-duplicated by content SHA-256.
- Injection: written as a real Codex thread through `ThreadStore` (rollout items, token estimate via `approx_tokens_from_byte_count`), so the user resumes it with the normal Codex resume UI. Also migrates config, hooks, MCP servers, plugins, memory.
- Notes for AO: shows what a vendor considers the portable subset (text turns plus textified tool calls, bounded tool output, an explicit marker, an import ledger).

#### 12. Sno Station (sno-ai/sno-station), A, V (README)
- 500 stars, created 2026-09-19, last push 2026-10-10. Claude Code, Codex, Hermes, OpenClaw, Cursor (`cursor-agent`).
- Mechanism: local encrypted shared memory plus an agent-to-agent mailbox ("Reach"). README: "One of them hits its cap. Say `sno reach call` from the other one; it picks up the task from the shared memory and the mailbox, context intact." No native session conversion. Task-level continuity, not turn-by-turn transcript.

#### 13. baton (blader/baton), A, V
- 60 stars, 2026-06-14. Skill for Claude Code and Codex. Agent writes `.baton/<date>-<slug>.md` (git-ignored) with fixed sections (Intent, State of Play verified against `git status`, Learnings and Landmines, Pointers, Next Steps, Open Questions). Next agent finds the latest with `ls -t .baton/`. Rule: verify against repo state, not the chat story. Highest-value content is dead ends already ruled out.

#### 14. project-continuity (seriousz158/project-continuity), A, V
- 99 stars, 2026-09-05 to 2026-09-22. CLI + Agent Skill for Codex, Claude Code, Kimi Code, ZCode, OpenCode. One file `.relay/CURRENT.md` (authoritative JSON + generated Markdown view), revision-checked writes, writer lease released by a formal handoff, 64 KiB cap with lossless archiving of old operation receipts. No chat replay, daemon or model call.

#### 15. claude-codex-switch (gitgoready/claude-codex-switch), A, V (README only)
- 0 stars, created around 2026-09, pushed 2026-09-22. Skill + Python CLI, Claude to Codex and back, "switch frameworks mid-task ... because one side's quota ran out". Titles converted sessions `From Claude - <name> - <timestamp>` and inserts them into `~/.codex/state_5.sqlite` so Codex lists them.

### B-class

#### ccswarm (nwiizo/ccswarm), B, V
- Rust, 153 stars, MIT, active (push 2026-10-09). Flow YAML `on_rate_limit: [{provider: codex}]`, per-stage `provider:`, and `promotion: [{at: N, provider, model}]` escalation.
- `crates/ccswarm/src/session/bridge.rs`: `apply_rate_limit_fallback` "Clears any session continuation — a new provider cannot resume another provider's" thread, and `fallback_notice_prompt` prepends: "A previous provider hit a rate limit mid-task. You are taking over fresh — no prior session context is available beyond this prompt." CHANGELOG: "switch clears session continuation, injects a fallback notice, and records ProviderError events." Codex runs via non-interactive `codex exec`; same-thread continuation via `codex exec resume <thread-id>`.
- Takeaway: the one orchestrator with automatic provider failover chooses not to carry the transcript, only the original prompt plus a notice.

#### agent-deck (asheshgoplani/agent-deck), B (account), V
- Go TUI over tmux, 1054 stars, active (push 2026-10-10). `agent-deck session switch-account <session> <account>` moves a Claude session to another Claude account "conversation included": stops the session, migrates the conversation file into the target `CLAUDE_CONFIG_DIR` (copy-only, destination backup, size verification, handles the several project-dir keys Claude may use for one cwd, newest-by-last-event wins), then restarts with `--resume`. `session fork` uses each tool's native fork (Claude, OpenCode, Pi, Codex, Oh My Pi). No Claude-to-Codex path.

#### CCS (kaitranntt/ccs), B, V (README) / I
- 2884 stars, push 2026-10-01. "Claude Codex Switch": launcher/profile manager (`ccs`, `ccs codex`, `ccs --target droid glm`) that points Claude Code, Codex or Droid at different accounts and providers via env and a local Anthropic-compatible proxy (CLIProxyAPI). Switching is at launch. `src/auth/resume-lane-warning.ts` warns when `claude --resume <id>` is run under a different `CLAUDE_CONFIG_DIR` lane and prints the exact command to resume in the right lane. No cross-agent transcript transfer.

#### Pi / pi-ai (earendil-works/pi, formerly badlogic/pi-mono), B, V
- 114,077 stars. `packages/ai/README.md` section "Cross-Provider Handoffs": the unified LLM API switches models mid-conversation; the shared `Context` is retransformed: user and tool-result messages pass through; assistant messages from the same provider are preserved; assistant messages from other providers have thinking blocks converted to `<thinking>` text; tool calls and text preserved. Test `cross-provider-handoff.test.ts` per provider family.
- This is the model-level analogue of what AO would do at harness level.

#### OpenCode, Crush, Goose, Aider, Gemini CLI, Kimi Code CLI, Qwen Code, Happy, CLIProxyAPI, B, V
- OpenCode: `/models` (ctrl+x m) in TUI docs (`packages/web/src/content/docs/tui.mdx`). Crush README: "switch LLMs mid-session while preserving context". Goose docs: "Use the /model command to change models without losing your session". Aider: `/model`, `/editor-model`, `/weak-model` (`aider/website/docs/usage/commands.md`). Gemini CLI: `/model` (`docs/cli/model.md`). Kimi Code: `/model` "Switch the LLM model used in the current session"; also `/import-from-cc-codex` imports Claude Code and Codex instructions, skills and MCP settings (config only, not conversations). Qwen Code README: "Switch at runtime." Happy README: "Switch models in the middle of a task." CLIProxyAPI: description says it wraps several agent backends as compatible APIs, so any harness can be re-pointed at another model (I).
- Common pattern: the harness owns the transcript, so switching the model is a re-send of the same history to another API. This works only because there is one harness.

## Adjacent patterns (not A or B but relevant)

- Same-agent handoff: Gas Town `gt handoff` (`internal/cmd/handoff.go`) collects hooked beads, inbox and ready work into mail, then `tmux respawn-pane -k` starts a fresh agent which runs `gt prime` on SessionStart; PreCompact hooks use `--cycle`/`--auto`. Amp's `handoff` replaces `/compact` with a draft of a new thread (third-party sources only, I). Codex `codex fork`, workmux `--fork`, agent-deck fork, ccmanager session copy are all same-agent.
- Delegation, not continuation: awslabs CAO `handoff` and `assign` MCP tools send a task to another agent's terminal and wait. openai/codex-plugin-cc delegates from Claude Code to Codex. CCB `/ask`.
- Terminal scraping: coder/agentapi (archived) runs the TUI in an in-memory terminal emulator, snapshots the screen before sending input, diffs after, and treats the new text as the agent's message. Nobody used it as a handoff source; its message log is lossy by construction.
- ACP: `session/load` replays history and `session/resume` does not; both are per-agent (agentclientprotocol.com RFD "session-resume-replay", `session/resume` stabilised 2026-04-22). A client switching agents has to supply the history itself.
- Vibe Kanban (sunsetting per README): docs say sessions in a workspace share files but not conversation, and "New sessions don't inherit conversation history", the standard workaround is to start fresh and let the new agent read the diff.

## Ideas worth stealing

1. Pick the context source by agent type, in this order: (a) AO's own recorded transcript (claw-orchestrator style) for sessions created in AO's Chat UI; (b) native session files (cli-continues parsers) for TUI sessions where AO never saw the turns; (c) agent-written handoff doc as a fallback when parsing fails. Nobody scrapes the terminal; do not start there.
2. Replay as a fenced, labelled block in the first message (`<conversation_history>`, per-turn fences so replies cannot spoof role tags) and say "verify against the workspace before trusting". Add a head-and-tail budget: keep the opening request, newest turns fill the rest, one line saying N middle turns omitted (claw-orchestrator, default 240k chars, about 60k tokens).
3. Tool activity as a summary section rather than raw replay: counts and sample commands per tool, files touched, errors (cli-continues), or textified tool calls with a 4,000-char tool-result cap (Codex import). Drop reasoning (all tools do; Codex's is encrypted).
4. Presets for size: minimal 3 / standard 10 / verbose 20 / full 50 messages (cli-continues). A single default (10 messages plus tool summary) with a "send more" option is enough.
5. Dual-delivery for launch: inline prompt argument for normal sizes, fall back to writing a file in the worktree and sending a one-line "read this file" prompt when the argument would be too long or the OS shell is fragile (cli-continues Windows rule). Add `--debug-prompt` to preview exactly what will be sent.
6. Always carry the source session's file path or id in the handoff, so the new agent (or the user) can trace back. Keep the old session intact and runnable (claw-orchestrator, porter `fork`).
7. Show a "Handing off to Codex..." notice and let the user preview/edit the handoff (Amp's editable draft thread; cli-continues `inspect`).
8. Converted-session titles: `From Claude - <original> - <timestamp>` (claude-codex-switch) so the target's history list shows provenance. Mark imported content with an explicit marker message (Codex `<EXTERNAL SESSION IMPORTED>`) and keep an import ledger keyed by content hash to make re-imports idempotent.
9. Redact secrets before any handoff is stored or sent (yuzushi session-handoff `redaction.py`, dotbabel eight patterns).
10. Provider failover should be explicit about what is lost: ccswarm's "you are taking over fresh, no prior context" notice is honest; combine it with a transcript block rather than dropping the history.
11. Keep a compatibility matrix and treat unverified target-version combos as "not-run" (yuzushi). Native session formats are undocumented and break with CLI releases; every converter ties itself to tested versions.
12. Mirror vs on-demand: session-sync and agent-session-bridge keep the other agents' session files in sync continuously (via hooks), so a switch after a crash or rate limit is instant. On-demand conversion (porter, cli-continues, yuzushi) is simpler but needs the source to be readable at switch time. For AO, on-demand from AO's own transcript is enough, with native-file parsing as the fallback.
13. Account-scoped conversation moves (agent-deck `switch-account`): copy-only, destination backup, size verification, never overwrite a newer target copy, and a receipt that lists every candidate considered.

## Gaps and caveats

- Not verified from source: authsec-bridge internals (blog only), session-handover injection path, session-sync writer detail (README only), agent-session-bridge hook details (README only), dotbabel Gemini support in `/handoff` (guide lists Claude, Copilot, Codex), Sno Station internals, claude-codex-switch internals, ruflo, codex-plugin-cc, jean, aoe.
- Codex `/model` mid-session was not confirmed: the fetched developer-commands page (learn.chatgpt.com, 100k of 132k chars read) lists no `/model` command, only a startup model flag. Droid and Amp official docs did not confirm mid-session model switching or Amp's handoff. oh-my-pi roles were not verified.
- Not investigated: zellij plugins, tmux-agents (nodeterm, smux, multi-agent-shogun), crew/CrewCLI (no relevant repo found), agent-farm (Dicklesworthstone/claude_code_agent_farm), agent-os, happy-cli (merged into slopus/happy), omnara internals, sidecar (no specific tool found), mnemo (Pilan-AI repo not found; cli-continues credits its Go adapters).
- GitHub search with multi-word queries often returned empty results (rate limit or AND semantics), so recall for tiny converters is incomplete. More A-class converters almost certainly exist (for example OpenMOSS-style protocols, hist-bridge, harnesshop, connectwithprakash/agent-session-bridge, pi-cross-agent-resumer, session-handover-style skills, all 0 to a few stars).
- Several repos moved or were archived: Pi is now earendil-works/pi; ruvnet/claude-flow became ruflo; steveyegge/gastown became gastownhall/gastown; coder/agentapi and MoonshotAI/kimi-cli are archived; Vibe Kanban is sunsetting; sst/opencode is now anomalyco/opencode; Goose is aaif-goose/goose.
- Stars and push dates are from 2026-10-10/11 API calls. Several A-class repos are very young (created 2026-04 to 2026-09), so none has long-term maintenance evidence except Codex's first-party import.
