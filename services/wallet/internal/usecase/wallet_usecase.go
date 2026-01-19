package usecase

import (
	"context"
	"fmt"

	"lick-scroll/pkg/logger"
	"lick-scroll/services/wallet/internal/entity"
	"lick-scroll/services/wallet/internal/repo/persistent"

	"github.com/redis/go-redis/v9"
)

type WalletUseCase interface {
	GetWallet(userID string) (*entity.Wallet, error)
	TopUp(userID string, amount int) (*entity.Wallet, error)
	DonateToPost(userID, postID string, amount int) (*entity.Wallet, error)
	GetTransactions(userID string, limit, offset int) ([]*entity.Transaction, error)
}

type walletUseCase struct {
	walletRepo  persistent.WalletRepository
	redisClient *redis.Client
	logger      *logger.Logger
}

func NewWalletUseCase(walletRepo persistent.WalletRepository, redisClient *redis.Client, logger *logger.Logger) WalletUseCase {
	return &walletUseCase{
		walletRepo:  walletRepo,
		redisClient: redisClient,
		logger:      logger,
	}
}

func (uc *walletUseCase) GetWallet(userID string) (*entity.Wallet, error) {
	wallet, err := uc.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		uc.logger.Error("Failed to get wallet: %v", err)
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return wallet, nil
}

func (uc *walletUseCase) TopUp(userID string, amount int) (*entity.Wallet, error) {
	wallet, err := uc.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		uc.logger.Error("Failed to get wallet: %v", err)
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	balanceBefore := wallet.Balance
	wallet.Balance += amount
	if err := uc.walletRepo.UpdateWallet(wallet); err != nil {
		uc.logger.Error("Failed to update wallet: %v", err)
		return nil, fmt.Errorf("failed to top up wallet: %w", err)
	}

	transaction := &entity.Transaction{
		UserID:        userID,
		Type:          entity.TransactionTypeEarn,
		Amount:        amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := uc.walletRepo.CreateTransaction(transaction); err != nil {
		uc.logger.Error("Failed to create transaction: %v", err)
	}

	return wallet, nil
}

func (uc *walletUseCase) DonateToPost(userID, postID string, amount int) (*entity.Wallet, error) {
	ctx := context.Background()
	postKey := fmt.Sprintf("post:%s", postID)
	creatorID, err := uc.redisClient.HGet(ctx, postKey, "creator_id").Result()
	if err != nil {
		return nil, fmt.Errorf("post not found")
	}

	if creatorID == userID {
		return nil, fmt.Errorf("cannot donate to your own post")
	}

	wallet, err := uc.walletRepo.GetOrCreateWallet(userID)
	if err != nil {
		uc.logger.Error("Failed to get wallet: %v", err)
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	if wallet.Balance < amount {
		return nil, fmt.Errorf("insufficient balance")
	}

	balanceBefore := wallet.Balance
	wallet.Balance -= amount
	if err := uc.walletRepo.UpdateWallet(wallet); err != nil {
		uc.logger.Error("Failed to update wallet: %v", err)
		return nil, fmt.Errorf("failed to process donation: %w", err)
	}

	donorTransaction := &entity.Transaction{
		UserID:        userID,
		PostID:        postID,
		Type:          entity.TransactionTypeDonation,
		Amount:        -amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  wallet.Balance,
	}
	if err := uc.walletRepo.CreateTransaction(donorTransaction); err != nil {
		uc.logger.Error("Failed to create transaction: %v", err)
	}

	creatorWallet, err := uc.walletRepo.GetOrCreateWallet(creatorID)
	if err != nil {
		uc.logger.Error("Failed to get creator wallet: %v", err)
	} else {
		creatorBalanceBefore := creatorWallet.Balance
		creatorWallet.Balance += amount
		if err := uc.walletRepo.UpdateWallet(creatorWallet); err != nil {
			uc.logger.Error("Failed to update creator wallet: %v", err)
		} else {
			creatorTransaction := &entity.Transaction{
				UserID:        creatorID,
				PostID:        postID,
				Type:          entity.TransactionTypeEarn,
				Amount:        amount,
				BalanceBefore: creatorBalanceBefore,
				BalanceAfter:  creatorWallet.Balance,
			}
			if err := uc.walletRepo.CreateTransaction(creatorTransaction); err != nil {
				uc.logger.Error("Failed to create creator transaction: %v", err)
			}
		}
	}

	return wallet, nil
}

func (uc *walletUseCase) GetTransactions(userID string, limit, offset int) ([]*entity.Transaction, error) {
	transactions, err := uc.walletRepo.GetTransactions(userID, limit, offset)
	if err != nil {
		uc.logger.Error("Failed to get transactions: %v", err)
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	return transactions, nil
}
