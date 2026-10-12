# Discovery 6 - Community and launch channels (provider switching / agent handoff)

Research date: 2026-10-10/11. Method: HN Algolia API (stories, Show HN, Launch HN, comments), GitHub (`gh api`, `gh search`), WebSearch/WebFetch of READMEs, product pages, dev.to, X search snippets.
Nothing was cloned or run. Everything on the web is treated as data.

Access limits (be aware when weighing evidence):
- Reddit: WebFetch and WebSearch both refuse reddit.com. No first-hand Reddit evidence; only one Reddit post (CodePass) surfaced through a search snippet. Reddit demand is therefore UNDER-sampled; HN, dev.to and X snippets stand in.
- X/Twitter: fetch returns 402. X items below come from search-result snippets only (dates decoded from tweet IDs).
- Product Hunt: only Moxie page was fetchable. YC launches were swept through Launch HN titles, not the YC directory.
- "VERIFIED" = official docs or the project's own README/source that I read. It means the source states it, NOT that I ran it. "CLAIMED" = marketing page, HN post text, third-party blog or search snippet only.

Class key: A = cross-agent handoff (same task continues on a different agent with context carried); B = model/provider picker inside one harness; C = side by side, no conversation handoff; F = automatic or semi-automatic failover (context note given); U = unclear / adjacent.

## Products found

### Class A - cross-agent handoff

