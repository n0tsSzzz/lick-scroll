package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"
	"lick-scroll/pkg/models"
	"lick-scroll/services/wallet/repository"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type WalletHandler struct {
	walletRepo  repository.WalletRepository
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewWalletHandler(walletRepo repository.WalletRepository, redisClient *redis.Client, logger *logger.Logger) *WalletHandler {
	return &WalletHandler{
		walletRepo:  walletRepo,
		redisClient: redisClient,
		logger:      logger,
	}
}

type TopUpRequest struct {
	Amount int `json:"amount" binding:"required,min=1"`
}

// GetWallet godoc
// @Summary      Get wallet
// @Description  Get wallet balance for the authenticated user
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.Wallet
// @Router       /wallet [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID := c.GetString("user_id")

	wallet, err := h.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		h.logger.Error("Failed to get wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// TopUp godoc
// @Summary      Top up wallet
// @Description  Add funds to user wallet
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body TopUpRequest true "Top up amount"
// @Success      200  {object}  models.Wallet
// @Router       /wallet/topup [post]
func (h *WalletHandler) TopUp(c *gin.Context) {
	userID := c.GetString("user_id")

	var req TopUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := h.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		h.logger.Error("Failed to get wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	// Update balance
	balanceBefore := wallet.Balance
	wallet.Balance += req.Amount
	if err := h.walletRepo.UpdateWallet(wallet); err != nil {
		h.logger.Error("Failed to update wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to top up wallet"})
		return
	}

	// Create transaction
	transaction := &models.Transaction{
		UserID:        userID,
		Type:          models.TransactionTypeEarn,
		Amount:        req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := h.walletRepo.CreateTransaction(transaction); err != nil {
		h.logger.Error("Failed to create transaction: %v", err)
	}

	c.JSON(http.StatusOK, wallet)
}

// PurchasePost godoc
// @Summary      Purchase post
// @Description  Purchase a post using wallet balance
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Router       /wallet/purchase/{post_id} [post]
func (h *WalletHandler) PurchasePost(c *gin.Context) {
	userID := c.GetString("user_id")
	postID := c.Param("post_id")

	// Get post price from cache
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", postID)
	priceStr, err := h.redisClient.HGet(ctx, postKey, "price").Result()
	if err != nil {
		// If not in cache, we would need to call post service
		// For MVP, return error
		c.JSON(http.StatusBadRequest, gin.H{"error": "Post not found or price not available. Please try again."})
		return
	}

	price, err := strconv.Atoi(priceStr)
	if err != nil || price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post price"})
		return
	}

	wallet, err := h.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		h.logger.Error("Failed to get wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	if wallet.Balance < price {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// Check if already purchased
	purchaseKey := fmt.Sprintf("purchase:%s:%s", userID, postID)
	alreadyPurchased, _ := h.redisClient.Exists(ctx, purchaseKey).Result()
	if alreadyPurchased > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Post already purchased"})
		return
	}

	// Deduct balance
	balanceBefore := wallet.Balance
	wallet.Balance -= price
	if err := h.walletRepo.UpdateWallet(wallet); err != nil {
		h.logger.Error("Failed to update wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process purchase"})
		return
	}

	// Create transaction
	transaction := &models.Transaction{
		UserID:        userID,
		PostID:        postID,
		Type:          models.TransactionTypePurchase,
		Amount:        -price,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := h.walletRepo.CreateTransaction(transaction); err != nil {
		h.logger.Error("Failed to create transaction: %v", err)
	}

	// Mark as purchased
	h.redisClient.Set(ctx, purchaseKey, "1", 0) // Never expire

	c.JSON(http.StatusOK, gin.H{
		"message": "Post purchased successfully",
		"wallet":  wallet,
	})
}

// GetTransactions godoc
// @Summary      Get transactions
// @Description  Get transaction history for the authenticated user
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        limit query int false "Number of transactions"
// @Param        offset query int false "Offset"
// @Success      200  {object}  map[string]interface{}
// @Router       /wallet/transactions [get]
func (h *WalletHandler) GetTransactions(c *gin.Context) {
	userID := c.GetString("user_id")
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	transactions, err := h.walletRepo.GetTransactions(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transactions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions, "count": len(transactions)})
}

