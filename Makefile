# Load environment variables from .env file
include .env
export

build:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs

run:
	docker compose up

ps:
	docker compose ps

generate:
	sqlc generate

user:
	go test ./internal/domain/user -v

migrate-customers:
	PGPASSWORD=password psql -h localhost -p 5434 -U user -d campaign_db -f migrations/001_create_customers_table.sql

db-tables:
	PGPASSWORD=password psql -h localhost -p 5434 -U user -d campaign_db -c "\dt"

migrate-campaigns:
	PGPASSWORD=password psql -h localhost -p 5434 -U user -d campaign_db -f migrations/002_create_campaigns_table.sql

migrate-messages:
	PGPASSWORD=password psql -h localhost -p 5434 -U user -d campaign_db -f migrations/003_create_outbound_messages_table.sql

verify-campaign_status:
	docker compose exec db psql -U user -d campaign_db -c "SELECT id, name, status FROM campaigns WHERE id = 1;"

verify-outbound-messages:
	docker compose exec db psql -U user -d campaign_db -c "SELECT id, campaign_id, customer_id, status FROM outbound_messages WHERE campaign_id = 1;"


verify-outbound-messages-details:
	docker compose exec db psql -U user -d campaign_db -c "SELECT id, status, retry_count, last_error FROM outbound_messages;"

worker-logs:
	docker compose logs -f worker


test-render-template:
	go test -v ./internal/domains/campaigns -run TestRenderTemplate

test-customer-preview-data:
	go test -v ./internal/domains/campaigns -run TestToCustomerPreviewData

test-pagination:
	go test -v ./internal/domains/campaigns -run TestPagination

test-worker:
	go test -v ./internal/worker -run TestWorker

test-preview:
	go test -v ./internal/domains/campaigns -run TestPersonalizedPreview

test-scheduler:
	go test -v ./internal/worker -run TestScheduler
