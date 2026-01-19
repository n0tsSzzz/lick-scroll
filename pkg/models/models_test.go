package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_BeforeCreate(t *testing.T) {
	user := &User{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password",
		Role:     RoleViewer,
		IsActive: true,
	}

	// BeforeCreate should set ID if empty
	err := user.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, user.ID)
}

func TestUser_BeforeCreate_WithID(t *testing.T) {
	existingID := "existing-id-123"
	user := &User{
		ID:       existingID,
		Email:    "test@example.com",
		Username: "testuser",
		Password: "password",
	}

	err := user.BeforeCreate(nil)
	assert.NoError(t, err)
	// ID should remain unchanged if already set
	assert.Equal(t, existingID, user.ID)
}

func TestPost_BeforeCreate(t *testing.T) {
	post := &Post{
		CreatorID: "creator-123",
		Title:     "Test Post",
		Type:      "photo",
		Status:    StatusPending,
	}

	// BeforeCreate should set ID if empty
	err := post.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, post.ID)
}

func TestPost_BeforeCreate_WithID(t *testing.T) {
	existingID := "existing-post-id"
	post := &Post{
		ID:        existingID,
		CreatorID: "creator-123",
		Title:     "Test Post",
		Type:      "photo",
		Status:    StatusPending,
	}

	err := post.BeforeCreate(nil)
	assert.NoError(t, err)
	// ID should remain unchanged if already set
	assert.Equal(t, existingID, post.ID)
}

func TestPostStatus_Constants(t *testing.T) {
	// Test that status constants are defined
	assert.Equal(t, PostStatus("pending"), StatusPending)
	assert.Equal(t, PostStatus("approved"), StatusApproved)
	assert.Equal(t, PostStatus("rejected"), StatusRejected)
}

func TestUserRole_Constants(t *testing.T) {
	// Test that role constants are defined
	assert.Equal(t, UserRole("viewer"), RoleViewer)
	assert.Equal(t, UserRole("creator"), RoleCreator)
	assert.Equal(t, UserRole("moderator"), RoleModerator)
}

func TestLike_BeforeCreate(t *testing.T) {
	like := &Like{
		UserID: "user-123",
		PostID: "post-123",
	}

	err := like.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, like.ID)
}

func TestSubscription_BeforeCreate(t *testing.T) {
	subscription := &Subscription{
		ViewerID:  "viewer-123",
		CreatorID: "creator-123",
	}

	err := subscription.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, subscription.ID)
}

func TestWallet_BeforeCreate(t *testing.T) {
	wallet := &Wallet{
		UserID: "user-123",
		Balance: 0,
	}

	err := wallet.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, wallet.ID)
}

func TestTransaction_BeforeCreate(t *testing.T) {
	transaction := &Transaction{
		UserID: "user-123",
		Type:   TransactionTypePurchase,
		Amount: 100,
	}

	err := transaction.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, transaction.ID)
}

func TestPostImage_BeforeCreate(t *testing.T) {
	postImage := &PostImage{
		PostID:   "post-123",
		ImageURL: "http://example.com/image.jpg",
	}

	err := postImage.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, postImage.ID)
}

func TestTransactionType_Constants(t *testing.T) {
	// Test that transaction type constants are defined
	assert.Equal(t, TransactionType("purchase"), TransactionTypePurchase)
	assert.Equal(t, TransactionType("earn"), TransactionTypeEarn)
	assert.Equal(t, TransactionType("refund"), TransactionTypeRefund)
	assert.Equal(t, TransactionType("donation"), TransactionTypeDonation)
}
