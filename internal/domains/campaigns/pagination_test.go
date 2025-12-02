package campaigns

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns/models"
)

// TestPagination_NoDuplicates, tests that pagination returns no duplicates across pages
func TestPagination_NoDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create 25 campaigns to have more than 2 pages (with page_size=10)
	campaignIDs := createTestCampaigns(t, ctx, repo, 25, "sms", "draft")

	// Fetch page 1
	page1, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{Valid: false},
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 1: %v", err)
	}

	// Fetch page 2
	page2, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{Valid: false},
		Limit:   10,
		Offset:  10,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 2: %v", err)
	}

	// Fetch page 3
	page3, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{Valid: false},
		Limit:   10,
		Offset:  20,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 3: %v", err)
	}

	// Check for duplicates across all pages
	seenIDs := make(map[int32]bool)
	allPages := [][]models.Campaign{page1, page2, page3}

	for pageNum, page := range allPages {
		for _, campaign := range page {
			if seenIDs[campaign.ID] {
				t.Errorf("Duplicate campaign ID %d found on page %d", campaign.ID, pageNum+1)
			}
			seenIDs[campaign.ID] = true
		}
	}

	// Verify we got all campaigns
	if len(seenIDs) != len(campaignIDs) {
		t.Errorf("Expected %d unique campaigns, got %d", len(campaignIDs), len(seenIDs))
	}
}

// TestPagination_ConsistentOrdering, tests that ordering is consistent across multiple queries
func TestPagination_ConsistentOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create 15 campaigns
	createTestCampaigns(t, ctx, repo, 15, "whatsapp", "draft")

	// Fetch the same page multiple times
	params := models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{Valid: false},
		Limit:   10,
		Offset:  0,
	}

	// First fetch
	firstFetch, err := repo.ListCampaigns(ctx, params)
	if err != nil {
		t.Fatalf("Failed first fetch: %v", err)
	}

	// Wait a moment to ensure time has passed
	time.Sleep(100 * time.Millisecond)

	// Second fetch
	secondFetch, err := repo.ListCampaigns(ctx, params)
	if err != nil {
		t.Fatalf("Failed second fetch: %v", err)
	}

	// Third fetch
	thirdFetch, err := repo.ListCampaigns(ctx, params)
	if err != nil {
		t.Fatalf("Failed third fetch: %v", err)
	}

	// Verify all fetches return the same order
	if len(firstFetch) != len(secondFetch) || len(firstFetch) != len(thirdFetch) {
		t.Fatalf("Fetches returned different counts: %d, %d, %d", len(firstFetch), len(secondFetch), len(thirdFetch))
	}

	for i := range firstFetch {
		if firstFetch[i].ID != secondFetch[i].ID {
			t.Errorf("Order mismatch at index %d: first=%d, second=%d", i, firstFetch[i].ID, secondFetch[i].ID)
		}
		if firstFetch[i].ID != thirdFetch[i].ID {
			t.Errorf("Order mismatch at index %d: first=%d, third=%d", i, firstFetch[i].ID, thirdFetch[i].ID)
		}
	}
}

// TestPagination_OrderByCreatedAtAndID, tests that campaigns are ordered by created_at DESC, id DESC
func TestPagination_OrderByCreatedAtAndID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create campaigns
	createTestCampaigns(t, ctx, repo, 20, "sms", "draft")

	// Fetch all campaigns
	campaigns, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{Valid: false},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch campaigns: %v", err)
	}

	// Verify ordering: created_at DESC, id DESC
	for i := 0; i < len(campaigns)-1; i++ {
		current := campaigns[i]
		next := campaigns[i+1]

		// If created_at is the same, ID should be descending
		if current.CreatedAt.Equal(next.CreatedAt) {
			if current.ID < next.ID {
				t.Errorf("ID ordering incorrect at index %d: current.ID=%d should be >= next.ID=%d", i, current.ID, next.ID)
			}
		} else {
			// created_at should be descending
			if current.CreatedAt.Before(next.CreatedAt) {
				t.Errorf("CreatedAt ordering incorrect at index %d: current=%v should be after next=%v", i, current.CreatedAt, next.CreatedAt)
			}
		}
	}
}