| Name | URL | Repo | Class | Claim | Verified/Claimed | Source+date |
|---|---|---|---|---|---|---|
| Skillsync + txcript (YC W26) | https://skillsync.com | https://github.com/skillsynchq/txcript (155 stars, pushed 2026-09-29) | A | Converts a session (messages, reasoning, tool calls) so it continues in another agent; "Pick up any of them in a different agent with Cmd-K". Deterministic, no AI in the conversion. Agents: Claude Code, Codex, Hermes, OpenClaw, Grok, Antigravity, pi, OpenCode, Cursor. Mac app + CLI + MCP. | VERIFIED (site + OSS core); founders' comments on HN | Launch HN https://news.ycombinator.com/item?id=49743049 (2026-09-17) |
| Continuo | https://usecontinuo.dev | not seen | A | "Converts the session into the target agent's native format," opens it where you left off. Claude Code, Codex, OpenCode. | CLAIMED (site) | site fetched 2026-10; mentioned by yoavfr in Skillsync thread (2026-09-17) |
| Baton | https://github.com/Mvp2o-ai/baton | same (0 stars, pushed 2026-09-17) | A | Sidecar: swap the CLI in the same terminal pane (Claude Code / Codex / Cursor CLI), runs txcript to carry the session, relaunches with resume flags; links personal skills across. | VERIFIED (README) | README, pushed 2026-09-17 |
| claude-codex-bridge (`agent-switch`) | https://github.com/nerln/claude-codex-bridge | same (2 stars, pushed 2026-08-12) | A | Moves a live conversation into a NEW NATIVE session of the other agent, both directions; same worktree. Marked experimental; version-gated; tested 2026-08-01 on Claude Code 2.1.219 / Codex 0.146.0-alpha.9. | VERIFIED (README) | README 2026-08 |
| OmniSession | https://github.com/bvolpato/omnisession | same (5 stars, pushed 2026-10-09) | A | Writes a new session in target agent's own format, read-back verifies, rolls back on failure; falls back to a "semantic handoff" file presenting redacted history as untrusted context. Claude Code, Codex, OpenCode, Pi... | VERIFIED (README) | README, pushed 2026-10-09 |
| continues (cli-continues) | https://github.com/yigitkonur/cli-continues | same (1,564 stars, pushed 2026-05-07, MIT) | A | Resume any of 16 CLIs' sessions in another; generates a structured handoff document (messages, file changes, tool samples) injected into the target. Presets minimal/standard/verbose/full. | VERIFIED (README) | Show HN https://news.ycombinator.com/item?id=47075089 (2026-02-19) |
| CatchUp | https://github.com/wilbeibi/catchup | same (77 stars, pushed 2026-09-28) | A | `catchup fork --into` seeds another agent with a transcript; strips successful tool calls and reasoning; 13 agents incl. Cursor, Cline, Kimi, Antigravity, Amp. | VERIFIED (README) | README 2026-09 |
| casr (Cross Agent Session Resumer) | https://github.com/Dicklesworthstone/cross_agent_session_resumer | same (122 stars, pushed 2026-09-22) | A | Converts Codex/Claude/Gemini sessions through a canonical IR and writes them in the target's native format. | VERIFIED (repo description) + third-party write-up | https://codex.danielvaughan.com/2026/06/01/codex-cli-cross-agent-session-portability-continues-casr-handoff/ (2026-06-01, upd. 2026-10-10) |
| Rses (`rses-cli`) | npm `rses-cli` (no repo URL in HN record) | GitHub link per author, not seen | A | Reads a Codex session (task, git log, recent turns, session pointer), builds a structured handoff, launches Claude with it; all 6 directions among Claude Code, Codex, OpenCode. ~1,200 lines. | CLAIMED (HN post) | https://news.ycombinator.com/item?id=47528990 (2026-03-26) |
| Handoff (sahir2k) | https://github.com/sahir2k/handoff | same (4 stars, pushed 2026-03-02) | A | Claude Code <-> Codex; reads full local history; full / focused / minimal detail levels; also passes CLAUDE.md and MEMORY.md. | CLAIMED (HN author text) | https://news.ycombinator.com/item?id=47206954 (2026-03-01) |
| OpenAI codex-plugin-cc: `/codex:transfer` | https://github.com/openai/codex-plugin-cc | same (34k stars, pushed 2026-07-08) | A | Creates a persistent Codex thread from the current Claude Code session via Codex's external-agent session importer; prints `codex resume <id>`. Also `/codex:rescue` (delegate). OFFICIAL. | VERIFIED (README) | README; X post about it 2026-08-25 https://x.com/Voxyz_ai/status/2092273872217186312 |
| xAI Grok Build: `grok import` + grok-build-plugin-cc | https://github.com/xai-org/grok-build-plugin-cc ; docs https://docs.x.ai/build/cli/reference | same (257 stars, pushed 2026-08-04) | A | `/grok-build:import` imports the current Claude transcript into Grok (`grok -r <id>`); CLI reference lists `grok import` from Claude Code. Third-party listing says release 0.2.101 (2026-07-13) picker shows Claude Code, Codex, Cursor sessions. OFFICIAL. | VERIFIED (README + docs search snippet) | README 2026-08; docs.x.ai |
| Orbital (zqiren) | https://github.com/zqiren/Orbital | same (313 stars, 969 commits, GPL-3.0, pushed 2026-10-10) | A (desktop app) | Desktop app (PyInstaller + pywebview + FastAPI). "Codex hits its usage limit mid-task. Switch to Claude Code in the same conversation." Agent picker shows remaining quota per agent. Context lives in project files (PROJECT_STATE.md, DECISIONS.md, LESSONS.md, queue.json), not in a transcript copy. Tagline: "Context is yours. Agents are replaceable." Manual switch, not automatic. | VERIFIED (README) | README; HN https://news.ycombinator.com/item?id=49740917 (2026-09-17) |
| Kangentic | https://github.com/Kangentic/kangentic | same (126 stars, pushed 2026-10-08) | A (desktop kanban) | Move a card between columns with different agents and "the next agent starts with the full history" (Claude, Codex, Gemini, Antigravity, Qwen, Kimi, Grok, OpenCode). Mechanism not documented. | VERIFIED (README claim) | README; GitKraken roundup https://gitkraken.com/blog/best-ades-for-multi-agent-coding-in-2026 (2026-10-01) |
| Codeg | https://github.com/xintaofei/codeg (docs.codeg.app) | same (3,881 stars, Apache-2.0, pushed 2026-10-10) | A/C | Desktop/server workspace aggregating sessions from Claude Code, Codex, OpenCode, Pi, Grok Build. Import/search/resume past sessions and "hand it to a different agent"; `@agent` delegation adds agents in the same thread. No auto failover. | VERIFIED (README) | README 2026-10 |
| Clinch | https://clinch.sh/ | not seen | A | Local-first Warp fork; "One-click session transfers between Claude Code and Codex"; auto-resume on restart; fork a session into a new tab. Transfer internals not described. | CLAIMED (site) | Show HN https://news.ycombinator.com/item?id=49353724 (2026-08-18) |
| SpecWeave 3 | https://github.com/anton-abyzov/specweave | same (165 stars, MIT, pushed 2026-10-10) | A + F | Say "hand off" in one tool and "pick up" in another (Claude Code, Codex, Grok): `specweave handoff` releases task claims, pushes branch + snapshot of uncommitted edits; state in committed files (ledger.jsonl, .specweave/memory). Optional `auto-handoff on`: hands off at 90% of the 5-hour or weekly limit (Claude via status line, Codex via Stop hook). | VERIFIED (README) | Show HN https://news.ycombinator.com/item?id=50026741 (2026-10-09) |
| ai-memory (akitaonrails) | https://github.com/akitaonrails/ai-memory | same (9,109 stars, pushed 2026-10-10) | A | Shared memory across 20+ harnesses; typed handoff protocol ("each one has an owner and is claimed exactly once"); `ai-memory run` relaunches the workstream in a different harness (Claude Code -> Codex). | VERIFIED (README) | README 2026-10 |
| sno-station | https://github.com/sno-ai/sno-station | same (500 stars, pushed 2026-10-10) | A (+ manual F) | Claude Code and Codex share encrypted memory + mailbox; when one nears quota, the other takes over via `sno reach call`. Outgoing agent writes a brief (done / half-done / exact commit); receiver reports size + checksum before starting; sender releases only after the ack. Evidence is one overnight example (~14 min handover). | VERIFIED (README) but single anecdote | README 2026-10 |
| Mida Context | https://github.com/damli40/mida-context | same (pushed 2026-10-09) | A | Hooks in Claude Code and Codex save a checkpoint; open the other agent and type "Continue." Checkpoint = verbatim original request, goal, progress, decisions + reasons, rejected approaches, constraints, next step. Encrypted, Monad-testnet permissions, pre-release. | VERIFIED (README) | HN comment https://news.ycombinator.com/item?id=49919704 (2026-10-01) |
| claw-orchestrator | https://github.com/Enderfga/claw-orchestrator | same (588 stars, pushed 2026-10-09) | A | `session_handoff` "replays a live conversation onto another engine in the same workspace" (Claude Code, Codex, Antigravity, Grok Build, OpenCode); original keeps running; can move to a cheaper model. No failover claim. | VERIFIED (README) | README 2026-10 |
| project-continuity | https://github.com/seriousz158/project-continuity | same (99 stars, pushed 2026-09-22) | A (file-based) | Single `.relay/CURRENT.md` (JSON + Markdown view) with Project / Tasks / Blockers / Evidence / Decisions; `save` releases a writer lease, `resume` acquires it. | VERIFIED (README) | README 2026-09 |
| Contextify | https://contextify.sh/ | not open (Mac app closed; server source-available) | A (pull model) | Indexes every Claude Code and Codex session locally; from inside either tool, search/pull the other's transcripts (slash command / `$` command / MCP). Author switches on rate limits. | CLAIMED (HN author text) | Show HN https://news.ycombinator.com/item?id=48777790 (2026-07-03) |
| Bridle | https://www.bridle.network/ | not seen | A (person to person) | Open protocol to hand context / queue a task / request a run in a teammate's agent; Claude Code, Codex, Cursor, Windsurf, Zed, Cline, OpenCode, Copilot. | CLAIMED (site) | Show HN https://news.ycombinator.com/item?id=49553877 (2026-09-03) |
| dsh-resume | https://github.com/aa2246740/dsh-resume | same (8 stars, pushed 2026-09-29) | A | "Continue Codex, Claude Code, and Cursor work in DeepSeek Harness." | CLAIMED (repo description only) | gh API 2026-10 |
| Handover MCP (44-pixels) | https://mcpservers.org/ru/servers/44-pixels/handover-mcp | not seen | A | Hosted MCP + CLI to move work between Claude Code, Codex and other agents; includes a Claude Code -> Codex recipe. | CLAIMED (directory snippet) | WebSearch snippet |
| ShardStitch | https://shardstitch.com/guides/codex-rate-limit-recovery/ | not seen | A (reconstruct from disk) | Builds a continuation packet from working tree, git history, notes, verification output; cannot recover choices that existed only in the lost chat. Manual run. | CLAIMED (site) | site 2026 (undated) |
| ai-handoff | https://il.linkedin.com/in/tamirnauman | not seen | A | Claude Code -> Codex first; pick which conversations to carry; writes AGENTS.md and .codex/handoff artifacts. | CLAIMED (LinkedIn snippet, weak) | WebSearch snippet |
| Switch Dev (skill) | https://claudskills.com/skills/switch-dev/ | not seen | A (via GitHub issues) | Switch between Claude Code, Cursor, Codex mid-task using GitHub-issue "Resume blocks". | CLAIMED (registry listing) | WebSearch snippet |
| cross-agent-handoff (jopsonaljohnkarl-stack) | https://github.com/jopsonaljohnkarl-stack/cross-agent-handoff | same (1 star, pushed 2026-07-23) | A (file-based) | "one HANDOFF.md baton" between Claude Code, Codex CLI. Several near-identical repos exist (OpenMOSS, zhpy2004, ParaGenie...). | CLAIMED (repo description) | gh search 2026-10 |

