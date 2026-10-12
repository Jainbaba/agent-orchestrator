-- +goose Up
CREATE TABLE tui_provider_switches (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    phase TEXT NOT NULL CHECK (phase IN ('prepared','source_stopping','target_starting','completed','failed','recovery_required')),
    record_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    UNIQUE (session_id, request_id)
);
CREATE UNIQUE INDEX tui_provider_switch_active ON tui_provider_switches(session_id)
WHERE phase NOT IN ('completed','failed');

-- +goose StatementBegin
CREATE TRIGGER tui_provider_switch_cdc_insert AFTER INSERT ON tui_provider_switches
BEGIN
    INSERT INTO change_log(project_id,session_id,event_type,payload,created_at)
    SELECT project_id,id,'session_updated',json_object('id',id,'providerSwitchPhase',NEW.phase),NEW.created_at
    FROM sessions WHERE id=NEW.session_id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER tui_provider_switch_cdc_update AFTER UPDATE ON tui_provider_switches
BEGIN
    INSERT INTO change_log(project_id,session_id,event_type,payload,created_at)
    SELECT project_id,id,'session_updated',json_object('id',id,'providerSwitchPhase',NEW.phase),CURRENT_TIMESTAMP
    FROM sessions WHERE id=NEW.session_id;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS tui_provider_switch_cdc_update;
DROP TRIGGER IF EXISTS tui_provider_switch_cdc_insert;
DROP TABLE tui_provider_switches;
