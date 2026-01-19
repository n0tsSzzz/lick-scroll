package usecase

import (
	"fmt"
	"io"

	"lick-scroll/pkg/jwt"
	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/queue"
	"lick-scroll/pkg/s3"
	"lick-scroll/services/auth/internal/entity"
	"lick-scroll/services/auth/internal/repo/persistent"

	"golang.org/x/crypto/bcrypt"
)

type AuthUseCase interface {
	Register(email, username, password string) (*entity.User, string, error)
	Login(email, password string) (*entity.User, string, error)
	GetUser(userID string) (*entity.User, error)
	UploadAvatar(userID string, fileReader io.Reader, fileKey string, contentType string) (*entity.User, error)
	GetSubscriptions(userID string) ([]*entity.Subscription, error)
	Subscribe(viewerID, creatorID string) error
	Unsubscribe(viewerID, creatorID string) error
	GetSubscriptionStatus(viewerID, creatorID string) (bool, error)
}

type authUseCase struct {
	userRepo   persistent.UserRepository
	jwtService *jwt.Service
	s3Client   *s3.Client
	queueClient *queue.Client
	logger     *logger.Logger
}

func NewAuthUseCase(
	userRepo persistent.UserRepository,
	jwtService *jwt.Service,
	s3Client *s3.Client,
	queueClient *queue.Client,
	logger *logger.Logger,
) AuthUseCase {
	return &authUseCase{
		userRepo:    userRepo,
		jwtService:  jwtService,
		s3Client:    s3Client,
		queueClient: queueClient,
		logger:      logger,
	}
}

func (uc *authUseCase) Register(email, username, password string) (*entity.User, string, error) {
	_, err := uc.userRepo.GetByEmail(email)
	if err == nil {
		return nil, "", fmt.Errorf("user with this email already exists")
	}

	_, err = uc.userRepo.GetByUsername(username)
	if err == nil {
		return nil, "", fmt.Errorf("username already taken")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		uc.logger.Error("Failed to hash password: %v", err)
		return nil, "", fmt.Errorf("failed to process registration")
	}

	user := &entity.User{
		Email:    email,
		Username: username,
		Password: string(hashedPassword),
		Role:     entity.RoleViewer,
		IsActive: true,
	}

	if err := uc.userRepo.Create(user); err != nil {
		uc.logger.Error("Failed to create user: %v", err)
		return nil, "", fmt.Errorf("failed to create user")
	}

	token, err := uc.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		uc.logger.Error("Failed to generate token: %v", err)
		return nil, "", fmt.Errorf("failed to generate token")
	}

	user.Password = ""
	return user, token, nil
}

func (uc *authUseCase) Login(email, password string) (*entity.User, string, error) {
	user, err := uc.userRepo.GetByEmail(email)
	if err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		return nil, "", fmt.Errorf("account is deactivated")
	}

	token, err := uc.jwtService.GenerateToken(user.ID, string(user.Role))
	if err != nil {
		uc.logger.Error("Failed to generate token: %v", err)
		return nil, "", fmt.Errorf("failed to generate token")
	}

	user.Password = ""
	return user, token, nil
}

func (uc *authUseCase) GetUser(userID string) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return user, nil
}

func (uc *authUseCase) UploadAvatar(userID string, fileReader io.Reader, fileKey string, contentType string) (*entity.User, error) {
	avatarURL, err := uc.s3Client.UploadFile(fileKey, fileReader, contentType)
	if err != nil {
		uc.logger.Error("Failed to upload avatar: %v", err)
		return nil, fmt.Errorf("failed to upload avatar")
	}

	user, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	user.AvatarURL = avatarURL
	if err := uc.userRepo.Update(user); err != nil {
		uc.logger.Error("Failed to update user: %v", err)
		return nil, fmt.Errorf("failed to update user")
	}

	user.Password = ""
	return user, nil
}

func (uc *authUseCase) GetSubscriptions(userID string) ([]*entity.Subscription, error) {
	return uc.userRepo.GetSubscriptions(userID)
}

func (uc *authUseCase) Subscribe(viewerID, creatorID string) error {
	existing, err := uc.userRepo.GetSubscription(viewerID, creatorID)
	if err == nil && existing != nil && existing.ID != "" {
		return fmt.Errorf("already subscribed")
	}

	if err := uc.userRepo.CreateSubscription(viewerID, creatorID); err != nil {
		uc.logger.Error("Failed to create subscription: %v", err)
		return fmt.Errorf("failed to subscribe")
	}

	// Send notification to creator about new subscription via RabbitMQ
	if uc.queueClient != nil {
		go func() {
			task := map[string]interface{}{
				"type":          "subscription",
				"user_id":       creatorID,
				"subscriber_id": viewerID,
				"priority":      4,
			}

			uc.logger.Info("[NOTIFICATION QUEUE] Publishing subscription notification task to RabbitMQ: subscriber_id=%s, creator_id=%s", viewerID, creatorID)
			if err := uc.queueClient.PublishNotificationTask(task); err != nil {
				uc.logger.Error("[NOTIFICATION QUEUE] Failed to publish subscription notification task to RabbitMQ: %v", err)
			} else {
				uc.logger.Info("[NOTIFICATION QUEUE] Successfully published subscription notification task to RabbitMQ")
			}
		}()
	}

	return nil
}

func (uc *authUseCase) Unsubscribe(viewerID, creatorID string) error {
	if err := uc.userRepo.DeleteSubscription(viewerID, creatorID); err != nil {
		uc.logger.Error("Failed to delete subscription: %v", err)
		return fmt.Errorf("failed to unsubscribe")
	}
	return nil
}

func (uc *authUseCase) GetSubscriptionStatus(viewerID, creatorID string) (bool, error) {
	subscription, err := uc.userRepo.GetSubscription(viewerID, creatorID)
	if err != nil {
		return false, nil
	}
	return subscription != nil && subscription.ID != "", nil
}
