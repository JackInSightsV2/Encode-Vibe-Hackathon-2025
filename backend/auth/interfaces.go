package auth

import (
	"context"
	"qt1-middleware/models"
)

// UserService defines the interface for user operations needed by OAuth2
type UserService interface {
	// Basic user operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
	
	// OAuth2-specific operations
	GetByOAuth2Provider(ctx context.Context, provider, providerUserID string) (*models.User, error)
	LinkOAuth2Account(ctx context.Context, userID int, provider, providerUserID, email string) error
	UnlinkOAuth2Account(ctx context.Context, userID int, provider string) error
	GetLinkedOAuth2Accounts(ctx context.Context, userID int) ([]*models.OAuth2Account, error)
}

// OAuth2AccountService defines operations for managing OAuth2 account links
type OAuth2AccountService interface {
	Create(ctx context.Context, account *models.OAuth2Account) error
	GetByUserID(ctx context.Context, userID int) ([]*models.OAuth2Account, error)
	GetByProvider(ctx context.Context, provider, providerUserID string) (*models.OAuth2Account, error)
	Update(ctx context.Context, account *models.OAuth2Account) error
	Delete(ctx context.Context, id int) error
	DeleteByUserIDAndProvider(ctx context.Context, userID int, provider string) error
}