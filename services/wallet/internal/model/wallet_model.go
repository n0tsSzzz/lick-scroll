package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletModel struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string    `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance   int       `gorm:"default:0" json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WalletModel) TableName() string {
	return "wallets"
}

func (w *WalletModel) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}

type TransactionModel struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID        string    `gorm:"type:uuid;not null;index" json:"user_id"`
	PostID        string    `gorm:"type:uuid;index" json:"post_id,omitempty"`
	Type          string    `gorm:"type:varchar(20);not null" json:"type"`
	Amount        int       `gorm:"not null" json:"amount"`
	BalanceBefore int       `json:"balance_before"`
	BalanceAfter  int       `json:"balance_after"`
	CreatedAt     time.Time `json:"created_at"`
}

func (TransactionModel) TableName() string {
	return "transactions"
}

func (t *TransactionModel) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}
