-- name: CreateOutboundMessage :one
INSERT INTO outbound_messages (
    campaign_id,
    customer_id,
    rendered_content,
    status
) VALUES (
    @campaign_id,
    @customer_id,
    @rendered_content,
    'pending'
)
ON CONFLICT (campaign_id, customer_id) DO NOTHING
RETURNING *;

-- name: CreateOutboundMessageBatch :many
INSERT INTO outbound_messages (
    campaign_id,
    customer_id,
    rendered_content,
    status
) 
SELECT 
    @campaign_id,
    unnest(@customer_ids::integer[]),
    @rendered_content,
    'pending'
ON CONFLICT (campaign_id, customer_id) DO NOTHING
RETURNING *;

-- name: GetOutboundMessage :one
SELECT * FROM outbound_messages
WHERE id = @id LIMIT 1;

-- name: UpdateOutboundMessageStatus :one
UPDATE outbound_messages
SET 
    status = @status::varchar,
    sent_at = CASE WHEN @status::varchar = 'sent' THEN CURRENT_TIMESTAMP ELSE sent_at END,
    failed_at = CASE WHEN @status::varchar = 'failed' THEN CURRENT_TIMESTAMP ELSE failed_at END,
    last_error = @error_message,
    retry_count = CASE WHEN @status::varchar = 'failed' THEN retry_count + 1 ELSE retry_count END
WHERE id = @id
RETURNING *;

-- name: GetPendingMessagesForCampaign :many
SELECT * FROM outbound_messages
WHERE campaign_id = @campaign_id 
AND status = 'pending'
ORDER BY created_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountOutboundMessagesByCampaign :one
SELECT COUNT(*) FROM outbound_messages
WHERE campaign_id = @campaign_id;

-- name: GetFailedMessagesWithRetry :many
SELECT * FROM outbound_messages
WHERE status = 'failed'
AND retry_count < @max_retries
AND (updated_at < CURRENT_TIMESTAMP - INTERVAL '5 minutes' OR updated_at IS NULL)
ORDER BY updated_at ASC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetOutboundMessageWithDetails :one
SELECT 
    om.id,
    om.campaign_id,
    om.customer_id,
    om.status,
    om.rendered_content,
    om.last_error,
    om.retry_count,
    om.provider_message_id,
    om.sent_at,
    om.failed_at,
    om.created_at,
    om.updated_at,
    c.phone as customer_phone,
    c.firstname as customer_firstname,
    c.lastname as customer_lastname,
    c.location as customer_location,
    c.prefered_product as customer_prefered_product,
    camp.base_template as campaign_base_template,
    camp.channel as campaign_channel
FROM outbound_messages om
INNER JOIN customer c ON om.customer_id = c.id
INNER JOIN campaigns camp ON om.campaign_id = camp.id
WHERE om.id = @id
LIMIT 1;

-- name: UpdateOutboundMessageWithRetry :one
UPDATE outbound_messages
SET 
    status = @status::varchar,
    sent_at = CASE WHEN @status::varchar = 'sent' THEN CURRENT_TIMESTAMP ELSE sent_at END,
    failed_at = CASE WHEN @status::varchar = 'failed' THEN CURRENT_TIMESTAMP ELSE failed_at END,
    last_error = sqlc.narg('last_error'),
    retry_count = CASE WHEN @status::varchar = 'failed' THEN retry_count + 1 ELSE retry_count END,
    provider_message_id = sqlc.narg('provider_message_id')
WHERE id = @id
RETURNING *;