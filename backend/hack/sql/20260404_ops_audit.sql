-- Migration: 20260404_ops_audit
-- Date: 2026-04-04
-- Description: Create ops_audit_reports and ops_audit_items tables for AIOps audit system

CREATE TABLE IF NOT EXISTS ops_audit_reports (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type     VARCHAR(32)     NOT NULL,           -- 'gpu_audit', 'capacity_planning'
    status          VARCHAR(16)     NOT NULL DEFAULT 'running', -- 'running', 'completed', 'failed'
    trigger_source  VARCHAR(32),                        -- 'cron', 'manual', 'agent'
    summary         JSONB,
    created_at      TIMESTAMPTZ     DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_audit_items (
    id              BIGSERIAL       PRIMARY KEY,
    report_id       UUID            NOT NULL REFERENCES ops_audit_reports(id),
    job_name        VARCHAR(255)    NOT NULL,
    user_id         VARCHAR(128),
    account_id      VARCHAR(128),
    username        VARCHAR(128),
    action_type     VARCHAR(32)     NOT NULL,           -- 'stop', 'notify', 'downscale'
    severity        VARCHAR(16)     NOT NULL,           -- 'critical', 'warning', 'info'
    gpu_utilization FLOAT,
    gpu_requested   INT,
    gpu_actual_used INT,
    analysis_detail JSONB,                              -- GPU analysis prompt output
    handled         BOOLEAN         DEFAULT FALSE,
    handled_at      TIMESTAMPTZ,
    handled_by      VARCHAR(128),
    created_at      TIMESTAMPTZ     DEFAULT NOW()
);

-- Indexes for ops_audit_items
CREATE INDEX idx_audit_items_report  ON ops_audit_items(report_id);
CREATE INDEX idx_audit_items_action  ON ops_audit_items(action_type, severity);
CREATE INDEX idx_audit_items_handled ON ops_audit_items(handled) WHERE NOT handled;

-- Indexes for ops_audit_reports
CREATE INDEX idx_audit_reports_type_status ON ops_audit_reports(report_type, status);
CREATE INDEX idx_audit_reports_created     ON ops_audit_reports(created_at DESC);
