package security

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	backupMagic         = "MTB1"
	BackupFormatVersion = 1
)

type BackupManifest struct {
	FormatVersion   int               `json:"formatVersion"`
	CreatedAt       string            `json:"createdAt"`
	UnitCount       int               `json:"unitCount"`
	AttachmentCount int               `json:"attachmentCount"`
	Checksums       map[string]string `json:"checksums"`
}

type BackupVerifyResult struct {
	Valid           bool   `json:"valid"`
	FormatVersion   int    `json:"formatVersion"`
	CreatedAt       string `json:"createdAt"`
	UnitCount       int    `json:"unitCount"`
	AttachmentCount int    `json:"attachmentCount"`
	FileCount       int    `json:"fileCount"`
	Message         string `json:"message"`
}

type BackupSource struct {
	MetaPath        string
	EncDBPath       string
	AttachmentsDir  string
	UnitCount       int
	AttachmentCount int
}

func CreateBackupFile(destinationPath string, recoveryRaw []byte, source BackupSource) (BackupManifest, error) {
	manifest, zipBytes, err := buildBackupZip(source)
	if err != nil {
		return BackupManifest{}, err
	}
	envelope, err := sealBackupEnvelope(recoveryRaw, zipBytes)
	if err != nil {
		return BackupManifest{}, err
	}
	if err := os.WriteFile(destinationPath, envelope, 0o600); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}

func VerifyBackupFile(backupPath string, recoveryRaw []byte) (BackupVerifyResult, error) {
	manifest, files, err := openBackupArchive(backupPath, recoveryRaw)
	if err != nil {
		return BackupVerifyResult{Valid: false, Message: err.Error()}, err
	}
	return BackupVerifyResult{
		Valid:           true,
		FormatVersion:   manifest.FormatVersion,
		CreatedAt:       manifest.CreatedAt,
		UnitCount:       manifest.UnitCount,
		AttachmentCount: manifest.AttachmentCount,
		FileCount:       len(files),
		Message:         "Backup verified successfully.",
	}, nil
}

func RestoreBackupFile(backupPath string, recoveryRaw []byte, vaultDir string) (BackupManifest, error) {
	manifest, files, err := openBackupArchive(backupPath, recoveryRaw)
	if err != nil {
		return BackupManifest{}, err
	}
	if _, ok := files["vault.meta.json"]; !ok {
		return BackupManifest{}, fmt.Errorf("backup missing vault.meta.json")
	}
	if _, ok := files["vault.db.enc"]; !ok {
		return BackupManifest{}, fmt.Errorf("backup missing vault.db.enc")
	}

	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return BackupManifest{}, err
	}

	// Remove previous vault payload (keep prices.db).
	_ = os.Remove(filepath.Join(vaultDir, "vault.meta.json"))
	_ = os.Remove(filepath.Join(vaultDir, "vault.db.enc"))
	_ = os.Remove(filepath.Join(vaultDir, "vault.session.db"))
	attachmentsDir := filepath.Join(vaultDir, "attachments")
	_ = os.RemoveAll(attachmentsDir)
	if err := os.MkdirAll(attachmentsDir, 0o700); err != nil {
		return BackupManifest{}, err
	}

	for relativePath, content := range files {
		if relativePath == "manifest.json" {
			continue
		}
		destination := filepath.Join(vaultDir, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return BackupManifest{}, err
		}
		if err := os.WriteFile(destination, content, 0o600); err != nil {
			return BackupManifest{}, err
		}
	}
	return manifest, nil
}

