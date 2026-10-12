# Provider switching competitor analysis: what AO should borrow

Research snapshot: 2026-10-10–11. Prepared using three parallel research agents plus a consolidation/source-check pass. **53 distinct product families/tools screened**, exceeding the requested approximately 30. Public documentation and selected implementations were inspected; competitors were not installed, authenticated, or exercised with live model calls. Source-verified implementation is not a claim of successful end-to-end testing or parity with the latest packaged release.

## 1. The conclusion

AO should keep **one durable AO conversation and workspace, with replaceable native-agent execution**. Use the existing canonical history to prepare a small deterministic handoff; do not translate every provider's private transcript or build a new shared agent harness.

The closest references are:

| Reference | Why it matters | What not to copy by default |
| --- | --- | --- |
| **Synara** | Same application thread, fresh target native session, canonical-history bootstrap, idle/approval checks. Closest simple architectural match. | Its older-history snippets are not native compaction summaries; its six completed messages are not five final-response exchanges. |
| **T3 Code** | Same application thread; explicit provider-thread ownership; budgeted context and retrieval; native resume plus missed-work delta on return. Strongest richer reference. | Its full orchestration/context-selection system is broader than AO's small implementation target. |
| **agentctl** | Actual Claude/Codex CLIs under one canonical session; exact native bindings, ownership leases, context-delivery receipts. | Every installed Codex version cannot be assumed to support its injection operation; native resume/delta adds complexity. |
| **Tandem CLI** | Real native TUI switching, including mirrored historical scrollback and paired provider sessions. | Native transcript/tool rewriting is a separate compatibility-heavy product, not a small bootstrap feature. |
| **Voyager/sessionFlow** | Stable task identity, canonical index, one-writer lease, shared continuation engine. | Search/ranking/transplant machinery; existing-target resume alone does not prove missed work was delivered. |