### Class F - failover / fallback (context note in each row)

| Name | URL | Repo | Class | Claim | Verified/Claimed | Source+date |
|---|---|---|---|---|---|---|
| Hydra | https://github.com/saadnvd1/hydra | same (20 stars, MIT, pushed 2026-04-21) | F (semi-auto) | Watches CLI output for rate-limit patterns, shows provider picker (one keypress); chain Claude Code -> OpenCode+Gemini -> Codex -> Pi. Context: copies conversation (parsed from Claude JSONL only), uncommitted diff, last 5 commits to CLIPBOARD; user pastes. | VERIFIED (README) | Show HN https://news.ycombinator.com/item?id=47851935 (2026-04-21) |
| Moxie | https://www.producthunt.com/posts/moxie-3 | not seen | F (accounts/models via proxy) | Mac menu bar local proxy for Claude Code; "switch accounts mid-task", same session keeps going; Claude Code "has no idea" the account/model changed. History carries only because the client session is unchanged; a commenter asked about history. | CLAIMED (Product Hunt maker) | Product Hunt, launched ~2026-10 (page says "8d ago") |
| AgentCodeGUI | https://github.com/UnrealFactory/AgentCodeGUI | same (82 stars, MIT, pushed 2026-09-13) | F + C | Multi-account desktop GUI for Claude Code and Codex; on limit continue on another account or auto-resume at reset; up to 6 panels with own engine/model/account. No cross-engine conversation handoff claimed. | VERIFIED (README, Korean) | Show HN https://news.ycombinator.com/item?id=49154498 (2026-08-03) |
| dario | https://github.com/askalf/dario | same (564 stars, pushed 2026-10-10) | F | Local endpoint for Claude + ChatGPT subscriptions; "A 429 on one plan is re-served by the other"; seat pools. README does not say whether history is re-sent on failover. | VERIFIED (README) | README 2026-10 |
| Pi multi-account | https://github.com/Sarrius/pi-multi-account | same (26 stars, pushed 2026-10-03) | F | Account failover/rotation inside Pi across Anthropic, OpenAI Codex, Qwen; `preserveInterruptedContext` (default true) rewrites the interrupted turn into a handoff record, auto-continue resumes from last safe point. Context carried: yes (same harness). | VERIFIED (README) | README 2026-10 |
| willynikes2/agent-orchestrator | https://github.com/willynikes2/agent-orchestrator | same (67 stars, pushed 2026-04-03) | F | Ordered role chains; "next-man-up failover" for Claude, Codex, Gemini. Context via shared knowledge-base server, transcript not stated. | VERIFIED (README) | README 2026-04 |
| CodePass | no URL found (Reddit r/ClaudeAI post) | not found | F | Runs agent in a PTY, detects rate limit/failure, switches to next in chain (Claude Code, Codex, Antigravity, opencode, Cline, Ollama). Author says transcripts are private formats so it keeps a live handoff file + git status/diff for the next tool. | CLAIMED (author Reddit post via search snippet; could not open) | https://r.datuan.dev/r/ClaudeAI/duplicates/1upzdel (Reddit mirror, 403 on fetch) |
| Edgee | https://www.edgee.ai/fallback-models | not seen | F (gateway) | On 429/5xx retries through next model in chain; lists Claude Code and Codex; "Edgee carries the session across the switch"; routing is a Team plan ($30/dev/mo). Same client, model swap only. | CLAIMED (marketing) | page, (c) 2026, undated |
| Inworld Router | https://docs.inworld.ai/router/guides/claude-code | not seen | F (gateway) | If Anthropic is rate-limited or down, fails over to another provider; Claude Code redirected via env vars. | CLAIMED (docs snippet) | WebSearch snippet |
| CCSwitcher / Agent Switch | https://github.com/XueshiQiao/CCSwitcher | same | F (accounts) | Switch Claude accounts; an in-flight session picks up new credentials on next API call. | CLAIMED (snippet) | WebSearch snippet |

