package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"lick-scroll/pkg/config"
	"lick-scroll/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	NotificationQueueName = "notification_queue"
	NotificationExchange  = "notifications"
)

type Client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *logger.Logger
}

func NewRabbitMQClient(cfg *config.Config, log *logger.Logger) (*Client, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		cfg.RabbitMQUser,
		cfg.RabbitMQPassword,
		cfg.RabbitMQHost,
		cfg.RabbitMQPort,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange for notifications
	err = channel.ExchangeDeclare(
		NotificationExchange, // name
		"direct",             // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare priority queue for notifications
	_, err = channel.QueueDeclare(
		NotificationQueueName, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		amqp.Table{
			"x-max-priority": 10, // Enable priority queue (0-10)
		},
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		NotificationQueueName, // queue name
		"new_post",            // routing key
		NotificationExchange,  // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	log.Info("Connected to RabbitMQ at %s:%s", cfg.RabbitMQHost, cfg.RabbitMQPort)

	return &Client{
		conn:    conn,
		channel: channel,
		logger:  log,
	}, nil
}

func (c *Client) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PublishNotificationTask publishes a notification task to the queue with priority
func (c *Client) PublishNotificationTask(task map[string]interface{}) error {
	priority := 1 // Default priority
	if p, ok := task["priority"].(int); ok {
		priority = p
		// Clamp priority to 0-10 range
		if priority < 0 {
			priority = 0
		}
		if priority > 10 {
			priority = 10
		}
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = c.channel.Publish(
		NotificationExchange, // exchange
		"new_post",           // routing key
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         taskJSON,
			Priority:     uint8(priority),
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		c.logger.Error("[RABBITMQ] Failed to publish message to exchange=%s, routing_key=%s: %v", NotificationExchange, "new_post", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	c.logger.Info("[RABBITMQ] Successfully published notification task to exchange=%s, routing_key=%s, queue=%s: %s", NotificationExchange, "new_post", NotificationQueueName, string(taskJSON))
	return nil
}

// ConsumeNotificationTasks consumes notification tasks from the queue
func (c *Client) ConsumeNotificationTasks(handler func(task map[string]interface{}) error) error {
	msgs, err := c.channel.Consume(
		NotificationQueueName, // queue
		"",                    // consumer
		false,                 // auto-ack (we'll manually ack after processing)
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.logger.Info("[RABBITMQ] Started consuming from notification queue: %s", NotificationQueueName)

	go func() {
		for msg := range msgs {
			c.logger.Info("[RABBITMQ] Received message from queue: %s, message_size=%d bytes", NotificationQueueName, len(msg.Body))
			
			var task map[string]interface{}
			if err := json.Unmarshal(msg.Body, &task); err != nil {
				c.logger.Error("[RABBITMQ] Failed to unmarshal notification task: %v, body=%s", err, string(msg.Body))
				msg.Nack(false, false) // Reject and don't requeue
				continue
			}

			c.logger.Info("[RABBITMQ] Successfully unmarshaled task: %+v", task)

			// Process task
			if err := handler(task); err != nil {
				c.logger.Error("[RABBITMQ] Handler failed to process notification task: %v, task=%+v", err, task)
				msg.Nack(false, true) // Reject and requeue
				continue
			}

			// Acknowledge message
			msg.Ack(false)
			c.logger.Info("[RABBITMQ] Successfully processed and acknowledged task: %+v", task)
		}
	}()

	return nil
}

// GetQueueLength returns the number of messages in the queue
func (c *Client) GetQueueLength() (int, error) {
	queue, err := c.channel.QueueInspect(NotificationQueueName)
	if err != nil {
		return 0, err
	}
	return queue.Messages, nil
}

