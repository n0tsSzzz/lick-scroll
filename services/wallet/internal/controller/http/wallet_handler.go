package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/wallet/internal/entity"
	"lick-scroll/services/wallet/internal/repo/persistent"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type WalletHandler struct {
	walletRepo  persistent.WalletRepository
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewWalletHandler(walletRepo persistent.WalletRepository, redisClient *redis.Client, logger *logger.Logger) *WalletHandler {
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
	transaction := &entity.Transaction{
		UserID:        userID,
		Type:          entity.TransactionTypeEarn,
		Amount:        req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := h.walletRepo.CreateTransaction(transaction); err != nil {
		h.logger.Error("Failed to create transaction: %v", err)
	}

	c.JSON(http.StatusOK, wallet)
}

type DonateRequest struct {
	Amount int `json:"amount" binding:"required,min=1"`
}

// DonateToPost godoc
// @Summary      Donate to post creator
// @Description  Donate to the creator of a post using wallet balance
// @Tags         wallet
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        post_id path string true "Post ID"
// @Param        request body DonateRequest true "Donation amount"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]string
// @Router       /wallet/donate/{post_id} [post]
func (h *WalletHandler) DonateToPost(c *gin.Context) {
	userID := c.GetString("user_id")
	postID := c.Param("post_id")

	var req DonateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get post creator from cache
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", postID)
	creatorID, err := h.redisClient.HGet(ctx, postKey, "creator_id").Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Post not found"})
		return
	}

	// Can't donate to yourself
	if creatorID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot donate to your own post"})
		return
	}

	wallet, err := h.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		h.logger.Error("Failed to get wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet"})
		return
	}

	if wallet.Balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	// Deduct balance from donor
	balanceBefore := wallet.Balance
	wallet.Balance -= req.Amount
	if err := h.walletRepo.UpdateWallet(wallet); err != nil {
		h.logger.Error("Failed to update wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process donation"})
		return
	}

	// Create transaction for donor
	donorTransaction := &entity.Transaction{
		UserID:        userID,
		PostID:        postID,
		Type:          entity.TransactionTypeDonation,
		Amount:        -req.Amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := h.walletRepo.CreateTransaction(donorTransaction); err != nil {
		h.logger.Error("Failed to create transaction: %v", err)
	}

	// Add balance to creator
	creatorWallet, err := h.walletRepo.GetOrCreateWallet(creatorID)
	if err != nil {
		h.logger.Error("Failed to get creator wallet: %v", err)
		// Don't fail the donation, just log the error
	} else {
		creatorBalanceBefore := creatorWallet.Balance
		creatorWallet.Balance += req.Amount
		if err := h.walletRepo.UpdateWallet(creatorWallet); err != nil {
			h.logger.Error("Failed to update creator wallet: %v", err)
		} else {
			// Create transaction for creator
			creatorTransaction := &entity.Transaction{
				UserID:        creatorID,
				PostID:        postID,
				Type:          entity.TransactionTypeEarn,
				Amount:        req.Amount,
				BalanceBefore: creatorBalanceBefore,
				BalanceAfter:  creatorWallet.Balance,
			}
			if err := h.walletRepo.CreateTransaction(creatorTransaction); err != nil {
				h.logger.Error("Failed to create creator transaction: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Donation sent successfully",
		"wallet":  wallet,
		"amount":  req.Amount,
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
