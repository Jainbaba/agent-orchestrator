package session

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/httpd/apierr"
	"github.com/aoagents/agent-orchestrator/backend/internal/sessionartifacts"
)

// WorkspacePathKind says which session-owned root a path belongs to.
type WorkspacePathKind string

const (
	// WorkspacePathKindWorkspace is a path inside the session worktree.
	WorkspacePathKindWorkspace WorkspacePathKind = "workspace"
	// WorkspacePathKindArtifact is a path inside the session artifact directory.
	WorkspacePathKindArtifact WorkspacePathKind = "artifact"
	// WorkspacePathKindOutside is a path in neither root.
	WorkspacePathKindOutside WorkspacePathKind = "outside"
)

// WorkspacePathResolution is a path reduced to the root that owns it. Path is
// relative to that root and empty for WorkspacePathKindOutside, so the caller
// never learns a local absolute path.
type WorkspacePathResolution struct {
	SessionID domain.SessionID
	Kind      WorkspacePathKind
	Path      string
}

// ResolveWorkspacePath classifies a path an agent wrote into chat. Agents link
// files by absolute path, but the renderer is never told where a session's
// worktree lives, so it asks here instead of guessing from the file catalog.
func (s *Service) ResolveWorkspacePath(ctx context.Context, id domain.SessionID, rawPath string) (WorkspacePathResolution, error) {
	rec, err := s.sessionWorkspaceRecord(ctx, id)
	if err != nil {
		return WorkspacePathResolution{}, err
	}
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return WorkspacePathResolution{}, apierr.Invalid("INVALID_WORKSPACE_PATH", "path is required", nil)
	}
	target := expandHomePath(trimmed)
	if !filepath.IsAbs(target) {
		// A rooted path that is not absolute here (a Unix path on a Windows
		// daemon, a Windows drive or UNC path on a Unix one, or a drive-relative
		// path) names somewhere this session does not own; it is not a
		// malformed relative path.
		if path.IsAbs(filepath.ToSlash(target)) || filepath.VolumeName(target) != "" || foreignAbsolutePath.MatchString(target) {
			return WorkspacePathResolution{SessionID: id, Kind: WorkspacePathKindOutside}, nil
		}
		rel, err := cleanWorkspaceRelativePath(trimmed)
		if err != nil {
			return WorkspacePathResolution{}, err
		}
		return WorkspacePathResolution{SessionID: id, Kind: WorkspacePathKindWorkspace, Path: rel}, nil
	}
	if rel, ok := relativeWithinRoot(rec.Metadata.WorkspacePath, target); ok {
		return WorkspacePathResolution{SessionID: id, Kind: WorkspacePathKindWorkspace, Path: rel}, nil
	}
	artifactDir := rec.Metadata.ArtifactDir
	if artifactDir == "" {
		artifactDir = sessionartifacts.Dir(s.dataDir, rec.ID)
	}
	if rel, ok := relativeWithinRoot(artifactDir, target); ok {
		return WorkspacePathResolution{SessionID: id, Kind: WorkspacePathKindArtifact, Path: rel}, nil
	}
	return WorkspacePathResolution{SessionID: id, Kind: WorkspacePathKindOutside}, nil
}

func expandHomePath(raw string) string {
	if raw != "~" && !strings.HasPrefix(raw, "~/") {
		return raw
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return raw
	}
	return filepath.Join(home, strings.TrimPrefix(raw, "~"))
}

// foreignAbsolutePath matches Windows drive (C:\x, C:/x) and UNC (\\host\share)
// spellings, which filepath does not treat as absolute on other platforms.
var foreignAbsolutePath = regexp.MustCompile(`^(?:[A-Za-z]:[\\/]|\\\\)`)

// relativeWithinRoot reports target relative to root when target is strictly
// inside it. Containment is decided on symlink-resolved paths, so a link inside
// the root that points out of it is not mistaken for a file the session owns;
// a target that does not exist yet is resolved through its deepest existing
// ancestor. The as-written spelling decides only when the root itself cannot
// be resolved.
func relativeWithinRoot(root, target string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" || !filepath.IsAbs(root) {
		return "", false
	}
	cleanRoot, cleanTarget := filepath.Clean(root), filepath.Clean(target)
	resolvedRoot, err := resolvedFilesystemPath(root)
	if err != nil {
		return relativeTo(cleanRoot, cleanTarget)
	}
	resolvedTarget, ok := resolveExistingPrefix(target)
	if !ok || !pathWithin(resolvedRoot, resolvedTarget) {
		return "", false
	}
	// Prefer the spelling the caller used: a link inside the root that stays
	// inside it is addressed by the link's own path, not its destination's.
	for _, base := range []string{cleanRoot, resolvedRoot} {
		if rel, ok := relativeTo(base, cleanTarget); ok {
			return rel, true
		}
	}
	return relativeTo(resolvedRoot, resolvedTarget)
}

func relativeTo(root, target string) (string, bool) {
	if !pathWithin(root, target) {
		return "", false
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// resolveExistingPrefix resolves symlinks in the longest existing ancestor of
// target and re-appends the missing tail, so a not-yet-created file under a
// symlinked directory still compares against a resolved root.
func resolveExistingPrefix(target string) (string, bool) {
	clean := filepath.Clean(target)
	tail := ""
	for current := clean; ; {
		if resolved, err := resolvedFilesystemPath(current); err == nil {
			return filepath.Join(resolved, tail), true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		tail = filepath.Join(filepath.Base(current), tail)
		current = parent
	}
}
