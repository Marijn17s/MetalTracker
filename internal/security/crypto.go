package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

const (
	dekSize           = 32
	saltSize          = 16
	nonceSize         = 12
	recoveryKeyBytes  = 32
	argon2Time        = 3
	argon2MemoryKiB   = 64 * 1024
	argon2Threads     = 4
	argon2KeyLength   = 32
	wrapInfoPIN             = "metaltracker-pin-wrap-v1"
	wrapInfoRecovery        = "metaltracker-recovery-wrap-v1"
	wrapInfoRecoveryKeyPIN  = "metaltracker-recovery-key-pin-wrap-v1"
	wrapInfoBackup          = "metaltracker-backup-v1"
)

var (
	ErrInvalidPIN      = errors.New("invalid PIN")
	ErrInvalidRecovery = errors.New("invalid recovery key")
	ErrVaultLocked     = errors.New("vault is locked")
	ErrVaultExists     = errors.New("vault already exists")
	ErrVaultMissing    = errors.New("vault does not exist")
	ErrWeakPIN         = errors.New("PIN must be exactly 6 digits")
)

type VaultMeta struct {
	Version               int    `json:"version"`
	Salt                  string `json:"salt"`
	WrappedDEKPIN         string `json:"wrappedDekPin"`
	WrappedDEKRecover     string `json:"wrappedDekRecovery"`
	WrappedRecoveryKeyPIN string `json:"wrappedRecoveryKeyPin,omitempty"`
	ArgonTime             uint32 `json:"argonTime"`
	ArgonMemory           uint32 `json:"argonMemory"`
	ArgonThreads          uint8  `json:"argonThreads"`
	FailedAttempts        int    `json:"failedAttempts"`
	LockUntilUnix         int64  `json:"lockUntilUnix"`
}

func ValidatePIN(pin string) error {
	if len(pin) != 6 {
		return ErrWeakPIN
	}
	for _, character := range pin {
		if character < '0' || character > '9' {
			return ErrWeakPIN
		}
	}
	return nil
}

func GenerateDEK() ([]byte, error) {
	key := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func GenerateRecoveryKey() (string, []byte, error) {
	raw := make([]byte, recoveryKeyBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", nil, err
	}
	return base64.RawStdEncoding.EncodeToString(raw), raw, nil
}

func ParseRecoveryKey(encoded string) ([]byte, error) {
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, ErrInvalidRecovery
	}
	if len(raw) != recoveryKeyBytes {
		return nil, ErrInvalidRecovery
	}
	return raw, nil
}

func EncodeRecoveryKey(raw []byte) string {
	return base64.RawStdEncoding.EncodeToString(raw)
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func DerivePINKey(pin string, salt []byte, timeCost uint32, memoryKiB uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(pin), salt, timeCost, memoryKiB, threads, argon2KeyLength)
}

func wrapKey(wrappingKey []byte, dek []byte, info string) (string, error) {
	derived, err := deriveWrapKey(wrappingKey, info)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(derived)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, dek, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func unwrapKey(wrappingKey []byte, wrapped string, info string) ([]byte, error) {
	derived, err := deriveWrapKey(wrappingKey, info)
	if err != nil {
		return nil, err
	}
	payload, err := base64.StdEncoding.DecodeString(wrapped)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(derived)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, errors.New("wrapped key too short")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func deriveWrapKey(secret []byte, info string) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, nil, []byte(info))
	key := make([]byte, dekSize)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func WrapDEKWithPIN(pin string, salt []byte, dek []byte) (string, error) {
	pinKey := DerivePINKey(pin, salt, argon2Time, argon2MemoryKiB, argon2Threads)
	return wrapKey(pinKey, dek, wrapInfoPIN)
}

func UnwrapDEKWithPIN(pin string, salt []byte, wrapped string, timeCost uint32, memoryKiB uint32, threads uint8) ([]byte, error) {
	pinKey := DerivePINKey(pin, salt, timeCost, memoryKiB, threads)
	dek, err := unwrapKey(pinKey, wrapped, wrapInfoPIN)
	if err != nil {
		return nil, ErrInvalidPIN
	}
	return dek, nil
}

func WrapDEKWithRecovery(recoveryRaw []byte, dek []byte) (string, error) {
	return wrapKey(recoveryRaw, dek, wrapInfoRecovery)
}

func UnwrapDEKWithRecovery(recoveryRaw []byte, wrapped string) ([]byte, error) {
	dek, err := unwrapKey(recoveryRaw, wrapped, wrapInfoRecovery)
	if err != nil {
		return nil, ErrInvalidRecovery
	}
	return dek, nil
}

func WrapRecoveryKeyWithPIN(pin string, salt []byte, recoveryRaw []byte) (string, error) {
	pinKey := DerivePINKey(pin, salt, argon2Time, argon2MemoryKiB, argon2Threads)
	return wrapKey(pinKey, recoveryRaw, wrapInfoRecoveryKeyPIN)
}

func UnwrapRecoveryKeyWithPIN(
	pin string,
	salt []byte,
	wrapped string,
	timeCost uint32,
	memoryKiB uint32,
	threads uint8,
) ([]byte, error) {
	pinKey := DerivePINKey(pin, salt, timeCost, memoryKiB, threads)
	recoveryRaw, err := unwrapKey(pinKey, wrapped, wrapInfoRecoveryKeyPIN)
	if err != nil {
		return nil, ErrInvalidPIN
	}
	if len(recoveryRaw) != recoveryKeyBytes {
		return nil, ErrInvalidPIN
	}
	return recoveryRaw, nil
}

func DeriveBackupKey(recoveryRaw []byte, salt []byte) ([]byte, error) {
	reader := hkdf.New(sha256.New, recoveryRaw, salt, []byte(wrapInfoBackup))
	key := make([]byte, dekSize)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func EncryptBytes(dek []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptBytes(dek []byte, payload []byte) ([]byte, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func ProgressiveDelaySeconds(failedAttempts int) int64 {
	if failedAttempts <= 0 {
		return 0
	}
	// 1s, 2s, 4s, ... capped at 60s
	delay := int64(1) << uint(min(failedAttempts-1, 5))
	if delay > 60 {
		return 60
	}
	return delay
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
