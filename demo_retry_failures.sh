#!/bin/bash

echo "=== Setting Up Retry and Failure Demonstration ==="
echo ""

# Create test outbound messages in the database
echo "Creating test outbound messages in database..."
docker compose exec db psql -U user -d campaign_db << 'EOF'
-- Insert test messages for demonstration
INSERT INTO outbound_messages (campaign_id, customer_id, rendered_content, status)
SELECT 
    1 as campaign_id,
    id as customer_id,
    'Test message for ' || firstname || ' ' || lastname as rendered_content,
    'pending' as status
FROM customer
WHERE id BETWEEN 1 AND 6
ON CONFLICT (campaign_id, customer_id) DO NOTHING;

SELECT 'Created ' || COUNT(*) || ' test messages' as result
FROM outbound_messages
WHERE campaign_id = 1 AND customer_id BETWEEN 1 AND 6;
EOF

echo ""
echo "Starting worker..."
docker compose up -d worker
sleep 3

echo ""
echo "Publishing messages to queue (this will trigger 95% success rate)..."
echo "With 95% success, we expect ~1 failure per 20 messages"
echo ""

# Publish 20 messages to increase chance of seeing failures
for i in {1..6}; do
    # Publish each message multiple times to trigger retries
    for retry in {1..4}; do
        docker compose exec rabbitmq rabbitmqadmin publish exchange=amq.default routing_key=campaign_sends payload="{\"outbound_message_id\": $i}" > /dev/null 2>&1
        echo -n "."
        sleep 0.2
    done
done
echo " Done!"

echo ""
echo "Waiting for processing (15 seconds)..."
sleep 15

echo ""
echo "=== Results ===" 
echo ""
echo "Message Status Summary:"
docker compose exec db psql -U user -d campaign_db -c "
SELECT 
    status,
    COUNT(*) as message_count,
    ROUND(AVG(retry_count), 2) as avg_retry_count,
    MAX(retry_count) as max_retry_count
FROM outbound_messages 
WHERE campaign_id = 1 AND customer_id BETWEEN 1 AND 6
GROUP BY status
ORDER BY status;
"

echo ""
echo "Detailed Message Status:"
docker compose exec db psql -U user -d campaign_db -c "
SELECT 
    id,
    customer_id,
    status,
    retry_count,
    CASE 
        WHEN last_error IS NOT NULL THEN LEFT(last_error, 40) || '...'
        ELSE 'No error'
    END as error_summary,
    CASE 
        WHEN sent_at IS NOT NULL THEN 'Yes'
        ELSE 'No'
    END as sent
FROM outbound_messages 
WHERE campaign_id = 1 AND customer_id BETWEEN 1 AND 6
ORDER BY customer_id, id;
"

echo ""
echo "Recent Worker Logs (showing retry behavior):"
docker compose logs worker --tail 30 | grep -E "(processing|sent|failed|retry)" || docker compose logs worker --tail 30

echo ""
echo "=== Explanation ===" 
echo "✓ 'sent' with retry_count=0: Succeeded on first try"
echo "✓ 'sent' with retry_count>0: Failed initially, then succeeded after retry"
echo "✗ 'failed' with retry_count<3: Currently failed, will retry"
echo "✗ 'failed' with retry_count=3: Permanently failed (max retries reached)"
echo ""
echo "Note: With 95% success rate, failures are rare. Run this script multiple times"
echo "or check the worker logs to see retry behavior when failures occur."
