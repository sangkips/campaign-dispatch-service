package campaigns

import (
	"context"

	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns/models"
)

type Repository interface {
	CreateCampaign(ctx context.Context, campaign models.CreateCampaignParams) (models.Campaign, error)
	GetCampaign(ctx context.Context, id int32) (models.Campaign, error)
	UpdateCampaignToSending(ctx context.Context, id int32) (models.Campaign, error)
	ListCampaigns(ctx context.Context, params models.ListCampaignsParams) ([]models.Campaign, error)
	CountCampaigns(ctx context.Context, params models.CountCampaignsParams) (int64, error)
	GetCampaignStats(ctx context.Context, id int32) (models.GetCampaignStatsRow, error)
	GetCampaignStatsBatch(ctx context.Context, campaignIDs []int32) ([]models.GetCampaignStatsBatchRow, error)
	GetCampaignsReadyToSend(ctx context.Context) ([]models.GetCampaignsReadyToSendRow, error)
}

type repository struct {
	q *models.Queries
}

func NewRepository(db models.DBTX) Repository {
	return &repository{q: models.New(db)}
}

func (r *repository) CreateCampaign(ctx context.Context, campaign models.CreateCampaignParams) (models.Campaign, error) {
	return r.q.CreateCampaign(ctx, campaign)
}

func (r *repository) GetCampaign(ctx context.Context, id int32) (models.Campaign, error) {
	return r.q.GetCampaign(ctx, id)
}

func (r *repository) UpdateCampaignToSending(ctx context.Context, id int32) (models.Campaign, error) {
	return r.q.UpdateCampaignToSending(ctx, id)
}

func (r *repository) ListCampaigns(ctx context.Context, params models.ListCampaignsParams) ([]models.Campaign, error) {
	return r.q.ListCampaigns(ctx, params)
}

func (r *repository) CountCampaigns(ctx context.Context, params models.CountCampaignsParams) (int64, error) {
	return r.q.CountCampaigns(ctx, params)
}

func (r *repository) GetCampaignStats(ctx context.Context, id int32) (models.GetCampaignStatsRow, error) {
	return r.q.GetCampaignStats(ctx, id)
}

func (r *repository) GetCampaignStatsBatch(ctx context.Context, campaignIDs []int32) ([]models.GetCampaignStatsBatchRow, error) {
	return r.q.GetCampaignStatsBatch(ctx, campaignIDs)
}

func (r *repository) GetCampaignsReadyToSend(ctx context.Context) ([]models.GetCampaignsReadyToSendRow, error) {
	return r.q.GetCampaignsReadyToSend(ctx)
}
