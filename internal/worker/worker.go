package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/campaigns"
	customersModels "github.com/sangkips/campaign-dispatch-service/internal/domains/customers/models"
	"github.com/sangkips/campaign-dispatch-service/internal/domains/messages"
	messagesModels "github.com/sangkips/campaign-dispatch-service/internal/domains/messages/models"
	"github.com/sangkips/campaign-dispatch-service/internal/queue"
)

type Worker struct {
	rabbitMQ *queue.RabbitMQ
	repo     messages.Repository
	sender   Sender
}

func NewWorker(rabbitMQ *queue.RabbitMQ, db messagesModels.DBTX, sender Sender) *Worker {
	return &Worker{
		rabbitMQ: rabbitMQ,
		repo:     messages.NewRepository(db),
		sender:   sender,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	msgs, err := w.rabbitMQ.Consume()
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	log.Info().Msg("worker started, waiting for messages")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("worker shutting down")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitMQ channel closed")
			}
			w.processMessage(ctx, d)
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, d amqp091.Delivery) {
	var msg queue.CampaignSendMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal message")
		d.Reject(false)
		return
	}

	log.Info().Int32("outbound_message_id", msg.OutboundMessageID).Msg("processing message")

	// Fetch message details
	details, err := w.repo.GetOutboundMessageWithDetails(ctx, msg.OutboundMessageID)
	if err != nil {
		log.Error().Err(err).Int32("outbound_message_id", msg.OutboundMessageID).Msg("failed to fetch message details")
		// If DB is down or record missing, maybe retry later. For now, nack with requeue if error
		// If record missing, reject.
		if err == sql.ErrNoRows {
			d.Reject(false)
		} else {
			d.Nack(false, true)
		}
		return
	}

	// Render template
	customerPreview := customersModels.GetCustomerForPreviewRow{
		ID:              details.CustomerID,
		Firstname:       details.CustomerFirstname,
		Lastname:        details.CustomerLastname,
		Phone:           details.CustomerPhone,
		Location:        details.CustomerLocation,
		PreferedProduct: details.CustomerPreferedProduct,
	}

	renderedContent := campaigns.RenderTemplate(details.CampaignBaseTemplate, customerPreview)

	// Send message
	providerMsgID, err := w.sender.Send(renderedContent, details.CustomerPhone)
	if err != nil {
		w.handleFailure(ctx, d, details, err)
		return
	}

	w.handleSuccess(ctx, d, details, providerMsgID)
}

func (w *Worker) handleSuccess(ctx context.Context, d amqp091.Delivery, details messagesModels.GetOutboundMessageWithDetailsRow, providerMsgID string) {
	_, err := w.repo.UpdateOutboundMessageWithRetry(ctx, messagesModels.UpdateOutboundMessageWithRetryParams{
		ID:     details.ID,
		Status: "sent",
		ProviderMessageID: sql.NullString{
			String: providerMsgID,
			Valid:  true,
		},
		LastError: sql.NullString{},
	})

	if err != nil {
		log.Error().Err(err).Int32("outbound_message_id", details.ID).Msg("failed to update status to sent")
		// If we send it but failed to update DB, we might resend duplicate.
		// Ideally we should have idempotency key on provider side.
		// For now, we ack because we did the job.
		d.Ack(false)
		return
	}

	log.Info().Int32("outbound_message_id", details.ID).Msg("message sent successfully")
	d.Ack(false)
}

func (w *Worker) handleFailure(ctx context.Context, d amqp091.Delivery, details messagesModels.GetOutboundMessageWithDetailsRow, sendErr error) {
	log.Warn().Err(sendErr).Int32("outbound_message_id", details.ID).Msg("failed to send message")

	// Check retry count
	if details.RetryCount >= 3 {
		_, err := w.repo.UpdateOutboundMessageWithRetry(ctx, messagesModels.UpdateOutboundMessageWithRetryParams{
			ID:     details.ID,
			Status: "failed",
			LastError: sql.NullString{
				String: sendErr.Error(),
				Valid:  true,
			},
		})
		if err != nil {
			log.Error().Err(err).Int32("outbound_message_id", details.ID).Msg("failed to update status to failed")
		}
		d.Ack(false)
		return
	}

	// returns the updated row.
	updated, err := w.repo.UpdateOutboundMessageWithRetry(ctx, messagesModels.UpdateOutboundMessageWithRetryParams{
		ID:     details.ID,
		Status: "failed",
		LastError: sql.NullString{
			String: sendErr.Error(),
			Valid:  true,
		},
	})
	if err != nil {
		log.Error().Err(err).Int32("outbound_message_id", details.ID).Msg("failed to update status to failed")
		d.Nack(false, true)
		return
	}

	if updated.RetryCount < 3 {
		log.Info().Int32("outbound_message_id", details.ID).Int32("retry_count", updated.RetryCount).Msg("requeueing for retry")
		// Sleep a bit to prevent tight loop
		time.Sleep(1 * time.Second)
		d.Nack(false, true)
	} else {
		log.Warn().Int32("outbound_message_id", details.ID).Msg("max retries reached, giving up")
		d.Ack(false)
	}
}