func BuildRecoveryKitHTML(recoveryKey string, createdAt string) string {
	escapedKey := htmlEscape(recoveryKey)
	escapedCreated := htmlEscape(createdAt)
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>MetalTracker recovery kit</title>
<style>
  body { font-family: Georgia, "Times New Roman", serif; max-width: 40rem; margin: 2rem auto; padding: 0 1.25rem; color: #1a1a1a; }
  h1 { font-size: 1.75rem; margin-bottom: 0.25rem; }
  .warn { border-left: 4px solid #b45309; padding: 0.75rem 1rem; background: #fffbeb; margin: 1.25rem 0; }
  .key { font-family: ui-monospace, Consolas, monospace; font-size: 1.05rem; word-break: break-all; padding: 1rem; border: 1px solid #d4d4d4; background: #fafafa; }
  ol { line-height: 1.5; }
  @media print { .noprint { display: none; } body { margin: 0; } }
</style>
</head>
<body>
  <h1>MetalTracker recovery kit</h1>
  <p>Created ` + escapedCreated + `</p>
  <div class="warn">
    <strong>Store offline.</strong> Anyone with this recovery key can reset your PIN and open your vault.
    Do not email it or store it in the same cloud folder as your backup file alone.
  </div>
  <h2>Recovery key</h2>
  <div class="key">` + escapedKey + `</div>
  <h2>Instructions</h2>
  <ol>
    <li>Keep this page (or a printed copy) in a safe place.</li>
    <li>On a new device: install MetalTracker -> use Forgot PIN / Restore with this key, or restore a <code>.mtbackup</code> file then unlock.</li>
    <li>After restore, verify holdings counts match your notes.</li>
    <li>Create a fresh backup from Settings after you confirm the migrate worked.</li>
  </ol>
  <p class="noprint"><button onclick="window.print()">Print</button></p>
</body>
</html>`
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(value)
}

func buildBackupZip(source BackupSource) (BackupManifest, []byte, error) {
	checksums := map[string]string{}
	files := map[string][]byte{}

	metaBytes, err := os.ReadFile(source.MetaPath)
	if err != nil {
		return BackupManifest{}, nil, fmt.Errorf("read vault meta: %w", err)
	}
	files["vault.meta.json"] = metaBytes
	checksums["vault.meta.json"] = sha256Hex(metaBytes)

	encBytes, err := os.ReadFile(source.EncDBPath)
	if err != nil {
		return BackupManifest{}, nil, fmt.Errorf("read encrypted vault db: %w", err)
	}
	files["vault.db.enc"] = encBytes
	checksums["vault.db.enc"] = sha256Hex(encBytes)

	attachmentCount := 0
	if source.AttachmentsDir != "" {
		entries, readErr := os.ReadDir(source.AttachmentsDir)
		if readErr != nil && !os.IsNotExist(readErr) {
			return BackupManifest{}, nil, readErr
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			content, fileErr := os.ReadFile(filepath.Join(source.AttachmentsDir, name))
			if fileErr != nil {
				return BackupManifest{}, nil, fileErr
			}
			relativePath := "attachments/" + name
			files[relativePath] = content
			checksums[relativePath] = sha256Hex(content)
			attachmentCount++
		}
	}
	if source.AttachmentCount > 0 {
		attachmentCount = source.AttachmentCount
	}

	manifest := BackupManifest{
		FormatVersion:   BackupFormatVersion,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		UnitCount:       source.UnitCount,
		AttachmentCount: attachmentCount,
		Checksums:       checksums,
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return BackupManifest{}, nil, err
	}
	files["manifest.json"] = manifestBytes

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, createErr := writer.Create(name)
		if createErr != nil {
			_ = writer.Close()
			return BackupManifest{}, nil, createErr
		}
		if _, writeErr := entry.Write(content); writeErr != nil {
			_ = writer.Close()
			return BackupManifest{}, nil, writeErr
		}
	}
	if err := writer.Close(); err != nil {
		return BackupManifest{}, nil, err
	}
	return manifest, buffer.Bytes(), nil
}

func sealBackupEnvelope(recoveryRaw []byte, zipBytes []byte) ([]byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}
	backupKey, err := DeriveBackupKey(recoveryRaw, salt)
	if err != nil {
		return nil, err
	}
	ciphertext, err := EncryptBytes(backupKey, zipBytes)
	if err != nil {
		return nil, err
	}
	envelope := make([]byte, 0, 4+4+len(salt)+len(ciphertext))
	envelope = append(envelope, []byte(backupMagic)...)
	versionBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(versionBytes, BackupFormatVersion)
	envelope = append(envelope, versionBytes...)
	envelope = append(envelope, salt...)
	envelope = append(envelope, ciphertext...)
	return envelope, nil
}

func openBackupArchive(backupPath string, recoveryRaw []byte) (BackupManifest, map[string][]byte, error) {
	envelope, err := os.ReadFile(backupPath)
	if err != nil {
		return BackupManifest{}, nil, err
	}
	if len(envelope) < 4+4+saltSize+nonceSize {
		return BackupManifest{}, nil, fmt.Errorf("backup file too short")
	}
	if string(envelope[:4]) != backupMagic {
		return BackupManifest{}, nil, fmt.Errorf("not a MetalTracker backup file")
	}
	formatVersion := binary.BigEndian.Uint32(envelope[4:8])
	if formatVersion != BackupFormatVersion {
		return BackupManifest{}, nil, fmt.Errorf("unsupported backup format version %d", formatVersion)
	}
	salt := envelope[8 : 8+saltSize]
	ciphertext := envelope[8+saltSize:]
	backupKey, err := DeriveBackupKey(recoveryRaw, salt)
	if err != nil {
		return BackupManifest{}, nil, err
	}
	zipBytes, err := DecryptBytes(backupKey, ciphertext)
	if err != nil {
		return BackupManifest{}, nil, ErrInvalidRecovery
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return BackupManifest{}, nil, fmt.Errorf("invalid backup archive: %w", err)
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if strings.Contains(name, "..") {
			return BackupManifest{}, nil, fmt.Errorf("invalid path in backup: %s", name)
		}
		opened, openErr := file.Open()
		if openErr != nil {
			return BackupManifest{}, nil, openErr
		}
		content, readErr := io.ReadAll(opened)
		_ = opened.Close()
		if readErr != nil {
			return BackupManifest{}, nil, readErr
		}
		files[name] = content
	}

	manifestBytes, ok := files["manifest.json"]
	if !ok {
		return BackupManifest{}, nil, fmt.Errorf("backup missing manifest.json")
	}
	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return BackupManifest{}, nil, fmt.Errorf("invalid manifest: %w", err)
	}
	for path, expected := range manifest.Checksums {
		content, exists := files[path]
		if !exists {
			return BackupManifest{}, nil, fmt.Errorf("missing file listed in manifest: %s", path)
		}
		if sha256Hex(content) != expected {
			return BackupManifest{}, nil, fmt.Errorf("checksum mismatch for %s", path)
		}
	}
	return manifest, files, nil
}

func sha256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
