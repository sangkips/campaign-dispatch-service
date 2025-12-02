-- migration_name: create_outbound_messages_table
CREATE TABLE outbound_messages (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    campaign_id INTEGER NOT NULL,
    customer_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    rendered_content TEXT NOT NULL,
    last_error TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    
    -- Add provider tracking for actual message sending
    provider_message_id VARCHAR(255),
    sent_at TIMESTAMP,
    failed_at TIMESTAMP,
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Foreign key constraints
    CONSTRAINT fk_campaign
        FOREIGN KEY (campaign_id)
        REFERENCES campaigns(id)
        ON DELETE CASCADE,
    
    CONSTRAINT fk_customer
        FOREIGN KEY (customer_id)
        REFERENCES customer(id)
        ON DELETE CASCADE,
    
    -- Status validation based on transitions
    CONSTRAINT valid_status 
        CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    
    -- Retry count validation
    CONSTRAINT valid_retry_count 
        CHECK (retry_count >= 0),
    
    -- Unique constraint: one message per customer per campaign
    CONSTRAINT unique_campaign_customer 
        UNIQUE (campaign_id, customer_id)
);


-- Trigger for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_outbound_messages_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_outbound_messages_updated_at
    BEFORE UPDATE ON outbound_messages
    FOR EACH ROW
    EXECUTE FUNCTION update_outbound_messages_updated_at();

-- Optional: Create a table for campaign send jobs/queue if you want to persist queue jobs
CREATE TABLE IF NOT EXISTS campaign_send_jobs (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    outbound_message_id INTEGER NOT NULL,
    campaign_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    scheduled_for TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (outbound_message_id) REFERENCES outbound_messages(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE INDEX idx_campaign_send_jobs_pending ON campaign_send_jobs(status, scheduled_for)
WHERE status = 'pending';

CREATE INDEX idx_campaign_send_jobs_campaign ON campaign_send_jobs(campaign_id, status);

-- Create indexes for efficient querying
CREATE INDEX idx_outbound_messages_campaign_id ON outbound_messages(campaign_id);
CREATE INDEX idx_outbound_messages_customer_id ON outbound_messages(customer_id);
CREATE INDEX idx_outbound_messages_status ON outbound_messages(status);
CREATE INDEX idx_outbound_messages_created_at ON outbound_messages(created_at DESC);
CREATE INDEX idx_outbound_messages_campaign_status ON outbound_messages(campaign_id, status);
CREATE INDEX idx_outbound_messages_pending_retry ON outbound_messages(status, retry_count) 
WHERE status IN ('pending', 'failed') AND retry_count < 3;

-- Index for getting pending messages for a campaign (used in Send Campaign)
CREATE INDEX idx_outbound_messages_campaign_pending ON outbound_messages(campaign_id, id)
WHERE status = 'pending';
