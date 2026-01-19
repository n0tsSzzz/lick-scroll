package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"time"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	"lick-scroll/pkg/s3"
	"lick-scroll/services/post/internal/entity"
	"lick-scroll/services/post/internal/repo/persistent"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type PostUseCase interface {
	CreatePost(userID string, title, description, postType, category string, mediaFile *multipart.FileHeader, imageFiles []*multipart.FileHeader) (*entity.Post, error)
	GetPost(postID, userID string) (*entity.Post, int64, bool, error)
	GetLikeCount(postID string) (int64, error)
	ListPosts(limit, offset int, category string) ([]*entity.Post, error)
	UpdatePost(postID, userID string, title, description, category *string) (*entity.Post, error)
	DeletePost(postID, userID string) error
	GetCreatorPosts(creatorID string, limit, offset int) ([]*entity.Post, error)
	LikePost(userID, postID string) (bool, error)
	IsLiked(userID, postID string) (bool, error)
	GetLikedPosts(userID string, limit, offset int) ([]*entity.Post, error)
	IncrementView(postID string) error
}

type postUseCase struct {
	postRepo    persistent.PostRepository
	s3Client    *s3.Client
	redisClient *redis.Client
	queueClient *queue.Client
	logger      *logger.Logger
}

func NewPostUseCase(
	postRepo persistent.PostRepository,
	s3Client *s3.Client,
	redisClient *redis.Client,
	queueClient *queue.Client,
	logger *logger.Logger,
) PostUseCase {
	return &postUseCase{
		postRepo:    postRepo,
		s3Client:    s3Client,
		redisClient: redisClient,
		queueClient: queueClient,
		logger:      logger,
	}
}

func (uc *postUseCase) CreatePost(userID string, title, description, postType, category string, mediaFile *multipart.FileHeader, imageFiles []*multipart.FileHeader) (*entity.Post, error) {
	var mediaURL string
	var postImages []entity.PostImage

	if postType == "video" {
		if mediaFile == nil {
			return nil, fmt.Errorf("media file is required for video posts")
		}

		src, err := mediaFile.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer src.Close()

		fileKey := fmt.Sprintf("posts/%s/%s%s", userID, uuid.New().String(), getFileExtension(mediaFile.Filename))
		contentType := mediaFile.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "video/mp4"
		}

		uploadedURL, err := uc.s3Client.UploadFile(fileKey, src, contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to S3: %w", err)
		}
		mediaURL = uploadedURL
	} else {
		if len(imageFiles) == 0 {
			return nil, fmt.Errorf("at least one image file is required for photo posts")
		}

		if len(imageFiles) > 10 {
			return nil, fmt.Errorf("maximum 10 images allowed per post")
		}

		for i, file := range imageFiles {
			src, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}

			fileKey := fmt.Sprintf("posts/%s/%s%s", userID, uuid.New().String(), getFileExtension(file.Filename))
			contentType := file.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "image/jpeg"
			}

			imageURL, err := uc.s3Client.UploadFile(fileKey, src, contentType)
			src.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to upload file to S3: %w", err)
			}

			postImages = append(postImages, entity.PostImage{
				ID:       uuid.New().String(),
				ImageURL: imageURL,
				Order:    i,
			})
		}
	}

	post := &entity.Post{
		CreatorID:   userID,
		Title:       title,
		Description: description,
		Type:        entity.PostType(postType),
		MediaURL:    mediaURL,
		Category:    category,
		Status:      entity.StatusPending,
		Images:      postImages,
	}

	if err := uc.postRepo.Create(post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	uc.cachePost(post)
	uc.addToFeed(post)

	if uc.queueClient != nil {
		go uc.publishNotification(post)
	}

	return post, nil
}

func (uc *postUseCase) GetPost(postID, userID string) (*entity.Post, int64, bool, error) {
	post, err := uc.postRepo.GetByID(postID)
	if err != nil {
		return nil, 0, false, err
	}

	likeCount, _ := uc.postRepo.GetLikeCount(postID)

	isLiked := false
	if userID != "" {
		isLiked, _ = uc.postRepo.IsLiked(userID, postID)
	}

	return post, likeCount, isLiked, nil
}

func (uc *postUseCase) ListPosts(limit, offset int, category string) ([]*entity.Post, error) {
	approvedPosts, err := uc.postRepo.List(limit*2, 0, category, entity.StatusApproved)
	if err != nil {
		return nil, err
	}

	pendingPosts, err := uc.postRepo.List(limit*2, 0, category, entity.StatusPending)
	if err != nil {
		return nil, err
	}

	result := approvedPosts
	result = append(result, pendingPosts...)

	start := offset
	end := offset + limit
	if start > len(result) {
		result = []*entity.Post{}
	} else {
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	}

	return result, nil
}

