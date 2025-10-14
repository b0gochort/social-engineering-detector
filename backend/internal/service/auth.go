package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"backend/internal/models"
	"backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

var ( // Define custom errors
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
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
	s.log.Infof("User %s token would be invalidated.", username)

	// TODO: Destroy Data Key (DK) from secure memory
	// For now, we'll just log it.
	s.log.Infof("User %s DK would be destroyed from memory.", username)

	// TODO: Log "User Logout" event to AuditLog
	s.log.Infof("User %s logged out successfully.", username)

	return nil
}

type authService struct {
	repo repository.AuthRepository
	log  *logrus.Logger
}

func NewAuthService(repo repository.AuthRepository, log *logrus.Logger) AuthService {
	return &authService{repo: repo, log: log}
}

func (s *authService) RegisterParent(username, password string) (*models.User, error) {
	// Check if a user already exists
	count, err := s.repo.CountUsers()
	if err != nil {
		s.log.Errorf("Failed to count users: %v", err)
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}
	if count > 0 {
		return nil, ErrUserAlreadyExists
	}

	// Hash the Master Password (MP)
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		s.log.Errorf("Failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate Data Key (DK)
	dk, err := generateRandomBytes(32) // 32 bytes for AES-256
	if err != nil {
		s.log.Errorf("Failed to generate DK: %v", err)
		return nil, fmt.Errorf("failed to generate data key: %w", err)
	}

	// Encrypt DK with MP (KEK process - placeholder for actual implementation)
	// For now, we'll just base64 encode the DK as a placeholder for DKenc
	// In a real scenario, this would involve deriving a key from MP and encrypting DK
	dkEncrypted := base64.StdEncoding.EncodeToString(dk)

	user := &models.User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "parent", // Hardcode role for now
		DKEncrypted:  dkEncrypted,
	}

	err = s.repo.CreateUser(user)
	if err != nil {
		s.log.Errorf("Failed to create user: %v", err)
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
		s.log.Errorf("Failed to get user by username: %v", err)
		return "", time.Time{}, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// Verify password
	if !s.verifyPassword(user.PasswordHash, password) {
		return "", time.Time{}, ErrInvalidCredentials
	}

	// TODO: Decrypt DKenc using MP to get DK and store in secure memory
	// For now, we'll just log that DK would be decrypted.
	s.log.Debugf("DK for user %s would be decrypted and stored in memory.", user.Username)

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
		s.log.Errorf("Failed to generate JWT token: %v", err)
		return "", time.Time{}, fmt.Errorf("failed to generate token: %w", err)
	}

	// TODO: Log "User Login" event to AuditLog
	s.log.Infof("User %s logged in successfully.", user.Username)

	return tokenString, expirationTime, nil
}

// hashPassword uses Argon2 to hash the password.
func (s *authService) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Store salt and hash together, e.g., $argon2id$v=19$m=65536,t=1,p=4$BASE64_SALT$BASE64_HASH
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, 64*1024, 1, 4, encodedSalt, encodedHash), nil
}

// verifyPassword compares a plaintext password with a hashed password.
func (s *authService) verifyPassword(hashedPassword, password string) bool {
	// Extract salt and parameters from the hashed password string
	var version int
	var m, t, p uint32
	var salt, hash []byte

	_, err := fmt.Sscanf(hashedPassword, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", &version, &m, &t, &p, &salt, &hash)
	if err != nil {
		s.log.Errorf("Failed to parse hashed password: %v", err)
		return false
	}

	decodedSalt, err := base64.RawStdEncoding.DecodeString(string(salt))
	if err != nil {
		s.log.Errorf("Failed to decode salt: %v", err)
		return false
	}
	decodedHash, err := base64.RawStdEncoding.DecodeString(string(hash))
	if err != nil {
		s.log.Errorf("Failed to decode hash: %v", err)
		return false
	}

	// Re-hash the provided password with the extracted parameters and salt
	comparisonHash := argon2.IDKey([]byte(password), decodedSalt, t, m, p, uint32(len(decodedHash)))

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
