-- Seed data for Campaign Dispatch Service
-- This script creates 10 sample customers and 3 sample campaigns for testing

-- Insert 10 sample customers
INSERT INTO customer (phone, firstname, lastname, location, prefered_product) VALUES
('+254712345001', 'John', 'Doe', 'Nairobi', 'Premium'),
('+254712345002', 'Jane', 'Smith', 'Mombasa', 'Standard'),
('+254712345003', 'Michael', 'Johnson', 'Kisumu', 'Basic'),
('+254712345004', 'Sarah', 'Williams', 'Nakuru', 'Premium'),
('+254712345005', 'David', 'Brown', 'Eldoret', 'Standard'),
('+254712345006', 'Emily', 'Jones', 'Nairobi', NULL),
('+254712345007', 'Daniel', 'Garcia', NULL, 'Premium'),
('+254712345008', 'Lisa', 'Martinez', 'Mombasa', 'Basic'),
('+254712345009', 'James', 'Rodriguez', 'Thika', NULL),
('+254712345010', 'Maria', 'Lopez', 'Nairobi', 'Standard');

-- Insert 3 sample campaigns
INSERT INTO campaigns (name, channel, status, scheduled_at, base_template) VALUES
('Welcome Campaign', 'sms', 'draft', NULL, 'Hello {first_name} {last_name}! Welcome to our store. We are located at: {location}'),
('Product Promotion', 'whatsapp', 'draft', NULL, 'Hi {first_name}, special offer just for you in {location}! Check out your preferred product {prefered_product}.'),
('Seasonal Offer', 'sms', 'draft', NULL, 'Dear {first_name}, enjoy our seasonal offers! Visit us at {location}.');
-- Display inserted data
SELECT 'Customers inserted:' as info;
SELECT id, phone, firstname, lastname, location, prefered_product FROM customer ORDER BY id;

SELECT 'Campaigns inserted:' as info;
SELECT id, name, channel, status, scheduled_at FROM campaigns ORDER BY id;
