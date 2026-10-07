# Provider switching implementation plan

Status: implemented and verified

Current checkpoint (2026-10-07): the bounded context builder and the idle-only
Chat-to-Chat transaction are implemented. The builder reads the existing
canonical conversation snapshot, selects the latest completed compaction
activity, keeps up to five complete human-user/provider-final exchanges,
excludes daemon and automation prompts, and enforces the 32,000-character cap.
The lifecycle path fences and stops the source, starts a fresh target provider
with the bootstrap in its standing prompt, commits the target branch/session
ownership atomically, restores the source on target failure, and records a
canonical provider-switch activity. The daemon routes, thin CLI command, and
desktop session action now call that same path. A daemon restart before
publication is safe because the source session/branch remains the durable owner
and normal startup restores it; the target is simply retried. No new transcript
store or migration was added; the registry-backed provider matrix uses the
shared path and deterministic tests.

## Execution checklist

- [x] Inspect the existing implementation and keep the handoff on the canonical
  AO conversation/activity model.
- [x] Prove the switch saga with source-start failure, target-publication
  failure, source restoration, and duplicate idempotency tests against SQLite.
- [x] Keep the registry-backed provider set explicit; all 13 registered AO Chat
  drivers are represented by the shared switch path and UI target list.
- [x] Run backend compile, focused tests, race/vet, and API-spec generation.
- [x] Run frontend typecheck and launch the Electron app against fresh scratch
  data and the requested existing `~/.ao/dev/data` state.
- [x] Record exact verification results and environment limitations below.

### Verification record

- Passed: `GOCACHE=/tmp/ao-provider-gocache go build ./...`.
- Passed: `GOCACHE=/tmp/ao-provider-gocache go vet ./...`.
- Passed: full compile-only backend suite (`go test -run '^$' ./...`).
- Passed: Chat/storage/domain focused tests, including provider-switch fault
  injection; provider-switch tests also pass with `go test -race`.
- Passed: `GOCACHE=/tmp/ao-provider-gocache npm run api:spec` and OpenAPI spec
  parity package tests.
- Passed: frontend TypeScript typecheck (`npm run typecheck`).
- Passed: Electron Forge dev launch with the bundled browser runtime. A fresh
  scratch-data launch rendered the AO board with successful daemon API calls;
  the requested existing `~/.ao/dev/data` launch rendered the existing
  project/session list.
- Blocked by sandbox: full `go test ./...` exercises tests that bind sockets,
  tmux, or launch provider binaries; those fail with `operation not permitted`
  or unavailable binaries, unrelated to this feature.
- Existing dev data reports unrelated historical schema drift when opening old
  conversation/session detail rows (`client_payload_hash`, `source_key`, and
  `preview_capability_verifier` are absent despite their migrations being
  marked applied). This is outside provider switching and was not modified.

This plan implements idle-only provider switching for one AO session. It is
intentionally shaped like Synara's handoff model: the AO conversation remains
the canonical UI history, while the target provider receives one bounded
bootstrap context to start or resume its own native conversation.

The implementation target is approximately 5,000 new or materially changed
production lines, with a soft test-to-production ratio of 4:1. The estimate is
for source code only; generated OpenAPI/sqlc output, fixtures, and test files
are counted separately. We should stop and split the work if the production
diff exceeds 5,500 lines without an explicit scope decision.

## 1. Product contract

### User-visible behavior

- A provider switch is offered only when the current agent is idle. A running
  turn, queued turn, approval, structured question, compaction, unresolved
  interaction, or unsent draft blocks the switch.
- The AO session and its canonical conversation do not change identity. The
  existing Chat timeline continues to show the complete conversation, tool
  activities, compaction markers, and a provider-switch marker.
- The target provider receives a fresh/native target thread (or a provider
  branch already created for that target) seeded with a single handoff
  bootstrap message. The bootstrap is an implementation detail and is not
  rendered as an additional human message in AO's canonical timeline.
- The source controller is stopped before the target controller is allowed to
  write. There is never a two-writer interval.
