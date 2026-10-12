-- +goose Up

-- One row per Terminal UI turn, fed by the prompt-submit and stop hooks. Chat
-- sessions keep their history in the conversation tables; a terminal agent owns
-- its own transcript, so this is AO's only durable record of those turns.
CREATE TABLE session_terminal_turns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    harness TEXT NOT NULL,
    native_session_id TEXT NOT NULL DEFAULT '',
    user_request TEXT NOT NULL,
    agent_response TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMP NOT NULL,
    responded_at TIMESTAMP
);
CREATE INDEX idx_session_terminal_turns_session ON session_terminal_turns (session_id, id);

-- The handoff note a Chat provider switch gave the branch's provider. It is
-- re-applied when that provider restarts and is never part of the timeline.
ALTER TABLE conversation_branches ADD COLUMN handoff_note TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE conversation_branches DROP COLUMN handoff_note;
DROP INDEX idx_session_terminal_turns_session;
DROP TABLE session_terminal_turns;
