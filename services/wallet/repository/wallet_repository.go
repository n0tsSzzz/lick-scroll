package repository

import (
	"lick-scroll/pkg/models"

	"gorm.io/gorm"
)

type WalletRepository interface {
	GetOrCreateWallet(userID string) (*models.Wallet, error)
	UpdateWallet(wallet *models.Wallet) error
	CreateTransaction(transaction *models.Transaction) error
	GetTransactions(userID string, limit, offset int) ([]*models.Transaction, error)
}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) GetOrCreateWallet(userID string) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new wallet
			wallet = models.Wallet{
				UserID:  userID,
				Balance: 0,
			}
			if err := r.db.Create(&wallet).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &wallet, nil
}

func (r *walletRepository) UpdateWallet(wallet *models.Wallet) error {
	return r.db.Save(wallet).Error
}

func (r *walletRepository) CreateTransaction(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *walletRepository) GetTransactions(userID string, limit, offset int) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