Also failover-adjacent, listed elsewhere: SpecWeave (auto-handoff at 90%), sno-station (manual at quota), Orbital (shows quota, manual switch), Skillsync/continues/Rses (built for the limit moment).

### Class B - model/provider picker inside one harness

| Name | URL | Repo | Class | Claim | Verified/Claimed | Source+date |
|---|---|---|---|---|---|---|
| Weave Router | https://news.ycombinator.com/item?id=49911500 (site weaveos.com/router) | github.com/weave-os/router (per HN text) | B | Plugs into Claude Code/Codex, "switches between LLMs" mid-session, escalates to stronger model when off course; claims ~48% lower cost. Benchmark numbers are the author's. | CLAIMED (HN) | Show HN 2026-09-30 |
| Happy | https://github.com/slopus/happy | same (24k stars, pushed 2026-10-10) | B | Desktop app for Claude Code, Codex, Grok + mobile; "Switch models in the middle of a task"; multi-model needs desktop app. | VERIFIED (README) | README 2026-10 |
| Arctic | https://github.com/arctic-cli/interface | same (154 stars, pushed 2026-06-23) | B | "Switch models mid-conversation"; unified limits/sessions across Codex, Gemini, Anthropic, Copilot. Cross-provider detail not stated. | VERIFIED (README, vague) | Show HN https://news.ycombinator.com/item?id=46466958 (2026-01-02) |
| Kepler (GitKraken) | https://gitkraken.com/kepler | n/a | B/C | Change agent per Action: "What you've configured carries over. New agent, same setup." No live-conversation handoff. | VERIFIED (product page) | roundup 2026-10-01 |
| cc-switch | https://enterprisedna.co/resources/ai-pulse/ai-pulse-2026-08-05-cc-switch-a-desktop-app-for-juggling-config-across-six-diffe/ | not seen | B | Desktop app swapping provider config across six coding CLIs (not conversations). | CLAIMED (blog) | 2026-08-05 |
| claude-code-with-codex | https://github.com/fcakyon/claude-code-with-codex | same | B | Local proxy so Claude Code uses a ChatGPT/Codex subscription; flip between Opus and the Codex model in one session. | CLAIMED (search snippet) | WebSearch snippet |

### Class C - side by side, no conversation handoff

