/**
 * One control for who answers the next message and how: agent, model, effort
 * and speed.
 *
 * Agents are icon tabs. The current agent's tab changes its settings straight
 * away, as before. Picking a model under another agent's tab changes nothing
 * yet: the choice is held as pending and the switch runs when the next message
 * is sent, so the handoff note is written at the moment it is needed and looking
 * through the list never restarts anything. Picking anything under the current
 * agent's tab again clears the pending choice.
 */

import { useQuery } from "@tanstack/react-query";
import type { TFunction } from "i18next";
import { Brain, ChevronLeft, ChevronRight, RefreshCw, RotateCcw, TriangleAlert, X, Zap } from "lucide-react";
import {
	Fragment,
	useCallback,
	useLayoutEffect,
	useMemo,
	useRef,
	useState,
	type KeyboardEvent,
	type PointerEvent,
	type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";
import { agentModelsQueryOptions } from "../../hooks/useAgentModelsQuery";
import { useAgentReadinessQuery } from "../../hooks/useAgentReadinessQuery";
import { agentModelDisplayLabel } from "../../lib/agent-model-choices";
import { agentLabel } from "../../lib/agent-options";
import { CHAT_SWITCH_TARGETS, TERMINAL_SWITCH_TARGETS } from "../../lib/agent-switch-targets";
import {
	collapseModelVariants,
	findModelVariant,
	listsModelVariants,
	modelGroupLabel,
	pickModelVariant,
	readableModelLabel,
} from "../../lib/model-variants";
import { cn } from "../../lib/utils";
import { formatEffortLabel } from "../settings/EffortPicker";
import {
	OptionMenu,
	OptionMenuContent,
	OptionMenuItem,
	OptionMenuLabel,
	OptionMenuTrigger,
} from "../ui/option-menu";
import { ModelMenuChoices } from "./ModelMenuChoices";
import { ProviderMark } from "./ProviderMark";

export interface PendingAgentSwitch {
	harness: string;
	/** Empty starts the agent on its own default model. */
	model?: string;
	modelLabel?: string;
	/** Sent with the switch. Unset when the model id already carries the effort. */
	effort?: string;
	/** Sent with the switch: start the agent on its faster tier. */
	fastMode?: boolean;
	/** What the chip shows; for agents whose model id carries these. */
	shownEffort?: string;
	fast?: boolean;
	thinking?: boolean;
	/** Terminal only: start Claude Code with Ultracode on. */
	ultracode?: boolean;
	/** Terminal only: where Claude Code compacts on its own, "200k" or "1m". */
	autoCompactWindow?: string;
}

/** What the chat surface passes down to turn the model control into this one. */
export interface AgentSwitchProps {
	projectId: string;
	hostId?: string;
	pending?: PendingAgentSwitch;
	onPendingChange: (next: PendingAgentSwitch | undefined) => void;
	/** Switching is refused while the agent is working. */
	disabled?: boolean;
}

export interface EffortChoice {
	value: string;
	label: string;
}

export interface PickerModel {
	id: string;
	label: string;
	/** Models from several makers are listed under their maker. */
	group?: string;
}

/** One agent's models and settings, with what each control does when used. */
export interface PickerSource {
	models: PickerModel[];
	modelId?: string;
	modelLabel: string;
	onPickModel?: (id: string) => void;
	efforts: EffortChoice[];
	effort?: string;
	onPickEffort?: (value: string) => void;
	/** Returns effort to the model's own default. Absent when it is already there. */
	onResetEffort?: () => void;
	/** Undefined when the agent has no speed setting. */
	fast?: boolean;
	onToggleFast?: (next: boolean) => void;
	/** Undefined when the model has no thinking choice. */
	thinking?: boolean;
	onToggleThinking?: (next: boolean) => void;
}

/** Kept as the name the settings bar builds for the running agent. */
export type CurrentAgentChoices = PickerSource;

type PickedVariant = { id: string; label: string; effort?: string; fast: boolean; thinking: boolean };

const TRIGGER_CLASS =
	"h-7 gap-1.5 bg-transparent rounded-lg px-2 text-[12px]! leading-none text-foreground hover:bg-interactive-active data-[state=open]:bg-interactive-active";
const MENU_CLASS = "chat-settings-menu w-80 text-[12px]!";

/**
 * A catalog that lists one entry per model setting, shown as one model per
 * family with effort, thinking and speed moved to their own controls.
 */
function variantSource(
	models: PickerModel[],
	selectedId: string | undefined,
	fallbackLabel: string,
	onPick: ((variant: PickedVariant) => void) | undefined,
	t: TFunction,
): PickerSource {
	const families = collapseModelVariants(models);
	const selected = findModelVariant(families, selectedId);
	const family = selected?.family;
	const variant = selected?.variant;
	const choose = onPick && family
		? (want: { effort?: string; thinking?: boolean; fast?: boolean }) => {
			const next = pickModelVariant(family, {
				effort: want.effort ?? variant?.effort,
				thinking: want.thinking ?? variant?.thinking,
				fast: want.fast ?? variant?.fast,
			});
			onPick({ id: next.id, label: family.label, effort: next.effort, fast: next.fast, thinking: next.thinking });
		}
		: undefined;
	const resettable = Boolean(onPick && family && variant && variant.id !== family.defaultVariant.id);
	return {
		models: families.map((entry) => ({ id: entry.id, label: entry.label, group: entry.group })),
		modelId: family?.id,
		modelLabel: family?.label ?? fallbackLabel,
		onPickModel: onPick
			? (id) => {
				const next = families.find((entry) => entry.id === id);
				if (!next) return;
				const chosen = next.defaultVariant;
				onPick({ id: chosen.id, label: next.label, effort: chosen.effort, fast: chosen.fast, thinking: chosen.thinking });
			}
			: undefined,
		efforts: (family?.efforts ?? []).map((value) => ({ value, label: formatEffortLabel(value, t) })),
		effort: variant?.effort,
		onPickEffort: choose ? (effort) => choose({ effort }) : undefined,
		onResetEffort: resettable && family
			? () => onPick?.({
				id: family.defaultVariant.id,
				label: family.label,
				effort: family.defaultVariant.effort,
				fast: family.defaultVariant.fast,
				thinking: family.defaultVariant.thinking,
			})
			: undefined,
		fast: family?.hasFast ? Boolean(variant?.fast) : undefined,
		onToggleFast: choose ? (fast) => choose({ fast }) : undefined,
		thinking: family?.hasThinkingChoice ? Boolean(variant?.thinking) : undefined,
		onToggleThinking: choose ? (thinking) => choose({ thinking }) : undefined,
	};
}

export function AgentModelPicker({
	currentHarness,
	current,
	agentSwitch,
	disabled,
	title,
	mark,
	extras,
	terminal = false,
	pendingFooter,
}: {
	/** Shown under the settings of the agent picked to switch to, such as its Switch and Cancel buttons. */
	pendingFooter?: ReactNode;
	currentHarness: string;
	current: PickerSource;
	agentSwitch: AgentSwitchProps;
	/**
	 * Over a terminal session. AO cannot change the running agent's model there,
	 * so the control names that agent and only offers a switch to another one.
	 */
	terminal?: boolean;
	/** The current agent's own settings cannot be changed right now. */
	disabled?: boolean;
	title?: string;
	/** A flag on the trigger, such as a substituted model. */
	mark?: ReactNode;
	/** The current agent's other settings, as menu rows under its effort. */
	extras?: ReactNode;
}) {
	const { t } = useTranslation();
	const { pending, onPendingChange, projectId, hostId } = agentSwitch;
	const readiness = useAgentReadinessQuery(true, hostId).data?.agents;
	const agents = useMemo(() => {
		const listed = (terminal ? TERMINAL_SWITCH_TARGETS : CHAT_SWITCH_TARGETS).flatMap((id) => {
			if (id === currentHarness) return [];
			const agent = readiness?.find((candidate) => candidate.id === id);
			// Every chat agent is shown so the row says what AO can switch to. One
			// that is not installed or not signed in is there but cannot be picked.
			// Unknown readiness stays selectable: the daemon checks the target
			// before it touches the running agent.
			const blocked = agent?.installation.state === "not_installed"
				? "not installed"
				: agent?.authentication.state === "unauthorized" ? "sign in required" : undefined;
			return [{ id: id as string, label: agent?.label ?? agentLabel(id), blocked }];
		});
		return [
			{ id: currentHarness, label: agentLabel(currentHarness), blocked: undefined as string | undefined },
			...listed.filter((agent) => !agent.blocked),
			...listed.filter((agent) => agent.blocked === "sign in required"),
			...listed.filter((agent) => agent.blocked === "not installed"),
		];
	}, [currentHarness, readiness, terminal]);
	const [openTab, setOpenTab] = useState<string>();
	// The chip is as wide as its text, and it sits against the right of the chat
	// box, so a new model or effort moves its left edge. The picker is held where
	// it opened by taking that movement back out.
	const trigger = useRef<HTMLButtonElement>(null);
	const [openLeft, setOpenLeft] = useState<number>();
	const [shift, setShift] = useState(0);
	useLayoutEffect(() => {
		if (openLeft === undefined) return;
		const left = trigger.current?.getBoundingClientRect().left;
		if (left !== undefined) setShift(Math.round(openLeft - left));
	});
	const tab = openTab ?? pending?.harness ?? currentHarness;
	const keepCurrent = () => {
		if (pending) onPendingChange(undefined);
	};
	const currentSource: PickerSource = (() => {
		const pickModel = current.onPickModel;
		const base = listsModelVariants(currentHarness)
			? variantSource(
				current.models,
				current.modelId,
				current.modelLabel,
				pickModel ? (variant) => pickModel(variant.id) : undefined,
				t,
			)
			: current;
		return disabled
			? {
				...base,
				onPickModel: undefined,
				onPickEffort: undefined,
				onResetEffort: undefined,
				onToggleFast: undefined,
				onToggleThinking: undefined,
			}
			: base;
	})();

	const shownHarness = pending?.harness ?? currentHarness;
	const extraRows = (pendingFooter && pending?.harness === tab ? 1 : 0) +
		((extras && tab === currentHarness) || (terminal && tab === "claude-code" && tab !== currentHarness) ? 1 : 0);
	const shownModel = pending ? pending.modelLabel ?? agentLabel(pending.harness) : currentSource.modelLabel;
	const pendingEffort = pending?.effort ?? pending?.shownEffort;
	const shownEffort = pending
		? pendingEffort ? formatEffortLabel(pendingEffort, t) : undefined
		: currentSource.efforts.find((effort) => effort.value === currentSource.effort)?.label;
	const shownUltra = isUltraEffort(pending ? pendingEffort : currentSource.effort);
	const shownFast = pending ? pending.fast : currentSource.fast;
	const shownThinking = pending ? pending.thinking : currentSource.thinking;

	return (
		<OptionMenu
			onOpenChange={(open) => {
				setOpenLeft(open ? trigger.current?.getBoundingClientRect().left : undefined);
				if (!open) {
					setOpenTab(undefined);
					setShift(0);
				}
			}}
		>
			<OptionMenuTrigger
				ref={trigger}
				disabled={disabled && agentSwitch.disabled}
				aria-label={terminal ? "Agent running in this terminal" : "Agent, model and reasoning effort for the next message"}
				title={title ?? (terminal ? "Switch to another agent" : "Agent, model and reasoning effort for the next message")}
				className={cn(TRIGGER_CLASS, terminal ? "w-full justify-start" : undefined)}
				showCaret={!terminal}
			>
				<ProviderMark provider={terminal ? currentHarness : shownHarness} />
				<span className={cn("min-w-0 max-w-[28ch] truncate", shownUltra && !terminal ? "ultra-text" : undefined)}>
					{terminal ? agentLabel(currentHarness) : shownModel}
				</span>
				{terminal ? <ChevronRight aria-hidden="true" className="ml-auto size-icon-sm shrink-0 text-muted-foreground" /> : (
					<>
						{shownFast ? <Zap aria-label="Fast mode" className="size-3 shrink-0 fill-current" /> : null}
						{shownThinking ? <Brain aria-label="Thinking" className="size-3 shrink-0" /> : null}
						{shownEffort ? (
							<span className={cn("shrink-0", shownUltra ? "ultra-text" : "text-muted-foreground")}>{shownEffort}</span>
						) : null}
						{mark}
					</>
				)}
			</OptionMenuTrigger>
			<OptionMenuContent align="start" alignOffset={shift} side={terminal ? "right" : "top"}
				// A footer bar or extra row adds its own height; it must not squeeze the model list.
				className={MENU_CLASS}
				style={extraRows > 0 ? { maxHeight: `calc(var(--size-select-menu-max) + ${extraRows * 2.85}rem)` } : undefined}
			>
				<AgentTabStrip>
					{agents.map((agent) => (
						<button
							key={agent.id}
							type="button"
							role="tab"
							aria-selected={agent.id === tab}
							aria-label={agent.label}
							title={agent.blocked ? `${agent.label} · ${agent.blocked}` : agent.label}
							disabled={Boolean(agent.blocked) || (agent.id !== currentHarness && agentSwitch.disabled)}
							onClick={() => setOpenTab(agent.id)}
							className={cn(
								"relative flex w-8 shrink-0 items-center justify-center disabled:cursor-not-allowed disabled:opacity-25 disabled:grayscale",
								agent.id === tab ? "opacity-100" : "opacity-60 hover:opacity-100",
							)}
						>
							<ProviderMark provider={agent.id} />
							{agent.id === tab ? (
								<span aria-hidden="true" className="absolute inset-x-1.5 bottom-0 h-0.5 rounded-full bg-foreground" />
							) : null}
						</button>
					))}
				</AgentTabStrip>
				{tab === currentHarness && terminal ? (
					<p className="px-3 py-2.5 text-xs text-muted-foreground">
						{agentLabel(currentHarness)} is running here. Change its model from inside the terminal.
					</p>
				) : tab === currentHarness ? (
					<>
						<SourcePanel
							source={currentSource}
							chosenId={pending ? undefined : currentSource.modelId}
							emptyLabel={agentLabel(currentHarness)}
							beforeChange={keepCurrent}
						/>
						{extras ? <div className="mt-1 border-t border-border pt-1">{extras}</div> : null}
					</>
				) : (
					<OtherAgentPanel
						key={tab}
						harness={tab}
						projectId={projectId}
						hostId={hostId}
						pending={pending?.harness === tab ? pending : undefined}
						onPick={onPendingChange}
						terminal={terminal}
					/>
				)}
				{pending?.harness === tab ? pendingFooter : null}
			</OptionMenuContent>
		</OptionMenu>
	);
}

/**
 * The row of agent tabs. It keeps the picker's width however many agents there
 * are: the row is exactly a whole number of tabs wide and scrolls sideways a tab
 * at a time with no scrollbar, so no icon is ever cut off. The arrows beside it
 * say there are more and move to them.
 */
function AgentTabStrip({ children }: { children: ReactNode }) {
	const strip = useRef<HTMLDivElement>(null);
	const [more, setMore] = useState({ left: false, right: false });
	const measure = useCallback(() => {
		const element = strip.current;
		if (!element) return;
		const left = element.scrollLeft > 1;
		const right = element.scrollLeft + element.clientWidth < element.scrollWidth - 1;
		setMore((previous) => (previous.left === left && previous.right === right ? previous : { left, right }));
	}, []);
	useLayoutEffect(measure);
	useLayoutEffect(() => {
		strip.current?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: "nearest", inline: "nearest" });
	}, []);
	const move = (by: number) => strip.current?.scrollBy({ left: by, behavior: "smooth" });
	const arrow = "flex w-4 items-center justify-center text-muted-foreground hover:text-foreground disabled:opacity-25 disabled:hover:text-muted-foreground";
	return (
		<div className="mb-1 flex shrink-0 items-stretch justify-between border-b border-border">
			<div
				ref={strip}
				role="tablist"
				aria-label="Agents"
				onScroll={measure}
				onKeyDown={(event) => event.stopPropagation()}
				// Tabs are 2rem wide and 0.125rem apart; the arrows take 2rem.
				style={{ width: "calc(round(down, 100% - 1.875rem, 2.125rem) - 0.125rem)" }}
				className="scrollbar-none flex h-9 flex-none snap-x snap-mandatory items-stretch gap-0.5 overflow-x-auto [&>*]:snap-start"
			>
				{children}
			</div>
			{more.left || more.right ? (
				<div className="flex shrink-0">
					<button type="button" aria-label="Earlier agents" disabled={!more.left} onClick={() => move(-136)} className={arrow}>
						<ChevronLeft aria-hidden="true" className="size-3.5" />
					</button>
					<button type="button" aria-label="More agents" disabled={!more.right} onClick={() => move(136)} className={arrow}>
						<ChevronRight aria-hidden="true" className="size-3.5" />
					</button>
				</div>
			) : null}
		</div>
	);
}

