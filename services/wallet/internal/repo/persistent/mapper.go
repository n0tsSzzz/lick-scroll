package persistent

import (
	"lick-scroll/services/wallet/internal/entity"
	"lick-scroll/services/wallet/internal/model"
)

func ToWalletEntity(m *model.WalletModel) *entity.Wallet {
	if m == nil {
		return nil
	}

	return &entity.Wallet{
		ID:        m.ID,
		UserID:    m.UserID,
		Balance:   m.Balance,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func ToWalletModel(e *entity.Wallet) *model.WalletModel {
	if e == nil {
		return nil
	}

	return &model.WalletModel{
		ID:        e.ID,
		UserID:    e.UserID,
		Balance:   e.Balance,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func ToTransactionEntity(m *model.TransactionModel) *entity.Transaction {
	if m == nil {
		return nil
	}

	return &entity.Transaction{
		ID:            m.ID,
		UserID:        m.UserID,
		PostID:        m.PostID,
		Type:          entity.TransactionType(m.Type),
		Amount:        m.Amount,
		BalanceBefore: m.BalanceBefore,
		BalanceAfter:  m.BalanceAfter,
		CreatedAt:     m.CreatedAt,
	}
}

func ToTransactionModel(e *entity.Transaction) *model.TransactionModel {
	if e == nil {
		return nil
	}

	return &model.TransactionModel{
		ID:            e.ID,
		UserID:        e.UserID,
		PostID:        e.PostID,
		Type:          string(e.Type),
		Amount:        e.Amount,
		BalanceBefore: e.BalanceBefore,
		BalanceAfter:  e.BalanceAfter,
		CreatedAt:     e.CreatedAt,
	}
}
