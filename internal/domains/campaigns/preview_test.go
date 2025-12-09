package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns/models"
	customersModels "github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
	messagesModels "github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
)

// Mock repositories for PersonalizedPreview tests
type mockCampaignRepo struct {
	campaign models.Campaign
	err      error
}

func (m *mockCampaignRepo) GetCampaign(ctx context.Context, id int32) (models.Campaign, error) {
	return m.campaign, m.err
}

func (m *mockCampaignRepo) CreateCampaign(ctx context.Context, params models.CreateCampaignParams) (models.Campaign, error) {
	return models.Campaign{}, errors.New("not implemented")
}

func (m *mockCampaignRepo) UpdateCampaignToSending(ctx context.Context, id int32) (models.Campaign, error) {
	return models.Campaign{}, errors.New("not implemented")
}

func (m *mockCampaignRepo) ListCampaigns(ctx context.Context, params models.ListCampaignsParams) ([]models.Campaign, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCampaignRepo) CountCampaigns(ctx context.Context, params models.CountCampaignsParams) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockCampaignRepo) GetCampaignStats(ctx context.Context, id int32) (models.GetCampaignStatsRow, error) {
	return models.GetCampaignStatsRow{}, errors.New("not implemented")
}

func (m *mockCampaignRepo) GetCampaignStatsBatch(ctx context.Context, campaignIDs []int32) ([]models.GetCampaignStatsBatchRow, error) {
	return nil, errors.New("not implemented")
}

func (m *mockCampaignRepo) GetCampaignsReadyToSend(ctx context.Context) ([]models.GetCampaignsReadyToSendRow, error) {
	return nil, nil
}

var _ Repository = (*mockCampaignRepo)(nil)

type mockCustomersRepo struct {
	customer customersModels.GetCustomerForPreviewRow
	err      error
}

func (m *mockCustomersRepo) GetCustomerForPreview(ctx context.Context, id int32) (customersModels.GetCustomerForPreviewRow, error) {
	return m.customer, m.err
}

var _ CustomersRepository = (*mockCustomersRepo)(nil)

type mockMessagesRepo struct{}

func (m *mockMessagesRepo) CreateOutboundMessageBatch(ctx context.Context, params messagesModels.CreateOutboundMessageBatchParams) ([]messagesModels.OutboundMessage, error) {
	return nil, errors.New("not implemented")
}

var _ MessagesRepository = (*mockMessagesRepo)(nil)

// Test: Basic template rendering with all fields
func TestPersonalizedPreview_BasicRendering(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           1,
			BaseTemplate: "Hello {first_name} {last_name}! You're in {location} and prefer {prefered_product}.",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:              100,
			Firstname:       "John",
			Lastname:        "Doe",
			Phone:           "+254712345678",
			Location:        sql.NullString{String: "Nairobi", Valid: true},
			PreferedProduct: sql.NullString{String: "Premium Plan", Valid: true},
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 100,
	}

	result, err := service.PersonalizedPreview(ctx, 1, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Hello John Doe! You're in Nairobi and prefer Premium Plan."
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}

	if result.UsedTemplate != campaignRepo.campaign.BaseTemplate {
		t.Errorf("Expected used template to be base template, got %q", result.UsedTemplate)
	}

	if result.Customer.ID != 100 {
		t.Errorf("Expected customer ID 100, got %d", result.Customer.ID)
	}

	if result.Customer.FirstName != "John" {
		t.Errorf("Expected customer first name 'John', got %q", result.Customer.FirstName)
	}
}

// Test: Template rendering with null fields
func TestPersonalizedPreview_NullFields(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           2,
			BaseTemplate: "Hi {first_name}! Location: {location}, Product: {prefered_product}",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:              200,
			Firstname:       "Jane",
			Lastname:        "Smith",
			Phone:           "+254723456789",
			Location:        sql.NullString{Valid: false}, // NULL
			PreferedProduct: sql.NullString{Valid: false}, // NULL
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 200,
	}

	result, err := service.PersonalizedPreview(ctx, 2, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Null fields should be replaced with empty strings
	expectedMessage := "Hi Jane! Location: , Product: "
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}

	// Customer data should have nil pointers for null fields
	if result.Customer.Location != nil {
		t.Errorf("Expected customer location to be nil, got %v", result.Customer.Location)
	}

	if result.Customer.PreferedProduct != nil {
		t.Errorf("Expected customer preferred product to be nil, got %v", result.Customer.PreferedProduct)
	}
}

