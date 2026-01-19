package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	"lick-scroll/services/interaction/internal/repo/persistent"

	"github.com/redis/go-redis/v9"
)

type InteractionUseCase interface {
	LikePost(userID, postID string) (bool, error)
	GetLikeCount(postID string) (int64, error)
	IsLiked(userID, postID string) (bool, error)
	GetLikedPosts(userID string, limit, offset int) ([]map[string]interface{}, error)
	IncrementView(userID, postID string) (bool, error)
	GetViewCount(postID string) (int64, error)
}

type interactionUseCase struct {
	interactionRepo persistent.InteractionRepository
	postRepo         persistent.PostRepository
	redisClient      *redis.Client
	queueClient      *queue.Client
	logger           *logger.Logger
}

func NewInteractionUseCase(
	interactionRepo persistent.InteractionRepository,
	postRepo persistent.PostRepository,
	redisClient *redis.Client,
	queueClient *queue.Client,
	logger *logger.Logger,
) InteractionUseCase {
	return &interactionUseCase{
		interactionRepo: interactionRepo,
		postRepo:         postRepo,
		redisClient:      redisClient,
		queueClient:      queueClient,
		logger:           logger,
	}
}

func (uc *interactionUseCase) LikePost(userID, postID string) (bool, error) {
	exists, err := uc.postRepo.PostExists(postID)
	if err != nil || !exists {
		return false, fmt.Errorf("post not found")
	}

	isLiked, err := uc.interactionRepo.IsLiked(userID, postID)
	if err != nil {
		uc.logger.Error("Failed to check like status: %v", err)
		return false, fmt.Errorf("failed to check like status: %w", err)
	}

	ctx := context.Background()
	redisKey := fmt.Sprintf("post:likes:%s", postID)

	if isLiked {
		if err := uc.interactionRepo.DeleteLike(userID, postID); err != nil {
			uc.logger.Error("Failed to delete like: %v", err)
			return false, fmt.Errorf("failed to unlike post: %w", err)
		}
		uc.redisClient.Decr(ctx, redisKey)
		return false, nil
	}

	if err := uc.interactionRepo.CreateLike(userID, postID); err != nil {
		uc.logger.Error("Failed to create like: %v", err)
		return false, fmt.Errorf("failed to like post: %w", err)
	}
	uc.redisClient.Incr(ctx, redisKey)

	creatorID, err := uc.postRepo.GetCreatorID(postID)
	if err == nil && creatorID != userID && uc.queueClient != nil {
		go func() {
			task := map[string]interface{}{
				"type":     "like",
				"user_id":  creatorID,
				"liker_id": userID,
				"post_id":  postID,
				"priority": 3,
			}

			uc.logger.Info("[NOTIFICATION QUEUE] Publishing like notification task to RabbitMQ: liker_id=%s, creator_id=%s, post_id=%s", userID, creatorID, postID)
			if err := uc.queueClient.PublishNotificationTask(task); err != nil {
				uc.logger.Error("[NOTIFICATION QUEUE] Failed to publish like notification task to RabbitMQ: %v", err)
			} else {
				uc.logger.Info("[NOTIFICATION QUEUE] Successfully published like notification task to RabbitMQ")
			}
		}()
	}

	return true, nil
}

func (uc *interactionUseCase) GetLikeCount(postID string) (int64, error) {
	ctx := context.Background()
	redisKey := fmt.Sprintf("post:likes:%s", postID)

	countStr, err := uc.redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		return count, nil
	}

	count, err := uc.interactionRepo.GetLikeCount(postID)
	if err != nil {
		return 0, fmt.Errorf("post not found")
	}

	uc.redisClient.Set(ctx, redisKey, count, 0)
	return count, nil
}

func (uc *interactionUseCase) IsLiked(userID, postID string) (bool, error) {
	return uc.interactionRepo.IsLiked(userID, postID)
}

func (uc *interactionUseCase) GetLikedPosts(userID string, limit, offset int) ([]map[string]interface{}, error) {
	return uc.interactionRepo.GetLikedPosts(userID, limit, offset)
}

func (uc *interactionUseCase) IncrementView(userID, postID string) (bool, error) {
	exists, err := uc.postRepo.PostExists(postID)
	if err != nil || !exists {
		return false, fmt.Errorf("post not found")
	}

	ctx := context.Background()
	viewKey := fmt.Sprintf("post_viewed:%s:%s", postID, userID)
	redisViewCountKey := fmt.Sprintf("post:views:%s", postID)

	set, err := uc.redisClient.SetNX(ctx, viewKey, "1", 365*24*3600*time.Second).Result()
	if err != nil {
		uc.logger.Error("Failed to set view key in Redis: %v", err)
		return false, fmt.Errorf("failed to track view: %w", err)
	}

	if set {
		if err := uc.interactionRepo.IncrementViews(postID); err != nil {
			uc.logger.Error("Failed to increment views: %v", err)
			return false, fmt.Errorf("failed to increment views: %w", err)
		}
		uc.redisClient.Incr(ctx, redisViewCountKey)
		return true, nil
	}

	return false, nil
}

func (uc *interactionUseCase) GetViewCount(postID string) (int64, error) {
	ctx := context.Background()
	redisKey := fmt.Sprintf("post:views:%s", postID)

	countStr, err := uc.redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		return count, nil
	}

	count, err := uc.interactionRepo.GetViewCount(postID)
	if err != nil {
		return 0, fmt.Errorf("post not found")
	}

	uc.redisClient.Set(ctx, redisKey, count, 0)
	return count, nil
}
