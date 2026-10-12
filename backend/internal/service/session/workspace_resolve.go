package session

import (
	"context"
	"os"
	"path"
	"path/filepath"
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
		// daemon, or a drive-relative one) names somewhere this session does
		// not own; it is not a malformed relative path.
		if path.IsAbs(filepath.ToSlash(target)) || filepath.VolumeName(target) != "" {
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

func expandHomePath(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// relativeWithinRoot reports target relative to root when target is strictly
// inside it. It compares both the symlink-resolved and the as-written
// spellings, because a root under a symlinked directory can be written either
// way and a target that does not exist yet cannot be resolved at all.
func relativeWithinRoot(root, target string) (string, bool) {
	root = strings.TrimSpace(root)
	if root == "" || !filepath.IsAbs(root) {
		return "", false
	}
	pairs := [][2]string{{filepath.Clean(root), filepath.Clean(target)}}
	if resolvedRoot, err := resolvedFilesystemPath(root); err == nil {
		pairs = append(pairs, [2]string{resolvedRoot, filepath.Clean(target)})
		if resolvedTarget, ok := resolveExistingPrefix(target); ok {
			pairs = append(pairs, [2]string{resolvedRoot, resolvedTarget})
		}
	}
	for _, pair := range pairs {
		if !pathWithin(pair[0], pair[1]) {
			continue
		}
		rel, err := filepath.Rel(pair[0], pair[1])
		if err != nil {
			continue
		}
		return filepath.ToSlash(rel), true
	}
	return "", false
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