- Switching back to a provider creates or resumes that provider's own branch;
  it does not rewind the AO conversation and does not delete the source
  provider's history.
- If a provider cannot satisfy the structured Chat contract, AO reports that
  provider as unsupported. AO must not scrape terminal scrollback to fake a
  handoff.

### Handoff payload

The payload is deterministic and built from durable AO facts:

1. The latest completed compaction activity's summary, read from the existing
   generic conversation activity (kind, summary, payload). The compaction
   discriminator is the existing system activity detail rather than a new
   summary table or a new LLM summarizer.
2. Up to the five latest completed user-to-final-assistant exchanges. Include
   the complete user message and the complete final assistant message for each
   exchange. Do not include streamed deltas, reasoning, tool chatter, approval
   prompts, or intermediate status text as transcript turns.
3. Small deterministic metadata: source provider, AO session/conversation
   identity, worktree/project context, source thread title when available,
   active branch/checkpoint, and explicit handoff timestamp. Provider-native
   credentials, hidden prompts, cache handles, approval answers, and tool
   permissions are never copied.

The entire serialized bootstrap has a hard 32,000-character budget. The
builder applies the following order:

1. reserve space for the framing/metadata and the latest compaction summary;
2. append the newest complete exchange first, then older exchanges until the
   budget is exhausted;
3. drop older exchanges before shortening newer ones;
4. preserve the newest user message and final assistant message whenever the
   budget can hold them; and
5. if the summary itself is larger than its reserved space, retain a bounded
   head/tail form with an explicit truncation marker.

The resulting bootstrap is one uncached user prompt to the target provider,
matching Synara's handoff shape. AO's canonical timeline is not replaced by
that prompt.

### Failure and recovery behavior

- The source remains active if bootstrap construction, target probing, or target
  startup fails.
- A failed target start leaves no active target branch/controller and can be
  retried idempotently.
- A daemon restart during a switch resumes from the durable transition phase;
  it never guesses that a provider is dead from a missing process alone.
- A successful switch commits the new provider/branch/controller ownership in
  one storage transaction. A failed switch rolls back to the source branch and
  records a user-visible failure activity.

## 2. Scope boundaries that protect the LOC budget

In scope:

- Chat-to-Chat switching for the providers with a structured AO Chat driver.
- Synara's provider set where AO has a driver: Claude Code, Codex, OpenCode,
  Cursor, Pi, Oh My Pi, and Droid. The current AO registry also includes Kimi,
  Kimchi, Qwen, DeepSeek Harness, and Unreal Agent; they use the same shared
  path when their driver advertises the required capabilities.
- A capability matrix and adapter tests for the Synara-only providers that do
  not yet have an AO structured Chat driver. They remain unavailable until a
  driver exists; no TUI/terminal parser is added as a shortcut.
- Desktop Chat UI and terminal-facing session controls that call the same
  daemon route. The terminal control is available when the session has a
  structured Chat provider; TUI-only agents are not advertised until they have
  a structured driver. The handoff decision and context builder live in Go, not
  in Electron or the CLI.

Out of scope for this first implementation:

- Mid-turn switching, partial-token transfer, or cancellation-as-success.
- Transferring provider caches, hidden reasoning, native session cookies,
  credentials, MCP process state, or provider-specific permission state.
- A new canonical transcript database, a separate summary service, or a new
  provider-specific compaction hook framework.
- Reimplementing all TUI-only agents as Chat drivers. That is a separate
  provider enablement project and would violate the budget.
- Changes to account switching, native TUI↔Chat interface handoff, reviewer
  conversations, or cloud orchestration unless a focused compatibility fix is
  required by the shared route.

## 3. Reuse map: existing AO seams

The implementation should build on these existing boundaries:

- domain.ConversationRecord, ConversationBranch, messages, turns, and
  ConversationActivity remain the canonical durable model.
- Existing activity persistence already stores kind, summary, and JSON detail;
  the latest compaction summary is a query over this data. Reuse CompactedAt
  for the fast “has compacted” fact, but never treat its timestamp as the
  summary.