| Name | URL | Repo | Class | Claim | Verified/Claimed | Source+date |
|---|---|---|---|---|---|---|
| Bellows | https://dev.to/bellows/i-use-three-coding-agents-so-i-built-one-desktop-app-to-run-them-all-under-one-set-of-rules-jll | n/a | C | Codex, Gemini CLI, Claude in tabs, shared permission rules; no mid-task transfer claimed. | CLAIMED (dev.to, 10-05) | 2026 (year not shown) |
| Metateam | https://www.metateam.ai/ | not seen | C | Many Claude/Codex/Gemini CLI instances as one crew; persona can resume "on a different provider"; no mechanism given. | CLAIMED (site) | Show HN https://news.ycombinator.com/item?id=47274120 (2026-03-06) |
| OpenMOSS claude-codex-handoff | https://github.com/OpenMOSS/claude-codex-handoff | same (42 stars, pushed 2026-07-04) | C | Async JSONL message streams between two LIVE agents in one workspace; message types include `handoff`; claims/leases; ~10 min heartbeat. Collaboration, not takeover. | VERIFIED (README) | README 2026-07 |
| Omnara (YC S25) | https://www.omnara.com/ | n/a | C | Run Claude Code and Codex from anywhere (remote). | CLAIMED (Launch HN title) | https://news.ycombinator.com/item?id=46991591 (2026-02-12) |
| Fundryi/claude-codex-handoff | https://github.com/Fundryi/claude-codex-handoff | same | C | Claude plans, Codex does, results come back (delegation). | CLAIMED (description) | gh search 2026-10 |

### Class U - unclear / adjacent

| Name | URL | Repo | Class | Claim | Verified/Claimed | Source+date |
|---|---|---|---|---|---|---|
| Termem | https://termem.com | https://github.com/leox255/termem (11 stars) | U | Indexes Claude Code, Codex, Gemini, opencode sessions; "Resume the exact past session in the right tool"; MCP lets one agent read another's sessions. | VERIFIED (site) | Show HN https://news.ycombinator.com/item?id=48546126 (2026-06-15) |
| Offload (`/offload`) | https://github.com/ToxicPine/offloads | same (14 stars, GPL-3.0) | U | Hands the task to a remote machine from Claude Code or Codex; does NOT carry chat history (suggests Entire for transcripts). | VERIFIED (README) | Show HN https://news.ycombinator.com/item?id=48555644 (2026-06-16) |
| Happier | https://guides.happier.dev/how-to-move-a-happier-session-to-another-machine-with-session-handoff | n/a | U | Session handoff across machines, same session ID; Claude Code/Codex/OpenCode. Not provider switching. | VERIFIED (guide) | 2026 |
| Entire | https://docs.entire.io/integrations/codex | https://github.com/entireio/cli | U | Git-linked checkpoints; `entire session resume BRANCH` restores the session log, same agent. | VERIFIED (docs snippet) | 2026 |
| claude-handoff-revive | https://github.com/sofumel/claude-handoff-revive | same | U | Claude-only: save minimal state at `.claude/handoff/current.md` before limit, start fresh session. | VERIFIED (README) | README 2026 |
| agent-limit-retry | https://github.com/error0702/agent-limit-retry | same (1 star, pushed 2026-09-25) | U | Claude Code only: exact reset times, handoff before the limit, headless runs resume after reset. | VERIFIED (README description) | Show HN https://news.ycombinator.com/item?id=49831324 (2026-09-24) |
| codex-claude-transfer (`cct`) | https://pkg.go.dev/github.com/ahmojo/codex-claude-transfer | n/a | U | Export/import `.codexbundle` between machines; some snippets also say cross-agent. | CLAIMED (pkg.go.dev) | snippet |
| Etymon | https://github.com/landoncrabtree/etymon | same | U | Converts CLAUDE.md / AGENTS.md / cursor rules / mcp.json into one `etymon.toml`; built because author hops Claude Code -> Codex -> OpenCode on limits. Config only. | CLAIMED (HN) | https://news.ycombinator.com/item?id=49926105 (2026-10-01) |
| Usage trackers (Armada, UsageBuddy, AI Gauge, CodexBar...) | https://armada.mgcrea.io/ , https://www.usagebuddy.com/ , https://github.com/jpajak/ai-gauge , https://github.com/steipete/CodexBar (22k stars) | various | U | Dozens of tools show Claude/Codex/Cursor limits on one screen. Demand signal, not switching. | VERIFIED (titles) | Show HN 2026-06 to 2026-09 |

Already-known names: VS Code docs https://code.visualstudio.com/docs/agents/concepts/sessions say a handoff "carries the full conversation history and context" when moving a session between harnesses (Copilot -> Claude/Codex). New evidence for an already-covered product. Amp: `/handoff` praised vs compaction by an HN user (see Why-break). Superset (Launch HN, YC P26, 2026-05-22) appears on the Launch HN list; no switching evidence found.

