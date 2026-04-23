-- Extend ops_audit_reports with richer report data
ALTER TABLE ops_audit_reports
  ADD COLUMN IF NOT EXISTS report_json   JSONB,
  ADD COLUMN IF NOT EXISTS period_start  TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS period_end    TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS comparison_report_id UUID,
  ADD COLUMN IF NOT EXISTS job_total     INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS job_success   INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS job_failed    INT DEFAULT 0,
  ADD COLUMN IF NOT EXISTS job_pending   INT DEFAULT 0;

-- Extend ops_audit_items with job detail fields
ALTER TABLE ops_audit_items
  ADD COLUMN IF NOT EXISTS category           VARCHAR(32),
  ADD COLUMN IF NOT EXISTS job_type           VARCHAR(32),
  ADD COLUMN IF NOT EXISTS owner              VARCHAR(128),
  ADD COLUMN IF NOT EXISTS namespace          VARCHAR(128),
  ADD COLUMN IF NOT EXISTS duration_seconds   INT,
  ADD COLUMN IF NOT EXISTS resource_requested JSONB,
  ADD COLUMN IF NOT EXISTS resource_actual    JSONB,
  ADD COLUMN IF NOT EXISTS exit_code          INT,
  ADD COLUMN IF NOT EXISTS failure_reason     VARCHAR(64);

-- Index for fast report listing by type and date
CREATE INDEX IF NOT EXISTS idx_audit_reports_type_created
  ON ops_audit_reports (report_type, created_at DESC);

-- Index for filtering items by category
CREATE INDEX IF NOT EXISTS idx_audit_items_category
  ON ops_audit_items (category) WHERE category IS NOT NULL;

-- Index for filtering items by failure reason
CREATE INDEX IF NOT EXISTS idx_audit_items_failure
  ON ops_audit_items (failure_reason) WHERE failure_reason IS NOT NULL;
