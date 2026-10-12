# Native bridges, handoff utilities, and routing controls: evidence notes

Research snapshot: 2026-10-10. These notes supplement the main competitor report. Verification means public documentation/source inspection, not running competitors with authenticated accounts. Repository snapshots were downloaded into a temporary research directory; no competitor code was executed. AO application code was not changed for this research.

## 1. agentctl — leofmarciano/agentctl

**Verdict:** verified logical-session switching between actual Claude Code and Codex native CLIs. Same canonical session and stable provider-specific native mappings; switching requires exiting the current CLI. It is not a hot replacement during a running turn.

Source snapshot: `7997916508dd31e1c3643204c00d1cc19d08986e`.

- [README](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/README.md): native CLI interaction, canonical store, provider mappings, and published smoke-test limitations.
- [Architecture](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/docs/architecture.md): exclusive worktree lease, launch/capture journal, stable identities, and uncertain-delivery handling.
- [Codex adapter](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/provider-codex/src/adapter.rs#L523-L593): `read_native_history` uses `thread/read`; `sync_context` submits public history via `thread/inject_items` and returns a receipt. This is the competitor's implementation against its supported installed Codex schema, not a promise that every Codex version provides that method.
- [Claude hooks](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/cli/src/native_hooks.rs#L703-L754): `deliver_staged_handoff` checks identity/digest and a 9,500-character limit, writes `UserPromptSubmit` additional context, then commits delivery. Output/receipt failures retain uncertainty.
- [Projection selection](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/transcript/src/projection.rs#L59-L143): sequence cursors, checkpoints, internal-event exclusion, and per-event receipts prevent replaying already synchronized history.

**AO implication:** the two directions need not have identical transport. Borrow one-writer ownership and delivery receipts. Delta synchronization could improve switch-back later, but adds mapping/cursor complexity. A receipt for context delivery must not be confused with proof the next model understood it. The published native smoke test submits no model prompt, so it establishes launch/mapping behavior rather than semantic handoff quality.

## 2. claude-codex-bridge — nerln/claude-codex-bridge

**Verdict:** verified bidirectional task-lane continuity through new native resumable sessions; not one unchanged native session or unified chat view.

Source snapshot: `56e0776b3aa8a8ffe42c55c78b1c508b8811440a`.

- [README](https://github.com/nerln/claude-codex-bridge/blob/56e0776b3aa8a8ffe42c55c78b1c508b8811440a/README.md): stable source snapshot, transcript/capsule modes, lineage ledger, default 120,000-character transcript budget.
- [Switch coordinator](https://github.com/nerln/claude-codex-bridge/blob/56e0776b3aa8a8ffe42c55c78b1c508b8811440a/skills/switch-agent/scripts/bridge.py#L2030-L2185): `cmd_switch` locks the project, selects the exact source, snapshots the workspace, chooses native transcript or capsule fallback, and hashes transfer inputs into a retry identity. A new transfer reserves a new target session/path.
- [Transcoder](https://github.com/nerln/claude-codex-bridge/blob/56e0776b3aa8a8ffe42c55c78b1c508b8811440a/skills/switch-agent/scripts/transcode.py#L895-L970): deterministic redaction then 30% head / 70% tail retention with an omission marker. `write_claude` and `write_codex` create private native-format JSONL; the Codex writer emits both model-history records and UI-event records. Source files are read, not rewritten.

**AO implication:** explicit source selection, transfer hashes, and dry-run inspection are useful. Native JSONL generation is a larger compatibility burden than AO's bounded bootstrap. The distinction between inference history and rendered scrollback is particularly relevant to TUI continuity. A native-looking transcript does not transfer provider cache, private reasoning, or live tool state.

## 3. catchup — wilbeibi/catchup

**Verdict:** verified cross-agent continuation into a fresh target invocation; native same-agent fork/resume is a separate path. Broad reader support does not mean every reader is a launchable target.

Source snapshot: `50ac5ca88628f5785f81318e4a7841a96f409585`.

- [Cross-agent launch](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/cli.go#L700-L748): `forkInto` reads and normalizes native history, optionally selects post-compaction/recent turns, renders it, then seeds the target. The dispatch at lines 921–973 supports Claude, Codex, OpenCode, Pi, Cline, Cursor, Copilot, Grok, and Antigravity forms, while refusing unsupported interactive targets such as Kimi in the inspected version.
- [Compaction selection](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/cli.go#L1242-L1273): `sinceCompact` keeps the last compaction marker, provider-marked retained earlier entries, and everything after it. Empty summary text generates a warning.
- [Codex reader](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/codex/codex.go#L375-L422): reads `replacement_history`; ignores opaque encrypted compaction items; preserves supported public retained content.
- [Clipping policy](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/clamp.go): user entries have a 4,096-byte ceiling; generated entries 32,768 bytes before clipping. This is a per-entry policy, not a 32k total budget.
- [Seed storage](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/seed.go): immutable content-hashed seed files are retained when the launch transport uses a file.

**AO implication:** this is an unusually relevant reference for the desired compaction-plus-recent design. Port the semantic distinction, not the whole parser suite: native compaction can retain older messages and can have no readable summary. Platform argument limits and source-read capability differ from target-launch capability.

## 4. Omnigent — omnigent-ai/omnigent

**Verdict:** cross-agent fork is available; current in-place switching is explicitly retired. Do not count an old issue describing a switch endpoint as evidence of current support.

Source snapshot: `ef0adfb2066582b9abc97072e0e4ca8051dae6ad`.

- [Current route](https://github.com/omnigent-ai/omnigent/blob/ef0adfb2066582b9abc97072e0e4ca8051dae6ad/omnigent/server/routes/sessions/routes_core.py#L3845-L3859) unconditionally returns HTTP 410 and directs callers to fork into another agent.
- [Changelog](https://github.com/omnigent-ai/omnigent/blob/ef0adfb2066582b9abc97072e0e4ca8051dae6ad/CHANGELOG.md) records retirement in PR #8795.
- [Fork behavior coverage](https://github.com/omnigent-ai/omnigent/blob/ef0adfb2066582b9abc97072e0e4ca8051dae6ad/tests/e2e_ui/fork_session/test_fork_switch_agent.py) covers carrying history into a different agent.

**AO implication:** reusable agent configuration and per-session ownership must remain distinct. This is a concrete warning against inferring current functionality from old designs, endpoint names, or test filenames.

## 5. agentproto

**Verdict:** official docs verify cross-harness checkpoint handoff, but it creates a new session and leaves the source running. It does not meet strict same-session replacement.

[Official handoff guide](https://agentproto.sh/docs/cli/guides/handoff) documents a real handoff asking an idle source to summarize, a read-only dry run using approximate transcript extraction, checkpoint sections, recent digest, native transcript path, and explicit continuation lineage. The target starts in the same working directory. The source is not stopped automatically. Provider/model compatibility is validated rather than silently selecting another billing route.

**AO implication:** adopt explicit approximation/provenance labels and installed/credential-ready suggestions. Its extra source summarization turn and two-live-session behavior conflict with AO's desired quick deterministic handoff and one-writer boundary.

## 6. Turntrail — yagyaanshK/turntrail

**Verdict:** documented cross-agent transfer through an export artifact and either a managed new/existing target CLI or clipboard. Logical project continuity; no verified single canonical chat replacement.

[Official README](https://github.com/yagyaanshK/turntrail) describes a local transcript ledger, workspace snapshot, exact source-session selection, deterministic deduplication/trimming, media stripping, secret redaction, and retrieval links for omitted tool output. Claude/Codex managed handoff can launch or resume; other transcript import support is broader than this managed path.

**AO implication:** readable truncation accounting and links to recover omitted details complement a small bootstrap. Avoid importing its account manager or separate ledger into AO, which already owns durable conversation state.

## 7. Agent Switcher — ibodc/agent-switcher

**Verdict:** documented Claude/Codex handoff into a new destination conversation, with logical lineage rather than one unchanged chat.

[Repository README](https://github.com/ibodc/agent-switcher) and [Marketplace listing](https://marketplace.visualstudio.com/items?itemName=ibodc.agent-switcher) describe SDK-backed Claude session reads, Codex app-server history reads, visible transcript plus current Git/file context, preview/redaction, and `.agentbridge/bridge.json` lineage. The destination normally receives a new native ID; Codex resume is only used when a corresponding persisted rollout exists.

**AO implication:** obtain source identity from AO's binding instead of searching latest files by directory. A compact optional preview is useful; full transcript and diffs are excessive as AO's default payload.

## 8. claude-code-with-codex — fcakyon/claude-code-with-codex

**Verdict:** documented same-conversation backend routing while Claude Code remains the harness. It is not replacement by native Codex.

[Official README](https://github.com/fcakyon/claude-code-with-codex) describes a local proxy routing Claude model requests to Claude access and GPT-family requests through Codex-authenticated access. The existing Claude Code agent loop and transcript remain in control.

**AO implication:** API translation can feel seamless, but changes the problem into protocol/auth compatibility and leaves the original harness responsible for tools. This is adjacent competition, not the architecture AO should copy for native provider switching.

## 9. CC Switch — farion1231/cc-switch

**Verdict:** multi-tool configuration/provider routing manager; reviewed public material does not establish cross-harness transcript handoff in one logical session.

[Official repository](https://github.com/farion1231/cc-switch) presents provider profiles, MCP/skills/prompts, and proxy features across coding tools. Those capabilities must not be counted as proof that a Claude Code conversation becomes a Codex conversation. [agent-provider-switch](https://github.com/minniexcode/agent-provider-switch) is another adjacent configuration-switching tool with the same classification concern.

**AO implication:** keep account selection, endpoint routing, model selection, and harness handoff distinct in both API naming and product claims.

## AO-specific finding exposed by this research

The working checkout is based on `4c75fe44356d1821833c271deea2b1c158051c2b` plus existing uncommitted implementation work. Inspection only:

- `backend/internal/service/chat/handoff_context.go` selects `ConversationActivity.Summary` for the last completed system activity whose detail event is `compaction`.
- `backend/internal/adapters/chatdriver/codexappserver/conversation.go`, `settleCompaction` / `compactionSummary`, produces a timeline label such as “Compacted the conversation history” or a count of freed tokens. That is not the provider's semantic conversation summary.
- The ACP `settleCompaction` path can also replace captured text with a reclaimed-token label. It therefore does not guarantee that `Activity.Summary` is portable semantic context.
- `backend/internal/session_manager/provider_switch_tui.go` calls optional `ports.AgentTUIHandoff.ReadTUIHandoff`; repository search finds a test implementation but no built-in production adapter implementing this method. The default TUI fallback uses `SessionInfo` and latest-message metadata, so it must not be described as universally carrying five complete exchanges plus a native compaction summary.

**Recommendation, not an implementation change:** represent semantic summary text, compaction occurrence, and UI label separately. Use a readable provider checkpoint when actually present; otherwise select durable public history deterministically and report reduced continuity. No extra LLM summary is required for that fallback. These findings need resolution before claiming complete compaction-based handoff coverage across providers.