func (uc *postUseCase) UpdatePost(postID, userID string, title, description, category *string) (*entity.Post, error) {
	post, err := uc.postRepo.GetByID(postID)
	if err != nil {
		return nil, err
	}

	if post.CreatorID != userID {
		return nil, fmt.Errorf("you can only update your own posts")
	}

	if title != nil {
		post.Title = *title
	}
	if description != nil {
		post.Description = *description
	}
	if category != nil {
		post.Category = *category
	}

	if err := uc.postRepo.Update(post); err != nil {
		return nil, err
	}

	return post, nil
}

func (uc *postUseCase) DeletePost(postID, userID string) error {
	post, err := uc.postRepo.GetByID(postID)
	if err != nil {
		return err
	}

	if post.CreatorID != userID {
		return fmt.Errorf("you can only delete your own posts")
	}

	return uc.postRepo.Delete(postID)
}

func (uc *postUseCase) GetCreatorPosts(creatorID string, limit, offset int) ([]*entity.Post, error) {
	return uc.postRepo.GetByCreatorID(creatorID, limit, offset)
}

func (uc *postUseCase) LikePost(userID, postID string) (bool, error) {
	isLiked, err := uc.postRepo.IsLiked(userID, postID)
	if err != nil {
		return false, err
	}

	if isLiked {
		if err := uc.postRepo.DeleteLike(userID, postID); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := uc.postRepo.CreateLike(userID, postID); err != nil {
		return false, err
	}
	return true, nil
}

func (uc *postUseCase) IsLiked(userID, postID string) (bool, error) {
	return uc.postRepo.IsLiked(userID, postID)
}

func (uc *postUseCase) GetLikedPosts(userID string, limit, offset int) ([]*entity.Post, error) {
	return uc.postRepo.GetLikedPosts(userID, limit, offset)
}

func (uc *postUseCase) IncrementView(postID string) error {
	return uc.postRepo.IncrementViews(postID)
}

func (uc *postUseCase) GetLikeCount(postID string) (int64, error) {
	return uc.postRepo.GetLikeCount(postID)
}

func (uc *postUseCase) cachePost(post *entity.Post) {
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", post.ID)
	postData := map[string]interface{}{
		"id":          post.ID,
		"creator_id":  post.CreatorID,
		"title":       post.Title,
		"description": post.Description,
		"type":        string(post.Type),
		"media_url":   post.MediaURL,
		"category":    post.Category,
		"status":      string(post.Status),
	}

	if len(post.Images) > 0 {
		imagesJSON, _ := json.Marshal(post.Images)
		postData["images"] = string(imagesJSON)
	}

	for k, v := range postData {
		uc.redisClient.HSet(ctx, postKey, k, v)
	}
	uc.redisClient.Expire(ctx, postKey, 24*time.Hour)
}

func (uc *postUseCase) addToFeed(post *entity.Post) {
	ctx := context.Background()
	globalFeedKey := "feed:global"
	uc.redisClient.LPush(ctx, globalFeedKey, post.ID)
	uc.redisClient.LTrim(ctx, globalFeedKey, 0, 9999)
	uc.redisClient.Expire(ctx, globalFeedKey, 7*24*time.Hour)

	if post.Category != "" {
		categoryFeedKey := fmt.Sprintf("feed:global:%s", post.Category)
		uc.redisClient.LPush(ctx, categoryFeedKey, post.ID)
		uc.redisClient.LTrim(ctx, categoryFeedKey, 0, 9999)
		uc.redisClient.Expire(ctx, categoryFeedKey, 7*24*time.Hour)
	}
}

func (uc *postUseCase) publishNotification(post *entity.Post) {
	task := map[string]interface{}{
		"type":       "new_post",
		"post_id":    post.ID,
		"creator_id": post.CreatorID,
		"category":   post.Category,
		"priority":   5,
	}

	uc.logger.Info("[NOTIFICATION QUEUE] Publishing notification task to RabbitMQ: post_id=%s, creator_id=%s", post.ID, post.CreatorID)
	if err := uc.queueClient.PublishNotificationTask(task); err != nil {
		uc.logger.Error("[NOTIFICATION QUEUE] Failed to publish notification task to RabbitMQ: %v (post_id=%s, creator_id=%s)", err, post.ID, post.CreatorID)
	} else {
		uc.logger.Info("[NOTIFICATION QUEUE] Successfully published notification task to RabbitMQ: post_id=%s, creator_id=%s", post.ID, post.CreatorID)
	}
}

func getFileExtension(filename string) string {
	if len(filename) == 0 {
		return ""
	}
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
	}
	return ""
}
