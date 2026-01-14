package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypePurchase TransactionType = "purchase"
	TransactionTypeEarn     TransactionType = "earn"
	TransactionTypeRefund   TransactionType = "refund"
	TransactionTypeDonation TransactionType = "donation"
)

type Wallet struct {
	ID        string    `gorm:"type:uuid;primary_key" json:"id"`
	UserID    string    `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance   int       `gorm:"default:0" json:"balance"` // Balance in internal currency
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID            string          `gorm:"type:uuid;primary_key" json:"id"`
	UserID        string          `gorm:"type:uuid;not null;index" json:"user_id"`
	PostID        string          `gorm:"type:uuid;index" json:"post_id,omitempty"`
	Type          TransactionType `gorm:"type:varchar(20);not null" json:"type"`
	Amount        int             `gorm:"not null" json:"amount"`
	BalanceBefore int             `json:"balance_before"`
	BalanceAfter  int             `json:"balance_after"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	return nil
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

