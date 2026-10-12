package acp

import (
	"context"
	"errors"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestACPDriverUsesProviderHibernationCheck(t *testing.T) {
	for _, configured := range []bool{false, true} {
		cfg := Config{Launch: func(context.Context, LaunchConfig) (Launch, error) { return Launch{Command: "fake"}, nil }}
		var gotID acpsdk.SessionId
		ready := false
		var checkErr error
		if configured {
			cfg.CanHibernate = func(_ context.Context, _ *acpsdk.ClientSideConnection, id acpsdk.SessionId) (bool, error) {
				gotID = id
				return ready, checkErr
			}
		}
		d := New(cfg, nil)
		d.useTestProcess(fakeSpawn(&fakeAgent{}))
		conv, err := d.Start(context.Background(), ports.ChatStartConfig{WorkspacePath: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		h := conv.(ports.ChatProviderHibernator)
		canSleep, err := h.CanHibernate(context.Background())
		if err != nil || canSleep == configured {
			t.Fatalf("configured=%v CanHibernate=%v, %v", configured, canSleep, err)
		}
		if configured {
			if string(gotID) != conv.ProviderConversationID() {
				t.Fatalf("check session ID=%q, want %q", gotID, conv.ProviderConversationID())
			}
			ready = true
			if canSleep, err = h.CanHibernate(context.Background()); err != nil || !canSleep {
				t.Fatalf("ended task CanHibernate=%v, %v", canSleep, err)
			}
			// Output outside an AO prompt is work AO has no turn for.
			text := acpsdk.UpdateAgentMessageText("tests passed, building")
			if err := conv.(*conversation).processUpdate(acpsdk.SessionNotification{SessionId: gotID, Update: text}); err != nil {
				t.Fatal(err)
			}
			if canSleep, err = h.CanHibernate(context.Background()); err != nil || canSleep {
				t.Fatalf("unprompted output CanHibernate=%v, %v", canSleep, err)
			}
			conv.(*conversation).unpromptedOutputAt = time.Now().Add(-unpromptedOutputQuiet)
			if canSleep, err = h.CanHibernate(context.Background()); err != nil || !canSleep {
				t.Fatalf("quiet agent CanHibernate=%v, %v", canSleep, err)
			}
			checkErr = errors.New("native registry unavailable")
			if _, err = h.CanHibernate(context.Background()); !errors.Is(err, checkErr) {
				t.Fatalf("native check error=%v, want %v", err, checkErr)
			}
		}
		_ = conv.Close()
	}
}
