// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package crypto

import (
	"encoding/binary"
	"fmt"
)

// Envelope is a relay-encrypted message for a specific recipient.
// Format: [4-byte payload length | 32-byte ephemeral public key | encrypted payload]
type Envelope struct {
	// EphemeralPublicKey is the sender's ephemeral X25519 key for ECDH.
	EphemeralPublicKey [32]byte

	// Payload is the encrypted data (nonce + ciphertext + tag).
	Payload []byte
}

// MarshalEnvelope serializes an envelope for relay transport.
func MarshalEnvelope(env *Envelope) []byte {
	payloadLen := len(env.Payload)
	buf := make([]byte, 4+32+payloadLen)
	binary.BigEndian.PutUint32(buf[0:4], uint32(payloadLen))
	copy(buf[4:36], env.EphemeralPublicKey[:])
	copy(buf[36:], env.Payload)
	return buf
}

// UnmarshalEnvelope deserializes an envelope from relay transport.
func UnmarshalEnvelope(data []byte) (*Envelope, error) {
	if len(data) < 36 {
		return nil, fmt.Errorf("envelope too short: %d bytes", len(data))
	}

	payloadLen := binary.BigEndian.Uint32(data[0:4])
	if int(payloadLen) != len(data)-36 {
		return nil, fmt.Errorf("envelope payload length mismatch: header says %d, actual %d", payloadLen, len(data)-36)
	}

	env := &Envelope{
		Payload: make([]byte, payloadLen),
	}
	copy(env.EphemeralPublicKey[:], data[4:36])
	copy(env.Payload, data[36:])
	return env, nil
}

// SealEnvelope encrypts plaintext for a recipient and returns a serialized envelope.
func SealEnvelope(plaintext []byte, recipientPublicKey [32]byte) ([]byte, error) {
	ephPub, encrypted, err := EncryptForRecipient(plaintext, recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("sealing envelope: %w", err)
	}

	env := &Envelope{
		EphemeralPublicKey: ephPub,
		Payload:            encrypted,
	}
	return MarshalEnvelope(env), nil
}

// OpenEnvelope decrypts a serialized envelope using the recipient's private key.
func OpenEnvelope(data []byte, recipientPrivateKey, recipientPublicKey [32]byte) ([]byte, error) {
	env, err := UnmarshalEnvelope(data)
	if err != nil {
		return nil, err
	}

	plaintext, err := DecryptFromSender(env.Payload, env.EphemeralPublicKey, recipientPrivateKey, recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("opening envelope: %w", err)
	}

	return plaintext, nil
}
