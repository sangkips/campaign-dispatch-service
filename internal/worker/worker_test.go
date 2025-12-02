package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages"
	messagesModels "github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
	"github.com/sangkips/campaign-dispatch-service/internal/queue"
)

// Mock Repository
type mockRepository struct {
	getMessageDetails      messagesModels.GetOutboundMessageWithDetailsRow
	getMessageDetailsError error
	updateMessageResult    messagesModels.OutboundMessage
	updateMessageError     error

	updateCalls []messagesModels.UpdateOutboundMessageWithRetryParams
	getCalls    []int32

	// Function hooks for dynamic mocking
	getOutboundMessageFunc func(ctx context.Context, id int32) (messagesModels.GetOutboundMessageWithDetailsRow, error)
	updateMessageFunc      func(ctx context.Context, params messagesModels.UpdateOutboundMessageWithRetryParams) (messagesModels.OutboundMessage, error)
	getPendingMessagesFunc func(ctx context.Context, params messagesModels.GetPendingMessagesForCampaignParams) ([]messagesModels.OutboundMessage, error)
}

func (m *mockRepository) GetOutboundMessageWithDetails(ctx context.Context, id int32) (messagesModels.GetOutboundMessageWithDetailsRow, error) {
	m.getCalls = append(m.getCalls, id)
	if m.getOutboundMessageFunc != nil {
		return m.getOutboundMessageFunc(ctx, id)
	}
	return m.getMessageDetails, m.getMessageDetailsError
}

func (m *mockRepository) UpdateOutboundMessageWithRetry(ctx context.Context, params messagesModels.UpdateOutboundMessageWithRetryParams) (messagesModels.OutboundMessage, error) {
	m.updateCalls = append(m.updateCalls, params)
	if m.updateMessageFunc != nil {
		return m.updateMessageFunc(ctx, params)
	}
	return m.updateMessageResult, m.updateMessageError
}

func (m *mockRepository) CreateOutboundMessage(ctx context.Context, params messagesModels.CreateOutboundMessageParams) (messagesModels.OutboundMessage, error) {
	return messagesModels.OutboundMessage{}, errors.New("not implemented")
}

func (m *mockRepository) CreateOutboundMessageBatch(ctx context.Context, params messagesModels.CreateOutboundMessageBatchParams) ([]messagesModels.OutboundMessage, error) {
	return nil, errors.New("not implemented")
}

func (m *mockRepository) CountOutboundMessagesByCampaign(ctx context.Context, campaignID int32) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockRepository) GetPendingMessagesForCampaign(ctx context.Context, params messagesModels.GetPendingMessagesForCampaignParams) ([]messagesModels.OutboundMessage, error) {
	if m.getPendingMessagesFunc != nil {
		return m.getPendingMessagesFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

var _ messages.Repository = (*mockRepository)(nil)

// Mock Sender
type mockSender struct {
	shouldFail   bool
	sendError    error
	sentMessages []sentMessage
}

type sentMessage struct {
	content string
	to      string
}

func (m *mockSender) Send(content string, to string) (string, error) {
	m.sentMessages = append(m.sentMessages, sentMessage{content: content, to: to})
	if m.shouldFail {
		return "", m.sendError
	}
	return "mock-provider-msg-123", nil
}

var _ Sender = (*mockSender)(nil)

// Mock Delivery tracker - tracks what happened to a delivery
type deliveryTracker struct {
	acked    bool
	nacked   bool
	requeued bool
	rejected bool
}

// Helper function to create a delivery and tracker
func createTestDelivery(messageID int32) (amqp091.Delivery, *deliveryTracker) {
	msg := queue.CampaignSendMessage{
		OutboundMessageID: messageID,
	}
	body, _ := json.Marshal(msg)

	tracker := &deliveryTracker{}

	delivery := amqp091.Delivery{
		Body:         body,
		Acknowledger: &mockAcknowledger{tracker: tracker},
	}

	return delivery, tracker
}

// Mock Acknowledger
type mockAcknowledger struct {
	tracker *deliveryTracker
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.tracker.acked = true
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple, requeue bool) error {
	m.tracker.nacked = true
	m.tracker.requeued = requeue
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	m.tracker.rejected = true
	m.tracker.requeued = requeue
	return nil
}

var _ amqp091.Acknowledger = (*mockAcknowledger)(nil)

// Test: Successful message processing
func TestWorker_ProcessMessage_Success(t *testing.T) {
	ctx := context.Background()

	// Setup mock repository
	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                      1,
			CampaignID:              100,
			CustomerID:              200,
			Status:                  "pending",
			RetryCount:              0,
			CustomerPhone:           "+254712345678",
			CustomerFirstname:       "John",
			CustomerLastname:        "Doe",
			CustomerLocation:        sql.NullString{String: "Nairobi", Valid: true},
			CustomerPreferedProduct: sql.NullString{String: "Premium", Valid: true},
			CampaignBaseTemplate:    "Hello {first_name} {last_name}! Your product: {prefered_product}",
			CampaignChannel:         "sms",
		},
		updateMessageResult: messagesModels.OutboundMessage{
			ID:     1,
			Status: "sent",
		},
	}

	// Setup mock sender (success)
	sender := &mockSender{shouldFail: false}

	// Create worker
	worker := &Worker{
		repo:   repo,
		sender: sender,
	}

	// Create test delivery
	delivery, tracker := createTestDelivery(1)

	// Process message
	worker.processMessage(ctx, delivery)

	// Assertions
	if !tracker.acked {
		t.Error("Expected message to be acknowledged")
	}

	if len(repo.getCalls) != 1 || repo.getCalls[0] != 1 {
		t.Errorf("Expected GetOutboundMessageWithDetails to be called with ID 1, got calls: %v", repo.getCalls)
	}

	if len(sender.sentMessages) != 1 {
		t.Fatalf("Expected 1 message to be sent, got %d", len(sender.sentMessages))
	}

	expectedContent := "Hello John Doe! Your product: Premium"
	if sender.sentMessages[0].content != expectedContent {
		t.Errorf("Expected rendered content %q, got %q", expectedContent, sender.sentMessages[0].content)
	}

	if sender.sentMessages[0].to != "+254712345678" {
		t.Errorf("Expected message sent to +254712345678, got %s", sender.sentMessages[0].to)
	}

	if len(repo.updateCalls) != 1 {
		t.Fatalf("Expected 1 update call, got %d", len(repo.updateCalls))
	}

	updateCall := repo.updateCalls[0]
	if updateCall.Status != "sent" {
		t.Errorf("Expected status 'sent', got %s", updateCall.Status)
	}

	if !updateCall.ProviderMessageID.Valid || updateCall.ProviderMessageID.String != "mock-provider-msg-123" {
		t.Errorf("Expected provider message ID to be set, got %v", updateCall.ProviderMessageID)
	}
}

