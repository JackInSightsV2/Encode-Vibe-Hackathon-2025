package auth

import (
	"context"
	"fmt"
	"time"
	"qt1-middleware/models"
	"qt1-middleware/repositories"

	"github.com/go-playground/validator/v10"
)

// Service implements the AuthService interface
type Service struct {
	config          *AuthConfig
	userRepo        repositories.UserRepository
	jwtManager      *JWTManager
	passwordManager *PasswordManager
	validator       *validator.Validate
}

// NewService creates a new authentication service
func NewService(config *AuthConfig, userRepo repositories.UserRepository) *Service {
	if config == nil {
		config = DefaultAuthConfig()
	}
	
	service := &Service{
		config:          config,
		userRepo:        userRepo,
		jwtManager:      NewJWTManager(config),
		passwordManager: NewPasswordManager(config),
		validator:       validator.New(),
	}
	
	return service
}

// UserService interface implementation for OAuth2 compatibility

// Create creates a new user
func (s *Service) Create(ctx context.Context, user *models.User) error {
	return s.userRepo.Create(ctx, user)
}

// GetByID gets a user by ID
func (s *Service) GetByID(ctx context.Context, id int) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetByUsername gets a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

// GetByEmail gets a user by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// Update updates a user
func (s *Service) Update(ctx context.Context, user *models.User) error {
	return s.userRepo.Update(ctx, user)
}

// Delete deletes a user
func (s *Service) Delete(ctx context.Context, id int) error {
	return s.userRepo.Delete(ctx, id)
}

// OAuth2-specific methods (placeholder implementations)

// GetByOAuth2Provider gets a user by OAuth2 provider and user ID
func (s *Service) GetByOAuth2Provider(ctx context.Context, provider, providerUserID string) (*models.User, error) {
	// TODO: Implement actual OAuth2 account lookup
	// This would require a new repository or table for OAuth2 account links
	return nil, fmt.Errorf("OAuth2 account not found")
}

// LinkOAuth2Account links an OAuth2 account to a user
func (s *Service) LinkOAuth2Account(ctx context.Context, userID int, provider, providerUserID, email string) error {
	// TODO: Implement actual OAuth2 account linking
	// This would store the OAuth2 account link in the database
	return nil
}

// UnlinkOAuth2Account unlinks an OAuth2 account from a user
func (s *Service) UnlinkOAuth2Account(ctx context.Context, userID int, provider string) error {
	// TODO: Implement actual OAuth2 account unlinking
	return nil
}

// GetLinkedOAuth2Accounts gets OAuth2 accounts linked to a user
func (s *Service) GetLinkedOAuth2Accounts(ctx context.Context, userID int) ([]*models.OAuth2Account, error) {
	// TODO: Implement actual OAuth2 account retrieval
	return []*models.OAuth2Account{}, nil
}

// Login authenticates a user and returns a token
func (s *Service) Login(username, password string) (*AuthResult, error) {
	ctx := context.Background()
	
	// Get user from database
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	
	// Check if user is active
	if !user.Active {
		return nil, ErrUserInactive
	}
	
	// Verify password
	if err := s.passwordManager.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, err
	}
	
	// Update last login time
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Warning: failed to update last login for user %d: %v\n", user.ID, err)
	}
	
	// Generate tokens
	token, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	
	refreshToken, err := s.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	return &AuthResult{
		User:         user,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.config.TokenExpiration),
	}, nil
}

// Logout invalidates a user's token (placeholder for future session management)
func (s *Service) Logout(userID int) error {
	// For now, this is a no-op since we're using stateless JWT tokens
	// In the future, we could add token blacklisting or session management
	return nil
}

// Register creates a new user account
func (s *Service) Register(req *RegisterRequest) (*models.User, error) {
	ctx := context.Background()
	
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUsernameExists
	}
	
	// Check if email already exists
	existingUser, err = s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailExists
	}
	
	// Hash password
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	
	// Set default role if not specified
	role := req.Role
	if role == "" {
		role = "user"
	}
	
	// Create user
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         role,
		Active:       true,
	}
	
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

// GenerateToken generates a JWT token for a user
func (s *Service) GenerateToken(user *models.User) (string, error) {
	return s.jwtManager.GenerateToken(user)
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}

// RefreshToken generates a new token using a refresh token
func (s *Service) RefreshToken(refreshTokenString string) (string, error) {
	ctx := context.Background()
	
	// Validate refresh token and get user ID
	userID, err := s.jwtManager.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", err
	}
	
	// Get user from database
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	
	if user == nil {
		return "", ErrUserNotFound
	}
	
	// Check if user is still active
	if !user.Active {
		return "", ErrUserInactive
	}
	
	// Generate new token
	return s.jwtManager.GenerateToken(user)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID int, oldPassword, newPassword string) error {
	ctx := context.Background()
	
	// Get user from database
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	if user == nil {
		return ErrUserNotFound
	}
	
	// Verify old password
	if err := s.passwordManager.VerifyPassword(user.PasswordHash, oldPassword); err != nil {
		return &AuthError{
			Code:    "INVALID_OLD_PASSWORD",
			Message: "Current password is incorrect",
		}
	}
	
	// Hash new password
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return err
	}
	
	// Update password in database
	if err := s.userRepo.ChangePassword(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	return nil
}

// HashPassword creates a bcrypt hash of the password
func (s *Service) HashPassword(password string) (string, error) {
	return s.passwordManager.HashPassword(password)
}

// VerifyPassword compares a hashed password with a plaintext password
func (s *Service) VerifyPassword(hashedPassword, password string) error {
	return s.passwordManager.VerifyPassword(hashedPassword, password)
}

// GetUserByToken retrieves a user from a JWT token
func (s *Service) GetUserByToken(tokenString string) (*models.User, error) {
	ctx := context.Background()
	
	// Validate token and get claims
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	
	// Get user from database
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if user == nil {
		return nil, ErrUserNotFound
	}
	
	// Check if user is still active
	if !user.Active {
		return nil, ErrUserInactive
	}
	
	return user, nil
}

// ValidatePasswordRequirements checks if a password meets the requirements
func (s *Service) ValidatePasswordRequirements(password string) error {
	return s.passwordManager.ValidatePassword(password)
}

// GetConfig returns the authentication configuration
func (s *Service) GetConfig() *AuthConfig {
	return s.config
}

// CreateDefaultAdmin creates a default admin user if none exists
func (s *Service) CreateDefaultAdmin() error {
	ctx := context.Background()
	
	// Check if any admin users exist
	adminCount, err := s.userRepo.CountByRole(ctx, "admin")
	if err != nil {
		return fmt.Errorf("failed to count admin users: %w", err)
	}
	
	if adminCount > 0 {
		return nil // Admin already exists
	}
	
	// Create default admin
	req := &RegisterRequest{
		Username: "admin",
		Email:    "admin@localhost.local",
		Password: "AdminPassword123!",
		Role:     "admin",
	}
	
	_, err = s.Register(req)
	if err != nil {
		return fmt.Errorf("failed to create default admin: %w", err)
	}
	
	fmt.Println("Created default admin user (username: admin, password: AdminPassword123!)")
	fmt.Println("WARNING: Change the default admin password immediately!")
	
	return nil
}