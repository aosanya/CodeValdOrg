package codevaldorg

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// generateToken produces a prefixed, high-entropy token and its SHA-256 hash.
// prefix should be one of "cv_ac_", "cv_at_", "cv_rt_", "cv_iv_".
func generateToken(prefix string) (plaintext, hashStr string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plaintext = prefix + base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(plaintext))
	hashStr = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

// hashSHA256 returns the base64url-encoded SHA-256 hash of s.
func hashSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// verifyPKCE checks that SHA-256(verifier) == challenge (S256 method).
func verifyPKCE(verifier, challenge string) bool {
	h := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// hashArgon2id hashes password using Argon2id and returns the PHC string.
func hashArgon2id(password string, t, m uint32, p uint8) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, t, m, p, 32)
	b64salt := base64.RawStdEncoding.EncodeToString(salt)
	b64hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", m, t, int(p), b64salt, b64hash), nil
}

// verifyArgon2id checks password against a PHC-format hash string.
func verifyArgon2id(password, phc string) (bool, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid PHC format")
	}
	var m, t, p uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, fmt.Errorf("invalid PHC params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid PHC salt: %w", err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid PHC hash: %w", err)
	}
	computed := argon2.IDKey([]byte(password), salt, t, m, uint8(p), uint32(len(expected)))
	return subtle.ConstantTimeCompare(computed, expected) == 1, nil
}
