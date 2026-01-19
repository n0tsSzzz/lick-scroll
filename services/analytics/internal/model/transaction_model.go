package model

import "time"

type TransactionModel struct {
	ID            string    `gorm:"column:id;type:uuid;primaryKey"`
	UserID        string    `gorm:"column:user_id;type:uuid;not null"`
	PostID        string    `gorm:"column:post_id;type:uuid"`
	Type          string    `gorm:"column:type;type:varchar(50);not null"`
	Amount        int       `gorm:"column:amount;type:integer;not null"`
	BalanceBefore int       `gorm:"column:balance_before;type:integer"`
	BalanceAfter  int       `gorm:"column:balance_after;type:integer"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamp"`
}

func (TransactionModel) TableName() string {
	return "transactions"
}
