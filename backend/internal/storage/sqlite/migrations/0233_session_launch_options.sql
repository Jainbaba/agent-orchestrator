-- +goose Up
-- Choices a terminal agent only reads when it is launched (fast tier,
-- Ultracode, auto-compact window), as JSON. '' means none.
ALTER TABLE sessions ADD COLUMN launch_options TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sessions DROP COLUMN launch_options;
