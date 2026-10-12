-- name: InsertTerminalTurn :exec
INSERT INTO session_terminal_turns (session_id, harness, native_session_id, user_request, requested_at)
VALUES (sqlc.arg(session_id), sqlc.arg(harness), sqlc.arg(native_session_id), sqlc.arg(user_request), sqlc.arg(requested_at));

-- name: LatestTerminalTurn :one
SELECT id, session_id, harness, native_session_id, user_request, agent_response, requested_at, responded_at
FROM session_terminal_turns
WHERE session_id = sqlc.arg(session_id)
ORDER BY id DESC
LIMIT 1;

-- name: AppendTerminalTurnRequest :exec
UPDATE session_terminal_turns
SET user_request = user_request || char(10) || char(10) || sqlc.arg(user_request)
WHERE id = sqlc.arg(id);

-- name: CloseTerminalTurn :exec
UPDATE session_terminal_turns
SET agent_response = sqlc.arg(agent_response), responded_at = sqlc.arg(responded_at)
WHERE id = sqlc.arg(id);

-- name: TrimTerminalTurns :exec
DELETE FROM session_terminal_turns
WHERE session_terminal_turns.session_id = sqlc.arg(session_id)
  AND session_terminal_turns.id NOT IN (
    SELECT kept.id FROM session_terminal_turns AS kept
    WHERE kept.session_id = sqlc.arg(session_id)
    ORDER BY kept.id DESC
    LIMIT sqlc.arg(keep)
  );

-- name: ListTerminalTurns :many
SELECT id, session_id, harness, native_session_id, user_request, agent_response, requested_at, responded_at
FROM session_terminal_turns
WHERE session_id = sqlc.arg(session_id)
ORDER BY id DESC
LIMIT sqlc.arg(max_rows);

-- name: DeleteTerminalTurnsForThread :exec
DELETE FROM session_terminal_turns
WHERE session_id = sqlc.arg(session_id)
  AND harness = sqlc.arg(harness)
  AND (native_session_id = sqlc.arg(native_session_id) OR native_session_id = '');

-- name: SetConversationBranchHandoffNote :exec
UPDATE conversation_branches SET handoff_note = sqlc.arg(handoff_note) WHERE id = sqlc.arg(id);

-- name: ConversationBranchHandoffNote :one
SELECT handoff_note FROM conversation_branches WHERE id = sqlc.arg(id);