// Test: Override template parameter
func TestPersonalizedPreview_OverrideTemplate(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           3,
			BaseTemplate: "Original template: Hello {first_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        300,
			Firstname: "Alice",
			Lastname:  "Johnson",
			Phone:     "+254734567890",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	overrideTemplate := "Override template: Hi {first_name} {last_name}!"
	req := PersonalizedPreviewRequest{
		CustomerID:       300,
		OverrideTemplate: &overrideTemplate,
	}

	result, err := service.PersonalizedPreview(ctx, 3, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Override template: Hi Alice Johnson!"
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}

	if result.UsedTemplate != overrideTemplate {
		t.Errorf("Expected used template to be override template %q, got %q", overrideTemplate, result.UsedTemplate)
	}
}

// Test: Empty override template (should use base template)
func TestPersonalizedPreview_EmptyOverrideTemplate(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           4,
			BaseTemplate: "Base template: Hello {first_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        400,
			Firstname: "Bob",
			Lastname:  "Williams",
			Phone:     "+254745678901",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	emptyOverride := ""
	req := PersonalizedPreviewRequest{
		CustomerID:       400,
		OverrideTemplate: &emptyOverride,
	}

	result, err := service.PersonalizedPreview(ctx, 4, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should use base template when override is empty
	expectedMessage := "Base template: Hello Bob"
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}

	if result.UsedTemplate != campaignRepo.campaign.BaseTemplate {
		t.Errorf("Expected used template to be base template, got %q", result.UsedTemplate)
	}
}

// Test: Nil override template (should use base template)
func TestPersonalizedPreview_NilOverrideTemplate(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           5,
			BaseTemplate: "Base: {first_name} {last_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        500,
			Firstname: "Charlie",
			Lastname:  "Brown",
			Phone:     "+254756789012",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID:       500,
		OverrideTemplate: nil, // Explicitly nil
	}

	result, err := service.PersonalizedPreview(ctx, 5, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Base: Charlie Brown"
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}

	if result.UsedTemplate != campaignRepo.campaign.BaseTemplate {
		t.Errorf("Expected used template to be base template, got %q", result.UsedTemplate)
	}
}

// Test: Campaign not found
func TestPersonalizedPreview_CampaignNotFound(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		err: sql.ErrNoRows,
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        600,
			Firstname: "David",
			Lastname:  "Miller",
			Phone:     "+254767890123",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 600,
	}

	result, err := service.PersonalizedPreview(ctx, 999, req)
	if err == nil {
		t.Fatal("Expected error for campaign not found, got nil")
	}

	if err.Error() != "campaign not found" {
		t.Errorf("Expected error 'campaign not found', got %q", err.Error())
	}

	if result != nil {
		t.Error("Expected nil result for campaign not found")
	}
}

// Test: Customer not found
func TestPersonalizedPreview_CustomerNotFound(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           6,
			BaseTemplate: "Hello {first_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		err: sql.ErrNoRows,
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 999,
	}

	result, err := service.PersonalizedPreview(ctx, 6, req)
	if err == nil {
		t.Fatal("Expected error for customer not found, got nil")
	}

	if err.Error() != "customer not found" {
		t.Errorf("Expected error 'customer not found', got %q", err.Error())
	}

	if result != nil {
		t.Error("Expected nil result for customer not found")
	}
}

