package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"lick-scroll/pkg/cache"
	"lick-scroll/pkg/config"
	"lick-scroll/pkg/database"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/pkg/s3"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "", "Path to config file")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	log := logger.New()
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Error("Failed to connect to database: %v", err)
		panic(err)
	}

	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Error("Failed to create S3 client: %v", err)
		panic(err)
	}

	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Error("Failed to connect to redis: %v", err)
		panic(err)
	}

	if err := seedDatabase(db, s3Client, redisClient, log); err != nil {
		log.Error("Failed to seed database: %v", err)
		panic(err)
	}

	log.Info("Database seeded successfully!")
}

func seedDatabase(db *gorm.DB, s3Client *s3.Client, redisClient *redis.Client, log *logger.Logger) error {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	testUsers := []struct {
		email    string
		username string
		password string
	}{
		{"alice@test.com", "alice_cat", "password123"},
		{"bob@test.com", "bob_cat", "password123"},
		{"charlie@test.com", "charlie_cat", "password123"},
		{"diana@test.com", "diana_cat", "password123"},
		{"eve@test.com", "eve_cat", "password123"},
	}

	userIDs := make([]string, 0, len(testUsers))

	for _, userData := range testUsers {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(userData.password), bcrypt.DefaultCost)

		user := &models.User{
			Email:    userData.email,
			Username: userData.username,
			Password: string(hashedPassword),
			Role:     models.RoleViewer,
			IsActive: true,
		}

		if err := user.BeforeCreate(nil); err != nil {
			return fmt.Errorf("failed to generate user ID: %w", err)
		}

		var existingUser models.User
		result := db.Where("email = ? OR username = ?", user.Email, user.Username).First(&existingUser)
		if result.Error == nil {
			log.Info("User %s already exists, skipping", user.Username)
			userIDs = append(userIDs, existingUser.ID)
			continue
		}

		if err := db.Create(user).Error; err != nil {
			log.Error("Failed to create user %s: %v", user.Username, err)
			continue
		}

		log.Info("Created user: %s (%s)", user.Username, user.Email)
		userIDs = append(userIDs, user.ID)

		wallet := &models.Wallet{
			UserID: user.ID,
			Balance: 1000,
		}
		if err := wallet.BeforeCreate(nil); err != nil {
			return fmt.Errorf("failed to generate wallet ID: %w", err)
		}
		if err := db.Create(wallet).Error; err != nil {
			log.Error("Failed to create wallet for user %s: %v", user.Username, err)
		}

		postsCount := 3 + (len(userIDs) % 3)
		log.Info("Creating %d posts for user %s", postsCount, user.Username)
		for i := 0; i < postsCount; i++ {
			if err := createPostWithCatImage(db, s3Client, redisClient, httpClient, user.ID, user.Username, i, log); err != nil {
				log.Error("Failed to create post %d for user %s: %v", i+1, user.Username, err)
				continue
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	for i := 0; i < len(userIDs); i++ {
		for j := i + 1; j < len(userIDs); j++ {
			subscriberID := userIDs[i]
			creatorID := userIDs[j]

			var existingSub models.Subscription
			result := db.Where("viewer_id = ? AND creator_id = ?", subscriberID, creatorID).First(&existingSub)
			if result.Error == nil {
				continue
			}

			subscription := &models.Subscription{
				ViewerID: subscriberID,
				CreatorID: creatorID,
			}
			if err := subscription.BeforeCreate(nil); err != nil {
				return fmt.Errorf("failed to generate subscription ID: %w", err)
			}

			if err := db.Create(subscription).Error; err != nil {
				log.Error("Failed to create subscription: %v", err)
				continue
			}
		}
	}

	log.Info("Created test subscriptions")
	return nil
}

func createPostWithCatImage(db *gorm.DB, s3Client *s3.Client, redisClient *redis.Client, httpClient *http.Client, userID, username string, index int, log *logger.Logger) error {
	cataasURL := "https://cataas.com/cat"
	if index%2 == 0 {
		cataasURL += fmt.Sprintf("/says/Hello from %s", username)
	}

	log.Info("Fetching cat image from %s", cataasURL)
	resp, err := httpClient.Get(cataasURL)
	if err != nil {
		return fmt.Errorf("failed to fetch cat image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cataas API returned status %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}

	if len(imageData) == 0 {
		return fmt.Errorf("received empty image data")
	}

	log.Info("Downloaded image: %d bytes", len(imageData))

	fileKey := fmt.Sprintf("posts/%s/seed_%d.jpg", userID, index)
	reader := bytes.NewReader(imageData)
	log.Info("Uploading image to S3: %s", fileKey)
	imageURL, err := s3Client.UploadFile(fileKey, reader, "image/jpeg")
	if err != nil {
		return fmt.Errorf("failed to upload image to S3: %w", err)
	}

	log.Info("Image uploaded successfully: %s", imageURL)

	post := &models.Post{
		CreatorID:   userID,
		Title:       fmt.Sprintf("Cat Post #%d by %s", index+1, username),
		Description: fmt.Sprintf("A cute cat from CATAAS API! Post #%d", index+1),
		Type:        models.PostTypePhoto,
		MediaURL:    imageURL,
		Category:    "cats",
		Status:      models.StatusApproved,
		Images: []models.PostImage{
			{
				ImageURL: imageURL,
				Order:    0,
			},
		},
	}

	if err := post.BeforeCreate(nil); err != nil {
		return fmt.Errorf("failed to generate post ID: %w", err)
	}

	for i := range post.Images {
		if err := post.Images[i].BeforeCreate(nil); err != nil {
			return fmt.Errorf("failed to generate image ID: %w", err)
		}
	}

	images := post.Images
	post.Images = nil

	if err := db.Create(post).Error; err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	if len(images) > 0 {
		for i := range images {
			images[i].PostID = post.ID
			if err := images[i].BeforeCreate(nil); err != nil {
				return fmt.Errorf("failed to generate image ID: %w", err)
			}
			if err := db.Create(&images[i]).Error; err != nil {
				log.Error("Failed to create post image: %v", err)
			}
		}
	}

	log.Info("Created post: %s by %s", post.Title, username)

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

	if len(images) > 0 {
		imagesJSON, _ := json.Marshal(images)
		postData["images"] = string(imagesJSON)
	}

	for k, v := range postData {
		redisClient.HSet(ctx, postKey, k, v)
	}
	redisClient.Expire(ctx, postKey, 24*time.Hour)

	globalFeedKey := "feed:global"
	redisClient.LPush(ctx, globalFeedKey, post.ID)
	redisClient.LTrim(ctx, globalFeedKey, 0, 9999)
	redisClient.Expire(ctx, globalFeedKey, 7*24*time.Hour)

	if post.Category != "" {
		categoryFeedKey := fmt.Sprintf("feed:global:%s", post.Category)
		redisClient.LPush(ctx, categoryFeedKey, post.ID)
		redisClient.LTrim(ctx, categoryFeedKey, 0, 9999)
		redisClient.Expire(ctx, categoryFeedKey, 7*24*time.Hour)
	}

	log.Info("Cached post %s in Redis feed", post.ID)
	return nil
}

