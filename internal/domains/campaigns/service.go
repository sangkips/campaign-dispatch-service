package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns/models"
	customersModels "github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
	messagesModels "github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
)

type Service struct {
	repo          Repository
	messagesRepo  MessagesRepository
	customersRepo CustomersRepository
	queue         QueuePublisher
}

func NewService(repo Repository, messagesRepo MessagesRepository, customersRepo CustomersRepository, queue QueuePublisher) *Service {
	return &Service{
		repo:          repo,
		messagesRepo:  messagesRepo,
		customersRepo: customersRepo,
		queue:         queue,
	}
}

type CreateCampaignRequest struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Channel      string     `json:"channel"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
	BaseTemplate string     `json:"base_template"`
}

type SendCampaignRequest struct {
	CustomerIDs []int32 `json:"customer_ids"`
}

type SendCampaignResponse struct {
	CampaignID     int32  `json:"campaign_id"`
	MessagesQueued int    `json:"messages_queued"`
	Status         string `json:"status"`
}

// MessagesRepository interface for message operations
type MessagesRepository interface {
	CreateOutboundMessageBatch(ctx context.Context, params messagesModels.CreateOutboundMessageBatchParams) ([]messagesModels.OutboundMessage, error)
}

// QueuePublisher interface for publishing messages to queue
type QueuePublisher interface {
	PublishCampaignSend(messageID int32) error
}

// CustomersRepository interface for customer operations
type CustomersRepository interface {
	GetCustomerForPreview(ctx context.Context, id int32) (customersModels.GetCustomerForPreviewRow, error)
}

// SendCampaign validates campaign and creates outbound messages
func (s *Service) SendCampaign(ctx context.Context, campaignID int32, req SendCampaignRequest) (*SendCampaignResponse, error) {
	// Validate customer_ids is not empty
	if len(req.CustomerIDs) == 0 {
		return nil, errors.New("customer_ids cannot be empty")
	}

	// Get campaign
	campaign, err := s.repo.GetCampaign(ctx, campaignID)
	if err != nil {
		return nil, errors.New("campaign not found")
	}

	// Validate campaign status
	if campaign.Status != "draft" && campaign.Status != "scheduled" {
		return nil, errors.New("campaign must be in draft or scheduled status")
	}

	// Create outbound messages for each customer
	messages, err := s.messagesRepo.CreateOutboundMessageBatch(ctx, messagesModels.CreateOutboundMessageBatchParams{
		CampaignID:      campaignID,
		CustomerIds:     req.CustomerIDs,
		RenderedContent: campaign.BaseTemplate, // For now, use template as-is
	})
	if err != nil {
		return nil, err
	}

	// Check if we should send immediately or if it's a scheduled campaign for the future
	shouldSendImmediately := true
	if campaign.ScheduledAt.Valid && campaign.ScheduledAt.Time.After(time.Now()) {
		shouldSendImmediately = false
	}

	if shouldSendImmediately {
		// Publish each message to the queue
		for _, msg := range messages {
			if err := s.queue.PublishCampaignSend(msg.ID); err != nil {
				// Log error but continue - we don't want to fail the entire operation
				// TODO, implement retry logic or dead letter queue
				return nil, errors.New("failed to publish messages to queue")
			}
		}

		// Update campaign status to sending
		updatedCampaign, err := s.repo.UpdateCampaignToSending(ctx, campaignID)
		if err != nil {
			return nil, err
		}

		return &SendCampaignResponse{
			CampaignID:     updatedCampaign.ID,
			MessagesQueued: len(messages),
			Status:         updatedCampaign.Status,
		}, nil
	}

	// For scheduled campaigns, we just return success with current status
	// The scheduler worker will pick it up when ready
	return &SendCampaignResponse{
		CampaignID:     campaignID,
		MessagesQueued: len(messages),
		Status:         campaign.Status,
	}, nil
}

type ListCampaignsParams struct {
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Channel  string `json:"channel"`
	Status   string `json:"status"`
}

type Pagination struct {
	Page       int32 `json:"page"`
	PageSize   int32 `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int32 `json:"total_pages"`
}

// CampaignWithStats includes campaign data and message statistics
type CampaignWithStats struct {
	ID           int32         `json:"id"`
	Name         string        `json:"name"`
	Channel      string        `json:"channel"`
	Status       string        `json:"status"`
	BaseTemplate string        `json:"base_template"`
	ScheduledAt  *time.Time    `json:"scheduled_at"`
	CreatedAt    time.Time     `json:"created_at"`
	Stats        CampaignStats `json:"stats"`
}

type ListCampaignsResponse struct {
	Data       []CampaignWithStats `json:"data"`
	Pagination Pagination          `json:"pagination"`
}

