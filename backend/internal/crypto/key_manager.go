package crypto

import (
	"encoding/base64"
	"errors"
	"os"
	"sync"
)

var (
	ErrMasterKeyNotSet      = errors.New("master key not set in environment")
	ErrInvalidMasterKey     = errors.New("invalid master key: must be 64 hex characters (32 bytes)")
	ErrDataKeyNotFound      = errors.New("data key not found for user")
	ErrDataKeyDecryptFailed = errors.New("failed to decrypt data key")
)

// KeyManager manages encryption keys for users
type KeyManager struct {
	masterKey []byte
	// In-memory cache of decrypted data keys (user_id -> data_key)
	dataKeys map[int64][]byte
	mu       sync.RWMutex
}

// NewKeyManager creates a new key manager with master key from environment
func NewKeyManager() (*KeyManager, error) {
	masterKeyHex := os.Getenv("MASTER_KEY")
	if masterKeyHex == "" {
		return nil, ErrMasterKeyNotSet
	}

	// Decode master key from base64
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyHex)
	if err != nil || len(masterKey) != 32 {
		return nil, ErrInvalidMasterKey
	}

	return &KeyManager{
		masterKey: masterKey,
		dataKeys:  make(map[int64][]byte),
	}, nil
}

// GenerateAndEncryptDataKey generates a new data key and encrypts it with master key
// Returns the encrypted data key (base64) that should be stored in the database
func (km *KeyManager) GenerateAndEncryptDataKey() (string, error) {
	// Generate new random data key
	dataKey, err := GenerateKey()
	if err != nil {
		return "", err
	}

	// Encrypt data key with master key
	encryptedDK, err := Encrypt(base64.StdEncoding.EncodeToString(dataKey), km.masterKey)
	if err != nil {
		return "", err
	}

	return encryptedDK, nil
}

// DecryptDataKey decrypts a user's data key using the master key
func (km *KeyManager) DecryptDataKey(encryptedDK string) ([]byte, error) {
	// Decrypt data key with master key
	decryptedDKBase64, err := Decrypt(encryptedDK, km.masterKey)
	if err != nil {
		return nil, ErrDataKeyDecryptFailed
	}

	// Decode from base64
	dataKey, err := base64.StdEncoding.DecodeString(decryptedDKBase64)
	if err != nil {
		return nil, ErrDataKeyDecryptFailed
	}

	return dataKey, nil
}

// CacheDataKey caches a decrypted data key for a user
func (km *KeyManager) CacheDataKey(userID int64, dataKey []byte) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.dataKeys[userID] = dataKey
}

// GetCachedDataKey retrieves a cached data key for a user
func (km *KeyManager) GetCachedDataKey(userID int64) ([]byte, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	dataKey, exists := km.dataKeys[userID]
	return dataKey, exists
}

// LoadDataKey loads and caches a user's data key
func (km *KeyManager) LoadDataKey(userID int64, encryptedDK string) ([]byte, error) {
	// Check cache first
	if dataKey, exists := km.GetCachedDataKey(userID); exists {
		return dataKey, nil
	}

	// Decrypt and cache
	dataKey, err := km.DecryptDataKey(encryptedDK)
	if err != nil {
		return nil, err
	}

	km.CacheDataKey(userID, dataKey)
	return dataKey, nil
}

// EncryptMessage encrypts a message using a user's data key
func (km *KeyManager) EncryptMessage(plaintext string, userID int64, encryptedDK string) (string, error) {
	dataKey, err := km.LoadDataKey(userID, encryptedDK)
	if err != nil {
		return "", err
	}

	return Encrypt(plaintext, dataKey)
}

// DecryptMessage decrypts a message using a user's data key
func (km *KeyManager) DecryptMessage(ciphertext string, userID int64, encryptedDK string) (string, error) {
	dataKey, err := km.LoadDataKey(userID, encryptedDK)
	if err != nil {
		return "", err
	}

	return Decrypt(ciphertext, dataKey)
}
