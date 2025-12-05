# Campaign Dispatch Service

A service for managing and dispatching personalized marketing campaigns via SMS and WhatsApp. The service supports immediate and scheduled campaign dispatch with template-based message personalization.

## Features

- **Campaign Management**: Create, list, and retrieve campaigns with pagination and filtering
- **Scheduled Dispatch**: Automatically send campaigns at a specified future time
- **Template Personalization**: Dynamic message rendering with customer data
- **Multi-Channel Support**: SMS and WhatsApp delivery
- **Retry Logic**: Automatic retry for failed messages (up to 3 attempts)
- **Health Monitoring**: Health check endpoint for database and queue connectivity
- **Preview Functionality**: Preview personalized messages before sending

## Architecture

The service consists of three main components:

1. **API Server** (`cmd/server`): REST API for campaign management
2. **Worker** (`cmd/worker`): Background worker for message delivery
3. **Scheduler**: Background job that dispatches scheduled campaigns

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.21+
- Make
- Node.js 20+

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/sangkips/campaign-dispatch-service.git
   ```

2. **Navigate to the project directory**:
   ```bash
   cd campaign-dispatch-service
   ```

### Running the Service

1. **Start all services** (API, Worker, Database, RabbitMQ):
   ```bash
   make build
   ```
2. **Access frontend**:
   ```bash
   http://localhost:3001
   ```

3. **View logs**:
   ```bash
   make logs
   ```

4. **Stop services**:
   ```bash
   make down
   ```

### Running Locally (Development)

1. **Start dependencies** (Database and RabbitMQ):
   ```bash
   docker compose up db rabbitmq -d
   ```

2. **Run migrations**:
   ```bash
   make migrate-customers
   make migrate-campaigns
   make migrate-messages
   ```

3. **Load seed data** (optional - creates 10 customers and 3 campaigns):

Make sure to have correct permissions to run the script.

```bash
chmod +x seed.sh
```

   ```bash
   ./seed.sh
   ```

This creates:
- **10 customers** with varied data (some with null locations/products)
- **3 campaigns**:
  - Welcome Campaign (SMS, draft)
  - Product Promotion (WhatsApp, draft)

The API will be available at `http://0.0.0.0:8080`.

## API Endpoints

### Campaigns

- `POST /campaigns` - Create a new campaign
- `GET /campaigns` - List campaigns (with pagination and filters)
- `GET /campaigns/{id}` - Get campaign details with statistics
- `POST /campaigns/{id}/send` - Send campaign to customers
- `POST /campaigns/{id}/personalized-preview` - Preview personalized message

### Customers

- `POST /customers` - Create a new customer

### Health

- `GET /health` - Health check (database and queue connectivity)

### Personalized Preview

- `POST /{id}/personalized-preview` - Allows previewing how a campaign message will render for a specific customer with template variable substitution

## Testing

### Run All Tests

```bash
# Unit tests
go test -v ./...

# Specific test suites
make test-render-template      # Template rendering tests
make test-customer-preview-data # Customer data conversion tests
make test-pagination           # Pagination tests (requires DB)
make test-worker              # Worker logic tests
make test-preview             # Personalized preview tests
```


## Mock Sender Implementation

The service uses a mock sender to simulate message delivery without actual external API calls.

### Behavior

- **Success Rate**: 95% of messages succeed
- **Failure Rate**: 5% of messages fail randomly to test retry logic
- **Response**: Returns a mock provider message ID
- **Logging**: Logs all send attempts with customer phone and message content

I have included a script to test the mock sender in `demo_retry_failures.sh`.
Run the script with:

```bash
./demo_retry_failures.sh
```
Make sure to have correct permissions to run the script.

```bash
chmod +x demo_retry_failures.sh
```

### Rationale

The mock sender allows:
- **Development**: Test the full workflow without external dependencies
- **Testing**: Verify retry logic and error handling
- **Demonstration**: Show the system's capabilities without API credentials


## Scheduled Dispatch

### How It Works

1. **Campaign Creation**: Create a campaign with `scheduled_at` in the future
2. **Send Endpoint**: Call `/campaigns/{id}/send` with customer IDs
   - Creates `outbound_messages` with status `pending`
   - Does NOT immediately publish to queue
   - Campaign status remains `scheduled`
3. **Scheduler**: Background job runs every 10 seconds
   - Polls for campaigns where `scheduled_at <= NOW()` and status = `scheduled`
   - Atomically updates campaign status to `sending` (using `FOR UPDATE SKIP LOCKED`)
   - Publishes pending messages to RabbitMQ

#### Using Postman
##### Step 1: Create campaign
If you are using Postman, you can use a Pre-request Script to automatically calculate the future time.