function OtherAgentPanel({
	harness,
	projectId,
	hostId,
	pending,
	onPick,
	terminal,
}: {
	harness: string;
	projectId: string;
	hostId?: string;
	pending?: PendingAgentSwitch;
	onPick: (next: PendingAgentSwitch) => void;
	/**
	 * A terminal agent takes these when it is launched, so Claude Code's
	 * Ultracode and auto-compact are offered here; in chat they are set on the
	 * running agent instead.
	 */
	terminal: boolean;
}) {
	const { t } = useTranslation();
	const query = useQuery(agentModelsQueryOptions(harness, projectId, hostId));
	const catalog = query.data?.models;
	const source = useMemo<PickerSource>(() => {
		const listed = (catalog ?? [])
			.filter((model) => !/^\[|\s/.test(model.id))
			.map((model) => ({
				id: model.id,
				label: readableModelLabel(agentModelDisplayLabel(harness, model.label)),
				group: modelGroupLabel(model.provider, harness),
			}));
		if (listsModelVariants(harness)) {
			return variantSource(listed, pending?.model, agentLabel(harness), (variant) => onPick({
				harness,
				model: variant.id,
				modelLabel: variant.label,
				shownEffort: variant.effort,
				fast: variant.fast,
				thinking: variant.thinking,
			}), t);
		}
		const chosen = pending ? pending.model ?? "" : undefined;
		// Effort levels belong to a model. With none picked they are the default's.
		const effortSource = catalog?.find((model) => (chosen ? model.id === chosen : model.isDefault)) ?? catalog?.[0];
		const defaultEffort = effortSource?.defaultEffort;
		const offersFast = Boolean(effortSource?.supportsFast);
		const fastOn = offersFast && Boolean(pending?.fastMode);
		const levels = (effortSource?.efforts ?? []).filter((value) => value && value.toLowerCase() !== "default");
		// Ultracode runs at Extra High, so it is only for models that have that level.
		const offersUltracode = terminal && harness === "claude-code" && levels.includes("xhigh");
		const ultracodeOn = offersUltracode && Boolean(pending?.ultracode);
		const keep = {
			harness,
			model: pending?.model,
			modelLabel: pending?.modelLabel,
			shownEffort: defaultEffort,
			fast: fastOn || undefined,
			fastMode: fastOn || undefined,
			autoCompactWindow: pending?.autoCompactWindow,
		};
		return {
			models: [{ id: "", label: "Default model" }, ...listed],
			modelId: chosen,
			modelLabel: pending?.modelLabel ?? agentLabel(harness),
			onPickModel: (id) => {
				// Fast mode carries over only to a model that also offers it.
				const next = catalog?.find((model) => (id ? model.id === id : model.isDefault));
				const fast = Boolean(next?.supportsFast && pending?.fastMode) || undefined;
				onPick({
					harness,
					model: id || undefined,
					modelLabel: listed.find((model) => model.id === id)?.label,
					shownEffort: next?.defaultEffort,
					fast,
					fastMode: fast,
					autoCompactWindow: pending?.autoCompactWindow,
				});
			},
			efforts: [
				...levels.map((value) => ({ value, label: formatEffortLabel(value, t) })),
				...(offersUltracode ? [{ value: "ultracode", label: "Ultracode" }] : []),
			],
			effort: ultracodeOn ? "ultracode" : pending?.effort ?? defaultEffort,
			onPickEffort: (effort) => onPick(effort === "ultracode"
				? { ...keep, effort: "xhigh", ultracode: true }
				: { ...keep, effort }),
			onResetEffort: ultracodeOn || (pending?.effort && pending.effort !== defaultEffort) ? () => onPick(keep) : undefined,
			fast: offersFast ? fastOn : undefined,
			onToggleFast: (fast) => onPick({ ...keep, effort: pending?.effort, ultracode: pending?.ultracode, fast: fast || undefined, fastMode: fast || undefined }),
		};
	}, [catalog, harness, onPick, pending, t, terminal]);

	// Claude Code only compacts early on its larger-window models.
	const offersAutoCompact = terminal && harness === "claude-code" && !/haiku|opus-4-5/.test(pending?.model ?? "");
	return (
		<>
			<SourcePanel source={source} chosenId={source.modelId} emptyLabel={agentLabel(harness)} />
			{offersAutoCompact ? (
				<div className="mt-1 border-t border-border pt-1">
					<AutoCompactRow
						control={{
							value: pending?.autoCompactWindow,
							onChange: (value) => onPick({ ...(pending ?? { harness }), autoCompactWindow: value }),
						}}
						title="When Claude Code compacts the conversation on its own."
					/>
				</div>
			) : null}
		</>
	);
}

/** Search, the model list and the settings under it, for one agent. */
function SourcePanel({
	source,
	chosenId,
	emptyLabel,
	beforeChange,
}: {
	source: PickerSource;
	chosenId?: string;
	emptyLabel: string;
	/** Runs before any change made here is applied. */
	beforeChange?: () => void;
}) {
	return (
		<>
			{source.models.length > 0 ? (
				<ModelMenuChoices
					models={source.models}
					alwaysSearch
					noSearch={source.models.filter((model) => model.id).length <= 10}
					divided
				>
					{(matches, searchActiveID, optionID) => matches.map((model, index) => {
						const active = model.id === chosenId;
						const key = model.id || "default";
						return (
							<Fragment key={key}>
								{model.group && model.group !== matches[index - 1]?.group ? (
									<OptionMenuLabel className="px-3 pt-2.5 pb-0.5 text-[10px]! text-muted-foreground/70">
										{model.group}
									</OptionMenuLabel>
								) : null}
								<OptionMenuItem
									id={optionID?.(key)}
									active={active}
									searchActive={model.id === searchActiveID}
									radio
									disabled={!source.onPickModel}
									onSelect={(event) => {
										// The menu stays open so effort and speed can be set too.
										event.preventDefault();
										beforeChange?.();
										if (!active) source.onPickModel?.(model.id);
									}}
									className="text-xs text-foreground"
								>
									{model.label}
								</OptionMenuItem>
							</Fragment>
						);
					})}
				</ModelMenuChoices>
			) : (
				<OptionMenuItem
					active={chosenId !== undefined || !beforeChange}
					radio
					onSelect={(event) => { event.preventDefault(); beforeChange?.(); }}
					className="text-xs"
				>
					{emptyLabel}
				</OptionMenuItem>
			)}
			<EffortPanel source={source} beforeChange={beforeChange} />
		</>
	);
}

// Half the thumb, so the thumb sits flush with either end of the track.
const SLIDER_INSET = 12;

/** The top level some agents have past Max: Codex's "ultra", Claude Code's "ultracode". */
function isUltraEffort(value: string | undefined): boolean {
	return Boolean(value?.toLowerCase().startsWith("ultra"));
}

/** Speed, effort and its reset, over a slider with one stop per level. */
function EffortPanel({ source, beforeChange }: { source: PickerSource; beforeChange?: () => void }) {
	const { efforts, effort, onPickEffort } = source;
	const track = useRef<HTMLDivElement>(null);
	const [dragIndex, setDragIndex] = useState<number>();
	const hasSlider = efforts.length > 0;
	if (!hasSlider && source.fast === undefined && source.thinking === undefined) return null;

	const selected = efforts.findIndex((choice) => choice.value === effort);
	const index = dragIndex ?? selected;
	const last = Math.max(efforts.length - 1, 1);
	const stopAt = (clientX: number) => {
		const rect = track.current?.getBoundingClientRect();
		if (!rect) return 0;
		const ratio = (clientX - rect.left - SLIDER_INSET) / Math.max(rect.width - SLIDER_INSET * 2, 1);
		return Math.min(efforts.length - 1, Math.max(0, Math.round(ratio * (efforts.length - 1))));
	};
	const commit = (next: number) => {
		const choice = efforts[next];
		if (!choice || choice.value === effort) return;
		beforeChange?.();
		onPickEffort?.(choice.value);
	};
	const position = (stop: number) => `calc(${SLIDER_INSET}px + (100% - ${SLIDER_INSET * 2}px) * ${stop / last})`;
	const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
		if (!onPickEffort) return;
		event.currentTarget.setPointerCapture(event.pointerId);
		setDragIndex(stopAt(event.clientX));
	};
	const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
		if (dragIndex !== undefined) setDragIndex(stopAt(event.clientX));
	};
	const onPointerUp = () => {
		if (dragIndex === undefined) return;
		commit(dragIndex);
		setDragIndex(undefined);
	};
	const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
		event.stopPropagation();
		if (!onPickEffort) return;
		const step = event.key === "ArrowRight" || event.key === "ArrowUp" ? 1
			: event.key === "ArrowLeft" || event.key === "ArrowDown" ? -1 : 0;
		if (step === 0) return;
		event.preventDefault();
		commit(Math.min(efforts.length - 1, Math.max(0, (selected < 0 ? 0 : selected) + step)));
	};
	const ultra = index >= 0 && isUltraEffort(efforts[index]?.value);
	const label = index >= 0 && efforts[index]
		? efforts[index].label
		: hasSlider ? "Effort" : source.fast ? "Fast" : "Standard";

	return (
		<div className="mt-1 border-t border-border px-2 pt-2 pb-1.5" onKeyDown={(event) => event.stopPropagation()}>
			<div className="mb-2 flex h-5 items-center gap-1">
				<span className="flex w-11 shrink-0 items-center gap-1">
					{source.fast === undefined ? null : (
						<PanelToggle
							label={source.fast ? "Turn fast mode off" : "Turn fast mode on"}
							pressed={source.fast}
							disabled={!source.onToggleFast}
							onPress={() => { beforeChange?.(); source.onToggleFast?.(!source.fast); }}
						>
							<Zap aria-hidden="true" className={cn("size-3.5", source.fast ? "fill-current" : undefined)} />
						</PanelToggle>
					)}
					{source.thinking === undefined ? null : (
						<PanelToggle
							label={source.thinking ? "Turn thinking off" : "Turn thinking on"}
							pressed={source.thinking}
							disabled={!source.onToggleThinking}
							onPress={() => { beforeChange?.(); source.onToggleThinking?.(!source.thinking); }}
						>
							<Brain aria-hidden="true" className="size-3.5" />
						</PanelToggle>
					)}
				</span>
				<span
					className={cn(
						"min-w-0 flex-1 truncate text-center text-xs font-medium",
						ultra ? "ultra-text" : index >= 0 ? "text-primary" : "text-muted-foreground",
					)}
				>
					{label}
				</span>
				<span className="flex w-11 shrink-0 items-center justify-end">
					{hasSlider ? (
						<PanelToggle
							label="Reset effort to the model's default"
							disabled={!source.onResetEffort}
							onPress={() => { beforeChange?.(); source.onResetEffort?.(); }}
						>
							<RotateCcw aria-hidden="true" className="size-3.5" />
						</PanelToggle>
					) : null}
				</span>
			</div>
			{hasSlider ? (
				<div
					ref={track}
					role="slider"
					tabIndex={onPickEffort ? 0 : -1}
					aria-label="Reasoning effort"
					aria-valuemin={0}
					aria-valuemax={efforts.length - 1}
					aria-valuenow={Math.max(index, 0)}
					aria-valuetext={index >= 0 ? efforts[index]?.label : undefined}
					aria-disabled={!onPickEffort}
					onPointerDown={onPointerDown}
					onPointerMove={onPointerMove}
					onPointerUp={onPointerUp}
					onPointerCancel={() => setDragIndex(undefined)}
					onKeyDown={onKeyDown}
					className={cn(
						"relative my-0.5 h-5 touch-none rounded-full bg-interactive-active outline-none focus-visible:ring-1 focus-visible:ring-ring",
						onPickEffort ? "cursor-pointer" : "cursor-not-allowed opacity-60",
					)}
				>
					{/* The fill ends under the thumb, which is wider than the track, so its end never shows. */}
					{index > 0 ? (
						<span
							aria-hidden="true"
							style={{ width: position(index) }}
							className={cn(
								"absolute inset-y-0 left-0 rounded-full transition-[width] ease-[cubic-bezier(0.22,1.1,0.36,1)]",
								ultra ? "ultra-fill" : "bg-primary",
								dragIndex === undefined ? "duration-200" : "duration-75",
							)}
						/>
					) : null}
					{efforts.map((choice, stop) => (
						<span
							key={choice.value}
							aria-hidden="true"
							style={{ left: position(stop) }}
							className={cn(
								"absolute top-1/2 size-1 -translate-x-1/2 -translate-y-1/2 rounded-full",
								stop <= index ? "bg-white/55" : "bg-muted-foreground/60",
							)}
						/>
					))}
					{index >= 0 ? (
						<span
							aria-hidden="true"
							style={{ left: position(index) }}
							className={cn(
								"absolute top-1/2 size-6 -translate-x-1/2 -translate-y-1/2 rounded-full bg-white shadow-[0_1px_2px_rgba(0,0,0,0.25)] transition-[left] ease-[cubic-bezier(0.22,1.1,0.36,1)]",
								dragIndex === undefined ? "duration-200" : "duration-75",
							)}
						/>
					) : null}
				</div>
			) : null}
		</div>
	);
}

