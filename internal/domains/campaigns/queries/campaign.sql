-- campaigns.sql
-- name: CreateCampaign :one
INSERT INTO campaigns (
    name,
    channel,
    status,
    scheduled_at,
    base_template
) VALUES (
    @name,
    @channel,
    CASE 
        WHEN sqlc.narg('scheduled_at')::TIMESTAMP IS NOT NULL AND sqlc.narg('scheduled_at')::TIMESTAMP > CURRENT_TIMESTAMP THEN 'scheduled'
        ELSE 'draft'
    END,
    sqlc.narg('scheduled_at'),
    @base_template
)
RETURNING *;

-- name: GetCampaign :one
SELECT * FROM campaigns
WHERE id = @id LIMIT 1;

-- name: UpdateCampaignStatus :one
UPDATE campaigns
SET status = @status
WHERE id = @id
RETURNING *;

-- name: UpdateCampaignToSending :one
UPDATE campaigns
SET status = 'sending'
WHERE id = @id AND status IN ('draft', 'scheduled')
RETURNING *;

-- name: ListCampaigns :many
SELECT * FROM campaigns
WHERE 
    (sqlc.narg('channel')::text IS NULL OR channel = sqlc.narg('channel'))
    AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCampaigns :one
SELECT COUNT(*) FROM campaigns
WHERE 
    (sqlc.narg('channel')::text IS NULL OR channel = sqlc.narg('channel'))
    AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'));

-- name: GetCampaignStats :one
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
    COUNT(CASE WHEN status = 'sending' THEN 1 END) as sending,
    COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
FROM outbound_messages
WHERE campaign_id = @campaign_id;

-- name: GetCampaignStatsBatch :many
SELECT
    campaign_id,
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
    COUNT(CASE WHEN status = 'sending' THEN 1 END) as sending,
    COUNT(CASE WHEN status = 'sent' THEN 1 END) as sent,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
FROM outbound_messages
WHERE campaign_id = ANY(sqlc.arg('campaign_ids')::int[])
GROUP BY campaign_id;


-- name: GetCampaignsReadyToSend :many
-- name: GetCampaignsReadyToSend :many
UPDATE campaigns
SET status = 'sending'
WHERE id IN (
    SELECT id FROM campaigns
    WHERE status = 'scheduled'
    AND scheduled_at <= CURRENT_TIMESTAMP
    ORDER BY scheduled_at ASC
    FOR UPDATE SKIP LOCKED
)
RETURNING id, name, channel, base_template;