// TestPagination_ChannelFilter, tests that channel filter is respected
func TestPagination_ChannelFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create campaigns with different channels
	createTestCampaigns(t, ctx, repo, 10, "sms", "draft")
	createTestCampaigns(t, ctx, repo, 8, "whatsapp", "draft")

	// Test SMS filter
	smsCampaigns, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "sms", Valid: true},
		Status:  sql.NullString{Valid: false},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch SMS campaigns: %v", err)
	}

	if len(smsCampaigns) != 10 {
		t.Errorf("Expected 10 SMS campaigns, got %d", len(smsCampaigns))
	}

	for _, campaign := range smsCampaigns {
		if campaign.Channel != "sms" {
			t.Errorf("Expected channel 'sms', got '%s'", campaign.Channel)
		}
	}

	// Test WhatsApp filter
	whatsappCampaigns, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "whatsapp", Valid: true},
		Status:  sql.NullString{Valid: false},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch WhatsApp campaigns: %v", err)
	}

	if len(whatsappCampaigns) != 8 {
		t.Errorf("Expected 8 WhatsApp campaigns, got %d", len(whatsappCampaigns))
	}

	for _, campaign := range whatsappCampaigns {
		if campaign.Channel != "whatsapp" {
			t.Errorf("Expected channel 'whatsapp', got '%s'", campaign.Channel)
		}
	}
}

// TestPagination_StatusFilter, tests that status filter is respected
func TestPagination_StatusFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create campaigns with different statuses
	createTestCampaigns(t, ctx, repo, 12, "sms", "draft")
	createTestCampaigns(t, ctx, repo, 7, "sms", "scheduled")
	createTestCampaigns(t, ctx, repo, 6, "sms", "sending")

	// Test draft filter
	draftCampaigns, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{String: "draft", Valid: true},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch draft campaigns: %v", err)
	}

	if len(draftCampaigns) != 12 {
		t.Errorf("Expected 12 draft campaigns, got %d", len(draftCampaigns))
	}

	for _, campaign := range draftCampaigns {
		if campaign.Status != "draft" {
			t.Errorf("Expected status 'draft', got '%s'", campaign.Status)
		}
	}

	// Test scheduled filter
	scheduledCampaigns, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{Valid: false},
		Status:  sql.NullString{String: "scheduled", Valid: true},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch scheduled campaigns: %v", err)
	}

	if len(scheduledCampaigns) != 7 {
		t.Errorf("Expected 7 scheduled campaigns, got %d", len(scheduledCampaigns))
	}

	for _, campaign := range scheduledCampaigns {
		if campaign.Status != "scheduled" {
			t.Errorf("Expected status 'scheduled', got '%s'", campaign.Status)
		}
	}
}

// TestPagination_CombinedFilters, tests that channel and status filters work together
func TestPagination_CombinedFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create campaigns with various combinations
	createTestCampaigns(t, ctx, repo, 5, "sms", "draft")
	createTestCampaigns(t, ctx, repo, 3, "sms", "scheduled")
	createTestCampaigns(t, ctx, repo, 4, "whatsapp", "draft")
	createTestCampaigns(t, ctx, repo, 2, "whatsapp", "scheduled")

	// Test SMS + draft filter
	smsDraft, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "sms", Valid: true},
		Status:  sql.NullString{String: "draft", Valid: true},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch SMS draft campaigns: %v", err)
	}

	if len(smsDraft) != 5 {
		t.Errorf("Expected 5 SMS draft campaigns, got %d", len(smsDraft))
	}

	for _, campaign := range smsDraft {
		if campaign.Channel != "sms" || campaign.Status != "draft" {
			t.Errorf("Expected SMS draft campaign, got channel='%s', status='%s'", campaign.Channel, campaign.Status)
		}
	}

	// Test whatsapp + scheduled filter
	whatsappScheduled, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "whatsapp", Valid: true},
		Status:  sql.NullString{String: "scheduled", Valid: true},
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch whatsapp scheduled campaigns: %v", err)
	}

	if len(whatsappScheduled) != 2 {
		t.Errorf("Expected 2 whatsapp scheduled campaigns, got %d", len(whatsappScheduled))
	}

	for _, campaign := range whatsappScheduled {
		if campaign.Channel != "whatsapp" || campaign.Status != "scheduled" {
			t.Errorf("Expected whatsapp scheduled campaign, got channel='%s', status='%s'", campaign.Channel, campaign.Status)
		}
	}
}

