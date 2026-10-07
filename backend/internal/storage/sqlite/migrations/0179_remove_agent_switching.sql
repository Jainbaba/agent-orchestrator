-- +goose Up

-- Provider/harness switching was removed from the product. Keep the historical
-- migrations immutable, but remove their durable schema from existing stores.
DROP TRIGGER IF EXISTS agent_switches_cdc_insert;
DROP TRIGGER IF EXISTS agent_switches_cdc_update;
DROP TRIGGER IF EXISTS agent_switches_target_native_scope_insert;
DROP TRIGGER IF EXISTS agent_switches_target_native_scope_update;
DROP TRIGGER IF EXISTS agent_switches_failed_recovery_marker_insert;
DROP TRIGGER IF EXISTS agent_switches_failed_recovery_marker_update;

DROP INDEX IF EXISTS idx_agent_switch_failure_outbox_pending;
DROP INDEX IF EXISTS idx_agent_switches_one_active_per_session;
DROP INDEX IF EXISTS idx_agent_switches_session_history;
DROP INDEX IF EXISTS idx_agent_native_sessions_native_identity;
DROP INDEX IF EXISTS idx_agent_native_sessions_resume_candidate;

DROP TABLE IF EXISTS agent_switch_failure_receipts;
DROP TABLE IF EXISTS agent_switch_failure_outbox;
DROP TABLE IF EXISTS agent_switch_failure_delivery_state;
DROP TABLE IF EXISTS agent_switch_failure_policy;
DROP TABLE IF EXISTS agent_switches;
DROP TABLE IF EXISTS agent_native_sessions;

-- +goose Down
SELECT 1;