func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func (s *Service) ListCampaigns(ctx context.Context, params ListCampaignsParams) (*ListCampaignsResponse, error) {
	// Set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	offset := (params.Page - 1) * params.PageSize

	// List campaigns
	campaigns, err := s.repo.ListCampaigns(ctx, models.ListCampaignsParams{
		Channel: stringToNullString(params.Channel),
		Status:  stringToNullString(params.Status),
		Limit:   params.PageSize,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	// Count total campaigns for pagination
	totalCount, err := s.repo.CountCampaigns(ctx, models.CountCampaignsParams{
		Channel: stringToNullString(params.Channel),
		Status:  stringToNullString(params.Status),
	})
	if err != nil {
		return nil, err
	}

	totalPages := int32(0)
	if totalCount > 0 {
		totalPages = int32((totalCount + int64(params.PageSize) - 1) / int64(params.PageSize))
	}

	// Extract campaign IDs for batch stats fetch
	campaignIDs := make([]int32, len(campaigns))
	for i, campaign := range campaigns {
		campaignIDs[i] = campaign.ID
	}

	// Fetch all stats in one query (eliminates N+1 problem)
	statsList, err := s.repo.GetCampaignStatsBatch(ctx, campaignIDs)
	if err != nil {
		// If batch stats fetch fails, we'll use zero values for all
		statsList = []models.GetCampaignStatsBatchRow{}
	}

	// Build a map for O(1) lookup
	statsMap := make(map[int32]models.GetCampaignStatsBatchRow)
	for _, stat := range statsList {
		statsMap[stat.CampaignID] = stat
	}

	// Build response with stats lookup
	campaignsWithStats := make([]CampaignWithStats, 0, len(campaigns))
	for _, campaign := range campaigns {
		stats := statsMap[campaign.ID] // O(1) lookup, zero value if not found

		var scheduledAt *time.Time
		if campaign.ScheduledAt.Valid {
			scheduledAt = &campaign.ScheduledAt.Time
		}

		campaignsWithStats = append(campaignsWithStats, CampaignWithStats{
			ID:           campaign.ID,
			Name:         campaign.Name,
			Channel:      campaign.Channel,
			Status:       campaign.Status,
			BaseTemplate: campaign.BaseTemplate,
			ScheduledAt:  scheduledAt,
			CreatedAt:    campaign.CreatedAt,
			Stats: CampaignStats{
				Total:   stats.Total,
				Pending: stats.Pending,
				Sending: stats.Sending,
				Sent:    stats.Sent,
				Failed:  stats.Failed,
			},
		})
	}

	return &ListCampaignsResponse{
		Data: campaignsWithStats,
		Pagination: Pagination{
			Page:       params.Page,
			PageSize:   params.PageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		},
	}, nil
}

type CampaignStats struct {
	Total   int64 `json:"total"`
	Pending int64 `json:"pending"`
	Sending int64 `json:"sending"`
	Sent    int64 `json:"sent"`
	Failed  int64 `json:"failed"`
}

type GetCampaignResponse struct {
	ID           int32         `json:"id"`
	Name         string        `json:"name"`
	Channel      string        `json:"channel"`
	Status       string        `json:"status"`
	BaseTemplate string        `json:"base_template"`
	ScheduledAt  *time.Time    `json:"scheduled_at"`
	CreatedAt    time.Time     `json:"created_at"`
	Stats        CampaignStats `json:"stats"`
}

func (s *Service) GetCampaign(ctx context.Context, id int32) (*GetCampaignResponse, error) {
	// Get campaign details
	campaign, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get campaign stats
	stats, err := s.repo.GetCampaignStats(ctx, id)
	if err != nil {
		return nil, err
	}

	var scheduledAt *time.Time
	if campaign.ScheduledAt.Valid {
		scheduledAt = &campaign.ScheduledAt.Time
	}

	return &GetCampaignResponse{
		ID:           campaign.ID,
		Name:         campaign.Name,
		Channel:      campaign.Channel,
		Status:       campaign.Status,
		BaseTemplate: campaign.BaseTemplate,
		ScheduledAt:  scheduledAt,
		CreatedAt:    campaign.CreatedAt,
		Stats: CampaignStats{
			Total:   stats.Total,
			Pending: stats.Pending,
			Sending: stats.Sending,
			Sent:    stats.Sent,
			Failed:  stats.Failed,
		},
	}, nil
}

// PersonalizedPreviewRequest represents the request body for personalized preview
type PersonalizedPreviewRequest struct {
	CustomerID       int32   `json:"customer_id"`
	OverrideTemplate *string `json:"override_template,omitempty"`
}

// PersonalizedPreviewResponse represents the response for personalized preview
type PersonalizedPreviewResponse struct {
	RenderedMessage string              `json:"rendered_message"`
	UsedTemplate    string              `json:"used_template"`
	Customer        CustomerPreviewData `json:"customer"`
}

// PersonalizedPreview generates a preview of how a message will render for a specific customer
func (s *Service) PersonalizedPreview(ctx context.Context, campaignID int32, req PersonalizedPreviewRequest) (*PersonalizedPreviewResponse, error) {
	// Get campaign
	campaign, err := s.repo.GetCampaign(ctx, campaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("campaign not found")
		}
		return nil, err
	}

	// Get customer data
	customer, err := s.customersRepo.GetCustomerForPreview(ctx, req.CustomerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("customer not found")
		}
		return nil, err
	}

	// Determine which template to use
	templateToUse := campaign.BaseTemplate
	if req.OverrideTemplate != nil && *req.OverrideTemplate != "" {
		templateToUse = *req.OverrideTemplate
	}

	// Render the template with customer data
	renderedMessage := RenderTemplate(templateToUse, customer)

	return &PersonalizedPreviewResponse{
		RenderedMessage: renderedMessage,
		UsedTemplate:    templateToUse,
		Customer:        ToCustomerPreviewData(customer),
	}, nil
}
