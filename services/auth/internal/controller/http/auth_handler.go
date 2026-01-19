package http

import (
	"fmt"
	"net/http"
	"path/filepath"

	"lick-scroll/services/auth/internal/entity"
	"lick-scroll/services/auth/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authUseCase usecase.AuthUseCase
}

func NewAuthHandler(authUseCase usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  *entity.User `json:"user"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with email, username, password and role
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration data"
// @Success      201  {object}  AuthResponse
// @Failure      400  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.authUseCase.Register(req.Email, req.Username, req.Password)
	if err != nil {
		if err.Error() == "user with this email already exists" || err.Error() == "username already taken" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200  {object}  AuthResponse
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      403  {object}  map[string]string
// @Router       /login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.authUseCase.Login(req.Email, req.Password)
	if err != nil {
		if err.Error() == "account is deactivated" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

// Me godoc
// @Summary      Get current user info
// @Description  Get information about the currently authenticated user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  entity.User
// @Failure      401  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Router       /me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.authUseCase.GetUser(userID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UploadAvatar godoc
// @Summary      Upload user avatar
// @Description  Upload avatar image for the current user
// @Tags         auth
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        avatar formData file true "Avatar image file"
// @Success      200  {object}  entity.User
// @Failure      400  {object}  map[string]string
// @Failure      401  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /avatar [post]
func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Avatar file is required"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format. Only jpg, jpeg, png, gif are allowed"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	defer src.Close()

	fileKey := fmt.Sprintf("avatars/%s/%s%s", userID.(string), uuid.New().String(), ext)
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	user, err := h.authUseCase.UploadAvatar(userID.(string), src, fileKey, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get user information by user ID (public endpoint for viewing other users' profiles)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200  {object}  entity.User
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /user/{id} [get]
func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.authUseCase.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetSubscriptions godoc
// @Summary      Get user subscriptions
// @Description  Get all subscriptions for the authenticated user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /users/{user_id}/subscriptions [get]
func (h *AuthHandler) GetSubscriptions(c *gin.Context) {
	userID := c.Param("user_id")
	currentUserID := c.GetString("user_id")

	if userID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own subscriptions"})
		return
	}

	subscriptions, err := h.authUseCase.GetSubscriptions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscriptions": subscriptions, "count": len(subscriptions)})
}

// Subscribe godoc
// @Summary      Subscribe to a creator
// @Tags         auth
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Param        creator_id path string true "Creator ID"
// @Router       /users/{user_id}/subscriptions/{creator_id} [post]
func (h *AuthHandler) Subscribe(c *gin.Context) {
	userID := c.Param("user_id")
	creatorID := c.Param("creator_id")
	currentUserID := c.GetString("user_id")

	if userID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only subscribe for yourself"})
		return
	}

	if err := h.authUseCase.Subscribe(userID, creatorID); err != nil {
		if err.Error() == "already subscribed" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Subscribed successfully"})
}

// Unsubscribe godoc
// @Summary      Unsubscribe from a creator
// @Tags         auth
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Param        creator_id path string true "Creator ID"
// @Router       /users/{user_id}/subscriptions/{creator_id} [delete]
func (h *AuthHandler) Unsubscribe(c *gin.Context) {
	userID := c.Param("user_id")
	creatorID := c.Param("creator_id")
	currentUserID := c.GetString("user_id")

	if userID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only unsubscribe for yourself"})
		return
	}

	if err := h.authUseCase.Unsubscribe(userID, creatorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribed successfully"})
}

// GetSubscriptionStatus godoc
// @Summary      Get subscription status
// @Tags         auth
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Param        creator_id path string true "Creator ID"
// @Router       /users/{user_id}/subscriptions/{creator_id}/status [get]
func (h *AuthHandler) GetSubscriptionStatus(c *gin.Context) {
	userID := c.Param("user_id")
	creatorID := c.Param("creator_id")
	currentUserID := c.GetString("user_id")

	if userID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only check your own subscription status"})
		return
	}

	subscribed, err := h.authUseCase.GetSubscriptionStatus(userID, creatorID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"subscribed": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"subscribed": subscribed})
}