- Open your POST /campaigns request.
- Go to the Pre-request Script tab.
- Add this code:
```bash
var date = new Date();
date.setMinutes(date.getMinutes() + 3); // Add 3 minutes
pm.environment.set("scheduled_at", date.toISOString());
```
- In the Body tab, use the variable to scheduled a campaign:
```bash
{
  "name": "Postman Scheduled Test",
  "channel": "sms",
  "base_template": "Hello {first_name}!",
  "scheduled_at": "{{scheduled_at}}"
}

```

##### Step 2: Add Recipients (Queue Messages)
Call the send endpoint. Since the campaign is scheduled for the future, this will only queue the messages and NOT send them immediately.
```bash
curl -X POST http://localhost:8080/campaigns/10/send \
  -H "Content-Type: application/json" \
  -d '{
    "customer_ids": [1, 2, 3]
  }'
  ```

Expected Response:
```bash
{
  "campaign_id": 10,
  "messages_queued": 3,
  "status": "scheduled"
}
```
##### Step 3: Verify Database State (Before Schedule)
Check that messages are pending and campaign is scheduled.
```bash
docker compose exec db psql -U user -d campaign_db -c "SELECT id, status FROM campaigns WHERE id = 10;"
docker compose exec db psql -U user -d campaign_db -c "SELECT id, status FROM outbound_messages WHERE campaign_id = 10;"

```
##### Step 4: Verify Database State (After Schedule)
Check that campaign status is sending (or sent if worker processed them) and messages are processed.

```bash
docker compose exec db psql -U user -d campaign_db -c "SELECT id, status FROM campaigns WHERE id = 10;"
docker compose exec db psql -U user -d campaign_db -c "SELECT id, status FROM outbound_messages WHERE campaign_id = 10;"
```
### Personalized Preview

Use this endpoint `POST /campaigns/{id}/personalized-preview`

##### Step 1: Basic Preview with Campaign Template
Request:

```bash
curl -X POST http://localhost:8080/campaigns/14/personalized-preview \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 3}'
  ```
Response:
```bash
{
  "rendered_message": "Hello Alice!",
  "used_template": "Hello {first_name}!",
  "customer": {
    "id": 3,
    "first_name": "Alice",
    "last_name": "Johnson",
    "phone": "+254712345678",
    "location": "Nairobi",
    "prefered_product": "Premium Package"
  }
}
```
###### Step 2: Preview with Override Template
Request:

```bash
curl -X POST http://localhost:8080/campaigns/14/personalized-preview \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 3,
    "override_template": "Hi {first_name}, special offer just for you in {location}!"
  }'
```
Response:

```bash
{
  "rendered_message": "Hi Alice, special offer just for you in Nairobi!",
  "used_template": "Hi {first_name}, special offer just for you in {location}!",
  "customer": {
    "id": 3,
    "first_name": "Alice",
    "last_name": "Johnson",
    "phone": "+254712345678",
    "location": "Nairobi",
    "prefered_product": "Shoes"
  }
}
```

### List Campaigns

##### 1. List All Campaigns (Page 1, Size 10)

```bash
curl -s "http://localhost:8080/campaigns?page=1&page_size=10"

```
Result:

```bash
{
  "data": [
    { "id": 12, "name": "Campaign 3", "channel": "sms", ... },
    ...
  ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total_count": 11,
    "total_pages": 2
  }

}
```

##### 2. Filter by Channel (SMS)

``` bash
curl -s "http://localhost:8080/campaigns?channel=sms"
```
Result:

```bash

{
  "data": [
    { "id": 12, "name": "Campaign 3", "channel": "sms", ... },
    ...
  ],
  "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 2,
      "total_pages": 1
  }
}

```

##### 3. Pagination (Page 2, Size 5)

```bash
curl -s "http://localhost:8080/campaigns?page=2&page_size=5"
```
Result:

```bash
{
  "data": [
    { "id": 7, "name": "Campaign 1", ... },
    ...
  ],
  "pagination": {
    "page": 2,
    "page_size": 5,
    "total_count": 12,
    "total_pages": 3
  }
}
```

## Environment Variables

Check whats in `.env.example` and copy it to your `.env`

## Monitoring

### Health Check

```bash
http://localhost:8080/health
```

**Response**:
```json
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy"
    },
    "queue": {
      "status": "healthy"
    }
  }
}
```

### Logs

- **Worker**: `docker compose logs -f worker`
- **Database**: `docker compose logs db`
- **RabbitMQ**: `docker compose logs rabbitmq`


## Troubleshooting

### Worker Not Processing Messages

1. Check RabbitMQ is running: `docker compose ps rabbitmq`
2. Check queue exists: Visit `http://localhost:15672` (guest/guest)
3. Check worker logs: `docker compose logs -f worker`

### Database Connection Issues

1. Verify DB is running: `docker compose ps db`
2. Check connection string in `.env`
3. Test connection: `make db-tables`

### NB

Check `Makefile` for database related commands

