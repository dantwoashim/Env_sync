package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"crypto/sha256"
)

const (
	// MagicBytes identifies an EnvSync encrypted file.
	MagicBytes = "ENVSYNC\x01"
	magicLen   = 8

	// NonceSize for XChaCha20-Poly1305 (24 bytes).
	NonceSize = 24

	// hkdfSalt for at-rest encryption key derivation.
	hkdfSalt = "envsync-at-rest-v1"

	// hkdfInfo for at-rest encryption key derivation.
	hkdfInfo = "local-storage-encryption"
)

// DeriveAtRestKey derives an encryption key from SSH private key bytes using HKDF-SHA256.
func DeriveAtRestKey(sshPrivateKeyBytes []byte) ([32]byte, error) {
	if len(sshPrivateKeyBytes) == 0 {
		return [32]byte{}, errors.New("empty key material")
	}

	hkdfReader := hkdf.New(sha256.New, sshPrivateKeyBytes, []byte(hkdfSalt), []byte(hkdfInfo))

	var key [32]byte
	if _, err := io.ReadFull(hkdfReader, key[:]); err != nil {
		return [32]byte{}, fmt.Errorf("HKDF key derivation failed: %w", err)
	}

	return key, nil
}

// Encrypt encrypts plaintext using XChaCha20-Poly1305 with the given key.
// Returns the EnvSync file format: magic (8) + nonce (24) + ciphertext + tag (16).
func Encrypt(plaintext []byte, key [32]byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, fmt.Errorf("creating XChaCha20-Poly1305: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	// Encrypt
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Assemble: magic + nonce + ciphertext (includes Poly1305 tag)
	result := make([]byte, 0, magicLen+NonceSize+len(ciphertext))
	result = append(result, MagicBytes...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts an EnvSync encrypted file using XChaCha20-Poly1305.
// Expects format: magic (8) + nonce (24) + ciphertext + tag (16).
func Decrypt(data []byte, key [32]byte) ([]byte, error) {
	minSize := magicLen + NonceSize + chacha20poly1305.Overhead
	if len(data) < minSize {
		return nil, fmt.Errorf("encrypted data too short: got %d bytes, minimum %d", len(data), minSize)
	}

	// Verify magic bytes
	if string(data[:magicLen]) != MagicBytes {
		return nil, errors.New("not an EnvSync encrypted file (invalid magic bytes)")
	}

	// Extract nonce and ciphertext
	nonce := data[magicLen : magicLen+NonceSize]
	ciphertext := data[magicLen+NonceSize:]

	aead, err := chacha20poly1305.NewX(key[:])
	if err != nil {
		return nil, fmt.Errorf("creating XChaCha20-Poly1305: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong key or corrupted data): %w", err)
	}

	return plaintext, nil
}

// EncryptForRecipient encrypts plaintext for a specific recipient using ephemeral ECDH.
// Returns: ephemeral public key (32) + nonce (24) + ciphertext + tag (16).
func EncryptForRecipient(plaintext []byte, recipientPublicKey [32]byte) (ephemeralPub [32]byte, encrypted []byte, err error) {
	// Generate ephemeral X25519 keypair
	// Note: curve25519.X25519() clamps internally, no manual clamping needed.
	var ephemeralPrivate [32]byte
	if _, err := rand.Read(ephemeralPrivate[:]); err != nil {
		return [32]byte{}, nil, fmt.Errorf("generating ephemeral key: %w", err)
	}

	// Compute ephemeral public key
	ephPub, err := curve25519X25519Base(ephemeralPrivate)
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("computing ephemeral public key: %w", err)
	}

	// ECDH: shared secret
	shared, err := DeriveSharedSecret(ephemeralPrivate, recipientPublicKey)
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// Derive encryption key via HKDF
	recipientFP := ComputeFingerprint(recipientPublicKey)
	hkdfReader := hkdf.New(sha256.New, shared[:], []byte("envsync-relay-v1"), []byte(recipientFP))

	var encKey [32]byte
	if _, err := io.ReadFull(hkdfReader, encKey[:]); err != nil {
		return [32]byte{}, nil, fmt.Errorf("HKDF for relay encryption: %w", err)
	}

	// Encrypt with derived key (no magic bytes for relay envelopes)
	aead, err := chacha20poly1305.NewX(encKey[:])
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("creating cipher: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return [32]byte{}, nil, fmt.Errorf("generating nonce: %w", err)
	}

	// Bind ephemeral public key as AAD to prevent key-substitution attacks
	ciphertext := aead.Seal(nil, nonce, plaintext, ephPub[:])

	// Assemble: nonce + ciphertext
	encrypted = make([]byte, 0, NonceSize+len(ciphertext))
	encrypted = append(encrypted, nonce...)
	encrypted = append(encrypted, ciphertext...)

	return ephPub, encrypted, nil
}

// DecryptFromSender decrypts data sent by a specific sender using ephemeral ECDH.
func DecryptFromSender(encrypted []byte, ephemeralPublicKey [32]byte, recipientPrivateKey [32]byte, recipientPublicKey [32]byte) ([]byte, error) {
	if len(encrypted) < NonceSize+chacha20poly1305.Overhead {
		return nil, errors.New("encrypted data too short")
	}

	// ECDH: shared secret using recipient's private key and sender's ephemeral public
	shared, err := DeriveSharedSecret(recipientPrivateKey, ephemeralPublicKey)
	if err != nil {
		return nil, fmt.Errorf("ECDH key agreement: %w", err)
	}

	// Derive decryption key via HKDF (must match sender's derivation)
	recipientFP := ComputeFingerprint(recipientPublicKey)
	hkdfReader := hkdf.New(sha256.New, shared[:], []byte("envsync-relay-v1"), []byte(recipientFP))

	var decKey [32]byte
	if _, err := io.ReadFull(hkdfReader, decKey[:]); err != nil {
		return nil, fmt.Errorf("HKDF for relay decryption: %w", err)
	}

	// Extract nonce and ciphertext
	nonce := encrypted[:NonceSize]
	ciphertext := encrypted[NonceSize:]

	aead, err := chacha20poly1305.NewX(decKey[:])
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	// AAD must match what was used during encryption (ephemeral public key)
	plaintext, err := aead.Open(nil, nonce, ciphertext, ephemeralPublicKey[:])
	if err != nil {
		return nil, errors.New("decryption failed: wrong key, corrupted data, or not intended for this recipient")
	}

	return plaintext, nil
}

// curve25519X25519Base computes the X25519 base point multiplication (public from private).
func curve25519X25519Base(privateKey [32]byte) ([32]byte, error) {
	pub, err := curve25519X25519(privateKey[:], curve25519Basepoint())
	if err != nil {
		return [32]byte{}, err
	}
	var result [32]byte
	copy(result[:], pub)
	return result, nil
}

func curve25519X25519(scalar, point []byte) ([]byte, error) {
	// Use golang.org/x/crypto/curve25519
	return curveX25519(scalar, point)
}

func curve25519Basepoint() []byte {
	// X25519 basepoint
	basepoint := [32]byte{9}
	return basepoint[:]
}
