-- Add source field to agent_sessions to distinguish chat vs non-chat sessions.
-- Values: 'chat' (default, user-initiated conversations shown in UI),
--         'ops_audit' (admin tool execution audit, hidden from chat UI),
--         'system' (background/automated agent operations),
--         'benchmark' (evaluation runs).
ALTER TABLE agent_sessions ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'chat';
UPDATE agent_sessions
SET source = 'chat'
WHERE source IS NULL OR BTRIM(source) = '';
UPDATE agent_sessions
SET source = 'ops_audit'
WHERE title LIKE '[audit] 审批%'
  AND source = 'chat';
CREATE INDEX IF NOT EXISTS idx_agent_sessions_source ON agent_sessions (source);

-- Add source field to agent_tool_calls to track execution origin.
-- Values: 'backend' (default, executed via Go backend API),
--         'local' (executed locally by Python agent via kubectl/prometheus),
--         'benchmark' (executed during evaluation with mock/live executor).
ALTER TABLE agent_tool_calls ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'backend';
UPDATE agent_tool_calls
SET source = 'backend'
WHERE source IS NULL OR BTRIM(source) = '';
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_source ON agent_tool_calls (source);