// TestPagination_FilteredNoDuplicates, tests no duplicates when using filters with pagination
func TestPagination_FilteredNoDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := connectTestDB(t)
	defer db.Close()

	tx := setupTestTx(t, db)
	defer tx.Rollback()

	ctx := context.Background()
	repo := NewRepository(tx)

	// Create 25 SMS draft campaigns
	createTestCampaigns(t, ctx, repo, 25, "sms", "draft")
	// Create some other campaigns to ensure filter works
	createTestCampaigns(t, ctx, repo, 10, "whatsapp", "draft")

	// Fetch page 1 with filter
	page1, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "sms", Valid: true},
		Status:  sql.NullString{String: "draft", Valid: true},
		Limit:   10,
		Offset:  0,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 1: %v", err)
	}

	// Fetch page 2 with filter
	page2, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "sms", Valid: true},
		Status:  sql.NullString{String: "draft", Valid: true},
		Limit:   10,
		Offset:  10,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 2: %v", err)
	}

	// Fetch page 3 with filter
	page3, err := repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: sql.NullString{String: "sms", Valid: true},
		Status:  sql.NullString{String: "draft", Valid: true},
		Limit:   10,
		Offset:  20,
	})
	if err != nil {
		t.Fatalf("Failed to fetch page 3: %v", err)
	}

	// Check for duplicates
	seenIDs := make(map[int32]bool)
	allPages := [][]models.Campaign{page1, page2, page3}

	for pageNum, page := range allPages {
		for _, campaign := range page {
			if seenIDs[campaign.ID] {
				t.Errorf("Duplicate campaign ID %d found on page %d", campaign.ID, pageNum+1)
			}
			seenIDs[campaign.ID] = true

			// Verify filter is applied
			if campaign.Channel != "sms" || campaign.Status != "draft" {
				t.Errorf("Filter not applied: got channel='%s', status='%s'", campaign.Channel, campaign.Status)
			}
		}
	}

	// Verify we got all 25 SMS draft campaigns
	if len(seenIDs) != 25 {
		t.Errorf("Expected 25 unique SMS draft campaigns, got %d", len(seenIDs))
	}
}


func connectTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Get database URL from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
		if dbURL == "" {
			t.Skip("TEST_DATABASE_URL or DB_URL not set, skipping integration test")
		}
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

func setupTestTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Clean up test data within the transaction
	// This ensures we start with a clean state for the test
	_, err = tx.Exec("DELETE FROM outbound_messages")
	if err != nil {
		t.Fatalf("Failed to clean up outbound_messages: %v", err)
	}

	_, err = tx.Exec("DELETE FROM campaigns")
	if err != nil {
		t.Fatalf("Failed to clean up campaigns: %v", err)
	}

	return tx
}

func createTestCampaigns(t *testing.T, ctx context.Context, repo Repository, count int, channel, status string) []int32 {
	t.Helper()

	var ids []int32
	for i := 0; i < count; i++ {
		// For scheduled status, set a future scheduled_at time
		var scheduledAt sql.NullTime
		if status == "scheduled" {
			futureTime := time.Now().Add(24 * time.Hour)
			scheduledAt = sql.NullTime{Time: futureTime, Valid: true}
		}

		campaign, err := repo.CreateCampaign(ctx, models.CreateCampaignParams{
			Name:         fmt.Sprintf("Test Campaign %s %s %d", channel, status, i),
			Channel:      channel,
			ScheduledAt:  scheduledAt,
			BaseTemplate: fmt.Sprintf("Test template for %s campaign", channel),
		})
		if err != nil {
			t.Fatalf("Failed to create test campaign: %v", err)
		}

		// If we need a specific status that's not draft or scheduled, update it
		if status != "draft" && status != "scheduled" {
			updatedCampaign, err := updateCampaignStatus(ctx, repo, campaign.ID, status)
			if err != nil {
				t.Fatalf("Failed to update campaign status: %v", err)
			}
			ids = append(ids, updatedCampaign.ID)
		} else {
			ids = append(ids, campaign.ID)
		}

		// Small delay to ensure different created_at timestamps
		time.Sleep(1 * time.Millisecond)
	}

	return ids
}

// updateCampaignStatus is a helper to update campaign status for testing
func updateCampaignStatus(ctx context.Context, repo Repository, id int32, status string) (models.Campaign, error) {
	// This is a workaround since we don't have a direct UpdateCampaignStatus in the Repository interface
	r := repo.(*repository)
	return r.q.UpdateCampaignStatus(ctx, models.UpdateCampaignStatusParams{
		ID:     id,
		Status: status,
	})
}
