package messages

import (
	"context"

	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
)

type Repository interface {
	CreateOutboundMessage(ctx context.Context, params models.CreateOutboundMessageParams) (models.OutboundMessage, error)
	CreateOutboundMessageBatch(ctx context.Context, params models.CreateOutboundMessageBatchParams) ([]models.OutboundMessage, error)
	CountOutboundMessagesByCampaign(ctx context.Context, campaignID int32) (int64, error)
	GetOutboundMessageWithDetails(ctx context.Context, id int32) (models.GetOutboundMessageWithDetailsRow, error)
	UpdateOutboundMessageWithRetry(ctx context.Context, params models.UpdateOutboundMessageWithRetryParams) (models.OutboundMessage, error)
	GetPendingMessagesForCampaign(ctx context.Context, params models.GetPendingMessagesForCampaignParams) ([]models.OutboundMessage, error)
}

type repository struct {
	q *models.Queries
}

func NewRepository(db models.DBTX) Repository {
	return &repository{q: models.New(db)}
}

func (r *repository) CreateOutboundMessage(ctx context.Context, params models.CreateOutboundMessageParams) (models.OutboundMessage, error) {
	return r.q.CreateOutboundMessage(ctx, params)
}

func (r *repository) CreateOutboundMessageBatch(ctx context.Context, params models.CreateOutboundMessageBatchParams) ([]models.OutboundMessage, error) {
	return r.q.CreateOutboundMessageBatch(ctx, params)
}

func (r *repository) CountOutboundMessagesByCampaign(ctx context.Context, campaignID int32) (int64, error) {
	return r.q.CountOutboundMessagesByCampaign(ctx, campaignID)
}

func (r *repository) GetOutboundMessageWithDetails(ctx context.Context, id int32) (models.GetOutboundMessageWithDetailsRow, error) {
	return r.q.GetOutboundMessageWithDetails(ctx, id)
}

func (r *repository) UpdateOutboundMessageWithRetry(ctx context.Context, params models.UpdateOutboundMessageWithRetryParams) (models.OutboundMessage, error) {
	return r.q.UpdateOutboundMessageWithRetry(ctx, params)
}

func (r *repository) GetPendingMessagesForCampaign(ctx context.Context, params models.GetPendingMessagesForCampaignParams) ([]models.OutboundMessage, error) {
	return r.q.GetPendingMessagesForCampaign(ctx, params)
}
