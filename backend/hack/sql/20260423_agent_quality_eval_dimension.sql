-- Add dimensions for multi-scope / multi-type agent quality evaluations.
-- eval_scope: session | turn
-- eval_type: full | dialogue | task

ALTER TABLE agent_quality_evals
  ADD COLUMN IF NOT EXISTS eval_scope VARCHAR(16) NOT NULL DEFAULT 'session',
  ADD COLUMN IF NOT EXISTS eval_type VARCHAR(32) NOT NULL DEFAULT 'full',
  ADD COLUMN IF NOT EXISTS target_id VARCHAR(128),
  ADD COLUMN IF NOT EXISTS metadata JSONB;

UPDATE agent_quality_evals
SET eval_scope = CASE WHEN turn_id IS NULL THEN 'session' ELSE 'turn' END
WHERE eval_scope IS NULL OR BTRIM(eval_scope) = '';

UPDATE agent_quality_evals
SET eval_type = 'full'
WHERE eval_type IS NULL OR BTRIM(eval_type) = '';

UPDATE agent_quality_evals
SET target_id = CASE
  WHEN eval_scope = 'turn' AND turn_id IS NOT NULL THEN turn_id::text
  ELSE session_id::text
END
WHERE target_id IS NULL OR BTRIM(target_id) = '';

CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_session_created
  ON agent_quality_evals (session_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_scope_type_status
  ON agent_quality_evals (eval_scope, eval_type, eval_status);

CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_target_created
  ON agent_quality_evals (eval_scope, target_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_quality_evals_active
  ON agent_quality_evals (eval_status, created_at DESC)
  WHERE eval_status IN ('pending', 'running');
