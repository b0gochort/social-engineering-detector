package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"backend/internal/crypto"
	"backend/internal/models"
	"backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
)

var ( // Define custom errors
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Secret key for JWT signing (for demonstration purposes only, use a strong, secure key in production)
var jwtSecret = []byte("supersecretjwtkey")

// GetJWTSecret returns the JWT secret key.
func GetJWTSecret() []byte {
	return jwtSecret
}

type AuthService interface {
	RegisterParent(username, password string) (*models.User, error)
	Login(username, password string) (string, time.Time, error) // Returns JWT token, expiration time, and error
	Logout(username string) error
	// TODO: Add ChangePassword methods
}

func (s *authService) Logout(username string) error {
	// TODO: Invalidate JWT token (e.g., add to a blacklist in Redis)
	// For now, we'll just log it.
	s.logger.Info("User token would be invalidated.", zap.String("username", username))

	// TODO: Destroy Data Key (DK) from secure memory
	// For now, we'll just log it.
	s.logger.Info("User DK would be destroyed from memory.", zap.String("username", username))

	// TODO: Log "User Logout" event to AuditLog
	s.logger.Info("User logged out successfully.", zap.String("username", username))

	return nil
}

type authService struct {
	repo       repository.AuthRepository
	logger     *zap.Logger
	keyManager *crypto.KeyManager
}

func NewAuthService(repo repository.AuthRepository, keyManager *crypto.KeyManager, logger *zap.Logger) AuthService {
	return &authService{
		repo:       repo,
		logger:     logger,
		keyManager: keyManager,
	}
}

func (s *authService) RegisterParent(username, password string) (*models.User, error) {
	// Check if a user already exists
	count, err := s.repo.CountUsers()
	if err != nil {
		s.logger.Error("Failed to count users", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}
	if count > 0 {
		return nil, ErrUserAlreadyExists
	}

	// Hash the Master Password (MP)
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate and encrypt Data Key (DK) with MASTER_KEY
	dkEncrypted, err := s.keyManager.GenerateAndEncryptDataKey()
	if err != nil {
		s.logger.Error("Failed to generate and encrypt DK", zap.Error(err))
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	user := &models.User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "parent", // Hardcode role for now
		DKEncrypted:  dkEncrypted,
	}

	err = s.repo.CreateUser(user)
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (s *authService) Login(username, password string) (string, time.Time, error) {
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, errors.New("sql: no rows in result set")) { // Specific error for no user found
			return "", time.Time{}, ErrUserNotFound
		}
		s.logger.Error("Failed to get user by username", zap.Error(err))
		return "", time.Time{}, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify password
	if !s.verifyPassword(user.PasswordHash, password) {
		return "", time.Time{}, ErrInvalidCredentials
	}

	// TODO: Decrypt DKenc using MP to get DK and store in secure memory
	// For now, we'll just log that DK would be decrypted.
	s.logger.Debug("DK for user would be decrypted and stored in memory.", zap.String("username", user.Username))

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		s.logger.Error("Failed to generate JWT token", zap.Error(err))
		return "", time.Time{}, fmt.Errorf("failed to generate token: %w", err)
	}

	// TODO: Log "User Login" event to AuditLog
	s.logger.Info("User logged in successfully.", zap.String("username", user.Username))

	return tokenString, expirationTime, nil
}

// hashPassword uses Argon2 to hash the password.
func (s *authService) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, uint8(4), 32)

	// Store salt and hash together, e.g., $argon2id$v=19$m=65536,t=1,p=4$BASE64_SALT$BASE64_HASH
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, 64*1024, 1, 4, encodedSalt, encodedHash), nil
}

// verifyPassword compares a plaintext password with a hashed password.
func (s *authService) verifyPassword(hashedPassword, password string) bool {
	// Parse the encoded hash format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	parts := []byte(hashedPassword)

	// Split by '$'
	sections := make([]string, 0)
	start := 0
	for i, b := range parts {
		if b == '$' {
			if i > start {
				sections = append(sections, string(parts[start:i]))
			}
			start = i + 1
		}
	}
	if start < len(parts) {
		sections = append(sections, string(parts[start:]))
	}

	// Expected format: ["argon2id", "v=19", "m=65536,t=1,p=4", "salt", "hash"]
	if len(sections) != 5 {
		s.logger.Error("Invalid hash format", zap.Int("sections", len(sections)))
		return false
	}

	// Parse parameters
	var version int
	fmt.Sscanf(sections[1], "v=%d", &version)

	var m, t, p uint32
	fmt.Sscanf(sections[2], "m=%d,t=%d,p=%d", &m, &t, &p)

	saltStr := sections[3]
	hashStr := sections[4]

	decodedSalt, err := base64.RawStdEncoding.DecodeString(saltStr)
	if err != nil {
		s.logger.Error("Failed to decode salt", zap.Error(err))
		return false
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(hashStr)
	if err != nil {
		s.logger.Error("Failed to decode hash", zap.Error(err))
		return false
	}

	// Re-hash the provided password with the extracted parameters and salt
	comparisonHash := argon2.IDKey([]byte(password), decodedSalt, t, m, uint8(p), uint32(len(decodedHash)))

	// Compare the generated hash with the stored hash
	return fmt.Sprintf("%x", comparisonHash) == fmt.Sprintf("%x", decodedHash)
}

// generateRandomBytes generates a cryptographically secure random byte slice.
func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
