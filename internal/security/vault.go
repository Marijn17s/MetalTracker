package security

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Vault struct {
	mu            sync.Mutex
	dir           string
	metaPath      string
	encDBPath     string
	plainDBPath   string
	dek           []byte
	unlocked      bool
	lastActivity  time.Time
	autoLockAfter time.Duration
}

type SetupResult struct {
	RecoveryKey string `json:"recoveryKey"`
}

func NewVault(dataDir string) (*Vault, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	return &Vault{
		dir:           dataDir,
		metaPath:      filepath.Join(dataDir, "vault.meta.json"),
		encDBPath:     filepath.Join(dataDir, "vault.db.enc"),
		plainDBPath:   filepath.Join(dataDir, "vault.session.db"),
		autoLockAfter: 15 * time.Minute,
	}, nil
}

func (vault *Vault) Exists() bool {
	_, err := os.Stat(vault.metaPath)
	return err == nil
}

func (vault *Vault) IsUnlocked() bool {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	return vault.unlocked
}

func (vault *Vault) PlainDBPath() string {
	return vault.plainDBPath
}

func (vault *Vault) Dir() string {
	return vault.dir
}

func (vault *Vault) AttachmentsDir() string {
	return filepath.Join(vault.dir, "attachments")
}

// EncryptPayload encrypts bytes with the unlocked vault DEK.
func (vault *Vault) EncryptPayload(plaintext []byte) ([]byte, error) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked || len(vault.dek) == 0 {
		return nil, ErrVaultLocked
	}
	return EncryptBytes(vault.dek, plaintext)
}

// DecryptPayload decrypts bytes with the unlocked vault DEK.
func (vault *Vault) DecryptPayload(payload []byte) ([]byte, error) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked || len(vault.dek) == 0 {
		return nil, ErrVaultLocked
	}
	return DecryptBytes(vault.dek, payload)
}

func (vault *Vault) SetAutoLockAfter(duration time.Duration) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	vault.autoLockAfter = duration
}

func (vault *Vault) TouchActivity() {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	vault.lastActivity = time.Now()
}

func (vault *Vault) CheckAutoLock() bool {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked || vault.autoLockAfter <= 0 {
		return false
	}
	if time.Since(vault.lastActivity) >= vault.autoLockAfter {
		vault.lockLocked()
		return true
	}
	return false
}

func (vault *Vault) Setup(pin string) (*SetupResult, error) {
	vault.mu.Lock()
	defer vault.mu.Unlock()

	if vault.Exists() {
		return nil, ErrVaultExists
	}
	if err := ValidatePIN(pin); err != nil {
		return nil, err
	}

	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}
	dek, err := GenerateDEK()
	if err != nil {
		return nil, err
	}
	recoveryEncoded, recoveryRaw, err := GenerateRecoveryKey()
	if err != nil {
		return nil, err
	}

	wrappedPIN, err := WrapDEKWithPIN(pin, salt, dek)
	if err != nil {
		return nil, err
	}
	wrappedRecovery, err := WrapDEKWithRecovery(recoveryRaw, dek)
	if err != nil {
		return nil, err
	}
	wrappedRecoveryKeyPIN, err := WrapRecoveryKeyWithPIN(pin, salt, recoveryRaw)
	if err != nil {
		return nil, err
	}

	meta := VaultMeta{
		Version:               1,
		Salt:                  base64.StdEncoding.EncodeToString(salt),
		WrappedDEKPIN:         wrappedPIN,
		WrappedDEKRecover:     wrappedRecovery,
		WrappedRecoveryKeyPIN: wrappedRecoveryKeyPIN,
		ArgonTime:             argon2Time,
		ArgonMemory:           argon2MemoryKiB,
		ArgonThreads:          argon2Threads,
	}
	if err := vault.writeMeta(meta); err != nil {
		return nil, err
	}

	// Plain DB is created by the storage layer after setup; Persist() encrypts it.
	_ = os.Remove(vault.plainDBPath)
	_ = os.Remove(vault.encDBPath)

	vault.dek = dek
	vault.unlocked = true
	vault.lastActivity = time.Now()

	return &SetupResult{RecoveryKey: recoveryEncoded}, nil
}

