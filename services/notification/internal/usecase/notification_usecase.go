package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	"lick-scroll/services/notification/internal/entity"
	"lick-scroll/services/notification/internal/repo/persistent"

	"github.com/redis/go-redis/v9"
)

type NotificationUseCase interface {
	SendNotification(userID, title, message, notificationType string, data map[string]interface{}) (*entity.Notification, error)
	BroadcastNotification(userIDs []string, title, message, notificationType string, data map[string]interface{}) (int, error)
	GetNotifications(userID string, limit, offset int) ([]entity.Notification, int64, error)
	DeleteNotificationByPostID(userID, postID string) (int, error)
	GetNotificationSettings(userID, creatorID string) (bool, error)
	EnableNotifications(userID, creatorID string) error
	DisableNotifications(userID, creatorID string) error
	ProcessNotificationQueue() (int64, error)
	HandleNewPostNotification(task map[string]interface{}) error
	HandleLikeNotification(task map[string]interface{}) error
	HandleSubscriptionNotification(task map[string]interface{}) error
}

type notificationUseCase struct {
	notificationRepo persistent.NotificationRepository
	redisClient       *redis.Client
	queueClient       *queue.Client
	logger            *logger.Logger
}

func NewNotificationUseCase(notificationRepo persistent.NotificationRepository, redisClient *redis.Client, queueClient *queue.Client, logger *logger.Logger) NotificationUseCase {
	return &notificationUseCase{
		notificationRepo: notificationRepo,
		redisClient:       redisClient,
		queueClient:       queueClient,
		logger:            logger,
	}
}