// Test: Success with nullable customer fields
func TestWorker_ProcessMessage_SuccessWithNullableFields(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                      2,
			CustomerPhone:           "+254723456789",
			CustomerFirstname:       "Jane",
			CustomerLastname:        "Smith",
			CustomerLocation:        sql.NullString{Valid: false},
			CustomerPreferedProduct: sql.NullString{Valid: false},
			CampaignBaseTemplate:    "Hi {first_name}! Location: {location}, Product: {prefered_product}",
		},
		updateMessageResult: messagesModels.OutboundMessage{ID: 2, Status: "sent"},
	}

	sender := &mockSender{shouldFail: false}
	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(2)

	worker.processMessage(ctx, delivery)

	if !tracker.acked {
		t.Error("Expected message to be acknowledged")
	}

	// Verify template rendering handles nulls correctly (replaced with empty strings)
	expectedContent := "Hi Jane! Location: , Product: "
	if len(sender.sentMessages) > 0 && sender.sentMessages[0].content != expectedContent {
		t.Errorf("Expected rendered content %q, got %q", expectedContent, sender.sentMessages[0].content)
	}
}

// Test: Send failure - first retry
func TestWorker_ProcessMessage_SendFailure_FirstRetry(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                   3,
			RetryCount:           0,
			CustomerPhone:        "+254734567890",
			CustomerFirstname:    "Alice",
			CustomerLastname:     "Johnson",
			CampaignBaseTemplate: "Hello {first_name}",
		},
		updateMessageResult: messagesModels.OutboundMessage{
			ID:         3,
			Status:     "failed",
			RetryCount: 1,
		},
	}

	sender := &mockSender{
		shouldFail: true,
		sendError:  errors.New("provider error: network timeout"),
	}

	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(3)

	worker.processMessage(ctx, delivery)

	// Should be nacked for retry
	if !tracker.nacked {
		t.Error("Expected message to be nacked")
	}

	if !tracker.requeued {
		t.Error("Expected message to be requeued")
	}

	// Verify status updated to failed
	if len(repo.updateCalls) != 1 {
		t.Fatalf("Expected 1 update call, got %d", len(repo.updateCalls))
	}

	updateCall := repo.updateCalls[0]
	if updateCall.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", updateCall.Status)
	}

	if !updateCall.LastError.Valid || updateCall.LastError.String != "provider error: network timeout" {
		t.Errorf("Expected error message to be stored, got %v", updateCall.LastError)
	}
}