func (vault *Vault) Unlock(pin string) error {
	vault.mu.Lock()
	defer vault.mu.Unlock()

	if !vault.Exists() {
		return ErrVaultMissing
	}
	meta, err := vault.readMeta()
	if err != nil {
		return err
	}
	if meta.LockUntilUnix > time.Now().Unix() {
		return errors.New("too many failed attempts; try again later")
	}

	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return err
	}
	dek, err := UnwrapDEKWithPIN(pin, salt, meta.WrappedDEKPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads)
	if err != nil {
		meta.FailedAttempts++
		meta.LockUntilUnix = time.Now().Unix() + ProgressiveDelaySeconds(meta.FailedAttempts)
		_ = vault.writeMeta(meta)
		return ErrInvalidPIN
	}

	if err := vault.decryptSession(dek); err != nil {
		return err
	}

	meta.FailedAttempts = 0
	meta.LockUntilUnix = 0
	if err := vault.writeMeta(meta); err != nil {
		return err
	}

	vault.dek = dek
	vault.unlocked = true
	vault.lastActivity = time.Now()
	return nil
}

func (vault *Vault) Lock() error {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	return vault.lockLocked()
}

func (vault *Vault) lockLocked() error {
	if !vault.unlocked {
		return nil
	}
	if err := vault.encryptSession(); err != nil {
		return err
	}
	vault.wipePlainDB()
	for index := range vault.dek {
		vault.dek[index] = 0
	}
	vault.dek = nil
	vault.unlocked = false
	return nil
}

func (vault *Vault) Persist() error {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked {
		return ErrVaultLocked
	}
	vault.lastActivity = time.Now()
	return vault.encryptSession()
}

func (vault *Vault) RecoverWithKey(recoveryKey string, newPIN string) error {
	vault.mu.Lock()
	defer vault.mu.Unlock()

	if !vault.Exists() {
		return ErrVaultMissing
	}
	if err := ValidatePIN(newPIN); err != nil {
		return err
	}

	meta, err := vault.readMeta()
	if err != nil {
		return err
	}
	recoveryRaw, err := ParseRecoveryKey(recoveryKey)
	if err != nil {
		return err
	}
	dek, err := UnwrapDEKWithRecovery(recoveryRaw, meta.WrappedDEKRecover)
	if err != nil {
		return err
	}

	salt, err := GenerateSalt()
	if err != nil {
		return err
	}
	wrappedPIN, err := WrapDEKWithPIN(newPIN, salt, dek)
	if err != nil {
		return err
	}

	wrappedRecoveryKeyPIN, wrapErr := WrapRecoveryKeyWithPIN(newPIN, salt, recoveryRaw)
	if wrapErr != nil {
		return wrapErr
	}

	meta.Salt = base64.StdEncoding.EncodeToString(salt)
	meta.WrappedDEKPIN = wrappedPIN
	meta.WrappedRecoveryKeyPIN = wrappedRecoveryKeyPIN
	meta.ArgonTime = argon2Time
	meta.ArgonMemory = argon2MemoryKiB
	meta.ArgonThreads = argon2Threads
	meta.FailedAttempts = 0
	meta.LockUntilUnix = 0
	if err := vault.writeMeta(meta); err != nil {
		return err
	}

	if err := vault.decryptSession(dek); err != nil {
		return err
	}
	vault.dek = dek
	vault.unlocked = true
	vault.lastActivity = time.Now()
	return nil
}

func (vault *Vault) ChangePIN(currentPIN string, newPIN string) error {
	vault.mu.Lock()
	defer vault.mu.Unlock()

	if !vault.unlocked {
		return ErrVaultLocked
	}
	if err := ValidatePIN(newPIN); err != nil {
		return err
	}

	meta, err := vault.readMeta()
	if err != nil {
		return err
	}
	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return err
	}
	dek, err := UnwrapDEKWithPIN(currentPIN, salt, meta.WrappedDEKPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads)
	if err != nil {
		return ErrInvalidPIN
	}

	newSalt, err := GenerateSalt()
	if err != nil {
		return err
	}
	wrappedPIN, err := WrapDEKWithPIN(newPIN, newSalt, dek)
	if err != nil {
		return err
	}

	if meta.WrappedRecoveryKeyPIN != "" {
		recoveryRaw, unwrapErr := UnwrapRecoveryKeyWithPIN(
			currentPIN,
			salt,
			meta.WrappedRecoveryKeyPIN,
			meta.ArgonTime,
			meta.ArgonMemory,
			meta.ArgonThreads,
		)
		if unwrapErr != nil {
			return unwrapErr
		}
		wrappedRecoveryKeyPIN, wrapErr := WrapRecoveryKeyWithPIN(newPIN, newSalt, recoveryRaw)
		if wrapErr != nil {
			return wrapErr
		}
		meta.WrappedRecoveryKeyPIN = wrappedRecoveryKeyPIN
	}

	meta.Salt = base64.StdEncoding.EncodeToString(newSalt)
	meta.WrappedDEKPIN = wrappedPIN
	meta.ArgonTime = argon2Time
	meta.ArgonMemory = argon2MemoryKiB
	meta.ArgonThreads = argon2Threads
	return vault.writeMeta(meta)
}