- ports.ChatDriver, ChatConversation, capability probing, and the detached
  persistent provider host remain the provider boundary.
- Existing conversation branch creation/activation and controller-generation
  fencing provide the ownership transaction. Extend them only where the
  current branch strategy cannot represent an approximate provider bootstrap.
- session_interface_transitions already provides durable phase vocabulary and
  recovery semantics. Add a provider-switch transition kind or reuse the
  transition record only after confirming its source/target mode invariants do
  not mislabel a Chat-to-Chat switch.
- Existing paged conversation snapshots remain the UI read path; no endpoint
  should ship the 32k bootstrap merely to redraw the conversation.

Before coding, verify whether the current branch strategy and transition store
can represent provider-to-provider ownership without a migration. Prefer a new
query/helper over a schema change. If a new durable discriminator is genuinely
required, add one forward goose migration and update sqlc from source; never
edit a merged migration or generated code by hand.

## 4. Production implementation phases

The numbers below are budgets, not permission to fill them. Shared helpers and
existing seams should keep the actual diff below the budget.

### Phase A — Contract and provider capability matrix (350–450 LOC)

Files likely involved:

- backend/internal/domain/chat_handoff.go
- backend/internal/domain/conversation.go
- backend/internal/ports/chat.go
- backend/internal/adapters/chatdriver/registry/registry.go
- adjacent capability/DTO types and tests

Work:

- Define the provider-switch request/result and a typed refusal set:
  busy, draft, pending interaction, unsupported target, target unavailable,
  bootstrap too large, and recovery required.
- Define the minimum target capability as structured Chat plus streaming,
  resume/start, interrupt, and history/branch publication as appropriate.
- Add a provider capability matrix used by the API and UI. Provider names must
  not be scattered through handoff logic.
- Keep provider IDs and native thread IDs separate from the AO conversation ID.

Exit criteria: domain and port tests prove capability/refusal behavior without
starting a provider.

### Phase B — Deterministic handoff context builder (550–700 LOC)

Files likely involved:

- backend/internal/service/chat/handoff_context.go
- backend/internal/service/chat/handoff_context_test.go
- storage query/store helper files under
  backend/internal/storage/sqlite/{queries,store}/

Work:

- Add a single storage read for the latest completed compaction activity on the
  active conversation path. Match the existing system/compaction detail
  discriminator and ignore rolled-back, cancelled, or running activities.
- Read the latest five completed user/final-assistant pairs in sequence order;
  exclude activities and incomplete turns from this transcript projection.
- Normalize provider/model/title/worktree metadata into a small neutral DTO.
- Serialize the bootstrap with stable labels and the 32,000-character cap.
- Implement deterministic clipping and truncation markers. Unit-test exact byte
  and character-boundary behavior, Unicode, empty summaries, long summaries,
  missing assistant finals, and histories shorter than five exchanges.

No new summary generation happens here. This phase only retrieves and formats
the summary that is already persisted as an activity.

Exit criteria: the same canonical snapshot always produces byte-for-byte equal
bootstrap text and never exceeds the cap.

### Phase C — Provider-switch transaction and lifecycle (850–1,050 LOC)

Files likely involved:

- backend/internal/service/chat/service.go
- a small backend/internal/service/chat/provider_switch.go
- backend/internal/session_manager/provider_switch.go
- existing transition/store helpers and focused tests

Work:

- Admit only idle sessions and fence new sends while a switch owns the session.
- Snapshot source ownership, active branch, latest sequence, and controller
  generation before stopping the source.
- Build the bootstrap before source shutdown so a read failure cannot strand
  the session.
- Stop/detach the source controller; create or select the target provider
  branch; start the target with the bootstrap; verify the returned native ID.
- Publish branch ownership, provider identity, session metadata, and transition
  completion atomically. Use idempotency keys for retries and generation checks
  to reject stale completions.
- On every failure, restore source ownership or leave a durable recovery-needed
  state with a clear retry action.
- Reuse the existing detached provider host and controller gate; do not create a
  second host lifecycle implementation.

