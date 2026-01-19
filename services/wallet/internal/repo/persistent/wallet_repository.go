package persistent

import (
	"lick-scroll/services/wallet/internal/entity"
	"lick-scroll/services/wallet/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletRepository interface {
	GetOrCreateWallet(userID string) (*entity.Wallet, error)
	UpdateWallet(wallet *entity.Wallet) error
	CreateTransaction(transaction *entity.Transaction) error
	GetTransactions(userID string, limit, offset int) ([]*entity.Transaction, error)
}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) GetOrCreateWallet(userID string) (*entity.Wallet, error) {
	var walletModel model.WalletModel
	if err := r.db.Where("user_id = ?", userID).First(&walletModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			walletModel = model.WalletModel{
				ID:      uuid.New().String(),
				UserID:  userID,
				Balance: 0,
			}
			if err := r.db.Create(&walletModel).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return ToWalletEntity(&walletModel), nil
}

func (r *walletRepository) UpdateWallet(wallet *entity.Wallet) error {
	walletModel := ToWalletModel(wallet)
	return r.db.Save(walletModel).Error
}

func (r *walletRepository) CreateTransaction(transaction *entity.Transaction) error {
	transactionModel := ToTransactionModel(transaction)
	if transactionModel.ID == "" {
		transactionModel.ID = uuid.New().String()
	}
	return r.db.Create(transactionModel).Error
}

func (r *walletRepository) GetTransactions(userID string, limit, offset int) ([]*entity.Transaction, error) {
	var transactionModels []model.TransactionModel
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&transactionModels).Error; err != nil {
		return nil, err
	}

	transactions := make([]*entity.Transaction, len(transactionModels))
	for i := range transactionModels {
		transactions[i] = ToTransactionEntity(&transactionModels[i])
	}
	return transactions, nil
}
