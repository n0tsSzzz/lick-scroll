package http

import (
	"net/http"
	"strconv"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/wallet/internal/usecase"

	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	walletUseCase usecase.WalletUseCase
	logger        *logger.Logger
}

func NewWalletHandler(walletUseCase usecase.WalletUseCase, logger *logger.Logger) *WalletHandler {
	return &WalletHandler{
		walletUseCase: walletUseCase,
		logger:        logger,
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

	wallet, err := h.walletUseCase.GetWallet(userID)
	if err != nil {
		h.logger.Error("Failed to get wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	wallet, err := h.walletUseCase.TopUp(userID, req.Amount)
	if err != nil {
		h.logger.Error("Failed to top up wallet: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	wallet, err := h.walletUseCase.DonateToPost(userID, postID, req.Amount)
	if err != nil {
		if err.Error() == "post not found" || err.Error() == "cannot donate to your own post" || err.Error() == "insufficient balance" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			h.logger.Error("Failed to donate: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
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

	transactions, err := h.walletUseCase.GetTransactions(userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get transactions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions, "count": len(transactions)})
}
