#!/bin/bash

# Seed database with sample data
# This script loads 100 customers and 3 campaigns for testing

set -e

echo "🌱 Seeding database with sample data..."

# Check if database is accessible
if ! docker compose exec -T db psql -U user -d campaign_db -c "SELECT 1" > /dev/null 2>&1; then
    echo "❌ Error: Database is not accessible. Make sure Docker Compose is running."
    echo "   Run: docker compose up -d"
    exit 1
fi

# Run seed script
docker compose exec -T db psql -U user -d campaign_db < migrations/004_seed_data.sql

echo "✅ Seed data loaded successfully!"
echo ""
echo "📊 Summary:"
echo "   - 100 customers created"
echo "   - 3 campaigns created (2 draft, 1 scheduled)"
echo ""
echo "🔍 Verify with:"
echo "   docker compose exec db psql -U user -d campaign_db -c 'SELECT COUNT(*) FROM customer;'"
echo "   docker compose exec db psql -U user -d campaign_db -c 'SELECT COUNT(*) FROM campaigns;'"
