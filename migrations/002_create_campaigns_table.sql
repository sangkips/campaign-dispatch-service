-- migration_name: create_campaigns_table

CREATE TABLE campaigns (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    scheduled_at TIMESTAMP,
    base_template TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Add constraints for enums
    CONSTRAINT valid_channel CHECK (channel IN ('sms', 'whatsapp')),
    CONSTRAINT valid_status CHECK (status IN ('draft', 'scheduled', 'sending', 'sent', 'failed'))
);

CREATE OR REPLACE FUNCTION update_campaign_status_from_scheduled()
RETURNS TRIGGER AS $$
BEGIN
    -- If scheduled_at is in the past and status is scheduled, change to sending
    IF NEW.scheduled_at <= CURRENT_TIMESTAMP AND NEW.status = 'scheduled' THEN
        NEW.status := 'sending';
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- This trigger runs before insert or update on campaigns
CREATE TRIGGER update_campaign_status_from_scheduled_trigger
    BEFORE INSERT OR UPDATE ON campaigns
    FOR EACH ROW
    EXECUTE FUNCTION update_campaign_status_from_scheduled();

-- Create a function to process scheduled campaigns (for a worker)
CREATE OR REPLACE FUNCTION get_campaigns_ready_to_send()
RETURNS TABLE (
    campaign_id INTEGER,
    campaign_name VARCHAR(255),
    channel VARCHAR(50),
    base_template TEXT
) AS $$
BEGIN
    RETURN QUERY
    UPDATE campaigns c
    SET status = 'sending'
    WHERE c.status = 'scheduled' 
      AND c.scheduled_at <= CURRENT_TIMESTAMP
    RETURNING c.id, c.name, c.channel, c.base_template;
END;
$$ language 'plpgsql';

-- For GET /campaigns with pagination and filtering
CREATE INDEX idx_campaigns_created_at_desc ON campaigns(created_at DESC, id DESC);
CREATE INDEX idx_campaigns_channel_status ON campaigns(channel, status, created_at DESC);
CREATE INDEX idx_campaigns_status_created_at ON campaigns(status, created_at DESC);

-- For customer queries (used in personalized preview)
CREATE INDEX idx_customer_id_quick ON customer(id) INCLUDE (firstname, lastname, location, prefered_product);

-- For outbound_messages statistics queries (GET /campaigns/{id})
CREATE INDEX idx_outbound_messages_campaign_status_for_stats ON outbound_messages(campaign_id, status);


-- Create indexes for common queries
CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_channel ON campaigns(channel);
CREATE INDEX idx_campaigns_scheduled_at ON campaigns(scheduled_at);
CREATE INDEX idx_campaigns_created_at ON campaigns(created_at DESC);
CREATE INDEX idx_campaigns_name ON campaigns(name);