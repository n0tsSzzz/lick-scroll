package entity

import "time"

type TransactionType string

const (
	TransactionTypePurchase TransactionType = "purchase"
	TransactionTypeEarn     TransactionType = "earn"
	TransactionTypeRefund   TransactionType = "refund"
	TransactionTypeDonation TransactionType = "donation"
)

type Wallet struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int       `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	PostID        string          `json:"post_id,omitempty"`
	Type          TransactionType  `json:"type"`
	Amount        int             `json:"amount"`
	BalanceBefore int             `json:"balance_before"`
	BalanceAfter  int             `json:"balance_after"`
	CreatedAt     time.Time       `json:"created_at"`
}
