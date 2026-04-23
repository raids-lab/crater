BEGIN;

ALTER TABLE agent_sessions
    ADD COLUMN IF NOT EXISTS last_orchestration_mode VARCHAR(32) DEFAULT 'single_agent';

UPDATE agent_sessions
SET last_orchestration_mode = 'single_agent'
WHERE last_orchestration_mode IS NULL
   OR BTRIM(last_orchestration_mode) = '';

ALTER TABLE agent_tool_calls
    ADD COLUMN IF NOT EXISTS turn_id UUID,
    ADD COLUMN IF NOT EXISTS tool_call_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS parent_event_id BIGINT,
    ADD COLUMN IF NOT EXISTS agent_role VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_turn_id ON agent_tool_calls(turn_id);
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_tool_call_id ON agent_tool_calls(tool_call_id);
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_agent_id ON agent_tool_calls(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_parent_event_id ON agent_tool_calls(parent_event_id);
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_agent_role ON agent_tool_calls(agent_role);

CREATE TABLE IF NOT EXISTS agent_turns (
    id BIGSERIAL PRIMARY KEY,
    turn_id UUID NOT NULL UNIQUE,
    session_id UUID NOT NULL,
    request_id VARCHAR(128),
    orchestration_mode VARCHAR(32) NOT NULL DEFAULT 'single_agent',
    root_agent_id VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'running',
    final_message_id BIGINT,
    metadata JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE agent_turns
    ADD COLUMN IF NOT EXISTS turn_id UUID,
    ADD COLUMN IF NOT EXISTS session_id UUID,
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS orchestration_mode VARCHAR(32),
    ADD COLUMN IF NOT EXISTS root_agent_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS status VARCHAR(32),
    ADD COLUMN IF NOT EXISTS final_message_id BIGINT,
    ADD COLUMN IF NOT EXISTS metadata JSONB,
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ended_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

ALTER TABLE agent_turns
    ALTER COLUMN orchestration_mode SET DEFAULT 'single_agent',
    ALTER COLUMN status SET DEFAULT 'running',
    ALTER COLUMN started_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_agent_turns_session_id ON agent_turns(session_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_turns_turn_id ON agent_turns(turn_id);
CREATE INDEX IF NOT EXISTS idx_agent_turns_request_id ON agent_turns(request_id);
CREATE INDEX IF NOT EXISTS idx_agent_turns_orchestration_mode ON agent_turns(orchestration_mode);
CREATE INDEX IF NOT EXISTS idx_agent_turns_root_agent_id ON agent_turns(root_agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_turns_status ON agent_turns(status);
CREATE INDEX IF NOT EXISTS idx_agent_turns_final_message_id ON agent_turns(final_message_id);
CREATE INDEX IF NOT EXISTS idx_agent_turns_started_at ON agent_turns(started_at);

CREATE TABLE IF NOT EXISTS agent_run_events (
    id BIGSERIAL PRIMARY KEY,
    turn_id UUID NOT NULL,
    session_id UUID NOT NULL,
    agent_id VARCHAR(128),
    parent_agent_id VARCHAR(128),
    agent_role VARCHAR(32),
    event_type VARCHAR(64) NOT NULL,
    event_status VARCHAR(32),
    title VARCHAR(255),
    content TEXT,
    metadata JSONB,
    sequence INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE agent_run_events
    ADD COLUMN IF NOT EXISTS turn_id UUID,
    ADD COLUMN IF NOT EXISTS session_id UUID,
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS parent_agent_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS agent_role VARCHAR(32),
    ADD COLUMN IF NOT EXISTS event_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS event_status VARCHAR(32),
    ADD COLUMN IF NOT EXISTS title VARCHAR(255),
    ADD COLUMN IF NOT EXISTS content TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB,
    ADD COLUMN IF NOT EXISTS sequence INTEGER,
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ended_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ;

ALTER TABLE agent_run_events
    ALTER COLUMN sequence SET DEFAULT 0,
    ALTER COLUMN created_at SET DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_agent_run_events_turn_id ON agent_run_events(turn_id);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_session_id ON agent_run_events(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_agent_id ON agent_run_events(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_parent_agent_id ON agent_run_events(parent_agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_agent_role ON agent_run_events(agent_role);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_event_type ON agent_run_events(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_event_status ON agent_run_events(event_status);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_sequence ON agent_run_events(sequence);
CREATE INDEX IF NOT EXISTS idx_agent_run_events_turn_sequence ON agent_run_events(turn_id, sequence);

COMMIT;

-- Add pinned_at column for session pinning feature
BEGIN;

ALTER TABLE agent_sessions
    ADD COLUMN IF NOT EXISTS pinned_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_agent_sessions_pinned_at ON agent_sessions(pinned_at);

COMMIT;