function PanelToggle({
	label,
	pressed,
	disabled,
	onPress,
	children,
}: {
	label: string;
	pressed?: boolean;
	disabled?: boolean;
	onPress: () => void;
	children: ReactNode;
}) {
	return (
		<button
			type="button"
			aria-label={label}
			title={label}
			aria-pressed={pressed}
			disabled={disabled}
			onClick={onPress}
			className={cn(
				"flex size-5 shrink-0 items-center justify-center rounded hover:bg-interactive-active disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent",
				pressed ? "text-foreground" : "text-muted-foreground hover:text-foreground",
			)}
		>
			{children}
		</button>
	);
}

/** The strip on top of the chat box while a switch waits for the next message. */
export function PendingAgentSwitchNotice({
	pending,
	switching,
	onCancel,
}: {
	pending: PendingAgentSwitch;
	switching?: boolean;
	onCancel: () => void;
}) {
	const target = [agentLabel(pending.harness), pending.modelLabel].filter(Boolean).join(" · ");
	return (
		<div
			role="status"
			className="-mb-2 flex min-w-0 items-center gap-2 rounded-t-(--radius-chat-composer) border-b border-border bg-surface px-3 py-2 text-xs text-foreground"
		>
			<RefreshCw aria-hidden="true" className={cn("size-3.5 shrink-0 text-muted-foreground", switching ? "animate-spin" : undefined)} />
			<span className="shrink-0">{switching ? "Handing off to" : "Next message goes to"}</span>
			<ProviderMark provider={pending.harness} className="-mx-0.5 size-3.5" />
			<span className="min-w-0 truncate">{switching ? `${target}…` : `${target} with a handoff note.`}</span>
			{switching ? null : (
				<button
					type="button"
					aria-label="Keep the current agent"
					title="Keep the current agent"
					onClick={onCancel}
					className="ml-auto shrink-0 rounded p-0.5 text-muted-foreground hover:bg-interactive-active hover:text-foreground"
				>
					<X aria-hidden="true" className="size-3.5" />
				</button>
			)}
		</div>
	);
}

