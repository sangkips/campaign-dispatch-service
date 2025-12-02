package worker

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages"
	messagesModels "github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
)

// Scheduler handles scheduled campaign dispatch
type Scheduler struct {
	campaignRepo campaigns.Repository
	messagesRepo messages.Repository
	queue        campaigns.QueuePublisher
	interval     time.Duration
	stopChan     chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(
	campaignRepo campaigns.Repository,
	messagesRepo messages.Repository,
	queue campaigns.QueuePublisher,
	interval time.Duration,
) *Scheduler {
	return &Scheduler{
		campaignRepo: campaignRepo,
		messagesRepo: messagesRepo,
		queue:        queue,
		interval:     interval,
		stopChan:     make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	log.Info().Msgf("starting scheduler with interval %v", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processReadyCampaigns()
		case <-s.stopChan:
			log.Info().Msg("stopping scheduler")
			return
		}
	}
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopChan)
}

func (s *Scheduler) processReadyCampaigns() {
	ctx := context.Background()

	// Fetch campaigns ready to send
	// This uses the stored function which atomically updates status to 'sending'
	// to prevent race conditions if multiple schedulers were running
	campaigns, err := s.campaignRepo.GetCampaignsReadyToSend(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch ready campaigns")
		return
	}

	if len(campaigns) == 0 {
		return
	}

	log.Info().Int("count", len(campaigns)).Msg("found campaigns ready to send")

	for _, campaign := range campaigns {
		s.processCampaign(ctx, campaign.ID)
	}
}

func (s *Scheduler) processCampaign(ctx context.Context, campaignID int32) {
	log.Info().Int32("campaign_id", campaignID).Msg("processing scheduled campaign")

	// Fetch pending messages for this campaign
	// We fetch in batches to avoid memory issues, though for now we'll just fetch a large batch
	// TODO, paginate this loop
	params := messagesModels.GetPendingMessagesForCampaignParams{
		CampaignID: campaignID,
		Limit:      10000,
		Offset:     0,
	}

	messages, err := s.messagesRepo.GetPendingMessagesForCampaign(ctx, params)
	if err != nil {
		log.Error().Err(err).Int32("campaign_id", campaignID).Msg("failed to fetch pending messages")
		return
	}

	log.Info().Int32("campaign_id", campaignID).Int("message_count", len(messages)).Msg("queuing messages")

	queuedCount := 0
	for _, msg := range messages {
		if err := s.queue.PublishCampaignSend(msg.ID); err != nil {
			log.Error().Err(err).Int32("message_id", msg.ID).Msg("failed to publish message")
			continue
		}
		queuedCount++
	}

	log.Info().Int32("campaign_id", campaignID).Int("queued", queuedCount).Msg("campaign processing complete")
}