func (uc *notificationUseCase) SendNotification(userID, title, message, notificationType string, data map[string]interface{}) (*entity.Notification, error) {
	notification := &entity.Notification{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Type:      notificationType,
		Data:      data,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := uc.sendNotificationToRedis(notification); err != nil {
		return nil, err
	}

	uc.logger.Info("Notification sent to user %s: %s", userID, title)
	return notification, nil
}

func (uc *notificationUseCase) BroadcastNotification(userIDs []string, title, message, notificationType string, data map[string]interface{}) (int, error) {
	sentCount := 0

	for _, userID := range userIDs {
		notification := &entity.Notification{
			UserID:    userID,
			Title:     title,
			Message:   message,
			Type:      notificationType,
			Data:      data,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		if err := uc.sendNotificationToRedis(notification); err != nil {
			uc.logger.Error("Failed to send notification to user %s: %v", userID, err)
			continue
		}
		sentCount++
	}

	uc.logger.Info("Broadcast notification sent to %d users: %s", sentCount, title)
	return sentCount, nil
}

func (uc *notificationUseCase) GetNotifications(userID string, limit, offset int) ([]entity.Notification, int64, error) {
	ctx := context.Background()
	userNotificationsKey := fmt.Sprintf("notifications:%s", userID)

	allNotifications, err := uc.redisClient.LRange(ctx, userNotificationsKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}

	var notifications []entity.Notification
	for _, notifJSON := range allNotifications {
		var notification entity.Notification
		if err := json.Unmarshal([]byte(notifJSON), &notification); err == nil {
			notifications = append(notifications, notification)
		}
	}

	totalCount, _ := uc.redisClient.LLen(ctx, userNotificationsKey).Result()

	return notifications, totalCount, nil
}

func (uc *notificationUseCase) DeleteNotificationByPostID(userID, postID string) (int, error) {
	ctx := context.Background()
	userNotificationsKey := fmt.Sprintf("notifications:%s", userID)

	allNotifications, err := uc.redisClient.LRange(ctx, userNotificationsKey, 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get notifications: %w", err)
	}

	var remainingNotifications []string
	deletedCount := 0
	for _, notifJSON := range allNotifications {
		var notification entity.Notification
		if err := json.Unmarshal([]byte(notifJSON), &notification); err == nil {
			if notification.Data != nil {
				if pID, ok := notification.Data["post_id"].(string); ok && pID == postID {
					deletedCount++
					continue
				}
			}
			remainingNotifications = append(remainingNotifications, notifJSON)
		} else {
			remainingNotifications = append(remainingNotifications, notifJSON)
		}
	}

	if deletedCount > 0 {
		if err := uc.redisClient.Del(ctx, userNotificationsKey).Err(); err != nil {
			uc.logger.Warn("Failed to delete old notifications list: %v", err)
		}

		if len(remainingNotifications) > 0 {
			for i := len(remainingNotifications) - 1; i >= 0; i-- {
				if err := uc.redisClient.LPush(ctx, userNotificationsKey, remainingNotifications[i]).Err(); err != nil {
					uc.logger.Warn("Failed to push notification back: %v", err)
				}
			}
		}

		if err := uc.redisClient.Expire(ctx, userNotificationsKey, 30*24*time.Hour).Err(); err != nil {
			uc.logger.Warn("Failed to set expiration: %v", err)
		}
	}

	return deletedCount, nil
}

func (uc *notificationUseCase) GetNotificationSettings(userID, creatorID string) (bool, error) {
	ctx := context.Background()
	settingsKey := fmt.Sprintf("notification_settings:%s:%s", userID, creatorID)

	enabled, err := uc.redisClient.Get(ctx, settingsKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get notification settings: %w", err)
	}

	return enabled == "true", nil
}

func (uc *notificationUseCase) EnableNotifications(userID, creatorID string) error {
	ctx := context.Background()
	settingsKey := fmt.Sprintf("notification_settings:%s:%s", userID, creatorID)

	if creatorID == "" {
		return fmt.Errorf("creator ID is required")
	}

	if err := uc.redisClient.Set(ctx, settingsKey, "true", 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	uc.logger.Info("Enabled notifications for user %s, creator %s", userID, creatorID)
	return nil
}

func (uc *notificationUseCase) DisableNotifications(userID, creatorID string) error {
	ctx := context.Background()
	settingsKey := fmt.Sprintf("notification_settings:%s:%s", userID, creatorID)

	if err := uc.redisClient.Del(ctx, settingsKey).Err(); err != nil {
		return fmt.Errorf("failed to disable notifications: %w", err)
	}

	return nil
}

func (uc *notificationUseCase) ProcessNotificationQueue() (int64, error) {
	if uc.queueClient == nil {
		return 0, fmt.Errorf("queue client is not available")
	}
	length, err := uc.queueClient.GetQueueLength()
	return int64(length), err
}

func (uc *notificationUseCase) HandleNewPostNotification(task map[string]interface{}) error {
	postID, _ := task["post_id"].(string)
	creatorID, _ := task["creator_id"].(string)

	if postID == "" || creatorID == "" {
		uc.logger.Error("[NOTIFICATION HANDLER] Invalid new_post task: missing post_id or creator_id, task=%+v", task)
		return fmt.Errorf("invalid task: missing post_id or creator_id")
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Processing new_post notification: post_id=%s, creator_id=%s", postID, creatorID)

	creatorUsername, err := uc.notificationRepo.GetCreatorUsername(creatorID)
	if err != nil {
		uc.logger.Warn("[NOTIFICATION HANDLER] Failed to get creator username for ID %s: %v", creatorID, err)
		creatorUsername = creatorID
	} else {
		uc.logger.Info("[NOTIFICATION HANDLER] Found creator: id=%s, username=%s", creatorID, creatorUsername)
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Querying subscribers for creator_id=%s", creatorID)
	viewerIDs, err := uc.notificationRepo.GetSubscribers(creatorID)
	if err != nil {
		uc.logger.Error("[NOTIFICATION HANDLER] Failed to get subscribers for creator %s: %v", creatorID, err)
		return err
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Found %d subscribers for creator %s (%s), post_id=%s", len(viewerIDs), creatorID, creatorUsername, postID)

	if len(viewerIDs) == 0 {
		uc.logger.Info("[NOTIFICATION HANDLER] No subscribers found for creator %s, skipping notifications", creatorID)
		return nil
	}

	notificationsSent := 0
	notificationsSkipped := 0
	ctx := context.Background()

	for i, userID := range viewerIDs {
		uc.logger.Info("[NOTIFICATION HANDLER] Processing subscriber %d/%d: viewer_id=%s, creator_id=%s", i+1, len(viewerIDs), userID, creatorID)

		settingsKey := fmt.Sprintf("notification_settings:%s:%s", userID, creatorID)
		enabled, err := uc.redisClient.Get(ctx, settingsKey).Result()
		if err == redis.Nil {
			uc.logger.Info("[NOTIFICATION HANDLER] No notification setting found for user %s from creator %s, defaulting to enabled", userID, creatorID)
		} else if err != nil {
			uc.logger.Warn("[NOTIFICATION HANDLER] Failed to check notification settings for user %s, creator %s: %v (assuming enabled)", userID, creatorID, err)
		} else if enabled == "false" {
			uc.logger.Info("[NOTIFICATION HANDLER] Notifications disabled for user %s from creator %s, skipping", userID, creatorID)
			notificationsSkipped++
			continue
		}

		notification := &entity.Notification{
			UserID:    userID,
			Title:     "New Post Alert!",
			Message:   fmt.Sprintf("Creator %s just posted new content!", creatorUsername),
			Type:      "new_post",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Data: map[string]interface{}{
				"post_id":    postID,
				"creator_id": creatorID,
			},
		}

		if err := uc.sendNotificationToRedis(notification); err != nil {
			uc.logger.Error("[NOTIFICATION HANDLER] Failed to send notification to user %s: %v", userID, err)
			continue
		}

		notificationsSent++
		uc.logger.Info("[NOTIFICATION HANDLER] Successfully processed notification for user %s about post %s", userID, postID)
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Completed processing notifications for post %s: sent=%d, skipped=%d, total_subscribers=%d", postID, notificationsSent, notificationsSkipped, len(viewerIDs))
	return nil
}

func (uc *notificationUseCase) HandleLikeNotification(task map[string]interface{}) error {
	userID, _ := task["user_id"].(string)    // Creator of the post (recipient)
	likerID, _ := task["liker_id"].(string) // User who liked
	postID, _ := task["post_id"].(string)

	if userID == "" || likerID == "" || postID == "" {
		uc.logger.Error("[NOTIFICATION HANDLER] Invalid like task: missing user_id, liker_id or post_id, task=%+v", task)
		return fmt.Errorf("invalid task: missing required fields")
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Processing like notification: user_id=%s, liker_id=%s, post_id=%s", userID, likerID, postID)

	likerUsername, err := uc.notificationRepo.GetLikerUsername(likerID)
	if err != nil {
		likerUsername = "Someone"
	}

	notification := &entity.Notification{
		UserID:    userID,
		Title:     "New Like!",
		Message:   fmt.Sprintf("%s liked your post", likerUsername),
		Type:      "like",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"post_id":  postID,
			"liker_id": likerID,
		},
	}

	if err := uc.sendNotificationToRedis(notification); err != nil {
		uc.logger.Error("[NOTIFICATION HANDLER] Failed to send like notification to user %s: %v", userID, err)
		return err
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Successfully sent like notification to user %s", userID)
	return nil
}

func (uc *notificationUseCase) HandleSubscriptionNotification(task map[string]interface{}) error {
	userID, _ := task["user_id"].(string)        // Creator (recipient)
	subscriberID, _ := task["subscriber_id"].(string) // User who subscribed

	if userID == "" || subscriberID == "" {
		uc.logger.Error("[NOTIFICATION HANDLER] Invalid subscription task: missing user_id or subscriber_id, task=%+v", task)
		return fmt.Errorf("invalid task: missing required fields")
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Processing subscription notification: user_id=%s, subscriber_id=%s", userID, subscriberID)

	subscriberUsername, err := uc.notificationRepo.GetSubscriberUsername(subscriberID)
	if err != nil {
		subscriberUsername = "Someone"
	}

	notification := &entity.Notification{
		UserID:    userID,
		Title:     "New Subscriber!",
		Message:   fmt.Sprintf("%s subscribed to you", subscriberUsername),
		Type:      "subscription",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"subscriber_id": subscriberID,
		},
	}

	if err := uc.sendNotificationToRedis(notification); err != nil {
		uc.logger.Error("[NOTIFICATION HANDLER] Failed to send subscription notification to user %s: %v", userID, err)
		return err
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Successfully sent subscription notification to user %s", userID)
	return nil
}

func (uc *notificationUseCase) sendNotificationToRedis(notification *entity.Notification) error {
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	uc.logger.Info("[NOTIFICATION HANDLER] Created notification JSON for user %s: %s", notification.UserID, string(notificationJSON))

	ctx := context.Background()
	userNotificationsKey := fmt.Sprintf("notifications:%s", notification.UserID)
	if err := uc.redisClient.LPush(ctx, userNotificationsKey, notificationJSON).Err(); err != nil {
		return fmt.Errorf("failed to LPush notification to Redis: %w", err)
	}
	uc.redisClient.LTrim(ctx, userNotificationsKey, 0, 99)
	uc.redisClient.Expire(ctx, userNotificationsKey, 30*24*time.Hour)
	uc.logger.Info("[NOTIFICATION HANDLER] Stored notification in Redis list: key=%s", userNotificationsKey)

	pubsubChannel := fmt.Sprintf("notifications:%s", notification.UserID)
	subscribers, err := uc.redisClient.Publish(ctx, pubsubChannel, notificationJSON).Result()
	if err != nil {
		return fmt.Errorf("failed to publish notification to Redis pub/sub channel %s: %w", pubsubChannel, err)
	}
	uc.logger.Info("[NOTIFICATION HANDLER] Published notification to Redis pub/sub channel=%s, subscribers=%d", pubsubChannel, subscribers)

	return nil
}
