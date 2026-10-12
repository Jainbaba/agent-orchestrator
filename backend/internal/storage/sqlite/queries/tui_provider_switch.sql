-- name: InsertTUIProviderSwitch :exec
INSERT INTO tui_provider_switches(id,session_id,request_id,phase,record_json,created_at)
VALUES (?,?,?,?,?,?);

-- name: UpdateTUIProviderSwitch :execrows
UPDATE tui_provider_switches SET phase=?,record_json=? WHERE id=? AND phase=?;

-- name: GetTUIProviderSwitch :one
SELECT record_json FROM tui_provider_switches WHERE session_id=? AND request_id=?;

-- name: LatestTUIProviderSwitch :one
SELECT record_json FROM tui_provider_switches WHERE session_id=? AND phase='completed' ORDER BY rowid DESC LIMIT 1;

-- name: ActiveTUIProviderSwitches :many
SELECT record_json FROM tui_provider_switches WHERE phase NOT IN ('completed','failed');

-- name: HasActiveTUIProviderSwitch :one
SELECT EXISTS(SELECT 1 FROM tui_provider_switches WHERE session_id=? AND phase NOT IN ('completed','failed'));

-- name: ChangeTUIProviderOwner :execrows
UPDATE sessions SET harness=sqlc.arg(target_harness),
    runtime_handle_id='',runtime_launch_id=sqlc.arg(target_launch_id),
    agent_session_id=sqlc.arg(target_native_id),agent_session_id_launch_id='',
    provider_conversation_id='',controller_generation='',
    native_identity_observed_at=NULL,native_transcript_path='',
    model=sqlc.arg(model),effort=sqlc.arg(effort),
    latest_user_prompt='',latest_assistant_update='',latest_assistant_update_at=NULL,
    conversation_checkpoint_state='empty',conversation_checkpoint_generation='',
    conversation_checkpoint_native_id='',conversation_checkpoint_turn_id='',
    conversation_checkpoint_unsettled=0,native_checkpoint_evidence='',
    claude_activity_facts='',codex_activity_facts='',first_signal_at=NULL,
    activity_state='idle',activity_last_at=sqlc.arg(now),updated_at=sqlc.arg(now)
WHERE id=sqlc.arg(id) AND harness=sqlc.arg(source_harness)
    AND runtime_launch_id=sqlc.arg(source_launch_id) AND session_mode='tui'
    AND is_terminated=0;

-- name: GetSessionLaunchOptions :one
SELECT launch_options FROM sessions WHERE id = ?;

-- name: SetSessionLaunchOptions :exec
UPDATE sessions SET launch_options = ? WHERE id = ?;
