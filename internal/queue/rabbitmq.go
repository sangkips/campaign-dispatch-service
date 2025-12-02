package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
)

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   amqp091.Queue
}

type CampaignSendMessage struct {
	OutboundMessageID int32 `json:"outbound_message_id"`
}

// NewRabbitMQ creates a new RabbitMQ connection and declares the campaign_sends queue
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	var conn *amqp091.Connection
	var err error

	// Retry connection up to 10 times with 2 second delay
	for i := 0; i < 10; i++ {
		conn, err = amqp091.Dial(url)
		if err == nil {
			break
		}
		log.Warn().Err(err).Msgf("failed to connect to RabbitMQ, retrying in 2s (%d/10)", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to RabbitMQ after retries")
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Error().Err(err).Msg("failed to open channel")
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the campaign_sends queue
	queue, err := channel.QueueDeclare(
		"campaign_sends", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		log.Error().Err(err).Msg("failed to declare queue")
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	log.Info().Msg("connected to RabbitMQ and declared campaign_sends queue")

	return &RabbitMQ{
		conn:    conn,
		channel: channel,
		queue:   queue,
	}, nil
}

// PublishCampaignSend publishes an outbound message ID to the campaign_sends queue
func (r *RabbitMQ) PublishCampaignSend(messageID int32) error {
	msg := CampaignSendMessage{
		OutboundMessageID: messageID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.Publish(
		"",           // exchange
		r.queue.Name, // routing key (queue name)
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			DeliveryMode: amqp091.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)
	if err != nil {
		log.Error().Err(err).Int32("message_id", messageID).Msg("failed to publish message")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Debug().Int32("message_id", messageID).Msg("published message to queue")
	return nil
}

// Consume returns a channel of deliveries for the campaign_sends queue
func (r *RabbitMQ) Consume() (<-chan amqp091.Delivery, error) {
	msgs, err := r.channel.Consume(
		r.queue.Name, // queue
		"",           // consumer
		false,        // auto-ack (we will manual ack)
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}
	return msgs, nil
}

// Ping checks if the RabbitMQ connection and channel are open
func (r *RabbitMQ) Ping() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	if r.channel == nil || r.channel.IsClosed() {
		return fmt.Errorf("channel is closed")
	}
	return nil
}

// Close closes the RabbitMQ connection and channel
func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close channel")
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close connection")
			return err
		}
	}
	log.Info().Msg("closed RabbitMQ connection")
	return nil
}