// Test: Send failure - max retries reached
func TestWorker_ProcessMessage_SendFailure_MaxRetriesReached(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                   4,
			RetryCount:           3, // Already at max retries
			CustomerPhone:        "+254745678901",
			CustomerFirstname:    "Bob",
			CustomerLastname:     "Williams",
			CampaignBaseTemplate: "Hello {first_name}",
		},
		updateMessageResult: messagesModels.OutboundMessage{
			ID:         4,
			Status:     "failed",
			RetryCount: 3,
		},
	}

	sender := &mockSender{
		shouldFail: true,
		sendError:  errors.New("provider error: invalid phone number"),
	}

	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(4)

	worker.processMessage(ctx, delivery)

	// Should be acked (not requeued) since max retries reached
	if !tracker.acked {
		t.Error("Expected message to be acknowledged (max retries reached)")
	}

	if tracker.nacked {
		t.Error("Expected message NOT to be nacked (max retries reached)")
	}

	// Verify status updated to failed
	if len(repo.updateCalls) != 1 {
		t.Fatalf("Expected 1 update call, got %d", len(repo.updateCalls))
	}

	updateCall := repo.updateCalls[0]
	if updateCall.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", updateCall.Status)
	}
}

// Test: Send failure - second retry
func TestWorker_ProcessMessage_SendFailure_SecondRetry(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                   5,
			RetryCount:           1, // First retry already happened
			CustomerPhone:        "+254756789012",
			CustomerFirstname:    "Charlie",
			CustomerLastname:     "Brown",
			CampaignBaseTemplate: "Hello {first_name}",
		},
		updateMessageResult: messagesModels.OutboundMessage{
			ID:         5,
			Status:     "failed",
			RetryCount: 2,
		},
	}

	sender := &mockSender{
		shouldFail: true,
		sendError:  errors.New("provider error: rate limit exceeded"),
	}

	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(5)

	worker.processMessage(ctx, delivery)

	// Should be nacked for another retry
	if !tracker.nacked {
		t.Error("Expected message to be nacked for retry")
	}

	if !tracker.requeued {
		t.Error("Expected message to be requeued")
	}
}

// Test: Invalid JSON in queue message
func TestWorker_ProcessMessage_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{}
	sender := &mockSender{}
	worker := &Worker{repo: repo, sender: sender}

	// Create delivery with invalid JSON
	tracker := &deliveryTracker{}
	delivery := amqp091.Delivery{
		Body:         []byte("invalid json {{{"),
		Acknowledger: &mockAcknowledger{tracker: tracker},
	}

	worker.processMessage(ctx, delivery)

	// Should be rejected without requeue
	if !tracker.rejected {
		t.Error("Expected message to be rejected")
	}

	if tracker.requeued {
		t.Error("Expected message NOT to be requeued (invalid JSON)")
	}

	// No repository or sender calls should be made
	if len(repo.getCalls) != 0 {
		t.Error("Expected no repository calls for invalid JSON")
	}

	if len(sender.sentMessages) != 0 {
		t.Error("Expected no sender calls for invalid JSON")
	}
}

// Test: Message not found in database
func TestWorker_ProcessMessage_MessageNotFound(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetailsError: sql.ErrNoRows,
	}

	sender := &mockSender{}
	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(999)

	worker.processMessage(ctx, delivery)

	// Should be rejected without requeue
	if !tracker.rejected {
		t.Error("Expected message to be rejected (not found)")
	}

	if tracker.requeued {
		t.Error("Expected message NOT to be requeued (record missing)")
	}

	// No sender calls should be made
	if len(sender.sentMessages) != 0 {
		t.Error("Expected no sender calls for missing record")
	}
}

// Test: Database error (transient)
func TestWorker_ProcessMessage_DatabaseError(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetailsError: errors.New("database connection timeout"),
	}

	sender := &mockSender{}
	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(6)

	worker.processMessage(ctx, delivery)

	// Should be nacked with requeue errors
	if !tracker.nacked {
		t.Error("Expected message to be nacked (database error)")
	}

	if !tracker.requeued {
		t.Error("Expected message to be requeued error")
	}
}

// Test: Template rendering with special characters
func TestWorker_ProcessMessage_TemplateWithSpecialCharacters(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		getMessageDetails: messagesModels.GetOutboundMessageWithDetailsRow{
			ID:                   8,
			CustomerPhone:        "+254778901234",
			CustomerFirstname:    "O'Brien",
			CustomerLastname:     "Smith-Jones",
			CustomerLocation:     sql.NullString{String: "Nairobi (CBD)", Valid: true},
			CampaignBaseTemplate: "Hello {first_name} {last_name} from {location}!",
		},
		updateMessageResult: messagesModels.OutboundMessage{ID: 8, Status: "sent"},
	}

	sender := &mockSender{shouldFail: false}
	worker := &Worker{repo: repo, sender: sender}
	delivery, tracker := createTestDelivery(8)

	worker.processMessage(ctx, delivery)

	if !tracker.acked {
		t.Error("Expected message to be acknowledged")
	}

	// Verify special characters are preserved
	expectedContent := "Hello O'Brien Smith-Jones from Nairobi (CBD)!"
	if len(sender.sentMessages) > 0 && sender.sentMessages[0].content != expectedContent {
		t.Errorf("Expected rendered content %q, got %q", expectedContent, sender.sentMessages[0].content)
	}
}
