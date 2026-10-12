package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
)

func resolveFixture(t *testing.T) (*Service, string, string) {
	t.Helper()
	base := t.TempDir()
	workspace := filepath.Join(base, "worktrees", "s")
	artifacts := filepath.Join(base, "artifacts", "s")
	for _, dir := range []string{filepath.Join(workspace, "src"), artifacts, filepath.Join(base, "worktrees", "s-2")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{filepath.Join(workspace, "src", "app.ts"), filepath.Join(artifacts, "report.md"), filepath.Join(base, "worktrees", "s-2", "app.ts")} {
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	st := newFakeStore()
	st.sessions["ao-1"] = domain.SessionRecord{ID: "ao-1", Metadata: domain.SessionMetadata{WorkspacePath: workspace, ArtifactDir: artifacts}}
	return &Service{store: st}, workspace, artifacts
}

func TestResolveWorkspacePathClassifiesPaths(t *testing.T) {
	svc, workspace, artifacts := resolveFixture(t)
	resolvedWorkspace, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		path     string
		wantKind WorkspacePathKind
		wantPath string
	}{
		{"absolute file in worktree", filepath.Join(workspace, "src", "app.ts"), WorkspacePathKindWorkspace, "src/app.ts"},
		{"symlink-resolved spelling of the worktree", filepath.Join(resolvedWorkspace, "src", "app.ts"), WorkspacePathKindWorkspace, "src/app.ts"},
		{"file that does not exist yet", filepath.Join(workspace, ".ao", "attachments", "new.png"), WorkspacePathKindWorkspace, ".ao/attachments/new.png"},
		{"relative path passes through cleaned", "./src/app.ts", WorkspacePathKindWorkspace, "src/app.ts"},
		{"artifact file", filepath.Join(artifacts, "report.md"), WorkspacePathKindArtifact, "report.md"},
		{"sibling sharing the root as a string prefix", workspace + "-2/app.ts", WorkspacePathKindOutside, ""},
		{"dot-dot escape", filepath.Join(workspace, "..", "s-2", "app.ts"), WorkspacePathKindOutside, ""},
		{"the worktree root itself", workspace, WorkspacePathKindOutside, ""},
		{"unrelated absolute path", "/etc/hosts", WorkspacePathKindOutside, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.ResolveWorkspacePath(context.Background(), "ao-1", tc.path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tc.wantKind || got.Path != tc.wantPath {
				t.Fatalf("ResolveWorkspacePath(%q) = {%s %q}, want {%s %q}", tc.path, got.Kind, got.Path, tc.wantKind, tc.wantPath)
			}
		})
	}
}

func TestResolveWorkspacePathRejectsEmptyAndEscapingRelativePaths(t *testing.T) {
	svc, _, _ := resolveFixture(t)
	for _, path := range []string{"", "   ", "../outside.txt"} {
		_, err := svc.ResolveWorkspacePath(context.Background(), "ao-1", path)
		assertAPIErrorCode(t, err, "INVALID_WORKSPACE_PATH")
	}
}

func TestResolveWorkspacePathUnknownSession(t *testing.T) {
	_, err := (&Service{store: newFakeStore()}).ResolveWorkspacePath(context.Background(), "missing", "/tmp/x")
	assertAPIErrorCode(t, err, "SESSION_NOT_FOUND")
}