func (vault *Vault) ExportRecoveryKey(pin string) (string, error) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked {
		return "", ErrVaultLocked
	}
	if err := ValidatePIN(pin); err != nil {
		return "", err
	}
	meta, err := vault.readMeta()
	if err != nil {
		return "", err
	}
	if meta.WrappedRecoveryKeyPIN == "" {
		return "", errors.New("recovery key is unavailable")
	}
	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return "", err
	}
	if _, err := UnwrapDEKWithPIN(pin, salt, meta.WrappedDEKPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads); err != nil {
		return "", ErrInvalidPIN
	}
	recoveryRaw, err := UnwrapRecoveryKeyWithPIN(pin, salt, meta.WrappedRecoveryKeyPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(recoveryRaw), nil
}

func (vault *Vault) ResolveRecoveryKey(pin string, recoveryKey string) ([]byte, error) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if strings.TrimSpace(recoveryKey) != "" {
		return ParseRecoveryKey(recoveryKey)
	}
	if !vault.unlocked {
		return nil, ErrVaultLocked
	}
	if err := ValidatePIN(pin); err != nil {
		return nil, err
	}
	meta, err := vault.readMeta()
	if err != nil {
		return nil, err
	}
	if meta.WrappedRecoveryKeyPIN == "" {
		return nil, errors.New("recovery key required")
	}
	salt, err := base64.StdEncoding.DecodeString(meta.Salt)
	if err != nil {
		return nil, err
	}
	if _, err := UnwrapDEKWithPIN(pin, salt, meta.WrappedDEKPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads); err != nil {
		return nil, ErrInvalidPIN
	}
	return UnwrapRecoveryKeyWithPIN(pin, salt, meta.WrappedRecoveryKeyPIN, meta.ArgonTime, meta.ArgonMemory, meta.ArgonThreads)
}

func (vault *Vault) MetaPath() string {
	return vault.metaPath
}

func (vault *Vault) EncDBPath() string {
	return vault.encDBPath
}

func (vault *Vault) EnsurePlainDBForSetup() error {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if !vault.unlocked {
		return ErrVaultLocked
	}
	_ = os.Remove(vault.plainDBPath)
	return nil
}

func (vault *Vault) decryptSession(dek []byte) error {
	encrypted, err := os.ReadFile(vault.encDBPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.Remove(vault.plainDBPath)
			return nil
		}
		return err
	}
	if len(encrypted) == 0 {
		_ = os.Remove(vault.plainDBPath)
		return nil
	}
	plain, err := DecryptBytes(dek, encrypted)
	if err != nil {
		return err
	}
	if len(plain) == 0 {
		_ = os.Remove(vault.plainDBPath)
		return nil
	}
	return os.WriteFile(vault.plainDBPath, plain, 0o600)
}

func (vault *Vault) encryptSession() error {
	if vault.dek == nil {
		return ErrVaultLocked
	}
	plain, err := os.ReadFile(vault.plainDBPath)
	if err != nil {
		if os.IsNotExist(err) {
			plain = []byte{}
		} else {
			return err
		}
	}
	encrypted, err := EncryptBytes(vault.dek, plain)
	if err != nil {
		return err
	}
	return os.WriteFile(vault.encDBPath, encrypted, 0o600)
}

func (vault *Vault) wipePlainDB() {
	_ = os.Remove(vault.plainDBPath)
}

func (vault *Vault) readMeta() (VaultMeta, error) {
	var meta VaultMeta
	data, err := os.ReadFile(vault.metaPath)
	if err != nil {
		return meta, err
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func (vault *Vault) writeMeta(meta VaultMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vault.metaPath, data, 0o600)
}
