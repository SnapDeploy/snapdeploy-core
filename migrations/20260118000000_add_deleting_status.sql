-- +goose Up
-- +goose StatementBegin

-- Add DELETING status to the status check constraint for async deletion support
ALTER TABLE deployments DROP CONSTRAINT IF EXISTS deployments_status_check;
ALTER TABLE deployments ADD CONSTRAINT deployments_status_check CHECK (
    status IN (
        'PENDING',
        'BUILDING',
        'DEPLOYING',
        'DEPLOYED',
        'FAILED',
        'ROLLED_BACK',
        'EXPIRED',
        'DELETING'
    )
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove DELETING status from the constraint
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

-- +goose StatementEnd
