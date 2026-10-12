import {
	AgentAvatar as ProductAgentAvatar,
	type AgentAvatarProps,
	type AgentLogoSources,
} from "@aoagents/product-ui";
import agyLogo from "../assets/agents/agy.svg";
import aiderLogo from "../assets/agents/aider.png";
import ampLogo from "../assets/agents/amp.svg";
import auggieLogo from "../assets/agents/auggie.svg";
import autohandLogo from "../assets/agents/autohand.svg";
import clineLogo from "../assets/agents/cline.svg";
import claudeLogo from "../assets/agents/claude.svg";
import codexLogo from "../assets/agents/codex.svg";
import continueLogo from "../assets/agents/continue.png";
import copilotLogo from "../assets/agents/copilot.svg";
import crushLogo from "../assets/agents/crush.png";
import cursorLogo from "../assets/agents/cursor.svg";
import devinLogo from "../assets/agents/devin.svg";
import deepseekHarnessLogo from "../assets/agents/deepseek-harness.svg";
import droidLogo from "../assets/agents/droid.svg";
import fxLogo from "../assets/agents/fx.svg";
import geminiLogo from "../assets/agents/gemini.svg";
import gooseLogo from "../assets/agents/goose.svg";
import grokLogo from "../assets/agents/grok.svg";
import kilocodeLogo from "../assets/agents/kilocode.svg";
import kimiLogo from "../assets/agents/kimi.png";
import kimchiLogo from "../assets/agents/kimchi.svg";
import kiroLogo from "../assets/agents/kiro.png";
import museLogo from "../assets/agents/muse.png";
import mimoCodeLogo from "../assets/agents/mimo-code.svg";
import ompLogo from "../assets/agents/omp.svg";
import opencodeLogo from "../assets/agents/opencode.svg";
import piLogo from "../assets/agents/pi.svg";
import primeAgentLogo from "../assets/agents/prime-agent.png";
import qwenLogo from "../assets/agents/qwen.png";
import unrealAgentLogo from "../assets/agents/unreal-agent.png";
import vibeLogo from "../assets/agents/vibe.png";
import { cn } from "../lib/utils";

// Real brand logos keyed by the harness name AO stores on session.provider.
// Agents without an asset fall back to a lettered tile (fake).
const LOGOS: AgentLogoSources = {
	codex: codexLogo,
	"claude-code": claudeLogo,
	claude: claudeLogo,
	cursor: cursorLogo,
	opencode: opencodeLogo,
	"opencode-v2": opencodeLogo,
	copilot: copilotLogo,
	aider: aiderLogo,
	grok: grokLogo,
	gemini: geminiLogo,
	droid: droidLogo,
	fx: fxLogo,
	crush: crushLogo,
	qwen: qwenLogo,
	goose: gooseLogo,
	continue: continueLogo,
	devin: devinLogo,
	kimi: kimiLogo,
	muse: museLogo,
	omp: ompLogo,
	kiro: kiroLogo,
	kilocode: kilocodeLogo,
	vibe: vibeLogo,
	pi: piLogo,
	kimchi: kimchiLogo,
	"prime-agent": primeAgentLogo,
	amp: ampLogo,
	cline: clineLogo,
	agy: agyLogo,
	auggie: auggieLogo,
	autohand: autohandLogo,
	"unreal-agent": unrealAgentLogo,
	"mimo-code": mimoCodeLogo,
	"deepseek-harness": deepseekHarnessLogo,
};

// Single-colour marks. They are painted in the text colour through a mask so
// they follow the theme; an <img> would keep one fixed colour.
const TINTED = new Set([
	"codex", "cursor", "opencode", "opencode-v2", "droid", "grok", "pi", "devin",
	"copilot", "cline", "goose", "kilocode", "auggie", "autohand",
]);

/**
 * Agent mark for board/task cards: the harness's real brand logo rendered bare —
 * no tile, border, or background — so each brand's own shape shows (codex,
 * claude, cursor, … carry their own rounded backgrounds). Agents without an
 * asset fall back to a bare initial. Kept small so the title stays the hero.
 *
 * The provider is exposed as the accessible name (alt / aria-label), not just a
 * hover title, so surfaces that show the logo in place of visible agent text —
 * e.g. the archive cards — still name the agent for screen readers.
 */
export function AgentAvatar({ provider, className, decorative = false }: AgentAvatarProps) {
	const tinted = TINTED.has(provider) ? LOGOS[provider] : undefined;
	if (tinted) {
		return (
			<span
				{...(decorative ? { "aria-hidden": true } : { "aria-label": provider, role: "img", title: provider })}
				className={cn("inline-block size-icon-xl shrink-0 bg-current text-foreground", className)}
				style={{ mask: `url("${tinted}") center / contain no-repeat` }}
			/>
		);
	}
	return <ProductAgentAvatar className={className} decorative={decorative} logoSources={LOGOS} provider={provider} />;
}

export type { AgentAvatarProps };
