// Package registry provides container image registry functionality.
package registry

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// EncryptedPrefix is the prefix for encrypted values.
	EncryptedPrefix = "encrypted:"
	// DefaultEncryptionKey is used when no key is provided (should be overridden in production).
	DefaultEncryptionKey = "cyp-registry-default-key!!!!!!!!"
)

// Credential represents a stored credential for a registry.
type Credential struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"` // Stored encrypted with "encrypted:" prefix
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CredentialStore represents the credential storage structure.
type CredentialStore struct {
	Credentials map[string]*Credential `json:"credentials"` // registry URL -> Credential
}

// CredentialManager handles credential storage and encryption.
type CredentialManager struct {
	storagePath   string
	encryptionKey []byte
	mu            sync.RWMutex
}

// NewCredentialManager creates a new CredentialManager.
func NewCredentialManager(storagePath string, encryptionKey string) (*CredentialManager, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create credential storage directory: %w", err)
	}

	key := encryptionKey
	if key == "" {
		key = DefaultEncryptionKey
	}

	// Derive a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(key))

	return &CredentialManager{
		storagePath:   storagePath,
		encryptionKey: hash[:],
	}, nil
}

// getCredentialFilePath returns the path to the credentials file.
func (cm *CredentialManager) getCredentialFilePath() string {
	return filepath.Join(cm.storagePath, "credentials.json")
}

// loadStore loads the credential store from disk.
func (cm *CredentialManager) loadStore() (*CredentialStore, error) {
	filePath := cm.getCredentialFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CredentialStore{
				Credentials: make(map[string]*Credential),
			}, nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var store CredentialStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	if store.Credentials == nil {
		store.Credentials = make(map[string]*Credential)
	}

	return &store, nil
}

// saveStore saves the credential store to disk.
func (cm *CredentialManager) saveStore(store *CredentialStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	filePath := cm.getCredentialFilePath()
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// encrypt encrypts plaintext using AES-GCM.
func (cm *CredentialManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-GCM.
func (cm *CredentialManager) decrypt(ciphertext string) (string, error) {
	// Remove encrypted prefix if present
	if len(ciphertext) > len(EncryptedPrefix) && ciphertext[:len(EncryptedPrefix)] == EncryptedPrefix {
		ciphertext = ciphertext[len(EncryptedPrefix):]
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// SaveCredential saves a credential for a registry with encrypted password.
func (cm *CredentialManager) SaveCredential(registryURL, username, password string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	store, err := cm.loadStore()
	if err != nil {
		return err
	}

	// Encrypt the password
	encryptedPassword, err := cm.encrypt(password)
	if err != nil {
		return fmt.Errorf("failed to encrypt password: %w", err)
	}

	now := time.Now().UTC()
	cred := &Credential{
		Username:  username,
		Password:  encryptedPassword,
		UpdatedAt: now,
	}

	// Preserve creation time if updating existing credential
	if existing, ok := store.Credentials[registryURL]; ok {
		cred.CreatedAt = existing.CreatedAt
	} else {
		cred.CreatedAt = now
	}

	store.Credentials[registryURL] = cred

	return cm.saveStore(store)
}

// GetCredential retrieves a credential for a registry with decrypted password.
func (cm *CredentialManager) GetCredential(registryURL string) (*Credential, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	store, err := cm.loadStore()
	if err != nil {
		return nil, err
	}

	cred, ok := store.Credentials[registryURL]
	if !ok {
		return nil, fmt.Errorf("credential not found for registry: %s", registryURL)
	}

	// Decrypt the password
	decryptedPassword, err := cm.decrypt(cred.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	return &Credential{
		Username:  cred.Username,
		Password:  decryptedPassword,
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	}, nil
}

// GetCredentialEncrypted retrieves a credential without decrypting the password.
func (cm *CredentialManager) GetCredentialEncrypted(registryURL string) (*Credential, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	store, err := cm.loadStore()
	if err != nil {
		return nil, err
	}

	cred, ok := store.Credentials[registryURL]
	if !ok {
		return nil, fmt.Errorf("credential not found for registry: %s", registryURL)
	}

	return &Credential{
		Username:  cred.Username,
		Password:  cred.Password, // Keep encrypted
		CreatedAt: cred.CreatedAt,
		UpdatedAt: cred.UpdatedAt,
	}, nil
}

// DeleteCredential removes a credential for a registry.
func (cm *CredentialManager) DeleteCredential(registryURL string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	store, err := cm.loadStore()
	if err != nil {
		return err
	}

	if _, ok := store.Credentials[registryURL]; !ok {
		return fmt.Errorf("credential not found for registry: %s", registryURL)
	}

	delete(store.Credentials, registryURL)

	return cm.saveStore(store)
}

// ListCredentials returns all stored credentials (with encrypted passwords).
func (cm *CredentialManager) ListCredentials() (map[string]*Credential, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	store, err := cm.loadStore()
	if err != nil {
		return nil, err
	}

	// Return a copy with masked passwords for security
	result := make(map[string]*Credential)
	for url, cred := range store.Credentials {
		result[url] = &Credential{
			Username:  cred.Username,
			Password:  "********", // Mask password in list
			CreatedAt: cred.CreatedAt,
			UpdatedAt: cred.UpdatedAt,
		}
	}

	return result, nil
}

// HasCredential checks if a credential exists for a registry.
func (cm *CredentialManager) HasCredential(registryURL string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	store, err := cm.loadStore()
	if err != nil {
		return false
	}

	_, ok := store.Credentials[registryURL]
	return ok
}

// IsPasswordEncrypted checks if a password string is encrypted.
func IsPasswordEncrypted(password string) bool {
	return len(password) > len(EncryptedPrefix) && password[:len(EncryptedPrefix)] == EncryptedPrefix
}