Counts (primary class; SpecWeave and sno-station counted in A, also noted under F): A = 27, F = 9, B = 6, C = 5, U = 9. Total distinct products and tools about 56.

## Why users switch

1. Usage / rate limits (dominant reason in 2026 sources)
   - Author of Rses, asked what the point is, said they are mainly hitting rate and quota limits. https://news.ycombinator.com/item?id=47528990 (2026-03-26)
   - "when u hit the limit switch 2 codex" - HN commenter 3dsnano; Claude plans/implements, Codex reviews when limit hits, alternate until clean. https://news.ycombinator.com/item?id=47969740 (2026-04-30)
   - vidarh prototypes on Claude until the weekly cap, moves to Codex, then to cheap open models; "The gap reduces each time." https://news.ycombinator.com/item?id=47701312 (2026-04-09)
   - Ask HN poster wants to move from Codex to Claude Code (or back) keeping task context, cannot afford $100-200 plans per agent. https://news.ycombinator.com/item?id=49904074 (2026-09-30)
   - Continues author: "losing the full conversation and spending 10 minutes re-explaining" after a limit. https://news.ycombinator.com/item?id=47075089 (2026-02-19)
   - Contextify author switches between Claude Code and Codex when he hits rate limits and expects to keep doing so as models launch. https://news.ycombinator.com/item?id=48777790 (2026-07-03)
   - Etymon author hops Claude Code -> Codex -> OpenCode+DeepSeek on limits; a reply says "This is definitely a real problem lol." https://news.ycombinator.com/item?id=49926105 (2026-10-01)
   - joshstrange hits 90-100% of a $200 plan and would move to Codex if promo limits end. https://news.ycombinator.com/item?id=49350033 (2026-08-18)
   - freedomben moved documentation planning to Codex after the 5-hour window forced planning around caps. https://news.ycombinator.com/item?id=47893113 (2026-04-24)
   - Limits are asymmetric: a free Codex tier lock-out lasts about a week vs Claude's 5-hour reset (Bishonen88, https://news.ycombinator.com/item?id=46868577, 2026-02-03). Users also think about which tool has headroom: "uses whichever one isn't at its limit" (muzani, 2025-09, https://news.ycombinator.com/item?id=45094674).
   - Skillsync's own launch lists "switching agents mid-task when hitting usage limits" as a first use case. https://news.ycombinator.com/item?id=49743049