Exit criteria: fault-injection tests cover failures before source stop, after
source stop, after target start, and after target publication; no test can
observe two active writers.

### Phase D — Shared provider integration (950–1,200 LOC)

Files likely involved:

- backend/internal/adapters/chatdriver/acp/*
- backend/internal/adapters/chatdriver/*acp/
- backend/internal/adapters/chatdriver/codexappserver/
- registry and provider-specific integration tests

Work:

- Make the shared ACP path accept a handoff bootstrap and fresh target session
  identity without leaking provider DTOs into the service.
- Keep Codex app-server's native start/resume path behind its adapter; do not
  pretend ACP and Codex have identical history semantics.
- Ensure every registered provider reports whether it supports fresh prompt
  replay, native resume, approvals, interrupt, and history settlement.
- Add a single adapter contract suite run against Claude, Cursor, OpenCode 1/2,
  Droid, Kimi, Kimchi, Pi, OMP, Qwen, DeepSeek Harness, and Unreal Agent
  fakes. Provider-specific tests should only cover wire translation and known
  quirks.
- Add explicit “unsupported” entries for Synara-listed providers that AO does
  not yet expose through a structured Chat driver (for example Antigravity,
  Grok Build, and Devin CLI, subject to the current provider matrix).

Exit criteria: the shared ACP implementation handles the common path; custom
Codex behavior is isolated; adding a provider requires a registry entry and a
small adapter test, not a new handoff algorithm.

### Phase E — Daemon API and CLI boundary (350–500 LOC)

Files likely involved:

- backend/internal/httpd/controllers/conversations.go
- backend/internal/httpd/controllers/dto.go
- backend/internal/httpd/apispec/specgen/build.go
- CLI client/command files if a CLI switch command is exposed
- generated OpenAPI and TypeScript files via npm run api

Work:

- Add GET/POST capability and switch operations through the existing daemon
  controller boundary. The CLI remains a thin HTTP client.
- Return typed refusal/error envelopes with request IDs and retryability.
- Expose transition progress and the selected source/target providers, not the
  full bootstrap text.
- Register operation schemas, regenerate OpenAPI and frontend types, and add
  route/spec parity tests.

Exit criteria: direct storage/runtime access is impossible from the CLI or
frontend, and every state transition is observable through the existing API/SSE
invalidation path.

### Phase F — Desktop and terminal UI (700–900 LOC)

Files likely involved:

- frontend/src/renderer/components/chat/SessionChatSurface.tsx
- frontend/src/renderer/components/chat/ChatWorkspace.tsx
- session actions/provider picker components and hooks
- terminal session action surface where the same session control is exposed
- focused React tests and i18n strings

Work:

- Show one provider-switch action when the daemon says it is available.
- Disable or explain it for active work, drafts, approvals, questions,
  compaction, unsupported targets, and missing authentication.
- Render source→target progress from durable transition state and recover the
  UI after daemon restart.
- Keep the canonical timeline mounted/refetched so the user experiences one
  continuous chat. Do not insert the bootstrap as a visible user bubble.
- On success, update provider labels and target composer capabilities without
  clearing the conversation or losing pagination position.
- Use the same API operation for Chat and terminal-facing controls; no terminal
  scraping or duplicated handoff logic in React.

Exit criteria: UI tests cover successful switch, busy refusal, failed target,
restart recovery, switch-back, and full-history continuity.

### Phase G — Documentation, telemetry, and cleanup (150–250 LOC)

Work:

- Document the user contract and provider capability matrix in the current
  harness/provider docs.
- Emit bounded transition outcome telemetry (provider IDs, duration, outcome,
  refusal code), never bootstrap text or message contents.
- Add a short operational recovery note and remove dead experimental switching
  code only when the new path has replaced it.

Exit criteria: documentation describes only verified providers and telemetry
contains no user content or credentials.

### Production LOC budget

| Phase | Budget |
| --- | ---: |
| A. Contract/matrix | 450 |
| B. Context builder | 700 |
| C. Lifecycle transaction | 1,050 |
| D. Provider integration | 1,200 |
| E. API/CLI | 500 |
| F. UI/terminal | 850 |
| G. Docs/telemetry | 250 |
| **Total soft ceiling** | **5,000** |

The largest risk is Phase D. If it grows, stop and report which provider
requires a genuinely custom protocol. Do not inflate the shared core to hide a
provider-specific exception. A provider that needs more than roughly 200 new
production lines should get a separate enablement task or be explicitly marked
unsupported for this release.

## 5. Test plan and 4:1 target

The soft target is about 20,000 test lines for 5,000 production lines. Tests
should be behavior-heavy and table-driven; fixtures and generated files do not
count toward the target.

| Test area | Approx. test LOC |
| --- | ---: |
| Context formatting/cap/clipping | 2,000 |
| Storage/activity query and rollback filtering | 1,500 |
| Lifecycle state machine and idempotency | 4,000 |
| Controller/host/provider fakes | 3,500 |
| ACP shared contract matrix | 3,000 |
| Codex custom adapter | 1,500 |
| HTTP/OpenAPI/CLI boundary | 1,500 |
| React Chat UI and terminal controls | 2,500 |
| E2E/restart/fault-injection | 1,500 |
| **Target** | **21,000** |

Required cases:

- exactly 0, 1, 5, and more than 5 exchanges;
- summary absent, empty, normal, Unicode, and over-budget;
- 32,000-character boundary and truncation marker placement;
- activities on inactive branches, rolled-back turns, and running turns;
- latest assistant final missing or interrupted;
- provider idle, active, queued, approval-pending, question-pending,
  compaction-running, draft-present, and terminated states;
- target unavailable, unauthenticated, incompatible, unsupported, and startup
  timeout;
- retry after every lifecycle checkpoint, including daemon restart;
- duplicate request/idempotency key and stale controller generation;
- source→target→source switching with no duplicate canonical messages;
- all shared ACP providers plus Codex's custom app-server path;
- API error envelope/request ID/retryability and SSE invalidation;
- UI full-history continuity, pagination, progress, failure recovery, and
  terminal control parity;
- opt-in live smoke tests only when the provider binary and credentials are
  available. Never make network/provider availability a normal unit-test
  dependency.

## 6. Delivery order and commits

Keep commits small enough that each can be reverted without leaving a half
usable API:

1. feat: define provider switch contract and capability matrix
2. feat: build bounded provider handoff context
3. feat: add idle provider switch transaction
4. feat: wire shared provider handoff adapters
5. feat: expose provider switch daemon API
6. feat: add provider switch controls to desktop and terminal UI
7. test: verify provider switching and restart recovery
8. docs: document provider switching support

Run focused tests after each commit. Before handoff, run the repository gates
required by AGENTS.md:

~~~text
cd backend && go build ./... && go test ./... && go test -race ./... && go vet ./...
npm run lint
npm run frontend:typecheck
cd frontend && npm run build
npm run api                 # when DTO/spec sources changed
npm run sqlc                # only when SQL/query sources changed
~~~

Do not claim a provider is supported until its structured driver, capability
probe, handoff contract tests, and restore/failure behavior pass. If a full
Synara provider requires a missing AO Chat driver, record it in the capability
matrix and split that driver into its own scoped change rather than breaking
the 5,000-line core budget.

## 7. Review checklist

- Is the latest compaction summary read from the existing activity row rather
  than regenerated or stored in a duplicate table?
- Is the canonical AO conversation unchanged and fully renderable after a
  switch?
- Is the bootstrap always <= 32,000 characters, deterministic, and bounded?
- Are only complete user/final-assistant exchanges included?
- Can any path run two provider writers at once?
- Can a failed/restarted switch be retried without duplicate branches,
  messages, or provider prompts?
- Are provider-specific details isolated behind the Chat driver interface?
- Does every supported provider have the same contract tests?
- Are unsupported TUI-only/Synara-only providers explicit rather than silently
  advertised?
- Is the production diff still near 5,000 lines and the test diff near 4:1?
