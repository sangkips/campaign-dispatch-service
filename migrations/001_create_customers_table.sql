-- migration_name: create_customer_table
CREATE TABLE customer (
    id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone VARCHAR(255) NOT NULL,
    firstname VARCHAR(255) NOT NULL,
    lastname VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    prefered_product VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_phone UNIQUE(phone),
    CONSTRAINT valid_phone_format CHECK (
        -- Must start with +
        phone LIKE '+%' 
        -- Must contain only numbers after + (except possible spaces/dashes)
        AND phone ~ '^\+[0-9\s\-\(\)]+$'
        -- Must be at least 8 characters total (including +)
        AND LENGTH(REPLACE(REPLACE(phone, ' ', ''), '-', '')) >= 8
    )
);

CREATE UNIQUE INDEX idx_customer_phone ON customer(phone);
CREATE INDEX idx_customer_location ON customer(location);