2. Second opinion / review by a different model
   - Dev.to summary of community talk: switching "usually starts in review, not generation"; draft in Claude, review in Codex. https://dev.to/dishant0406/why-developers-are-switching-from-claude-code-to-codex-3ba0 (Oct 2026)
   - buu700 relays code between Grok, Claude and Gemini so they settle an issue; Grok reply as second opinion (2025-08, https://news.ycombinator.com/item?id=44809787). teruakohatu switches models for a second opinion when Opus limits hit (2025-08, https://news.ycombinator.com/item?id=44806942).
3. Model strengths: Gemini for codebase-wide analysis/planning, Claude for coding (buu700 above); Claude for momentum, Codex for second look (dev.to above).
4. Cost: cheaper open models once expensive quota runs out (vidarh); z.ai when Claude limit hit (azuanrb, 2025-12, https://news.ycombinator.com/item?id=46364523).
5. Stuck agent: only anecdotal. Rses skeptic faangguyindia argues the same context yields the same errors and backtracking, i.e. a handoff to a new model may not unstick it. https://news.ycombinator.com/item?id=47528990 No strong "I switched because Claude was stuck" quote found in my sample (Reddit was unreachable).
6. Preference/personality: X user toli switched all agents from Opus to Codex and wrote a "soul mod" to make Codex sound like Opus - agent personality is part of what users want to carry. https://x.com/tolibear_/status/2040290342956581317 (2026-04-04, snippet only)

## How users hand off by hand today

- Ask the outgoing agent for HANDOFF.md, read/fix it, start the next agent with "Read HANDOFF.md and continue". Dev.to guide: https://dev.to/heylittlepan/how-to-hand-off-a-task-from-claude-code-to-codex-and-back-54ne (2026). pwython recommends HANDOFF.md between compactions: https://news.ycombinator.com/item?id=49038414 (2026-07-24).
- Ask the current chat for a summary plus a copy-paste prompt (quinncom does this from ChatGPT into Codex): https://news.ycombinator.com/item?id=47075089
- Copy terminal output (did not carry tool-use context, per continues author): same thread.
- Shared markdown "session journals" read by both Claude and Codex (elenaviter, 2026-07-23, https://news.ycombinator.com/item?id=49028053); a design-doc/markdown file the agent maintains so it can restart or hand off anytime (nijave, 2026-06-14, https://news.ycombinator.com/item?id=48526447).
- `/compact` then continue; "handoff documents summarizing prior sessions are common" and Claude automates it with /compact (jazzypants, https://news.ycombinator.com/item?id=48075973, 2026-05-09).
- Amp's `/handoff` (summarise context toward a stated goal into a fresh thread) is held up as better than compaction: https://news.ycombinator.com/item?id=47097741 (2026-02-21). Glen Maddern asked if Claude Code/OpenCode/Codex have an Amp-style handoff: https://x.com/glenmaddern/status/2021079347130662990 (2026-02-10, snippet).
- Auto handoff hook: Alex Hillman, "Auto handoff is the way" - hook near context headroom, cheap Haiku subagent writes a carry-over file by custom instructions, new session picks it up with pointers. https://x.com/alexhillman/status/2106405507556606425 (2026-10-03, snippet)
- Clipboard bundle of history + diff + last commits (Hydra): https://github.com/saadnvd1/hydra
- Text-snippet launchers: serkanhaslak stores the continues prompt in Raycast snippets (47075089).
- Plan in Claude, paste into Codex (several guides); merge with `@AGENTS.md` inside CLAUDE.md so both read the same rules (dev.to guide above).

## What breaks

- New agent starts blank: "you lose your entire conversational context and start from scratch"; copy-paste relies on hoping "the receiving agent infers enough context". https://codex.danielvaughan.com/2026/06/01/codex-cli-cross-agent-session-portability-continues-casr-handoff/ (2026-06-01)
- Redoing finished work if the note does not say it is complete; approvals/permissions do not carry; MCP servers, skills, hooks and custom commands must be set up on both sides; full transcripts are large and mostly tool output. https://dev.to/heylittlepan/how-to-hand-off-a-task-from-claude-code-to-codex-and-back-54ne
- Compaction loses detail: "compactions tend to be full of a lot of useless stuff" (SatvikBeri, https://news.ycombinator.com/item?id=47097741). pigpop: summaries drop important details, turn auto-compaction off, keep per-session logs (https://news.ycombinator.com/item?id=46521933, 2026-01-07).
- Hidden reasoning, permission state, background processes and tool state are "not transferable"; tool results are dropped in compact mode; middle of long transcripts trimmed at 120,000 chars. https://github.com/nerln/claude-codex-bridge
- Successful tool activity and reasoning are stripped from cross-agent forks. https://github.com/wilbeibi/catchup ; tool activity reduced to aggregates and handoff consumes receiving context window. continues write-up above.
- Undocumented, version-fragile session formats; bridge tool is version-gated and says run `doctor` after every Claude/Codex upgrade. Skillsync users hit gaps too: Codex "archived" chats are not auto-discovered (solfox, in https://news.ycombinator.com/item?id=49743049).
- Summary vs immutable log: agentdev001 calls markdown-journal sync "a garbage approach" because it distils the session; mmykola87 carries only conclusions and verified items; cat-whisperer: "the next session wants conclusions, not the whole transcript" (same thread). Continues author argues raw agentic flow teaches more than a sanitised dump (47075089).
- Mistakes carry over: same context may reproduce the same errors (faangguyindia, 47528990).
- Race during takeover: sno-station example had the receiver start editing before the brief was released; had to be paused and the brief rewritten. https://github.com/sno-ai/sno-station
- Local transcripts expire: Claude Code deletes them after 30 days by default (Contextify author, 48777790).
- Model swap vs reasoning: a Product Hunt commenter noted that when you swap models the memory transfers but the reasoning resets (via search snippet of https://www.producthunt.com/posts/moxie-3).
- Manual steps: Hydra needs a paste; continues says rate-limit detection is manual (47075089, 47851935).
- Cross-machine: Offload does not carry chat history to the remote agent. https://github.com/ToxicPine/offloads
- Lost background work: noworriesnate asked whether sub-agent work was lost when a 5-hour limit hit mid-run (https://news.ycombinator.com/item?id=48510079, 2026-06-12); a Show HN exists solely to resume subagents killed by a limit (https://news.ycombinator.com/item?id=49831324).

## What users wish existed

- Switch agents while keeping the same task context, without paying for two high-tier plans (Ask HN, 49904074). "found some tools... none have much traction" (paraphrase of the same post) - the space is crowded but unconsolidated.
- Move ChatGPT-chat context into Codex/agent UI (quinncom, 47075089); brainstorm on phone/browser chat then continue in a coding agent (Skillsync founder, 49743049).
- Amp-style goal-directed `/handoff` in Claude Code/Codex/OpenCode (Glen Maddern, X 2026-02-10).
- One config for many harnesses; 20+ files make a project non-portable (Etymon, 49926106).
- Visibility of remaining quota per agent before choosing (gosasan asked for visible usage tracking, https://news.ycombinator.com/item?id=45264284 (2025-09); Orbital's picker shows it; dozens of usage-meter Show HNs, e.g. https://news.ycombinator.com/item?id=49832165).
- Auto-triggered handoff before the limit (SpecWeave 90% threshold; agent-limit-retry; Alex Hillman's hook).
- Expectation that harnesses can switch models mid-session natively (aerzen asked whether Rses helps with pi, which can switch models mid-session, 47528990).
- Handoff that preserves "what failed": Mida/damli40 explicitly stores rejected approaches.
- Searchable history across tools (Contextify, Termem, Skillsync).

## Handoff document recipes

Consolidated from sources (each section appears in at least one source):
1. Goal in 1-2 sentences (dev.to guide; claude-handoff-revive; Mida stores original request verbatim; later user instructions override earlier).
2. Done, and how it was verified (dev.to guide; session-handoff skills "Verification").
3. Remaining work in order, with a concrete next action including file and exact command (claude-handoff-revive `next_action`).
4. Decisions and why (all sources).
5. Rejected approaches / what failed and why (Mida; damli40 hook; claude-handoff-revive `lessons_learned`; session-handoff skills).
6. Key files and what is uncommitted; auto-fill from git (branch, base commit, recent commits, diff) so it is checkable (session-handoff skills; Hydra; ShardStitch "files risky to touch").
7. Constraints and blockers/open questions (project-continuity; skills).
8. Smoke-test info: build and launch steps, changed flows, what "working" means (dev.to guide).
9. Hygiene: write it from the agent that has the context; human reads and corrects it; skip anything readable from the code; keep it small (claude-handoff-revive: ~1-3k tokens); strip secrets (Mida; skills); mark staleness vs git history (skills); commit or gitignore the file.
10. Protocol hygiene (stronger systems): receiver echoes size + checksum and sender releases only after ack (sno-station); single-writer lease (project-continuity); handoff claimed exactly once (ai-memory).
11. Detail levels: minimal / standard / verbose / full (continues); full / focused / minimal (sahir2k).
Sources: https://dev.to/heylittlepan/how-to-hand-off-a-task-from-claude-code-to-codex-and-back-54ne ; https://github.com/sofumel/claude-handoff-revive ; https://github.com/damli40/mida-context ; https://github.com/sno-ai/sno-station ; https://github.com/seriousz158/project-continuity ; skill registries https://skillselion.com/skills/davila7/claude-code-templates/session-handoff (secondary, low usage).

## Ideas worth stealing

1. Same-pane harness swap with an explicit "switch" step that runs a conversion then relaunches with resume flags (Baton). Fits AO's per-session worktree: keep the worktree, swap the agent process.
2. Native-session transcoding with read-back verification and rollback (OmniSession), with a "semantic handoff" fallback when native import is impossible. Offer both paths and say which was used.
3. Official import paths now exist: Codex has an external-agent session importer (used by `/codex:transfer`), Grok has `grok import`. AO can call vendor importers instead of reverse-engineering formats where available.
4. Version gating plus a `doctor` check for session-format compatibility (claude-codex-bridge), because formats are undocumented and break on upgrades.
5. Detail-level control for how much conversation carries (continues presets, Skillsync "controls on how much of the conversation"). Default to a goal-directed summary plus recent turns, offer "full transcript" as an option.
6. Handoff carries structured sections incl. verbatim original request, decisions, rejected approaches, constraints, next step (Mida) and git-derived state; show it to the user for edit before sending (dev.to guide advice).
7. Show remaining quota per agent in the agent picker (Orbital) and offer a threshold-triggered suggestion (SpecWeave 90%), but keep the switch user-confirmed (Hydra shows the picker rather than auto-switching, since limits are account-wide).
8. Keep project state in files the user owns (Orbital: PROJECT_STATE / DECISIONS / LESSONS) so any agent can read it on demand rather than pasting a blob; AO could write a per-session decisions log that both agents see.
9. Ack-before-release protocol for the handoff (sno-station): receiver restates what it understood (size/checksum, summary of next step) before the old agent is released or the new one edits. Avoids the early-editing race.
10. Preserve a link to the original session ("full path to original session file" in continues; Skillsync "whole transcript still exists") so the new agent can look things up instead of redoing work.
11. Mark handoff boundaries in the chat UI (Kangentic's card/column move; Pi multi-account's "handoff record" in the turn) so the user sees where the agent changed.
12. Keep personality/instruction parity: users port CLAUDE.md / AGENTS.md / skills (Baton links personal skills; Etymon unifies config; `@AGENTS.md` inside CLAUDE.md). A switch should check the target agent has equivalent instructions, MCP servers and approvals, since those do not carry.
13. For failover, treat account-wide limits as global: one detection should prompt all open sessions (Hydra's `hydra switch` signals every open session).

## Gaps

- No Reddit data (blocked). No Medium/Substack deep read beyond dev.to and a few roundups. X only via snippets.
- Product Hunt and YC coverage is thin; only Skillsync (YC W26) and Superset/Omnara surfaced in Launch HN; no other YC company found claiming agent switching.
- CodePass has no findable repo; Kangentic and Codeg mechanisms undocumented; Clinch transfer internals unknown; Continuo not seen beyond its landing page; Weave Router numbers are the author's and unverified.
- I did not run any tool; "VERIFIED" means a README or official doc says it.
- GitHub search returned empty for several multi-word queries, so the long tail on GitHub is likely larger (many near-duplicate HANDOFF.md repos with 0-2 stars).