/** An error from a switch, shown as a strip on top of the chat box. */
export function ComposerErrorNotice({
	title,
	message,
	onDismiss,
	dismissLabel = "Dismiss",
}: {
	title: string;
	message: string;
	onDismiss?: () => void;
	dismissLabel?: string;
}) {
	return (
		<div
			role="alert"
			className="-mb-2 flex min-w-0 items-start gap-2 rounded-t-(--radius-chat-composer) border-b border-destructive/30 bg-destructive/10 px-3 py-2 text-xs"
		>
			<TriangleAlert aria-hidden="true" className="mt-0.5 size-3.5 shrink-0 text-destructive" />
			<div className="min-w-0 flex-1">
				<p className="font-medium text-foreground">{title}</p>
				<p className="break-words text-muted-foreground">{message}</p>
			</div>
			{onDismiss ? (
				<button
					type="button"
					aria-label={dismissLabel}
					title={dismissLabel}
					onClick={onDismiss}
					className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-interactive-active hover:text-foreground"
				>
					<X aria-hidden="true" className="size-3.5" />
				</button>
			) : null}
		</div>
	);
}

export interface AutoCompactControl {
	/** "200k", "1m", or unset to leave it to Claude Code. */
	value?: string;
	/** The agent is working or already restarting. */
	disabled?: boolean;
	onChange: (value: string | undefined) => void;
}

const AUTO_COMPACT_CHOICES = [
	{ value: undefined, label: "Auto" },
	{ value: "200k", label: "200k" },
	{ value: "1m", label: "1M" },
] as const;

/** One row under the effort slider: where Claude Code compacts on its own. */
export function AutoCompactRow({ control, title }: { control: AutoCompactControl; title: string }) {
	return (
		<div className="flex h-9 items-center gap-2 px-3 text-xs" title={title}>
			<span className="text-muted-foreground">Auto-compact</span>
			<span role="radiogroup" aria-label="Auto-compact" className="ml-auto flex rounded-md bg-interactive-active p-0.5">
				{AUTO_COMPACT_CHOICES.map((choice) => (
					<button
						key={choice.label}
						type="button"
						role="radio"
						aria-checked={control.value === choice.value}
						disabled={control.disabled || control.value === choice.value}
						onClick={() => control.onChange(choice.value)}
						className={cn(
							"rounded px-2 py-0.5 disabled:cursor-default",
							control.value === choice.value ? "bg-settings-menu text-foreground" : "text-muted-foreground hover:text-foreground",
						)}
					>
						{choice.label}
					</button>
				))}
			</span>
		</div>
	);
}
