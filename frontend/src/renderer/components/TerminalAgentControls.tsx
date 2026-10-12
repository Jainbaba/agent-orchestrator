import { MoreHorizontal, X } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useProviderSwitchCommand } from "../hooks/useConversation";
import { agentLabel } from "../lib/agent-options";
import { apiErrorMessage } from "../lib/api-client";
import { useUiStore } from "../stores/ui-store";
import type { WorkspaceSession } from "../types/workspace";
import { AgentModelPicker, type PendingAgentSwitch } from "./chat/AgentSwitchPicker";
import { ProviderMark } from "./chat/ProviderMark";
import { NewTerminalIcon, RoundedChatIcon } from "./SessionInterfaceSwitch";
import { TopbarButton } from "./TopbarButton";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "./ui/dropdown-menu";

const NO_CHOICES = { models: [], modelLabel: "", efforts: [] };

export interface TerminalSwitchAction {
	target: "chat" | "tui";
	disabled: boolean;
	reason?: string;
	run: () => void;
}

/**
 * The menu on a terminal session's tab: the running agent, which opens the
 * agent picker beside the menu, the switch to the chat view, and a new terminal.
 * Picking another agent adds Switch and Cancel under its settings; the switch
 * runs when Switch is pressed, because a terminal has no next message to wait on.
 */
export function TerminalSessionMenu({
	session,
	hostId,
	blocked,
	switchAction,
	switchControl,
	onNewTerminal,
}: {
	session: WorkspaceSession;
	hostId?: string;
	/** The agent is working, so it cannot be replaced right now. */
	blocked: boolean;
	switchAction?: TerminalSwitchAction;
	/** Shown in place of the menu while a switch to chat is running. */
	switchControl?: ReactNode;
	onNewTerminal?: () => void;
}) {
	const { t } = useTranslation();
	const [open, setOpen] = useState(false);
	const [pending, setPending] = useState<PendingAgentSwitch>();
	const commands = useProviderSwitchCommand(session.id, hostId);
	const showGlobalToast = useUiStore((state) => state.showGlobalToast);
	const agentSwitch = useMemo(
		() => ({ projectId: session.workspaceId, hostId, pending, onPendingChange: setPending }),
		[hostId, pending, session.workspaceId],
	);
	if (!switchAction && switchControl) return <div className="inline-flex shrink-0 items-center">{switchControl}</div>;
	const switchNow = () => {
		if (!pending) return;
		void commands.switchProvider(pending.harness, pending.model, pending.effort, pending.fastMode, {
			ultracode: pending.ultracode,
			autoCompactWindow: pending.autoCompactWindow,
		})
			.then(() => { setPending(undefined); setOpen(false); })
			.catch((error) => showGlobalToast("Agent switch failed", apiErrorMessage(error), "error"));
	};
	const target = pending ? [agentLabel(pending.harness), pending.modelLabel].filter(Boolean).join(" · ") : "";
	// One bordered bar under the picked agent's settings: what it switches to, Switch, cancel.
	const footer = pending ? (
		<div className="mt-1 box-border flex h-10 shrink-0 items-center border-t border-border px-1.5 pt-1">
			<div role="status" className="flex h-7 min-w-0 flex-1 items-center gap-1.5 rounded-lg border border-border bg-interactive-active/50 pl-2 pr-1 text-xs text-foreground">
				<ProviderMark provider={pending.harness} className="size-3.5" />
				<span className="min-w-0 flex-1 truncate">{target}</span>
				<button
					type="button"
					disabled={blocked || commands.switchProviderPending}
					title={blocked ? "The agent is working. Switch when it is idle." : `Switch this terminal to ${target}`}
					onClick={switchNow}
					className="shrink-0 rounded-md bg-primary px-2 py-px text-[11px] font-medium text-primary-foreground hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-40"
				>
					{commands.switchProviderPending ? "Switching…" : "Switch"}
				</button>
				<button
					type="button"
					aria-label="Cancel"
					title="Cancel"
					disabled={commands.switchProviderPending}
					onClick={() => setPending(undefined)}
					className="shrink-0 rounded p-0.5 text-muted-foreground hover:bg-interactive-active hover:text-foreground"
				>
					<X aria-hidden="true" className="size-3.5" />
				</button>
			</div>
		</div>
	) : null;
	return (
		<DropdownMenu open={open} onOpenChange={setOpen}>
			<DropdownMenuTrigger asChild>
				<TopbarButton
					aria-label={t("session.actionsMenu")}
					className="size-7 !bg-transparent text-muted-foreground hover:!bg-transparent active:!bg-transparent focus:!bg-transparent data-[state=open]:!bg-transparent hover:text-foreground"
					data-session-actions-trigger
					title={t("session.actionsMenu")}
					type="button"
					variant="icon"
				>
					<MoreHorizontal aria-hidden="true" className="size-icon-md" />
				</TopbarButton>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="start" className="min-w-48">
				<AgentModelPicker terminal currentHarness={session.provider} current={NO_CHOICES} agentSwitch={agentSwitch} pendingFooter={footer} />
				{switchAction ? (
					<DropdownMenuItem disabled={switchAction.disabled} title={switchAction.reason} onSelect={switchAction.run}>
						<RoundedChatIcon className="size-icon-lg" />
						Switch to chat UI
					</DropdownMenuItem>
				) : null}
				{onNewTerminal ? (
					<DropdownMenuItem onSelect={onNewTerminal}>
						<NewTerminalIcon className="size-icon-lg" />
						{t("shortcut.new-shell-terminal")}
					</DropdownMenuItem>
				) : null}
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
