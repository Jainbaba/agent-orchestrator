# First-party session import / handoff mechanisms (researched 2026-10-11)

Legend: VERIFIED = seen in source or official docs (path/URL given). INFERRED = my deduction, not seen directly.
Repo paths are relative to the repo root on `main`/HEAD as of 2026-10-10. Crate and file names below are the real ones (the `cla`/`cur` abbreviations are Codex's own file naming).

---

## Target 1: OpenAI Codex `external-agent-migration` session importer, and `thread/inject_items`

### 1a. Overall verdict on the reported claims
Mostly right, with these corrections:
- VERIFIED: it exists, reads Claude Code and Cursor transcripts, flattens tools into tagged prose, drops thinking/sidechain/meta, adds a marker, creates a real persisted thread, keeps a content-hash ledger, and refuses append once compaction exists.
- CORRECTION: the marker is NOT in the model-visible history. It is only a UI event (`EventMsg::AgentMessage`), never a `ResponseItem`, and the append planner strips it.
- CORRECTION: the thread is NOT created through `thread/start`. It is built directly in the thread store (`CreateThreadParams` + `append_items`) inside the app-server process.
- CORRECTION: the import response does not return a thread id. Callers must read it from the ledger file or from the completed-notification.
- CORRECTION: the app-server will not import an arbitrary path. The path must be a `.jsonl` under `<HOME>/.claude/projects` (or `<HOME>/.cursor/projects` for Cursor mode).

### 1b. Code locations (all VERIFIED, `openai/codex`)
- App-server glue: `codex-rs/app-server/src/external_agent_migration/{processor.rs,session_importer.rs,protocol.rs}`
- Conversion + ledger crate: `codex-rs/external-agent-migration/src/sessions/{mod.rs,export.rs,records_common.rs,records_cla.rs,records_cur.rs,ledger.rs,append.rs}`
- Discovery: `codex-rs/external-agent-migration/src/detect/sessions/{cla.rs,cur.rs,common.rs}`, limits in `src/model.rs` lines 5-24
- Source selector: `src/migration_source.rs` (lines ~51-75), home resolution `src/service.rs` ~798-803
- TUI: `codex-rs/tui/src/external_agent_config_migration/flow.rs` (detect loop ~255-300), `codex-rs/tui/src/slash_command.rs` (`SlashCommand::Import`)
- Protocol methods: `codex-rs/app-server-protocol/src/protocol/common.rs` (~1436-1455 and ~889)
- History: Cursor source added by PR #33426 (2026-07-15); append-to-existing-import added by #36356 (2026-07-31). Both are commits on `session_importer.rs`/`append.rs`.

### 1c. Where it reads from
- Claude Code: `<HOME>/.claude/projects/<any-dir>/*.jsonl` (`detect/sessions/cla.rs` ~26-58). `HOME` or `USERPROFILE` of the app-server process; no `CLAUDE_CONFIG_DIR` support here (`service.rs` `default_external_agent_home`).
- Cursor: `<HOME>/.cursor/projects/<project>/agent-transcripts/**` (`detect/sessions/cur.rs` ~37-57). Selected by `migrationSource: "cursor"` (case-insensitive); anything else defaults to Claude.
- Detection limits (VERIFIED `model.rs`): sessions modified in last 30 days, max 50 sessions; request params `maxSessionAgeDays` and `maxSessions` can override.
- Claude sessions must have a `cwd` in the records and that directory must still exist (`require_existing_cwd`). Cursor mode falls back to the `cwd` passed in `details.sessions[].cwd` when records lack one, and to file mtime when records lack timestamps (`SessionMetadataMode::MigrationFallback`).

### 1d. What is kept and dropped
Claude record parsing (`records_cla.rs`, `conversation_message_from_owned_record`, ~150-200):
- Kept: records with `type` of `user` or `assistant` that have `message.content`. Plain strings and content blocks.
- Dropped: `isMeta: true`, `isSidechain: true` records (subagent transcripts); every other record type (summary, system, attachments, file-history snapshots, etc.); `thinking` blocks (`records_common.rs`: `Some("thinking") => {}`); empty text.
- Custom/AI titles are read from `custom-title` / `ai-title` records; fallback title is the first user message (title max 120 chars).
- `attributionMcpServer` fields are read for connector attribution only.
- A user record containing only `tool_result` blocks is re-labelled as role=assistant (`only_tool_result`). Cursor `<user_query>` wrappers are unwrapped.

Cursor parsing (`records_cur.rs`): same shape, keyed on `role` of `user`/`assistant`, `timestamp_ms` supported.

Flattening (`records_common.rs`, VERIFIED):
- `tool_use` becomes text: `[external_agent_tool_call: NAME]` then `description:`, `command:`, `file:` lines (if present), else `input: <json truncated to 2,000 chars>`, then `[/external_agent_tool_call]`.
- `tool_result` becomes `[external_agent_tool_result]` (or `[external_agent_tool_result: error]`) + text truncated to 4,000 chars + closing tag. Only text parts of result arrays are kept (images dropped).
- Unknown block types become `[external unsupported block: TYPE]`.
- Constants: `NOTE_MAX_LEN = 2_000`, `TOOL_RESULT_MAX_LEN = 4_000`. Plain user/assistant text is NOT truncated. There is no total size cap in the importer.

### 1e. How it becomes a Codex thread (`export.rs` `rollout_items_from_messages`, ~85-150)
- Each user message starts a synthetic turn `external-import-turn-N` (TurnStarted, UserMessage event, plus a model-visible `ResponseItem::Message role=user InputText`). Assistant text becomes `AgentMessage` event + `ResponseItem::Message role=assistant OutputText`. Assistant text before the first user message is skipped.
- So the imported history is model-visible plain messages only (no function_call items), with tool activity embedded as tagged text.
- A token-count record is added using an approximate bytes-to-tokens estimate.
- Marker: `EXTERNAL_SESSION_IMPORTED_MARKER = "<EXTERNAL SESSION IMPORTED>"`, appended as a final `EventMsg::AgentMessage` in the last turn. UI-visible only; not a `ResponseItem`, and `append.rs` `is_import_marker` filters it out of appended suffixes.
- Thread creation (`session_importer.rs` `persist_session`, ~290-330): loads config for the session `cwd`, picks the default model, creates `ThreadId::new()`, writes through `ThreadStore::create_thread`/`append_items`. Base instructions come from current config/model.

### 1f. Ledger / dedupe (`ledger.rs`, `mod.rs`)
- File: `$CODEX_HOME/external_agent_session_imports.json` (const `SESSION_IMPORT_LEDGER_FILE`). Record fields: `source_path` (canonicalized), `content_sha256` (SHA-256 of the raw source file bytes), `imported_thread_id`, `imported_at`, `source_modified_at`, `connector_names`, `title`.
- `prepare_validated_session_import_with_metadata_mode` (`sessions/mod.rs`):
  - no record: create new thread.
  - record with same hash: skip silently (no thread created, no error).
  - record with different hash and exactly one record for the path: attempt incremental append to the same thread.
  - more than one record for the path (ambiguous): skip.
- Detection also skips sources whose mtime matches the ledger (`detect/sessions/common.rs`).
- Concurrency: an import semaphore of 1 per request batch and 5 sessions in flight (`SESSION_IMPORT_CONCURRENCY`).

### 1g. Append and compaction refusal (`append.rs`, VERIFIED)
- Append is allowed only if the existing Codex thread is cold (not currently loaded in the ThreadManager), not archived, readable, and its complete model-visible transcript is an exact prefix of the new source transcript. Only the missing suffix is appended, then the ledger hash is checkpointed.
- It fails closed (returns false, ledger unchanged) when either side contains `RolloutItem::Compacted`, `EventMsg::ContextCompacted`, `EventMsg::ThreadRolledBack`, `TurnContext`, `InterAgentCommunication` or `WorldState` (`source_model_items`, `history_model_items`, ~230-280). So the reported "refuses incremental append once a compaction exists" is CONFIRMED, and it also refuses after any divergence or after the thread has been used (a new Codex turn adds `TurnContext`).
- INFERRED: practical consequence is that once a user has continued the imported thread in Codex, later changes in the Claude source can never be merged. Re-import is not possible either (hash changed, thread exists, append rejected), so nothing happens.

### 1h. How the user invokes it
- TUI: slash command `/import` ("import setup, this project, and recent chats from Claude Code", `slash_command.rs`). The flow calls `externalAgentConfig/detect` once per source (Claude, Cursor) and shows a checklist including "Chat sessions (N), Last 30 days of chats" (snapshot `..._external_agent_config_migration_prompt.snap`). Also shown as a setup prompt; `flow.rs` tells the user to "run /import again" for leftovers.
- No `codex import` CLI subcommand found in `codex-rs/cli` (searched tree; only `migrate_rollouts.rs`, which is unrelated). Codex App has the same importer (stated in codex-plugin-cc README).
- Bridge: `/codex:transfer` in Claude Code (Target 2).

### 1i. App-server methods (all VERIFIED, `common.rs` and generated TS schema in `codex-rs/app-server-protocol/schema/typescript/v2/`)
- `externalAgentConfig/detect` params: `includeHome?, cwds?, maxSessionAgeDays?, maxSessions?, migrationSource?, source?(deprecated, ignored)`.
- `externalAgentConfig/import` params: `{ migrationItems: ExternalAgentConfigMigrationItem[], source?, providerId?, migrationSource? }`. Response: `{ importId }` only.
- Migration item for sessions: `{ itemType: "SESSIONS", description, cwd: null, details: { sessions: [{ path, cwd, title }], plugins:[], mcpServers:[], hooks:[], subagents:[], commands:[] } }`.
- Notifications: `externalAgentConfig/import/progress`, `externalAgentConfig/import/completed` (`{ importId, itemTypeResults[{ itemType, successes[{cwd, source, target, title}], failures[] }] }`). In the code, `target` is set to `imported_thread_id` for sessions (`record_success_with_cwd`, session_importer.rs ~131-136). INFERRED that `target` therefore carries the thread id, because codex-plugin-cc instead re-reads the ledger.
- Also `externalAgentConfig/import/recordHistory` and `/readHistories`.
- Not marked `#[experimental]` in `common.rs`. Needs a Codex build from mid-July 2026 or later for Cursor, later still for append.
- Path validation (`service.rs` `external_agent_session_source_path`, lines 175-190): extension must be `jsonl`, canonical path must start with `<external_agent_home>/projects`. Otherwise error `session_not_detected`. The import does not require a prior `detect` call (validation is path-only), VERIFIED by reading `processor.rs` `validate_pending_session_imports` ~448-512.

### 1j. Can a third-party host drive it?
- VERIFIED yes in practice: OpenAI's own codex-plugin-cc does exactly this from a Node process: spawns `codex app-server`, sends `externalAgentConfig/import` with one SESSIONS item, waits for `externalAgentConfig/import/completed` (2-minute timeout), then reads the thread id from `external_agent_session_imports.json` (`plugins/codex/scripts/lib/codex.mjs`, `importExternalAgentSession` ~1058-1093; `requestExternalAgentSessionImport` ~700-730; `importedThreadIdForSource` ~660-675).
- Limit: it can only import JSONL files laid out like Claude Code (or Cursor) transcripts under `~/.claude/projects` (`~/.cursor/projects` for Cursor mode), relative to the HOME env var of the app-server process.
- INFERRED recipe for AO to seed Codex from ANY source: write a synthetic Claude-format JSONL (`{"type":"user"|"assistant","cwd":<existing dir>,"timestamp":<RFC3339>,"message":{"content":[...]}}`, one per line) into a staging dir `<STAGE>/.claude/projects/ao/<id>.jsonl`, spawn `codex app-server` with `HOME=<STAGE>` (and `CODEX_HOME` set to the real one), call `externalAgentConfig/import`, read `target` from the completed notification (or the ledger), then `thread/resume`. Caveats: only text is model-visible (tool activity must be pre-flattened by AO or will be flattened with Codex's tags and caps), the `<EXTERNAL SESSION IMPORTED>` UI marker appears, the thread shows as an ordinary Codex thread with synthetic turn ids, and HOME override may affect other Codex lookups. Not tested.

### 1k. `thread/inject_items` (VERIFIED)
- Exact name: `thread/inject_items` (`ThreadInjectItems => "thread/inject_items"`, `common.rs` ~889). Doc comment: "Append raw Responses API items to the thread history without starting a user turn."
- Params: `{ threadId: string, items: Array<JsonValue> }`; "Raw Responses API items to append to the thread's model-visible history." Response: `{}` (empty).
- Introduced in PR #17703 "Add turn item injection API", commit `d4be06ade`, 2026-04-13. First stable release containing it: `rust-v0.121.0` (published 2026-04-15; compare of d4be06ade...rust-v0.121.0 is "ahead"; `0.121.0-alpha.8` is also ahead, `-alpha.4` is diverged). Note the PR text calls it `turn/inject_items` and mentions a typo alias `turn/injet_items`, but the code in that commit (and now) registers `thread/inject_items`. I did not find the alias in current `common.rs`.
- Behaviour (`app-server/src/request_processors/turn_processor.rs` ~994-1022; `core/src/codex_thread.rs` ~822-870): requires an already-loaded thread (`load_thread`) and `ensure_direct_input_allowed`; each item is parsed as `ResponseItem` (error `items[i] is not a valid response item`), image URLs validated, empty list rejected ("items must not be empty"). Records items, then checkpoints/flushes the rollout. No turn is started and no model call is made (test asserts `response_mock.requests().is_empty()`).
- Items can be any `ResponseItem` shape: messages with role user/assistant/developer, `function_call_output`, etc. (test `app-server/tests/suite/v2/thread_inject_items.rs`). This is the only first-party path found that accepts structured tool calls/results.
- Side effect: the test asserts the thread memory mode becomes `"polluted"` when `[memories] disable_on_external_context = true` is configured; INFERRED that injected context excludes the thread from memory generation under that setting.
- No numeric size or count limit found in the handler. Limits would be the model context window and whatever `ResponseItem` serde accepts. INFERRED: unvalidated large injections can overflow context and trigger Codex auto-compaction.
- Third-party use: T3 Code references it in `apps/server/src/orchestration-v2/Adapters/CodexAdapterV2.ts` (+ tests `CodexAdapterV2.test.ts`, `ThreadMergeBack.integration.test.ts`) per GitHub code search (file match on the string; I did not read those files). Codex's own TUI uses it (`tui/src/app_server_session.rs` `thread_inject_items`, "TUI side conversation setup"). openclaw also uses it.
- Visibility: INFERRED that injected raw items are not turned into UI turns, so a thread seeded this way may look empty in a Codex UI while still carrying model-visible context. Not verified.

---

## Target 2: OpenAI `codex-plugin-cc` and `/codex:transfer`

VERIFIED sources: `openai/codex-plugin-cc` (pushed 2026-07-08): `plugins/codex/commands/transfer.md`, `plugins/codex/scripts/codex-companion.mjs` (`executeTransfer` ~625-640, `handleTransfer` ~825-838), `plugins/codex/scripts/lib/claude-session-transfer.mjs`, `plugins/codex/scripts/lib/codex.mjs`, `README.md` lines ~165-178. Introduced by commit `430e2a863` "feat: add Claude session transfer command (#374)", 2026-06-23.

- Direction: Claude Code to Codex only.
- What: "Transfer the current Claude Code session into a resumable Codex thread." It does not translate anything itself. It calls Codex's own importer (Target 1) via `externalAgentConfig/import`, so conversion rules, truncations, marker, ledger and dedupe are exactly Target 1's. README: "follows the same conversion rules as importing Claude history in the Codex App and creates visible turns".
- Source: path comes from the `SessionStart` hook (`session-lifecycle-hook.mjs` line 79 exports `CODEX_COMPANION_TRANSCRIPT_PATH` from the hook input's `transcript_path`) or `--source <jsonl>`. Must be `.jsonl` under `~/.claude/projects` (realpath check).
- Output: prints `Codex session ID: <id>` and `codex resume <id>`. Thread id is recovered by reading `$CODEX_HOME/external_agent_session_imports.json` and matching source path + current file SHA-256 (not from the RPC response).
- Dedupe: handled entirely by Codex's ledger (same hash: no new thread, and then the plugin returns the existing id because it matches path+hash; changed hash: Codex tries in-place append, else nothing and the plugin errors "did not record an imported thread" only if no record matches current hash).
- INFERRED edge: after a changed-and-rejected append the plugin looks for a ledger record whose hash equals the current file hash, finds none, and reports an error.
- Trigger: slash command `/codex:transfer [--source <path>]`, `disable-model-invocation: true` (user-only). Needs a Codex CLI new enough to expose `externalAgentConfig/import`; the plugin maps JSON-RPC `-32601` to "update Codex".
- Programmatic: yes, same RPC any host can call (see Target 1j). Companion CLI form: `node scripts/codex-companion.mjs transfer [--source <jsonl>] [--json]`.

---

## Target 3: xAI `grok import` and `xai-org/grok-build-plugin-cc`

VERIFIED sources: `xai-org/grok-build-plugin-cc` (pushed 2026-08-04) `plugins/grok-build/commands/import.md`, `scripts/lib/grok.mjs` `runImport` ~279-345, `scripts/lib/claude-session-transfer.mjs`, `README.md` ~116-125; official docs `https://docs.x.ai/build/cli/reference` (table row `grok import [targets...]`, "Import sessions from Claude Code"); open source TUI repo `xai-org/grok-build`.

- Official CLI: `grok import [targets...]` described only as importing sessions from Claude Code. Flags `--list` and `--json` appear in the plugin's invocation (`grok import --list`, `grok import <path> --json`). No other flags documented in the reference page. The resulting session is resumed with `grok -r <session-id>` (docs: `-r, --resume [<ID>]`).
- Plugin `/grok-build:import [--source <claude-jsonl>]`: same path rule as codex-plugin-cc (`.jsonl` under `~/.claude/projects`, env `GROK_CC_TRANSCRIPT_PATH` from the SessionStart hook). It shells out to `grok import <abs-path> --json`, parses JSON lines for `sessionId | session_id | id | importedSessionId | threadId`, else regexes a UUID from stdout. Prints "Grok session ID" and `grok -r <id>`.
- Other agents: the CLI reference says Claude Code only. Separately, the Grok shell changelog (`crates/codegen/xai-grok-shell/CHANGELOG.md`) says 0.2.96 (July 10 2026) added a welcome-screen ctrl+u one-click resume for a recent Claude Code session, and 0.2.100 (2026-07-13) "Session picker now discovers and resumes recent Claude Code, Codex, and Cursor sessions". Source for that listing is `crates/codegen/xai-grok-foreign-sessions` (metadata-only; limits `MAX_SESSIONS_PER_TOOL = 50`, `MAX_SESSION_AGE = 30 days`, `MAX_TITLE_CHARS = 200`; reads `$CLAUDE_CONFIG_DIR` or `~/.claude`, `$CODEX_HOME` or `~/.codex` incl. its sqlite state DB / rollout files). In that public tree the Cursor scanner is stubbed to return nothing (`lib.rs`: `let scan_cursor = |_,_| Vec::new()`), so Cursor listing is not implemented in the published source despite the changelog line (discrepancy; INFERRED that published source is a trimmed snapshot).
- What is kept/dropped, size limits, marking, dedupe: NOT VERIFIED. The conversion code behind `grok import` and the picker's "resume foreign session" is not in the public `xai-org/grok-build` tree (searched; `xai-grok-pager-bin/src/main.rs` has no import subcommand; only config import `/import-claude`, which imports settings/permissions/MCP/hooks and not conversations). Third-party guide (continuumcode.ai, unofficial) says conversation is imported as handoff context, no repo files/permissions/process state, and that tool-call fidelity, size limits and dedupe are undocumented. Treat as UNVERIFIED.
- Trigger: CLI `grok import`, TUI welcome-screen ctrl+u / session picker, Claude Code `/grok-build:import`.
- Programmatic host use: `grok import <jsonl> --json` is a plain CLI that a host can spawn; output JSON includes the new session id. Also `grok agent stdio` runs Grok as an ACP agent (docs). INFERRED: no documented API to import non-Claude sources via CLI; the host would have to emit Claude-format JSONL (same trick as Codex).
- Mirrors codex-plugin-cc: the plugin README says it mirrors the OpenAI one (another community plugin, `TolyK/grok-plugin-cc`, does too).

---

## Target 4: Anthropic (Claude Code / Claude Agent SDK)

Conclusion: NO import of another agent's session exists in any official doc I found. There are, however, documented ways to resume a transcript the host supplies. The transcript entry format is declared internal and unstable.

VERIFIED from official docs:
- `https://code.claude.com/docs/en/sessions` (local copy fetched): "`claude --resume <transcript-path>` resumes the session stored in the `.jsonl` transcript file at that absolute path". Transcripts live at `~/.claude/projects/<project>/<session-id>.jsonl` where `<project>` is the cwd with non-alphanumerics replaced by `-` (truncated plus hash at 200 chars). Quote: "The entry format is internal to Claude Code and changes between versions, so scripts that parse these files directly can break on any release." Features documented: `--continue`, `--resume`, `--from-pr`, `/export` (plain text), branching. No import command. The docs index (`https://code.claude.com/docs/llms.txt`, 566 lines) has no import/migrate page for foreign sessions.
- Agent SDK sessions `https://code.claude.com/docs/en/agent-sdk/sessions`: `continue`, `resume`, `forkSession`/`fork_session`; options `resumeSessionAt`, `sessionId` (use a specific UUID), `persistSession`; functions `listSessions`, `getSessionMessages`, `getSessionInfo`, `forkSession({upToMessageId})`. Resume is "Same machine only" unless a SessionStore is used. Doc says "Capture the results you need ... pass them into a fresh session's prompt" as an alternative to shipping transcripts.
- SDK `SessionStore` adapter `https://code.claude.com/docs/en/agent-sdk/session-storage`: interface `append(key, entries)`, `load(key) -> entries|null`, optional `listSessions`, `listSessionSummaries`, `delete`, `listSubkeys`. `SessionKey = {projectKey, sessionId, subpath?}`. Entries are "`{ type: string; ... }` ... opaque JSON-safe values". When `resume` + `sessionStore` are passed, "the SDK asks the store for a transcript before it spawns the subprocess ... writes it into a temporary config directory" and runs Claude Code against it. Dedupe by `entry.uuid` is the adapter's job. Requires `resume` from a cwd matching the original `projectKey`. TS: `sessionStore` incompatible with `persistSession:false` and file checkpointing.
- Streaming input: `query({prompt: AsyncIterable<SDKUserMessage>})` (TS reference). SDKUserMessage is a user message (with optional `uuid`, which `forkSession.upToMessageId` can reference). I found no documented way to stream assistant-role or tool-result history into a new session (the page was only partially read; reference text beyond 100k chars unread). Mark INFERRED: only user turns can be fed, so synthetic prior assistant turns would need to go through the transcript/SessionStore route.
- ACP (`https://agentclientprotocol.com/protocol/session-setup`): `session/new` (cwd, mcpServers), `session/load` (agent MUST replay the whole conversation via `session/update`; client gives only `sessionId`; requires `agentCapabilities.loadSession`), `session/resume` (no replay; `sessionCapabilities.resume`). No client-supplied history anywhere. The Claude ACP adapter is `agentclientprotocol/claude-agent-acp` (tree listed; I did not read it) and delegates to Claude's own transcripts.
- INFERRED recipe for AO: write a Claude-format JSONL (or implement a SessionStore whose `load()` returns synthetic entries) with a chosen `sessionId`, then `claude --resume <abs-path>` or SDK `resume: <id>` with matching cwd. Works today in practice because Codex's importer and Grok's importer read this same on-disk format, but Anthropic declares the format unstable. Needs fields like `type`, `uuid`, `parentUuid`, `sessionId`, `cwd`, `version`, `message{role,content}`, which I did not verify against Claude Code source (closed). Treat as brittle and version-pin.

---

## Target 5: VS Code + GitHub Copilot

### 5a. Changing session target / harness on a live chat ("handoff")
Official doc VERIFIED: `https://code.visualstudio.com/docs/agents/run/agent-harnesses`: "You can change the harness of an ongoing session with the Session Target dropdown. ... Handoff continues ongoing work with a different agent configuration and carries the conversation history and context with it." Destinations Copilot, Claude, Codex, Cloud. Only Local sessions show the dropdown (agent host sessions don't). Tools, permissions and models might change; hooks follow the chosen harness. Fork creates an independent session. The tutorial page (`/docs/agents/agents-handoff-tutorial`) repeats "carries the conversation history and context" without saying whether it is a summary. Short link in source: `https://aka.ms/vscode-agent-handoff`.

Source VERIFIED (`microsoft/vscode`), `src/vs/workbench/contrib/chat/browser/actions/chatContinueInAction.ts`:
- `buildDelegationTranscript(requests, maxLength = MAX_DELEGATION_TRANSCRIPT_LENGTH = 20_000)` (~lines 470-485): plain text `User: <text>\nAssistant: <markdown>` per turn joined by blank lines; if longer than 20,000 chars it keeps the LAST 20,000 characters (`substring(len - maxLength)`). Only the visible user text and the response markdown: no tool calls/results, no thinking.
- `createDelegationTranscriptAttachment` (~490-505): wraps it as a paste attachment named "Previous conversation" with text "The following is the conversation history from a previous {sourceName} session. Continue working on it.\n\n{transcript}".
- Cross-type delegation (~lines 680-790): opens a NEW session of the target type (the source local session stays in history; the source editor is replaced in place so it feels like the same chat). For agent-host targets (Claude, Codex, Copilot CLI host) the transcript is attached as context; for other targets (Cloud coding agent) it is inlined in the prompt: "...\n\nUser: {prompt}".
- Special path for Copilot agent host (main window): `importedTurnsFromChatModel(chatModel)` (`.../agentSessions/agentHost/importLocalConversationToAgentSession.ts`) seeds the new session with structured "real, editable turns" including tool-call response parts (tool name, input JSON, text output, error flag, sub-agent summary), passed as `importConversation: { turns, model }`, and keeps the selected model. This structured import is used only when target is `SessionType.AgentHostCopilot` (source comment lines ~705-711). For Claude and Codex the text transcript attachment is used (INFERRED from the `importConversationTurns` condition; I did not find a Claude/Codex consumer of `importConversation`).
- Repository: repo `owner/repo` is extracted from the source session and passed as session option.
- Dedupe: none (each handoff opens a new session). No imported marker beyond the "Previous conversation" attachment name.
- `agentHost` housekeeping: `src/vs/platform/agentHost/common/agentHostConversationContext.ts` `buildConversationContext(turns, {maxChars, framing})` is a separate text builder: "User request:\n...\n\nAgent response:\n..." joined by `---`, tools/reasoning ignored, middle-truncated with `...` (`truncateMiddle`), used for utility-model prompts (title etc.), not the handoff. Capability meta key `vscode.importSession` (`agentHostSessionImportMeta.ts`) shows the agent host protocol has an "import external session" capability (details not read).

### 5b. Delegation to Copilot CLI / cloud uses an LLM summary (older extension path)
VERIFIED `microsoft/vscode-copilot-chat` (also mirrored under `extensions/copilot/` in microsoft/vscode):
- `src/extension/chatSessions/copilotcli/common/delegationSummaryService.ts` (106 lines): `summarize(context, token)` delegates to `ChatSummarizer.provideChatSummary`; `trackSummaryUsage` stores the summary in globalState keyed by session id (first 100 chars only, entries pruned after 7 days) and returns a `Delegation Summary` prompt reference (summaries under 100 chars are shown inline); `extractPrompt` rebuilds the prompt as "...Complete the task as described in the [summary](copilot-delegated-chat-summary:...)".
- Summarizer: `src/extension/prompt/node/summarizer.ts` `ChatSummarizerProvider.provideChatSummary`: returns '' if no successful turn; uses the `copilot-fast` endpoint; renders `ConversationHistorySummarizationPrompt` with `maxToolResultLength: 2000`, `maxSummaryTokens: 7_000`, `triggerSummarize: false`; strips surrounding quotes.
- Prompt: `src/extension/prompts/node/agent/summarizedConversationHistory.tsx` `SummaryPrompt` (lines ~50-150): "create a comprehensive, detailed summary of the entire conversation that captures all essential information needed to seamlessly continue the work without any loss of context", with an `<analysis>` block (chronological review, intent mapping, technical inventory, code archaeology, progress, recent commands) and a `<summary>` block (Conversation Overview, Technical Foundation, Codebase Status, ...). Final user message asks to focus on the most recent agent commands and tool results.
- Use sites: `copilotCLIChatSessionsContribution.ts` `handleDelegationFromAnotherChat` (~1801-1870): progress "Analyzing chat history"; the prompt becomes `"{userPrompt}\n**Summary**\n{summary}"`; opens a new CLI session via `workbench.action.chat.openSessionWithPrompt.copilotcli`. `copilotCloudSessionsProvider.ts` `delegate` (~1528-1600): the cloud agent gets the summary appended to the PR context ("This chat's history will be summarized and appended to the pull request as context", BASE_MESSAGE ~291). Skipped when there is no non-empty response in history (`hasHistoryToSummarize`).
- So: older/extension handoff = LLM summary as prompt text; newer agent-host handoff = raw-ish transcript tail (20k chars) or structured turns for Copilot host.

### 5c. Custom-agent `handoffs:` front-matter (VERIFIED, `https://code.visualstudio.com/docs/copilot/customization/custom-agents`)
Each item: `label` (button text), `agent` (agent identifier to switch to), `prompt` (text pre-filled for the target), `send` (optional boolean, default false: auto-submit), `model` (optional, `Model Name (vendor)`). Buttons appear as suggestions after a response completes. Clicking switches to the target agent "with the conversation context and a pre-filled prompt". Docs do not say whether history is full or summarized, nor whether it is the same chat session; the wording and the source `chatSuggestNextWidget.ts` / `chatExecuteActions.ts` (file names from search; not read) suggest same conversation. Intra-chat agent-mode switch, distinct from the harness/session-type handoff above. Third-party host cannot drive it: it is a VS Code UI feature.

---

## Target 6: Cursor

- Mid-chat model change (VERIFIED, `https://cursor.com/blog/continually-improving-agent-harness`, published 2026-04-30). Key points: Cursor swaps in the new model's own harness (prompts and tools). History from another model is "out of distribution from what it was trained on". Quote: "we add custom instructions that tell the model when it's taking over mid-chat from another model", and further instructions steer it away from calling tools that appear in the history but not in its own toolset. Caches are provider/model specific, "so switching means a cache miss and a slower, more expensive first turn". They tested summarizing at switch time, but "if the user is deep into a complex task, the summary can lose important details". Alternative: use a subagent (fresh context, can pin a model). Recommendation: stay on one model per conversation unless there is a reason. The exact text of the takeover instruction is not published in the post. So Cursor keeps the full native history and adds a preamble, rather than summarizing.
- Local to cloud and back (docs/changelog, VERIFIED at the level of the pages read): Cursor CLI 2026-01-16 `&` prefix: "Push your local conversation to a Cloud Agent and let it keep running while you're away", resumable at cursor.com/agents (`https://cursor.com/changelog/cli-jan-16-2026`). Cursor 3 (April 2026, `https://cursor.com/blog/cursor-3`) and Agents Window docs (`https://cursor.com/docs/agent/agents-window`) describe moving agents between local and cloud in both directions; changelog 3.7 (2026-06-17, per search snippet) improved reliability. The Cursor CLI changelog says cloud transfers preserve the selected model and workspace path. What exactly is serialized beyond that is not documented. A forum report says cloud-origin agents lack a "Move to Local" option and moving from cloud-origin sessions loses chat history (unofficial, not verified).
- Programmatic use: none documented for conversation handoff. Cursor is closed source. Its local transcripts (`~/.cursor/projects/<project>/agent-transcripts/`) are read by Codex's importer (Target 1) and listed by Grok.

---

## Target 7: Gemini CLI, OpenCode, Pi

- Gemini CLI (`google-gemini/gemini-cli`): no first-party import of other agents' sessions found. Docs `docs/cli/session-management.md` cover only its own sessions (`~/.gemini/tmp/<project_hash>/chats/`, `--resume`, `/resume`, `/chat`); `memoryImportProcessor.ts` is for `@file` includes in GEMINI.md. ACP mode supports `loadSession` (`docs/cli/acp-mode.md`) but with no client-supplied history. VERIFIED by tree and doc search.
- OpenCode (`anomalyco/opencode`): no import of other agents' sessions, but it has a structured native import usable by a host: `opencode session import <file|url>` (`packages/cli/src/commands/handlers/session/import.ts`) posts `SessionTransfer.Data = { info: Session.Info, messages: SessionMessage.Info[] }` (`packages/schema/src/session-transfer.ts`) to `POST /api/experimental/session/import` on the OpenCode server, with `location.directory`; HTTP 409 means "Session already exists" (dedupe by session id). Experimental path. Server contract in `packages/protocol/src/groups/session.ts`. This allows seeding OpenCode with full structured messages (incl. parts/tool calls per `SessionMessage` schema, which I did not read).
- Pi (`earendil-works/pi`, formerly badlogic/pi-mono): no first-party import of other agents' sessions found in README, `docs/cli.md`, or tree. It has `--session <path|id>`, `--fork`, `--session-id`, `--export` (to HTML), and RPC `new_session`, `switch_session {sessionPath}`, `fork` (`packages/coding-agent/docs/rpc-commands.md`). A community package `pi-opencode-session-import` exists on pi.dev (search result; not first-party).

---

## What a third-party host like AO can use

Goal: seed a target agent with real history (not a text prompt).

1. Codex (best option; first-party and proven by OpenAI's own plugin)
   - `externalAgentConfig/import` on `codex app-server` (Target 1i/1j). AO emits a Claude-format (or Cursor-format) JSONL for the normalized session, places it under `<HOME>/.claude/projects/...` (staging HOME for the spawned app-server, or the real one), imports, learns the thread id from the completed notification `target`/ledger, then `thread/resume`.
   - Caveats: text-only model-visible history, tool activity flattened with `[external_agent_tool_call]` tags and 2,000/4,000-char caps (AO may pre-flatten with its own rules); UI shows `<EXTERNAL SESSION IMPORTED>`; ledger dedupes by source path + SHA-256, and append only works while the Codex thread is cold and uncompacted/unused; `cwd` must exist; HOME override is untested; the `.jsonl` path restriction is enforced server-side.
   - Alternative: `thread/start` then `thread/inject_items` with real `ResponseItem`s (Target 1k). Keeps structured user/assistant messages and function_call/function_call_output pairs, no tagged flattening, no marker, no ledger. Caveats: raw item schema must match the Codex build; thread must be loaded; may mark memories "polluted"; unknown whether UI shows them; no size guard (may trigger compaction); available since rust-v0.121.0 (2026-04-15).

2. Claude Code (no import API; resume of host-supplied transcript is possible but format unstable)
   - Write a Claude JSONL with a chosen session UUID in `~/.claude/projects/<encoded-cwd>/` and run `claude --resume <abs path>` / SDK `resume` + `sessionId`, or implement an SDK `SessionStore` whose `load()` returns the synthetic entries (`resume` with `sessionStore`, cwd must match `projectKey`). Needs Claude's internal entry shape (`uuid`, `parentUuid`, `type`, `message`, `cwd`, `sessionId`, ...), which Anthropic says is unstable; pin Claude Code version and test per release. SDK streaming input accepts user messages only (INFERRED); ACP `session/load` cannot take client history.

3. xAI Grok Build
   - Spawn `grok import <claude-format.jsonl> --json` (what grok-build-plugin-cc does) and then `grok -r <id>`. Source format is Claude's; fidelity, limits and dedupe are undocumented/unverified, and conversion code is not in the public repo. Treat as a text-ish handoff.

4. OpenCode
   - `POST /api/experimental/session/import` with `SessionTransfer.Data` (full structured messages) on a running OpenCode server; 409 on duplicate id. Experimental, schema in `packages/schema`.

5. Copilot CLI / Cursor / Gemini CLI / Pi
   - Copilot CLI: no host API found; VS Code's own path is the LLM summary (extension) or raw 20k-char transcript attachment / structured `importConversation` turns inside VS Code's agent host (internal, not callable from outside).
   - Cursor: closed, no import; its model-switch "taking over" instruction is a pattern AO can copy (keep native history, add a preamble naming the previous agent and warning about unavailable tools).
   - Gemini CLI, Pi: no import; would require writing their native session files (Pi documents `switch_session {sessionPath}` over RPC; Gemini stores under `~/.gemini/tmp/<hash>/chats/`), format stability unverified.

6. Design hints borrowed from vendors (all VERIFIED above)
   - Flatten tools into delimited text with per-call input cap 2,000 chars and result cap 4,000 chars (Codex).
   - Drop thinking, sidechain and meta records (Codex).
   - Mark imports with a UI-only marker, not model-visible (Codex).
   - Hash source content and keep a ledger of source to target thread, and refuse in-place sync if the target has diverged or been compacted (Codex).
   - Keep only the tail of the transcript, 20,000 chars, user and assistant prose only (VS Code); or use a fast-model structured summary with `<analysis>`/`<summary>` sections, tool results capped at 2,000 chars and 7,000 summary tokens (Copilot extension).
   - When switching models, prepend takeover instructions and steer away from unknown tools (Cursor).