These five have inspected implementations retaining an **outer work identity**, but they are not equivalent: Synara/T3 retain an application conversation; agentctl/Tandem retain a paired logical session; Voyager retains a task group without one shared chat UI. Context-bridge also claims linked-session continuity, but its source was not audited enough to include it in this source-verified shortlist. [Synara reactor](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/Layers/ProviderCommandReactor.ts#L6777), [T3 planner](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/apps/server/src/orchestration-v2/ProviderSwitchService.ts), [agentctl architecture](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/docs/architecture.md), [Tandem architecture](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/docs/how-it-works.md), [Voyager engine](https://github.com/HarryHeYu/sessionFlow/blob/ee86d8b079e927ea064a4860e0d6e87982d746b9/voyager/continuity.py#L1250).

**Important correction to our earlier discussion:** Synara's generic activity `summary` is often a display label such as “Context compacted,” not the semantic summary of the conversation. Its handoff builder does not consume compaction activities. AO's inspected Codex path has the same display-label distinction. The small checkpoint-based design remains sensible, but we must use real checkpoint text when available and an explicit deterministic-history fallback when it is not. Details and local evidence are in section 6.

## 2. What “switching” means in this report

Four different operations are frequently advertised under the same name:

1. **Harness switching:** native Claude Code stops controlling the work; native Codex takes over. Tools, system instructions, permissions and provider-native state may differ.
2. **Backend/model switching:** Cline, Pi, Cursor or another shared harness keeps control and uses another inference backend. This does not launch native Codex just because a model is named Codex.
3. **Continuation/fork:** a fresh target chat or terminal receives source context; the original remains a distinct conversation.
4. **Configuration/surface switching:** change API profiles, accounts, terminal/mobile control or visible tabs. This alone proves no conversation transfer.

For AO, the strict UX target is: **same AO session ID, same canonical conversation, same workspace, full old UI history, another actual provider for subsequent work**. The native provider session ID does not need to stay the same. Account switching within one provider is a separate compatibility/security question.

Evidence labels:

- **S:** implementation inspected, often with tests/fixtures; not runtime-tested here.
- **D:** explicit first-party documentation establishes the stated behavior; internals not fully audited.
- **U:** relevant integrations/claims exist, but exact continuity is unverified.
- **H:** historical/archived implementation; not counted as a maintained current competitor.

“Unverified” means insufficient evidence in this hunt, not proof a product can never support it. Current `main`/`dev` and current docs can be ahead of downloadable releases. Provider integration counts are not counts of tested handoff pairs.

## 3. Complete discovery catalog

### 3.1 Actual native-agent switching and seeded continuation: 21 candidates

| # | Product | Agents / scope found | Verified behavior and identity | Evidence |
| --- | --- | --- | --- | --- |
| 1 | **Synara** | Ten provider kinds: Claude, Codex, Cursor, Antigravity, Grok, Droid, OpenCode, Pi, Devin, Oh My Pi | **Same app thread.** Stops source, abandons native cursor, starts target fresh, replays canonical context. Native switch-back is fresh too. | S — [reactor](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/Layers/ProviderCommandReactor.ts#L6777), [provider enum](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/packages/contracts/src/orchestration.ts#L77) |
| 2 | **T3 Code** | Native provider instances including Claude/Codex; capabilities determine transport | **Same app thread.** Portable history into new target; can reuse prior target-native thread with missed-run delta. | S — [planner](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/apps/server/src/orchestration-v2/ProviderSwitchService.ts), [policy](https://github.com/pingdotgg/t3code/blob/main/docs/user/portable-handoffs.md) |
| 3 | **Tandem CLI — Bhavya6187** | Claude Code, Codex, OpenCode | **Same paired session/native TUI rotation.** Continuously mirrors translated history into each provider's shadow storage; resumes paired target. | S — [architecture](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/docs/how-it-works.md) |
| 4 | **agentctl — leofmarciano** | Claude Code, Codex | **Same canonical logical session.** Exit source CLI, capture history, deliver delta, launch/resume mapped target. Provider files not rewritten. | S — [architecture](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/docs/architecture.md) |
| 5 | **Voyager / sessionFlow** | Eight readers; direct bundle targets include Claude, Codex, Grok | **Same WorkThread/task group, not one chat UI.** Canonical index plus native session members; continuation bundle or native resume. Round-trip freshness needs route-specific testing. | S — [engine](https://github.com/HarryHeYu/sessionFlow/blob/ee86d8b079e927ea064a4860e0d6e87982d746b9/voyager/continuity.py#L1250) |
| 6 | **Orca** | Broad native launcher catalog | **New tab/session**, source unchanged. Transcript pointer with focused/full read instructions, or bounded terminal capture. | S — [launcher](https://github.com/stablyai/orca/blob/7de3710dbed7466ebf0ff7edc84738ce9906c4f7/src/renderer/src/lib/launch-agent-session-continuation.ts#L66) |
| 7 | **Superset** | Native terminal agents; inspected history readers: Claude/Codex | **New split/tab.** Native transcript preferred; stream/screen fallback; same-agent fork is separate. | S/D — [sessions guide](https://docs.superset.sh/agent-sessions), [builder](https://github.com/superset-sh/superset/blob/4ec5b6e287d380480a71b4b2ad98976354c82d4f/packages/shared/src/terminal-session-handoff.ts) |
| 8 | **Conductor** | Explicit Claude↔Codex behavior | **New chat tab** with conversation summary. Summary producer/limits not established by inspected changelog. | D — [0.28.7/0.28.4 changelog](https://www.conductor.build/changelog/page/11) |
| 9 | **Vibeyard** | Claude Code, Codex CLI, Gemini CLI | **New app session record** with prompt to read source transcript. Missing transcript asks user to restate goal. | S — [creation](https://github.com/elirantutia/vibeyard/blob/19bc19f0fac04f11e7d9e5cf68ed75ac7d291566/src/renderer/state.ts#L820) |
| 10 | **1Code** | Claude/Codex | **New subchat** within same parent/worktree; bounded history attachment. Provider mutation blocked after messages exist. Archived when checked. | S/H — [active chat](https://github.com/21st-dev/1code/blob/9f1bc76fa4372c18c565b5a4f8daf38ae3595f0e/src/renderer/features/agents/main/active-chat.tsx) |
| 11 | **VS Code agent harnesses** | Local → available Copilot/Claude/Codex/Cloud destinations | **Constrained cross-harness handoff.** Only Local sessions may initiate; Agent Host sessions are destinations. Arbitrary native Claude→Codex→Claude and stable target chat identity are not established. | D — [handoff restrictions](https://code.visualstudio.com/docs/agents/run/agent-harnesses#hand-off-a-session) |
| 12 | **showagent** | Codex, Claude, Gemini, OpenCode, jcode, Pi | **New native session through text conversion.** No single shared app chat; MCP can return commands rather than launch. | S — [conversion](https://github.com/aytzey/showagent/blob/39a97bda8203314a0b3f43992b8da29c00d2cb21/internal/session/handoff.go#L17), [compatibility](https://github.com/aytzey/showagent/blob/39a97bda8203314a0b3f43992b8da29c00d2cb21/docs/compatibility.md) |
| 13 | **claude-codex-bridge — nerln** | Claude↔Codex | **New native resumable target** under logical task lineage. Stable source snapshot; native JSONL conversion or capsule fallback. | S — [switch coordinator](https://github.com/nerln/claude-codex-bridge/blob/56e0776b3aa8a8ffe42c55c78b1c508b8811440a/skills/switch-agent/scripts/bridge.py#L2030-L2185) |
| 14 | **catchup** | Broad native readers; inspected launch dispatch has Claude, Codex, OpenCode, Pi, Cline, Cursor, Copilot, Grok, Antigravity | **Fresh cross-agent invocation**, optional last-compaction/recent selection. Reader support is broader than launch support. | S — [cross-agent path](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/cli.go#L700-L748) |
| 15 | **continues / cli-continues** | 16 declared agent integrations | **Context document plus target launch.** Native same-agent resume is separate; no stable cross-provider chat identity established. Direction-count claims not live-tested. | D — [repository](https://github.com/yigitkonur/cli-continues) |
| 16 | **Omnigent** | Multi-agent sessions | **Cross-agent fork**, not current in-place switching. Existing switch route explicitly returns 410/retired; old feature descriptions are stale. | S — [current route](https://github.com/omnigent-ai/omnigent/blob/ef0adfb2066582b9abc97072e0e4ca8051dae6ad/omnigent/server/routes/sessions/routes_core.py#L3845-L3859) |
| 17 | **agentproto** | Documented Claude→Codex, supported target agents | **New checkpoint-linked session**, source remains running. Real handoff asks source to summarize; dry run uses approximate extraction. | D — [handoff guide](https://agentproto.sh/docs/cli/guides/handoff) |
| 18 | **Turntrail** | Claude/Codex managed handoff; broader imports | **Export artifact + new/existing target/clipboard.** Local ledger and exact source selection; no single canonical chat replacement established. | D — [repository](https://github.com/yagyaanshK/turntrail) |
| 19 | **Agent Switcher — ibodc** | Claude↔Codex | **New target conversation**, history/workspace preview and lineage. Existing Codex resume requires a persisted rollout. | D — [repository](https://github.com/ibodc/agent-switcher) |
| 20 | **context-bridge — SerdarDB** | Claude, Codex, Grok, Antigravity, OpenCode | **Claims linked lanes with delta synchronization** and provider-native resume. Promising direct competitor; source-level delivery/round-trip audit remains open. | U/D — [primary README](https://github.com/serdardb/context-bridge) |
| 21 | **claude-to-codex — Amal-David** | Claude→Codex | **One-way new interactive target.** Source skill prepares disk context and launches Codex with a bounded pointer prompt. | D — [repository](https://github.com/Amal-David/claude-to-codex) |

**Filter result:** five source-traced outer-work-identity implementations (#1–5), fifteen continuation/fork implementations or explicit documented workflows (#6–19 and #21; #10 historical), and one linked-session claim requiring deeper audit (#20). Only #1–2 establish the exact same-application-chat architecture from this source audit. This is not a claim that all provider pairs or released binaries work.

### 3.2 Manual handoff, retrieval, and delegation: five candidates

| # | Product | What it actually does | Why it is not strict same-session switching | Evidence |
| --- | --- | --- | --- | --- |
| 22 | **claude-session-handoff — fvhreimert** | Codex skill discovers/selects local Claude JSONL and prepares context. | Recipient-side recovery/context import, not a stable outer-session switch controller. | D — [repository](https://github.com/fvhreimert/claude-session-handoff) |
| 23 | **AgentHandoff — aceandro2812** | Structured packet; target reads Markdown or sections through MCP. | Context transport; not verified native runtime ownership transfer. Cost-saving numbers not validated. | D — [repository](https://github.com/aceandro2812/AgentHandoff) |
| 24 | **claude-codex-handoff — zhpy2004** | Mirrored skills exchange `HANDOFF.md`. | Explicitly no automatic target launch. | D — [repository](https://github.com/zhpy2004/claude-codex-handoff) |
| 25 | **cross-agent-handoff — rwineman** | Claude writes handoff docs, calls `codex exec`, then picks up results. | Delegation/review with source still coordinating, not replacement. | D — [repository](https://github.com/rwineman/cross-agent-handoff) |
| 26 | **open-grove/handoff** | Immutable portable artifact; optional target-session generator. | Explicitly no native-state restoration; document-oriented continuation. | D — [repository](https://github.com/open-grove/handoff) |

These are useful evidence for compact packet structure and retrieval, but should not inflate the strict switching shortlist.

### 3.3 Own-harness/backend switching: 14 references

| # | Product | Same-conversation finding | Native Claude Code→Codex? | Evidence |
| --- | --- | --- | --- | --- |
| 27 | **OpenCode** | Current public V2 contract switches session model/backend for future turns. Moving dev contract; release parity not tested. | No: OpenCode remains harness. | S/D — [session model API](https://dev.opencode.ai/v2/docs/api/session/v2-session-switchmodel/), [models](https://dev.opencode.ai/v2/docs/models/) |
| 28 | **Kilo Code** | `/models` changes current session; model/effort remembered per agent in session. | Not proved by a Codex model label. | D — [model selection](https://kilo.ai/docs/code-with-ai/agents/model-selection) |
| 29 | **Cline** | API-configuration update replaces active task's provider/model shim. | No native harness handoff established. | S — [update handler](https://github.com/cline/cline/blob/main/apps/vscode/src/core/controller/models/updateApiConfiguration.ts) |
| 30 | **Aider** | `/model` retains work; changed edit format can trigger summarization of incompatible old completed history. | No: Aider remains harness. | S/D — [commands](https://github.com/Aider-AI/aider/blob/main/aider/commands.py), [coder transfer](https://github.com/Aider-AI/aider/blob/main/aider/coders/base_coder.py) |
| 31 | **Goose** | `/model --provider` switches API providers. Current CLI explicitly refuses ACP/context-owning providers. | **Not supported by this CLI switch path**, despite native-agent integrations. | S/D — [CLI source](https://github.com/block/goose/blob/main/crates/goose-cli/src/session/mod.rs), [commands](https://goose-docs.ai/docs/guides/goose-cli-commands/) |
| 32 | **OpenHands** | Supporting backends allow explicit `/model <saved-profile>` switching with history/workspace/task retained. Editing a saved profile is not the same operation. | No: OpenHands remains harness. | D — [profiles](https://github.com/OpenHands/docs/blob/main/openhands/usage/agent-canvas/llm-profiles.mdx) |
| 33 | **Continue** | CLI `/model` changes configured model; compact/resume/fork are separate. | No native replacement established. | D — [TUI mode](https://docs.continue.dev/cli/tui-mode) |
| 34 | **Crush** | Explicitly preserves context when switching LLMs mid-session. | No: Crush remains harness. | S/D — [repository](https://github.com/charmbracelet/crush) |
| 35 | **Pi** | Cross-provider model switching; explicit session tree and real compaction entries. | No: Pi inference integration is not native Codex CLI. | S/D — [session format](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/session-format.md), [provider normalization](https://github.com/earendil-works/pi/blob/main/packages/ai/README.md) |
| 36 | **Cursor** | Official model picker can change during chat/agent use. Internal projection mechanism is closed/unverified. | No native replacement established. | D — [available models](https://prod.cursor.com/help/models-and-usage/available-models) |
| 37 | **Happy / Happy Harness** | Current shared runtime changes inference provider between responses in one durable session. Distinct from older native wrappers. | No untouched CLI swap: Codex-style Pi inference/tools and Claude SDK inference use shared runtime behavior. | D — [runtime architecture](https://github.com/slopus/happy-agent), [overview](https://github.com/slopus/happy) |
| 38 | **Zed** | Native agent supports model selection. This audit establishes external threads have separate context, not strict cross-external transfer. | **Unverified for external-agent handoff.** Native picker alone is insufficient. | D/U — [parallel agents](https://zed.dev/docs/ai/parallel-agents), [external agents](https://zed.dev/docs/ai/external-agents) |
| 39 | **claude-code-with-codex — fcakyon** | Same Claude Code conversation routes model requests through a proxy to Claude/GPT access. | No: Claude Code keeps orchestration/tools. | D — [repository](https://github.com/fcakyon/claude-code-with-codex) |
| 40 | **Roo Code** | Explicit profile activation updates current task's API handler/history binding. **Historical:** official shutdown notice. | No native replacement established. | S/H — [source](https://github.com/RooCodeInc/Roo-Code/blob/main/src/core/webview/ClineProvider.ts), [shutdown notice](https://github.com/RooCodeInc/Roo-Code/blob/main/README.md) |

This group contains twelve active documented/source-confirmed backend-routing examples (#27–37 and #39), one native-picker/external-transfer qualification (#38), and one historical reference (#40). It is **not twelve native-agent migration systems**.

### 3.4 Adjacent, restricted, or unverified products: 13 candidates

| # | Product | Relevant capability | Narrow switching verdict / evidence |
| --- | --- | --- | --- |
| 41 | **Windsurf / Devin Desktop** | Claude/GPT/in-house model selection. Current Windsurf docs redirect to Devin Desktop. | Partial: exact old-history replay semantics not established. Count lineage once. [Current model docs](https://docs.devin.ai/desktop/models) |
| 42 | **Amp** | Dial modes/model routing; Claude SDK mode. | Current docs lock mode after first message; new thread needed to change it. Do not use older switching claims. [The Dial](https://ampcode.com/docs/the-dial) |
| 43 | **Gemini CLI** | `/model` affects future interactions. | Gemini-family switching, not cross-provider evidence. [Model docs](https://geminicli.com/docs/cli/model/) |
| 44 | **HAPI** | Multi-agent local/mobile supervision; Codex shared-session surfaces. | U: remote/surface handoff does not establish cross-provider context handoff. [Agent guide](https://hapi.run/docs/guide/agents), [shared Codex sessions](https://hapi.run/docs/guide/codex-shared-sessions) |
| 45 | **AionUi** | ACP agents and parallel conversations. | Agent-bound conversations in reviewed docs; no transferred existing conversation verified. [ACP setup](https://github.com/iOfficeAI/AionUi/wiki/ACP-Setup) |
| 46 | **JetBrains IDE ACP / Air / Junie** | Multiple agents in the project and ACP integration. | U: no source-verified existing native conversation transfer in this hunt. Grouped ecosystem, not three invented confirmations. [Agent controls](https://www.jetbrains.com/help/phpstorm/agents-in-phpstorm.html), [ACP architecture](https://blog.jetbrains.com/idea/2026/08/how-to-use-ai-agents-in-intellij-idea-with-acp/) |
| 47 | **Vibe Kanban** | Many native agents, shared workspace files. | Explicit separate conversations: new sessions do not inherit previous chat history. [Workspace sessions](https://vibekanban.com/docs/workspaces/sessions) |
| 48 | **Nimbalyst / historical Crystal** | Native agents; sibling-session brief command. | Started native-provider change blocked; new-session brief is different. Count predecessor once. [Provider rules](https://github.com/nimbalyst/nimbalyst/blob/main/docs/AI_PROVIDER_TYPES.md), [new-session brief](https://github.com/nimbalyst/nimbalyst/blob/main/packages/extensions/planning/claude-plugin/commands/launch-new-session.md) |
| 49 | **CodexMonitor** | Codex app-server workspace/thread supervisor. | Reviewed upstream is single native provider; Codex model picker is not Claude→Codex. [README](https://github.com/Dimillian/CodexMonitor/blob/main/README.md) |
| 50 | **Opcode / Claudia** | Claude Code UI and custom agents. | Reviewed scope is Claude, not shipped cross-provider handoff. [Repository](https://github.com/winfunc/opcode) |
| 51 | **CloudCLI / Claude Code UI** | Multiple native providers' sessions in web/mobile UI. | U: multi-provider availability does not establish transfer. [Repository](https://github.com/siteboon/claudecodeui) |
| 52 | **Emdash** | Multi-agent workspaces, Chat/TUI and durable conversations. | U: no exact cross-agent continuation mechanism established in reviewed material. [Repository](https://github.com/generalaction/emdash), [architecture](https://github.com/generalaction/emdash/blob/main/CONTEXT.md) |
| 53 | **CC Switch** | Tool profiles, endpoint/proxy routing, MCP/skills/prompts. | Configuration switching, not verified cross-harness conversation transfer. [Official repository](https://github.com/farion1231/cc-switch) |

We did not count related forks, historical names or similarly named unrelated repos twice. Examples: Crystal/Nimbalyst, Claudia/Opcode, Windsurf/Devin Desktop, overlapping `livesam/AI-continues`/`cli-continues`, and unrelated repositories also named Tandem. This is a substantial discovery set, **not an exhaustive census of every commercial/private tool**.

## 4. How the important implementations really work

### Synara: simple fresh native execution, stable app history

1. While agents work, Synara projects messages, activities and current provider/native-session facts into thread-keyed SQLite tables. Full stored history is separate from the bootstrap window.
2. Switching validates idle state, approvals/questions and target compatibility. Same-provider instance/account handoffs are rejected by this path and still require a new thread. The reactor also checks source-specific pending sends/background tasks.
3. It clears the native resume cursor and stops the source, binds a target session to the same application thread, then adds prior canonical transcript to the next turn.
4. Return to Claude uses fresh native execution, not automatic revival of the first Claude native thread. If target startup fails, it restores source selection but prepares canonical bootstrap for the next fresh source session.

Sources: [projection schema](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/persistence/Migrations/005_Projections.ts#L22), [validation](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/decider.ts#L149), [runtime transition/recovery](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/Layers/ProviderCommandReactor.ts#L6777).

The exact builder uses **six completed individual messages**, each capped at 2,400 characters, plus older deterministic snippets capped at 320 characters apiece. Total transcript bootstrap is capped at 32,000 characters and can be reduced at turn assembly. These JavaScript lengths count UTF-16 code units, with surrogate-safe clipping, not tokens or Unicode scalar values. The filter is completed/nonempty user/assistant messages, not proven assistant-final-only. Older snippets are not an LLM summary. [Builder](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/handoff.ts).

### T3 Code: stable app history with optional native reuse and delta

Verified current source at the pinned commit; packaged-release parity and live recall were not tested.

T3 separates application thread, provider-native thread and live provider process. Its planner checks instance/continuation compatibility and capabilities. Returning can select a previous provider thread; orchestration prepares the runs that target missed rather than pretending old native history already includes them. Context preparation selects canonical history deterministically; delivery can inject public historical messages on capable Codex versions or render attributed context. Delivery persists `pending`, `injected` and `inline` states and treats ambiguous acceptance as requiring fresh native state. [Planner](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/apps/server/src/orchestration-v2/ProviderSwitchService.ts), [preparation](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/apps/server/src/orchestration-v2/ContextHandoffService.ts), [delivery](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/apps/server/src/orchestration-v2/ContextHandoffDelivery.ts).

Its budget is capacity-aware rather than a five-exchange rule. It preserves selected messages, useful outcomes, original intent and references to omitted history; the current request stays separate. The configurable default allowance is 16,000 with conservative one-token-per-UTF-8-byte accounting, attribution/escaping overhead and a separate 64,000-byte import ceiling. These are bounds/estimates, not measured tokenizer counts. This is richer than AO needs initially. [Budget source](https://github.com/pingdotgg/t3code/blob/dacd2cb649eea06a00244bd181eb400735e923d5/packages/provider-core/src/server/handoffBudget.ts), [user contract](https://github.com/pingdotgg/t3code/blob/main/docs/user/portable-handoffs.md).

### agentctl: native sessions plus durable delivery receipts

One canonical session maps to Claude and Codex native IDs. After source exit it captures public transcript delta. Claude receives staged context at the next actual `UserPromptSubmit`; Codex reads native history and uses a version-supported injection method. Cursors/receipts prevent blind duplicate replay, and exclusive workspace leases prevent two active owners. Its Claude payload has a 9,500-character bound. Published zero-prompt smoke checks are not evidence of semantic recall. [Claude delivery](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/cli/src/native_hooks.rs#L703-L754), [Codex adapter](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/provider-codex/src/adapter.rs#L523-L593), [cursor projection](https://github.com/leofmarciano/agentctl/blob/7997916508dd31e1c3643204c00d1cc19d08986e/crates/transcript/src/projection.rs#L59-L143).

### Tandem: native scrollback comes from rewriting native history

Tandem tails active history, normalizes it, translates completed tool pairs, and writes provider-native shadow histories with directional sync cursors. A switch normally waits using provider-specific completion signals, exits source, synchronizes, then resumes target's paired native ID. Its documentation warns that older Claude versions can flip immediately and Codex can use quiescence/timeout fallbacks; this is not a universal hard idle guarantee. This is why its target native TUI can display old conversation. It is not something a bootstrap prompt automatically achieves. Excluded reasoning and generic compaction notices are not recovered hidden state. Version compatibility, orphan tool handling, atomic writes and target boot caching all matter. [Boundary limitations](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/docs/how-it-works.md#L49-L64), [sync engine](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/src/tandem/sync.py#L43), [flip loop](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/src/tandem/flip.py#L152), [converter](https://github.com/Bhavya6187/tandem/blob/d5ff75e995085d06acb2ce05c414d51bb9099bdb/src/tandem/converter.py#L46).

### Orca/Superset/1Code: good handoff, different conversation identity

Orca prefers an exact source transcript pointer and focused/full read instructions; fallback is terminal capture. Superset actually reads supported native transcripts first, then terminal stream/screen, sanitizes/caps them and opens a new target. Its prompt builder credits Orca. 1Code exports bounded Markdown and opens a sibling subchat. All are useful practical references, but none should be described as the same durable chat merely because the workspace is unchanged. [Orca preparation](https://github.com/stablyai/orca/blob/7de3710dbed7466ebf0ff7edc84738ce9906c4f7/src/renderer/src/components/terminal-pane/terminal-agent-session-continuation.ts#L49), [Superset retrieval](https://github.com/superset-sh/superset/blob/4ec5b6e287d380480a71b4b2ad98976354c82d4f/packages/host-service/src/terminal/terminal.ts), [1Code export](https://github.com/21st-dev/1code/blob/9f1bc76fa4372c18c565b5a4f8daf38ae3595f0e/src/renderer/features/agents/lib/export-chat.ts).

### catchup/Pi: the most useful compaction-format references

catchup's post-compaction selector keeps the last checkpoint, explicitly retained earlier entries and newer material. Its Codex reader handles supported public replacement history and skips opaque encrypted compaction items; a compaction occurrence need not yield readable summary text. Pi's own session tree explicitly stores semantic summary, retained-entry boundary and provenance. These are better references for checkpoint semantics than a generic timeline activity. [catchup selection](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/cli/cli.go#L1242-L1273), [Codex reader](https://github.com/wilbeibi/catchup/blob/50ac5ca88628f5785f81318e4a7841a96f409585/internal/codex/codex.go#L375-L422), [Pi format](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/session-format.md).

## 5. Budget and switching tradeoffs

| Product/path | Bound actually found | Payload | Important qualification |
| --- | --- | --- | --- |
| Synara | 32,000 transcript-bootstrap characters; turn assembly can reduce it | Six individual recent messages ≤2,400 each + older snippets ≤320 each | Not native checkpoint selection or final-only exchanges. |
| T3 Code | Default 16,000 conservative budget units; configurable; separate 64,000 imported bytes | Budget-selected attributed history/outcomes plus recovery references | One UTF-8 byte conservatively charged as one token; capacity reserves can reduce allowance. |
| agentctl Claude | 9,500 characters in hook-delivered context | Undelivered canonical projection | Transport-specific cap, not a universal provider token budget. |
| Orca | 36,000 fallback transcript characters | Pointer or newest terminal text | Pointer/full-read mode does not cap eventual model input; wrapper overhead is separate. |
| Superset | Normally 36,000 transcript characters | Parsed native text or terminal fallback | Transcript bound, not strict total prompt size; no automatic secret redaction in inspected builder. |
| 1Code | 50,000-character transcript-tail selection before added overhead | Tail-retained Markdown attachment | Header/omission text can exceed the tail budget; attachment consumes context; archived. |
| claude-codex-bridge | Default 120,000 transcript characters | Native synthetic history; head 30%/tail 70% after redaction | Different continuity/cost target, substantial schema coupling. |
| Voyager | Presets 4k/20k/100k estimated tokens | Ranked continuation sections | Uses chars/4 estimation; not exact tokens or a universal hard ceiling. |
| catchup | Per-entry byte limits and large-payload warning | Selected native public history | Not a 32k total-bootstrap guarantee. |

Sources and exact code paths for every row are in the four evidence notebooks linked below. These bounds are not directly comparable without their units, surrounding overhead and target context reserves.

**No competitor benchmark was run here.** A small path-only prompt is not necessarily cheap: the target may then read a huge transcript. No inspected handoff establishes transfer of another provider's remote prompt cache. Native resume may preserve useful provider-specific state, but cache hits, billed tokens and first-answer latency depend on actual provider behavior and must be measured. Do not claim “zero-cost switch” because preparation itself performs no LLM call.

## 6. The compaction-summary finding that matters for AO

Three distinct values must not be conflated:

| Value | Example | Valid as semantic checkpoint? |
| --- | --- | --- |
| Lifecycle/display label | “Context compacted,” “Freed 11k tokens” | **No.** Says an event happened. |
| Deterministic history digest | Snippets from older user/assistant messages | Useful fallback, but **not** provider compaction. |
| Readable native semantic checkpoint | Actual condensed goals, decisions, work and constraints, with covered-history boundary | **Yes**, within its captured coverage and completeness limits. |

Synara's activity projection uses the first kind for routine compaction events; its handoff builder uses the second kind. It does not select the third kind from its activities. A payload may hold provider-dependent detail, but its existence is not proof of semantic checkpoint capture or use. [Activity projection](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/providerRuntimeActivityProjection.ts#L1125), [handoff builder](https://github.com/Emanuele-web04/synara/blob/3e3e41c3b312ad95f62c33b155354b099bf36e2e/apps/server/src/orchestration/handoff.ts).

Read-only inspection of this AO checkout (`4c75fe44356d1821833c271deea2b1c158051c2b` plus pre-existing uncommitted work) found:

- [Handoff builder](../../backend/internal/service/chat/handoff_context.go) selects the last completed compaction activity's `Summary` as `CompactionSummary`.
- [Codex driver](../../backend/internal/adapters/chatdriver/codexappserver/conversation.go), `settleCompaction`, overwrites `ev.Summary` with `compactionSummary(before, after)`. The helper explicitly labels a timeline row; it is not the semantic checkpoint body.
- [ACP driver](../../backend/internal/adapters/chatdriver/acp/conversation.go), `settleCompaction`, can replace captured compaction text with a freed-token label when counts are present. Semantic text in that activity field is therefore not guaranteed.
- [TUI switch code](../../backend/internal/session_manager/provider_switch_tui.go) calls optional `AgentTUIHandoff.ReadTUIHandoff`. Repository search found a test implementation but no built-in production adapter implementing it. Its metadata fallback is not proof of universal checkpoint-plus-five-exchange capture.

These are specific gaps, not a request to rebuild everything. **Recommended correction:** keep semantic checkpoint text separate from UI labels, identify its provider/native generation/coverage and preserve capture capability honestly. If unreadable, missing or encrypted, fall back to bounded canonical public history. Do not add a separate model summarization call merely to satisfy the name `summary`. No application code was changed by this research.

A native compaction is also not necessarily “everything before this timestamp replaced by one string.” It may retain earlier messages or multiple public items. Final assistant answers are not guaranteed comprehensive work summaries either. Carry only facts we actually have and provide recovery references for omitted evidence.

## 7. Ideal approach for AO, given our constraints

This is a recommendation, not a new implementation plan or proof the current code already satisfies it. Keep the agreed roughly **5,000 production LOC soft target** and approximately **1:4 production:test LOC aspiration**, without inflating test code to chase a numeric ratio.

### 7.1 Product contract

- One AO session, canonical conversation and workspace throughout. Switching changes execution ownership/native generation, not history storage location.
- Full existing AO chat history remains visible and append-only across provider boundaries. Bootstrap clipping never deletes stored/UI history.
- Idle-only switching: no active turn, unresolved approval/question, or uncontrolled source background mutation. Fence late source events after ownership changes.
- Use actual native provider adapters, not a new AO inference proxy masquerading as Codex/Claude.
- One daemon switch operation for desktop chat, terminal entry points and CLI. Keep frontend thin.
- Refresh provider label, model/effort, permissions, config, skills/tools and relevant capabilities together. A stale Claude selector on Codex is a correctness issue, not cosmetic polish.

### 7.2 Default handoff contents

1. **Tiny stable metadata/intent anchor:** AO session and source attribution, task title/original goal where available, exact cwd/worktree/branch and essential policy constraints. Do not silently translate approval settings into broader permission.
2. **Latest readable semantic checkpoint**, if actually captured, with coverage/provenance. If absent, a clearly labelled deterministic older-history digest; do not call it native compaction.
3. **Newest five completed human-user/final-assistant exchanges**: up to ten messages, not five individual messages. Exclude tool chatter, streaming deltas, private reasoning and synthetic automation prompts. Preserve full text when it fits.
4. **Small omission/recovery note:** what was clipped/unavailable and how to retrieve exact saved public history if necessary, **only when an existing session-scoped tool/path is actually accessible to the target**. Otherwise explicitly say exact-history retrieval is unavailable; do not emit a fake pointer. An optional compact test/changed-path fact can help when durable data already exists; do not create a second expensive summarization subsystem.

A long task's original constraints can fall outside the last five exchanges and be absent from a clipped checkpoint. A tiny task-intent anchor is worth more than arbitrary verbose metadata. **There can also be a middle gap:** important exchanges after the latest checkpoint but before the newest five are covered by neither. The packet must mark that uncovered range and give a real recovery route where available, otherwise disclose reduced continuity; checkpoint-plus-five is intentionally budgeted context, not complete conversational memory. Adding an agent-accessible paged reader is a possible next step, not an assumed existing capability. If a stopped/interrupted turn changed files without a final answer, acknowledge that gap instead of inventing a final response; current files/git remain authoritative.

### 7.3 Budget policy

Keep the agreed **32,000-character maximum for the complete serialized bootstrap**, including role labels, headings, metadata, separators and omission markers. Define the character unit consistently across Go/frontend, avoid breaking UTF-8/grapheme content, and report byte/token estimates separately. Target-specific request/context limits can require a smaller effective budget.

Priority should follow the user's earlier preference: tiny mandatory metadata, newest complete exchanges at full length, then as much real checkpoint text as fits. Trim expendable metadata and fallback digest first; reduce checkpoint text with explicit truncation, then drop oldest complete exchanges if still necessary. If even one newest exchange cannot fit, deterministic within-exchange clipping is unavoidable and must be marked. Never shorten the user's *new* prompt to hide an oversized handoff. Stored source messages remain intact. This ordering is a recommended interpretation of the prior full-recent-exchange preference, not a claim all checkpoints can always be preserved intact.

Add a target-capacity safety check/reserve when information is available. Characters are not tokens—especially with non-English text, code, escaping and attachments. The 32k cap alone cannot guarantee every target model accepts the request.

### 7.4 Switch-back and TUI scope

**Initial simple default:** fresh native target on each cross-provider switch, with canonical bootstrap. It is easier to reason about stale state and fits the chosen minimal architecture. This is a recommendation, not authorization to remove a working native-resume path.

**Later measured optimization:** resume a mapped prior native thread only if compatibility is known and missed canonical work is delivered once, with coverage/cursors/receipts. Merely remembering a native session ID is insufficient. T3/agentctl are references for this step.

**Terminal honesty:** switching actual native TUIs in the same AO pane is feasible. Identical old messages inside each provider's own native scrollback is a different requirement. Either expose AO's retained timeline/history alongside the terminal, or explicitly invest in Tandem-style transcript synthesis as a separate feature. Do not promise the latter from a bootstrap-only implementation.

### 7.5 Provider breadth without a second architecture

Use Synara's current ten-kind enum as a comparison checklist, not a guarantee every integration is installed/ready or that all 90 directions are tested. Map it against AO's existing adapters before committing to new ones. For each adapter record:

| Capability | Why it must be explicit |
| --- | --- |
| Canonical human/final-message capture | Determines whether the five-exchange policy is actually available. |
| Readable semantic checkpoint capture | A compaction signal alone is insufficient. |
| Idle/approval/background-task detection | Prevents unsafe source ownership transfer. |
| Fresh launch and bootstrap acceptance | “Process started” is not “context delivered.” |
| Native resume and missed-work delta | Optional optimization; needs compatibility and idempotency. |
| Model/config/permission catalog refresh | Prevents stale source-provider controls. |
| Desktop chat vs native TUI support | Similar provider names can expose different protocol capabilities. |
| Fixture, native-reader and live recall evidence | Makes support claims auditable instead of multiplying a provider count. |

Unsupported checkpoint capture should degrade to honest history fallback, not hide the switch option universally. Unsupported safe ownership/bootstrap transport may require disabling that specific path. Reuse AO's existing ledger, driver boundaries and service API; no separate transcript index, full tool transcoder or provider proxy is required for the baseline.

## 8. Ideas retained from the hunt

Priority meanings: **Now** fixes the promised minimal contract; **Next** adds modest utility after correctness; **Later** requires measured value/extra maintenance. Effort is qualitative, not a fabricated LOC estimate.

| Idea | Origin | AO value / limitation | Priority; effort |
| --- | --- | --- | --- |
| Typed checkpoint provenance/coverage | Pi, catchup; Synara counterexample | Avoid passing a display label as memory; handle retained/opaque checkpoint material. | Now; small–medium |
| Stable session + generation fencing | Synara, T3 | Old callbacks cannot mutate new provider state; same UI history survives. | Now; medium |
| Exact source/native binding | Orca, agentctl, Agent Switcher | Never scrape “latest transcript in this directory” and import another task/account. | Now; small |
| Shared daemon switch command | Voyager, OpenCode's session API | Desktop and TUI obey identical validation and failure rules. | Now; small–medium |
| Launch vs delivery vs uncertain acceptance states | Orca, T3, agentctl | Avoid reporting success before context was accepted; safe retry after disconnect. | Now; medium |
| Whole-bootstrap size accounting and visible omission counts | Synara, Superset, bridge | Bound cost without silently dropping crucial messages. | Now; small |
| Refresh every provider-owned selector | Kilo, Roo, Goose, VS Code | Prevent stale models/effort/config/permissions after switch. | Now; small |
| Original-task anchor | T3 | Preserve long-running intent that last five exchanges miss. | Now; small |
| Historical text treated as data | Superset | Source messages/tool output must not become privileged instructions; disk is authoritative. | Now; small |
| Compatibility evidence levels | showagent, Tandem | Separate parsing, native rendering, launch, delivery and actual recall claims. | Now; tests/docs |
| Compact optional “what transfers” preview | Superset, Turntrail, Agent Switcher | Characters, estimates, checkpoint availability, omitted items; avoid a complex editor first. | Next; small–medium |
| Read-only paged recovery of omitted AO history | T3, AgentHandoff, Turntrail | Keep default tiny; retrieve needed evidence rather than replay everything. Access-check against same session. | Next; medium; reuse existing tools if possible |
| Resume prior target + exactly-once delta | T3, agentctl, context-bridge claim | Potentially lower repeated history/native setup; stronger crash/retry obligations. | Later; medium–large |
| Optional focused/full history modes | Orca | Explicit user tradeoff, not silent unlimited reads. Preserve hard default. | Later; small–medium |
| Prewarm target process | Tandem | Could reduce latency; provider boot caching and two-owner risks. Measure first. | Later; medium |
| Native-TUI historical scrollback transplantation | Tandem, showagent, bridge | True native history continuity; private-format and tool-vocabulary maintenance. | Separate later project; large |

Ideas deliberately rejected for the baseline: an extra LLM summary on every switch (latency/cost), uncontrolled full-transcript reads (unbounded eventual input), rewriting foreign tool calls as live target state (compatibility/safety), a new independent canonical ledger (AO already owns one), and auto-switching during active work (ownership ambiguity).

## 9. What should be verified before AO claims this is complete

This is a proposed acceptance matrix, not a report that these tests ran during research.

| Test | What it catches |
| --- | --- |
| Claude A → Codex B → Claude; unique decision introduced only during B must be recalled | Stale native resume, missing delta/bootstrap, wrong source binding. |
| Five/six human queries with intermediate commentary/tool activity, then switch | Final-only selection vs accidentally replaying every assistant update. |
| Readable automatic/manual checkpoint; missing checkpoint; generic label; opaque native item | False semantic-summary assumptions and provider-specific gaps. |
| Important decision after checkpoint but older than newest five exchanges | Unacknowledged middle-history gap; inability to recover omitted constraints. |
| Oversized final answer/summary/metadata; non-English and emoji content | Wrong units, malformed Unicode, broken serialized 32k ceiling, silent clipping. |
| Pending turn, approval, user question and background task | Unsafe switch while source still owns work. |
| Target unavailable/auth failure/start failure/acceptance disconnect; retry/restart | False success, duplicate context, lost input, two active writers, unrecoverable state. |
| Late source events after target ownership and app restart | Generation-fencing and durable ownership recovery. |
| Model/effort/config/skills/permission selectors immediately after switch and restart | Stale source-provider UI/catalogs. |
| Full AO history before/after switch; source storage integrity | Bootstrap trimming accidentally treated as history retention/deletion. |
| TUI directions on each actually installed provider | Safe process exit/start, actual bootstrap delivery, honest scrollback promises. |
| Warm/cold repeat measurements: prepare/start/first response, input/cache/uncached tokens where reported | Whether resume/delta/prewarm improves real cost or latency. |

For an unsupported/unavailable provider: verify adapter contract/fixtures, document the live-test gap, and do not present it as authenticated end-to-end success. Cover adapter capabilities and representative pair interactions rather than inventing evidence from N×(N−1).

## 10. Evidence notebooks and reproducibility

Full per-product findings, additional source links, qualifications and research-derived ideas:

- [Canonical clients, model switching, and compaction formats](provider-switching-canonical-findings.md) — 14 products.
- [Native handoff tools, Synara/Tandem/Orca/Voyager deep traces](provider-switching-native-findings.md) — 14 products.
- [Desktop, ACP and terminal supervisors](provider-switching-desktop-findings.md) — 16 families.
- [Bridges, routing utilities, and AO-specific source gaps](provider-switching-bridge-findings.md) — nine additional tools.

Selected public source snapshots were shallow-cloned to temporary directories, read without dependency installs/builds/execution, and linked by checked commit:

| Repository | Inspected commit |
| --- | --- |
| Synara | `3e3e41c3b312ad95f62c33b155354b099bf36e2e` |
| T3 Code | `dacd2cb649eea06a00244bd181eb400735e923d5` |
| Tandem CLI | `d5ff75e995085d06acb2ce05c414d51bb9099bdb` |
| agentctl | `7997916508dd31e1c3643204c00d1cc19d08986e` |
| Voyager/sessionFlow | `ee86d8b079e927ea064a4860e0d6e87982d746b9` |
| Orca | `7de3710dbed7466ebf0ff7edc84738ce9906c4f7` |
| Superset | `4ec5b6e287d380480a71b4b2ad98976354c82d4f` |
| 1Code | `9f1bc76fa4372c18c565b5a4f8daf38ae3595f0e` |
| Vibeyard | `19bc19f0fac04f11e7d9e5cf68ed75ac7d291566` |
| showagent | `39a97bda8203314a0b3f43992b8da29c00d2cb21` |
| claude-codex-bridge | `56e0776b3aa8a8ffe42c55c78b1c508b8811440a` |
| catchup | `50ac5ca88628f5785f81318e4a7841a96f409585` |
| Omnigent | `ef0adfb2066582b9abc97072e0e4ca8051dae6ad` |

Other primary-source links can move. Closed-source internals, source-only features' released availability, and unaudited linked-session claims remain explicitly qualified. User sign-ins, paid trials and live competitor tests were skipped; none was required to complete this evidence report. Existing AO application changes were preserved, not modified or committed by this research.

**Bottom line:** borrow Synara's stable chat/fresh-runtime simplicity, T3's ownership/budget/recovery discipline, agentctl's delivery honesty and Pi/catchup's real checkpoint semantics. Treat Tandem-style native history mirroring as an optional separate investment. AO's advantage can be a predictable, small, real-agent handoff that works consistently in chat and terminal without pretending to transfer hidden state or provider caches.
