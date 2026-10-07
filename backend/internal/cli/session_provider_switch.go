package cli

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

type providerSwitchRequest struct {
	TargetHarness  string `json:"targetHarness"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
}

type providerSwitchResponse struct {
	SessionID            string `json:"sessionId"`
	ConversationID       string `json:"conversationId"`
	BranchID             string `json:"branchId"`
	SourceHarness        string `json:"sourceHarness"`
	TargetHarness        string `json:"targetHarness"`
	ProviderConversation string `json:"providerConversationId"`
}

func newSessionSwitchProviderCommand(ctx *commandContext) *cobra.Command {
	var idempotencyKey string
	cmd := &cobra.Command{
		Use:   "switch-provider <id> <target-harness>",
		Short: "Switch an idle Chat session to another provider",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := normalizeSessionID(args[0])
			if err != nil {
				return err
			}
			return ctx.switchSessionProvider(cmd.Context(), cmd, id, args[1], idempotencyKey)
		},
	}
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Stable retry key for this handoff")
	return cmd
}

func (c *commandContext) switchSessionProvider(ctx context.Context, cmd *cobra.Command, id, target, key string) error {
	path := "sessions/" + url.PathEscape(id) + "/provider-switch"
	var response providerSwitchResponse
	if err := c.postJSON(ctx, path, providerSwitchRequest{TargetHarness: target, IdempotencyKey: key}, &response); err != nil {
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "session %s switched %s -> %s (branch %s)\n", response.SessionID, response.SourceHarness, response.TargetHarness, response.BranchID)
	return err
}
