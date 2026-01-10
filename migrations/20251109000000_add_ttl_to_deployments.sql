-- +goose Up
-- +goose StatementBegin

-- Add TTL fields to deployments table for ephemeral deployment support
ALTER TABLE deployments ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE deployments ADD COLUMN extended_count INTEGER DEFAULT 0 NOT NULL;

-- Add index for efficient expired deployment queries
CREATE INDEX idx_deployments_expires_at ON deployments (expires_at) WHERE expires_at IS NOT NULL;

-- Add EXPIRED status to the status check constraint
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS deployments_status_check;
ALTER TABLE deployments ADD CONSTRAINT deployments_status_check CHECK (
    status IN (
        'PENDING',
        'BUILDING',
        'DEPLOYING',
        'DEPLOYED',
        'FAILED',
        'ROLLED_BACK',
        'EXPIRED'
    )
);

-- Backfill existing DEPLOYED deployments with expires_at = created_at + 6 hours
UPDATE deployments 
SET expires_at = created_at + INTERVAL '6 hours'
WHERE status = 'DEPLOYED' AND expires_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove the new columns
ALTER TABLE deployments DROP COLUMN IF EXISTS expires_at;
ALTER TABLE deployments DROP COLUMN IF EXISTS extended_count;

-- Restore original status constraint
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS deployments_status_check;
ALTER TABLE deployments ADD CONSTRAINT deployments_status_check CHECK (
    status IN (
        'PENDING',
        'BUILDING',
        'DEPLOYING',
        'DEPLOYED',
        'FAILED',
        'ROLLED_BACK'
    )
);

-- Drop the index
DROP INDEX IF EXISTS idx_deployments_expires_at;

-- +goose StatementEnd


