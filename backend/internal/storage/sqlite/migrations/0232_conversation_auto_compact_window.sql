-- +goose Up
-- When Claude Code compacts on its own: '' leaves it to Claude Code.
ALTER TABLE conversations ADD COLUMN auto_compact_window TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE conversations DROP COLUMN auto_compact_window;
