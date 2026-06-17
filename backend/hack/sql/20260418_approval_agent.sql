-- Migration: Add agent approval fields to approval_orders
-- Date: 2026-04-18

ALTER TABLE approval_orders ADD COLUMN IF NOT EXISTS review_source VARCHAR(32) DEFAULT '';
ALTER TABLE approval_orders ADD COLUMN IF NOT EXISTS agent_report TEXT DEFAULT '';

-- Backfill existing records
UPDATE approval_orders SET review_source = 'system_auto'
WHERE review_notes LIKE '%approved due to system%' AND review_source = '';

UPDATE approval_orders SET review_source = 'admin_manual'
WHERE reviewer_id > 0 AND review_source = '' AND status IN ('Approved', 'Rejected');