// Test: Multiple customers with same campaign
func TestPersonalizedPreview_DifferentCustomers(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           7,
			BaseTemplate: "Dear {first_name}, welcome to {location}!",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, nil, nil)

	testCases := []struct {
		name            string
		customer        customersModels.GetCustomerForPreviewRow
		expectedMessage string
	}{
		{
			name: "Customer 1",
			customer: customersModels.GetCustomerForPreviewRow{
				ID:        701,
				Firstname: "Emma",
				Lastname:  "Davis",
				Phone:     "+254778901234",
				Location:  sql.NullString{String: "Mombasa", Valid: true},
			},
			expectedMessage: "Dear Emma, welcome to Mombasa!",
		},
		{
			name: "Customer 2",
			customer: customersModels.GetCustomerForPreviewRow{
				ID:        702,
				Firstname: "Frank",
				Lastname:  "Wilson",
				Phone:     "+254789012345",
				Location:  sql.NullString{String: "Kisumu", Valid: true},
			},
			expectedMessage: "Dear Frank, welcome to Kisumu!",
		},
		{
			name: "Customer 3 with null location",
			customer: customersModels.GetCustomerForPreviewRow{
				ID:        703,
				Firstname: "Grace",
				Lastname:  "Taylor",
				Phone:     "+254790123456",
				Location:  sql.NullString{Valid: false},
			},
			expectedMessage: "Dear Grace, welcome to !",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Update the mock to return this customer
			customersRepo := &mockCustomersRepo{
				customer: tc.customer,
			}
			service.customersRepo = customersRepo

			req := PersonalizedPreviewRequest{
				CustomerID: tc.customer.ID,
			}

			result, err := service.PersonalizedPreview(ctx, 7, req)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if result.RenderedMessage != tc.expectedMessage {
				t.Errorf("Expected rendered message %q, got %q", tc.expectedMessage, result.RenderedMessage)
			}

			if result.Customer.ID != tc.customer.ID {
				t.Errorf("Expected customer ID %d, got %d", tc.customer.ID, result.Customer.ID)
			}
		})
	}
}

// Test: Override template with different placeholders
func TestPersonalizedPreview_OverrideWithDifferentPlaceholders(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           8,
			BaseTemplate: "Base: {first_name} {last_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:              800,
			Firstname:       "Henry",
			Lastname:        "Anderson",
			Phone:           "+254701234567",
			Location:        sql.NullString{String: "Nakuru", Valid: true},
			PreferedProduct: sql.NullString{String: "Gold", Valid: true},
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	// Override uses only phone and product
	overrideTemplate := "Call {phone} for {prefered_product} details"
	req := PersonalizedPreviewRequest{
		CustomerID:       800,
		OverrideTemplate: &overrideTemplate,
	}

	result, err := service.PersonalizedPreview(ctx, 8, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Call +254701234567 for Gold details"
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}
}

// Test: Special characters in customer data
func TestPersonalizedPreview_SpecialCharacters(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           9,
			BaseTemplate: "Hello {first_name} {last_name} from {location}!",
		},
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        900,
			Firstname: "O'Brien",
			Lastname:  "Smith-Jones",
			Phone:     "+254712345678",
			Location:  sql.NullString{String: "Nairobi (CBD)", Valid: true},
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 900,
	}

	result, err := service.PersonalizedPreview(ctx, 9, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMessage := "Hello O'Brien Smith-Jones from Nairobi (CBD)!"
	if result.RenderedMessage != expectedMessage {
		t.Errorf("Expected rendered message %q, got %q", expectedMessage, result.RenderedMessage)
	}
}

// Test: Database error for campaign
func TestPersonalizedPreview_CampaignDatabaseError(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		err: errors.New("database connection timeout"),
	}

	customersRepo := &mockCustomersRepo{
		customer: customersModels.GetCustomerForPreviewRow{
			ID:        1000,
			Firstname: "Test",
			Lastname:  "User",
			Phone:     "+254700000000",
		},
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 1000,
	}

	result, err := service.PersonalizedPreview(ctx, 10, req)
	if err == nil {
		t.Fatal("Expected error for database error, got nil")
	}

	if err.Error() != "database connection timeout" {
		t.Errorf("Expected error 'database connection timeout', got %q", err.Error())
	}

	if result != nil {
		t.Error("Expected nil result for database error")
	}
}

// Test: Database error for customer
func TestPersonalizedPreview_CustomerDatabaseError(t *testing.T) {
	ctx := context.Background()

	campaignRepo := &mockCampaignRepo{
		campaign: models.Campaign{
			ID:           11,
			BaseTemplate: "Hello {first_name}",
		},
	}

	customersRepo := &mockCustomersRepo{
		err: errors.New("database connection timeout"),
	}

	service := NewService(campaignRepo, &mockMessagesRepo{}, customersRepo, nil)

	req := PersonalizedPreviewRequest{
		CustomerID: 1100,
	}

	result, err := service.PersonalizedPreview(ctx, 11, req)
	if err == nil {
		t.Fatal("Expected error for database error, got nil")
	}

	if err.Error() != "database connection timeout" {
		t.Errorf("Expected error 'database connection timeout', got %q", err.Error())
	}

	if result != nil {
		t.Error("Expected nil result for database error")
	}
}
