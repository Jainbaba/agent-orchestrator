package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apispec"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/envelope"
	chatsvc "github.com/aoagents/agent-orchestrator/backend/internal/service/chat"
)

func (c *ConversationsController) providerSwitchService(w http.ResponseWriter, r *http.Request) (providerSwitchConversationService, bool) {
	svc, ok := c.Svc.(providerSwitchConversationService)
	if !ok {
		apispec.NotImplemented(w, r, r.Method, r.URL.Path)
	}
	return svc, ok
}

func (c *ConversationsController) switchProvider(w http.ResponseWriter, r *http.Request) {
	svc, ok := c.providerSwitchService(w, r)
	if !ok {
		return
	}
	var req ProviderSwitchRequest
	if !decodeConversationBody(w, r, &req) {
		return
	}
	result, err := svc.SwitchProvider(r.Context(), domain.SessionID(chi.URLParam(r, "sessionId")), domain.ProviderSwitchRequest{
		TargetHarness: domain.AgentHarness(req.TargetHarness), IdempotencyKey: req.IdempotencyKey,
		Model:    capActivityMeta(strings.TrimSpace(domain.SanitizeControlChars(req.Model))),
		Effort:   capActivityMeta(strings.TrimSpace(domain.SanitizeControlChars(req.Effort))),
		FastMode: req.FastMode, Ultracode: req.Ultracode, AutoCompactWindow: req.AutoCompactWindow,
	})
	if err != nil {
		writeProviderSwitchError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, ProviderSwitchResponse{
		SessionID: string(result.SessionID), ConversationID: result.ConversationID, BranchID: result.BranchID,
		SourceHarness: string(result.SourceHarness), TargetHarness: string(result.TargetHarness),
		ProviderConversation: result.ProviderConversation,
	})
}

func (c *ConversationsController) providerSwitchCapability(w http.ResponseWriter, r *http.Request) {
	svc, ok := c.providerSwitchService(w, r)
	if !ok {
		return
	}
	capability, err := svc.ProviderSwitchCapabilities(r.Context(), domain.SessionID(chi.URLParam(r, "sessionId")), domain.AgentHarness(chi.URLParam(r, "targetHarness")))
	if err != nil {
		writeConversationError(w, r, err)
		return
	}
	envelope.WriteJSON(w, http.StatusOK, ProviderSwitchCapabilityResponse{
		Harness: string(capability.Harness), Supported: capability.Supported,
		Reason: string(capability.Reason), Retryable: capability.Retryable,
	})
}

func writeProviderSwitchError(w http.ResponseWriter, r *http.Request, err error) {
	var refusal *chatsvc.ProviderSwitchError
	if !errors.As(err, &refusal) {
		writeConversationError(w, r, err)
		return
	}
	status, category := http.StatusConflict, "conflict"
	switch refusal.Kind {
	case domain.ProviderSwitchRefusalUnsupported:
		status, category = http.StatusBadRequest, "validation"
	case domain.ProviderSwitchRefusalUnavailable:
		status, category = http.StatusServiceUnavailable, "unavailable"
	case domain.ProviderSwitchRefusalBootstrapTooLarge:
		status, category = http.StatusRequestEntityTooLarge, "validation"
	}
	envelope.WriteAPIError(w, r, status, category, "PROVIDER_SWITCH_"+string(refusal.Kind), refusal.Error(), map[string]any{
		"retryable": refusal.Kind.Retryable(),
	})
}

const (
	sessionHistoryDefaultLimit = 10
	sessionHistoryMaxLimit     = 50
)

// sessionHistory serves `ao session history`: the turns AO holds for a session
// in full, newest first, so an agent that took over can read what its handoff
// note left out.
func (c *ConversationsController) sessionHistory(w http.ResponseWriter, r *http.Request) {
	svc, ok := c.Svc.(sessionHistoryService)
	if !ok {
		apispec.NotImplemented(w, r, r.Method, r.URL.Path)
		return
	}
	offset, limit := 0, sessionHistoryDefaultLimit
	for name, target := range map[string]*int{"offset": &offset, "limit": &limit} {
		raw := r.URL.Query().Get(name)
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || (name == "limit" && value == 0) {
			envelope.WriteAPIError(w, r, http.StatusBadRequest, "validation", "INVALID_HISTORY_PAGE", name+" must be a non-negative integer", nil)
			return
		}
		*target = value
	}
	limit = min(limit, sessionHistoryMaxLimit)
	turns, total, err := svc.SessionHistory(r.Context(), domain.SessionID(chi.URLParam(r, "sessionId")), offset, limit)
	if err != nil {
		writeConversationError(w, r, err)
		return
	}
	out := SessionHistoryResponse{Turns: make([]SessionHistoryTurn, 0, len(turns)), Total: total, Offset: offset}
	for _, turn := range turns {
		out.Turns = append(out.Turns, SessionHistoryTurn{
			UserRequest: turn.UserRequest, AgentResponse: turn.AgentResponse, At: turn.At, Harness: string(turn.Harness),
		})
	}
	envelope.WriteJSON(w, http.StatusOK, out)
}
