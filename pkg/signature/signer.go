// Package signature provides image signing and verification utilities.
package signature

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Signer provides image signing capabilities.
type Signer struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	keyID      string
	keyPath    string
}

// SignerConfig holds signer configuration.
type SignerConfig struct {
	KeyPath string
	KeyID   string
}

// SignaturePayload represents the data to be signed.
type SignaturePayload struct {
	ImageRef  string    `json:"image_ref"`
	Digest    string    `json:"digest"`
	Timestamp time.Time `json:"timestamp"`
	Signer    string    `json:"signer"`
	KeyID     string    `json:"key_id"`
}

// Signature represents a digital signature.
type Signature struct {
	Payload   SignaturePayload `json:"payload"`
	Signature string           `json:"signature"`
	Algorithm string           `json:"algorithm"`
}

// NewSigner creates a new Signer instance.
func NewSigner(config *SignerConfig) (*Signer, error) {
	s := &Signer{
		keyPath: config.KeyPath,
		keyID:   config.KeyID,
	}

	// Try to load existing key
	if err := s.loadKey(); err != nil {
		// Generate new key if not exists
		if err := s.generateKey(); err != nil {
			return nil, err
		}
		if err := s.saveKey(); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Sign signs an image digest.
func (s *Signer) Sign(imageRef, digest, signer string) (*Signature, error) {
	if s.privateKey == nil {
		return nil, errors.New("no private key available")
	}

	payload := SignaturePayload{
		ImageRef:  imageRef,
		Digest:    digest,
		Timestamp: time.Now(),
		Signer:    signer,
		KeyID:     s.keyID,
	}

	// Create hash of payload
	hash := s.hashPayload(payload)

	// Sign the hash
	r, ss, err := ecdsa.Sign(rand.Reader, s.privateKey, hash)
	if err != nil {
		return nil, err
	}

	// Encode signature
	sigBytes := append(r.Bytes(), ss.Bytes()...)
	sigStr := base64.StdEncoding.EncodeToString(sigBytes)

	return &Signature{
		Payload:   payload,
		Signature: sigStr,
		Algorithm: "ECDSA-P256-SHA256",
	}, nil
}

// Verify verifies a signature.
func (s *Signer) Verify(sig *Signature) (bool, error) {
	if s.publicKey == nil {
		return false, errors.New("no public key available")
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(sig.Signature)
	if err != nil {
		return false, err
	}

	if len(sigBytes) != 64 {
		return false, errors.New("invalid signature length")
	}

	// Extract r and s
	r := new(big.Int).SetBytes(sigBytes[:32])
	ss := new(big.Int).SetBytes(sigBytes[32:])

	// Hash payload
	hash := s.hashPayload(sig.Payload)

	// Verify
	valid := ecdsa.Verify(s.publicKey, hash, r, ss)
	return valid, nil
}

// hashPayload creates a hash of the signature payload.
func (s *Signer) hashPayload(payload SignaturePayload) []byte {
	data := payload.ImageRef + payload.Digest + payload.Timestamp.String() + payload.Signer + payload.KeyID
	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// generateKey generates a new ECDSA key pair.
func (s *Signer) generateKey() error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	s.privateKey = privateKey
	s.publicKey = &privateKey.PublicKey

	if s.keyID == "" {
		s.keyID = s.generateKeyID()
	}

	return nil
}

// generateKeyID generates a key ID from the public key.
func (s *Signer) generateKeyID() string {
	if s.publicKey == nil {
		return ""
	}

	pubBytes := elliptic.Marshal(s.publicKey.Curve, s.publicKey.X, s.publicKey.Y)
	hash := sha256.Sum256(pubBytes)
	return base64.StdEncoding.EncodeToString(hash[:8])
}

// loadKey loads the key from disk.
func (s *Signer) loadKey() error {
	if s.keyPath == "" {
		return errors.New("no key path specified")
	}

	privPath := filepath.Join(s.keyPath, "private.pem")
	privData, err := os.ReadFile(privPath)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(privData)
	if block == nil {
		return errors.New("failed to decode PEM block")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	s.privateKey = privateKey
	s.publicKey = &privateKey.PublicKey

	if s.keyID == "" {
		s.keyID = s.generateKeyID()
	}

	return nil
}

// saveKey saves the key to disk.
func (s *Signer) saveKey() error {
	if s.keyPath == "" || s.privateKey == nil {
		return errors.New("no key to save")
	}

	// Ensure directory exists
	os.MkdirAll(s.keyPath, 0700)

	// Save private key
	privBytes, err := x509.MarshalECPrivateKey(s.privateKey)
	if err != nil {
		return err
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privBytes,
	})

	privPath := filepath.Join(s.keyPath, "private.pem")
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return err
	}

	// Save public key
	pubBytes, err := x509.MarshalPKIXPublicKey(s.publicKey)
	if err != nil {
		return err
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	pubPath := filepath.Join(s.keyPath, "public.pem")
	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		return err
	}

	return nil
}

// GetPublicKey returns the public key in PEM format.
func (s *Signer) GetPublicKey() (string, error) {
	if s.publicKey == nil {
		return "", errors.New("no public key available")
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(s.publicKey)
	if err != nil {
		return "", err
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return string(pubPEM), nil
}

// GetKeyID returns the key ID.
func (s *Signer) GetKeyID() string {
	return s.keyID
}
